//go:build integration

package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type invFixture struct {
	db       *gorm.DB
	repo     *gormUserInvitationRepository
	tenantA  *entity.Tenant
	tenantB  *entity.Tenant
	roleA    uuid.UUID
	inviterA uuid.UUID
}

func newInvFixture(t *testing.T) *invFixture {
	t.Helper()
	db := setupTestDB(t)
	f := &invFixture{db: db, repo: NewGormUserInvitationRepository(db).(*gormUserInvitationRepository)}
	f.tenantA = createTestTenant(t, db, "inv-a-"+uuid.NewString())
	f.tenantB = createTestTenant(t, db, "inv-b-"+uuid.NewString())
	f.roleA = uuid.New()
	require.NoError(t, db.Create(&entity.Role{ID: f.roleA, TenantID: f.tenantA.ID, Name: "admin"}).Error)
	f.inviterA = createTestUser(t, db, f.tenantA.ID, "inviter+"+uuid.NewString()+"@example.com").ID
	return f
}

func (f *invFixture) newInv(createdAt time.Time) *entity.UserInvitation {
	return &entity.UserInvitation{
		ID: uuid.New(), TenantID: f.tenantA.ID, Email: "inv+" + uuid.NewString() + "@example.com",
		RoleID: f.roleA, InvitedBy: f.inviterA, TokenHash: "th-" + uuid.NewString(),
		ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func (f *invFixture) mustCreate(t *testing.T, createdAt time.Time) *entity.UserInvitation {
	t.Helper()
	inv := f.newInv(createdAt)
	require.NoError(t, f.repo.Create(context.Background(), inv))
	return inv
}

func TestGormUserInvitationRepository_TenantIsolation(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	inv := f.mustCreate(t, time.Now().UTC())

	got, err := f.repo.GetByID(c, f.tenantA.ID, inv.ID)
	require.NoError(t, err)
	require.Equal(t, inv.ID, got.ID)

	_, err = f.repo.GetByID(c, f.tenantB.ID, inv.ID)
	require.ErrorIs(t, err, domainerrors.ErrInvitationNotFound)

	list, total, err := f.repo.List(c, f.tenantB.ID, repository.InvitationListFilter{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, list)
	require.EqualValues(t, 0, total)

	require.ErrorIs(t, f.repo.MarkRevoked(c, f.tenantB.ID, inv.ID, uuid.New(), time.Now().UTC()), domainerrors.ErrInvitationNotPending)
	require.ErrorIs(t, f.repo.UpdateToken(c, f.tenantB.ID, inv.ID, uuid.New(), "x", time.Now().Add(time.Hour)), domainerrors.ErrInvitationNotPending)
	require.ErrorIs(t, f.repo.MarkAccepted(c, f.tenantB.ID, inv.ID, time.Now().UTC()), domainerrors.ErrInvitationNotPending)

	// Row untouched under its own tenant.
	got, err = f.repo.GetByID(c, f.tenantA.ID, inv.ID)
	require.NoError(t, err)
	require.Nil(t, got.RevokedAt)
	require.Nil(t, got.AcceptedAt)
	require.Equal(t, inv.TokenHash, got.TokenHash)
}

func TestGormUserInvitationRepository_ConditionalUpdates(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	now := time.Now().UTC()

	t.Run("second accept fails", func(t *testing.T) {
		inv := f.mustCreate(t, now)
		require.NoError(t, f.repo.MarkAccepted(c, f.tenantA.ID, inv.ID, now))
		require.ErrorIs(t, f.repo.MarkAccepted(c, f.tenantA.ID, inv.ID, now), domainerrors.ErrInvitationNotPending)
	})
	t.Run("revoke after accept fails", func(t *testing.T) {
		inv := f.mustCreate(t, now)
		require.NoError(t, f.repo.MarkAccepted(c, f.tenantA.ID, inv.ID, now))
		require.ErrorIs(t, f.repo.MarkRevoked(c, f.tenantA.ID, inv.ID, uuid.New(), now), domainerrors.ErrInvitationNotPending)
	})
	t.Run("update token after revoke fails", func(t *testing.T) {
		inv := f.mustCreate(t, now)
		require.NoError(t, f.repo.MarkRevoked(c, f.tenantA.ID, inv.ID, uuid.New(), now))
		require.ErrorIs(t, f.repo.UpdateToken(c, f.tenantA.ID, inv.ID, uuid.New(), "new", now.Add(time.Hour)), domainerrors.ErrInvitationNotPending)
		require.ErrorIs(t, f.repo.MarkAccepted(c, f.tenantA.ID, inv.ID, now), domainerrors.ErrInvitationNotPending)
	})
	t.Run("update token while pending", func(t *testing.T) {
		inv := f.mustCreate(t, now)
		exp := now.Add(48 * time.Hour)
		require.NoError(t, f.repo.UpdateToken(c, f.tenantA.ID, inv.ID, uuid.New(), "rotated-"+inv.ID.String(), exp))
		got, err := f.repo.GetByID(c, f.tenantA.ID, inv.ID)
		require.NoError(t, err)
		require.Equal(t, "rotated-"+inv.ID.String(), got.TokenHash)
		require.WithinDuration(t, exp, got.ExpiresAt, time.Second)
	})
	t.Run("unknown id", func(t *testing.T) {
		require.ErrorIs(t, f.repo.MarkRevoked(c, f.tenantA.ID, uuid.New(), uuid.New(), now), domainerrors.ErrInvitationNotPending)
	})
}

func TestGormUserInvitationRepository_GetByTokenHash(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	inv := f.mustCreate(t, time.Now().UTC())

	got, err := f.repo.GetByTokenHash(c, inv.TokenHash)
	require.NoError(t, err)
	require.Equal(t, inv.ID, got.ID)
	require.Equal(t, f.tenantA.ID, got.TenantID)

	_, err = f.repo.GetByTokenHash(c, "no-such-hash-"+uuid.NewString())
	require.ErrorIs(t, err, domainerrors.ErrInvitationNotFound)
}

func TestGormUserInvitationRepository_ListPagination(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	base := time.Now().UTC().Add(-time.Hour)
	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		ids = append(ids, f.mustCreate(t, base.Add(time.Duration(i)*time.Minute)).ID)
	}
	// A row in tenant B must not be counted.
	// (created via raw insert: role/inviter FKs belong to tenant A's rows, which is fine for FK purposes.)

	page1, total, err := f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Limit: 2})
	require.NoError(t, err)
	require.EqualValues(t, 5, total)
	require.Len(t, page1, 2)
	require.Equal(t, ids[4], page1[0].ID, "newest first")
	require.Equal(t, ids[3], page1[1].ID)

	page3, total, err := f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Limit: 2, Offset: 4})
	require.NoError(t, err)
	require.EqualValues(t, 5, total)
	require.Len(t, page3, 1)
	require.Equal(t, ids[0], page3[0].ID)
}

func TestGormUserInvitationRepository_RollbackLeavesNoRow(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	inv := f.newInv(time.Now().UTC())
	sentinel := errors.New("force rollback")

	err := database.NewGormTransactor(f.db).WithinTransaction(c, func(txCtx context.Context) error {
		require.NoError(t, f.repo.Create(txCtx, inv))
		got, err := f.repo.GetByID(txCtx, f.tenantA.ID, inv.ID)
		require.NoError(t, err, "visible inside the tx")
		require.Equal(t, inv.ID, got.ID)
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	_, err = f.repo.GetByID(c, f.tenantA.ID, inv.ID)
	require.ErrorIs(t, err, domainerrors.ErrInvitationNotFound)
}

func TestGormOrgReferenceRepository_TenantScoped(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormOrgReferenceRepository(db)
	c := context.Background()
	tA := createTestTenant(t, db, "org-a-"+uuid.NewString())
	tB := createTestTenant(t, db, "org-b-"+uuid.NewString())
	dept, pos := uuid.New(), uuid.New()
	require.NoError(t, db.Exec("INSERT INTO departments (id, tenant_id, name) VALUES (?, ?, 'Eng')", dept, tA.ID).Error)
	require.NoError(t, db.Exec("INSERT INTO positions (id, tenant_id, name) VALUES (?, ?, 'Dev')", pos, tA.ID).Error)

	ok, err := repo.DepartmentExists(c, tA.ID, dept)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.DepartmentExists(c, tB.ID, dept)
	require.NoError(t, err)
	require.False(t, ok, "foreign tenant must not see department")
	ok, err = repo.DepartmentExists(c, tA.ID, uuid.New())
	require.NoError(t, err)
	require.False(t, ok)

	found, _, err := repo.LockPositionShared(database.WithTx(c, db), tA.ID, pos)
	require.NoError(t, err)
	require.True(t, found)
	found, _, err = repo.LockPositionShared(database.WithTx(c, db), tB.ID, pos)
	require.NoError(t, err)
	require.False(t, found, "foreign tenant must not see position")
	found, _, err = repo.LockPositionShared(database.WithTx(c, db), tA.ID, dept)
	require.NoError(t, err)
	require.False(t, found, "a department id is not a position")

	inactivePos := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO positions (id, tenant_id, name, is_active) VALUES (?, ?, 'Old', FALSE)", inactivePos, tA.ID).Error)
	found, active, err := repo.LockPositionShared(database.WithTx(c, db), tA.ID, inactivePos)
	require.NoError(t, err)
	require.True(t, found)
	require.False(t, active)
	found, active, err = repo.LockPositionShared(database.WithTx(c, db), tA.ID, pos)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, active)
}

func TestGormUserInvitationRepository_ListFilterSearchJoins(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	now := time.Now().UTC()

	mk := func(email string, mut func(*entity.UserInvitation)) *entity.UserInvitation {
		inv := f.newInv(now.Add(-time.Minute))
		inv.Email = email
		if mut != nil {
			mut(inv)
		}
		require.NoError(t, f.repo.Create(c, inv))
		return inv
	}
	past := now.Add(-time.Hour)
	pending := mk("pending@x.com", nil)
	mk("expired@x.com", func(i *entity.UserInvitation) { i.ExpiresAt = past })
	mk("accepted@x.com", func(i *entity.UserInvitation) { i.AcceptedAt = &past })
	mk("revoked@x.com", func(i *entity.UserInvitation) { i.RevokedAt = &past })
	mk("50%_off@x.com", nil)
	mk("50xxoff@x.com", nil)

	for st, want := range map[entity.InvitationStatus]int{
		entity.InvitationStatusPending: 3, entity.InvitationStatusExpired: 1,
		entity.InvitationStatusAccepted: 1, entity.InvitationStatusRevoked: 1,
	} {
		_, total, err := f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Status: st, Now: now, Limit: 50})
		require.NoError(t, err)
		require.EqualValues(t, want, total, st)
	}

	items, total, err := f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Search: "PENDING@", Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, pending.ID, items[0].ID)
	require.Equal(t, "admin", items[0].RoleName)
	require.NotEmpty(t, items[0].InvitedByEmail)

	// LIKE wildcards are literal.
	_, total, err = f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Search: "50%_off", Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	_, total, err = f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Search: "%", Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)

	// Tenant B sees nothing of tenant A's, even with a matching search.
	_, total, err = f.repo.List(c, f.tenantB.ID, repository.InvitationListFilter{Search: "pending", Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
}

func TestGormUserInvitationRepository_ListHidesSoftDeletedJoinNames(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	now := time.Now().UTC()

	inv := f.mustCreate(t, now)
	items, _, err := f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "admin", items[0].RoleName)
	require.NotEmpty(t, items[0].InvitedByEmail)

	require.NoError(t, f.db.Exec("UPDATE roles SET deleted_at = NOW() WHERE id = ?", f.roleA).Error)
	require.NoError(t, f.db.Exec("UPDATE users SET deleted_at = NOW() WHERE id = ?", f.inviterA).Error)

	items, total, err := f.repo.List(c, f.tenantA.ID, repository.InvitationListFilter{Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total, "invitation must still be listed")
	require.Equal(t, inv.ID, items[0].ID)
	require.Empty(t, items[0].RoleName)
	require.Empty(t, items[0].InvitedByFirst)
	require.Empty(t, items[0].InvitedByLast)
	require.Empty(t, items[0].InvitedByEmail)
}

func TestGormUserInvitationRepository_AcceptedAndRevokeUnacceptedForEmail(t *testing.T) {
	f := newInvFixture(t)
	c := context.Background()
	email := "re+" + uuid.NewString() + "@example.com"
	mk := func(tenant uuid.UUID) *entity.UserInvitation {
		inv := f.newInv(time.Now().UTC())
		inv.Email, inv.TenantID = email, tenant
		if tenant == f.tenantB.ID {
			inv.RoleID = uuid.New()
			require.NoError(t, f.db.Create(&entity.Role{ID: inv.RoleID, TenantID: tenant, Name: "r"}).Error)
			inv.InvitedBy = createTestUser(t, f.db, tenant, "i+"+uuid.NewString()+"@example.com").ID
		}
		require.NoError(t, f.repo.Create(c, inv))
		return inv
	}
	pending, expired, other := mk(f.tenantA.ID), mk(f.tenantA.ID), mk(f.tenantB.ID)
	require.NoError(t, f.db.Model(&entity.UserInvitation{}).Where("id = ?", expired.ID).
		Update("expires_at", time.Now().UTC().Add(-time.Hour)).Error)

	ok, err := f.repo.HasAcceptedForEmail(c, f.tenantA.ID, email)
	require.NoError(t, err)
	require.False(t, ok)

	n, err := f.repo.RevokeUnacceptedForEmail(c, f.tenantA.ID, email, f.inviterA, time.Now().UTC())
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
	for _, id := range []uuid.UUID{pending.ID, expired.ID} {
		got, err := f.repo.GetByID(c, f.tenantA.ID, id)
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt)
	}
	// Other tenant's invitation for the same email is untouched.
	got, err := f.repo.GetByID(c, f.tenantB.ID, other.ID)
	require.NoError(t, err)
	require.Nil(t, got.RevokedAt)

	// Accepted detection is tenant-scoped.
	require.NoError(t, f.repo.MarkAccepted(c, f.tenantB.ID, other.ID, time.Now().UTC()))
	ok, err = f.repo.HasAcceptedForEmail(c, f.tenantA.ID, email)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = f.repo.HasAcceptedForEmail(c, f.tenantB.ID, email)
	require.NoError(t, err)
	require.True(t, ok)
}
