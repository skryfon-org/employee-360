//go:build integration

package persistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

func withTestTx(c context.Context, tx *gorm.DB) context.Context { return database.WithTx(c, tx) }

func TestGormTenantRepository_RenameAndLock(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantRepository(db)
	c := context.Background()
	tn := createTestTenant(t, db, "mut-"+uuid.NewString())
	other := createTestTenant(t, db, "mut-other-"+uuid.NewString())
	editor := uuid.New()
	now := time.Now().UTC()

	got, err := repo.LockByID(c, tn.ID)
	require.NoError(t, err)
	require.Equal(t, tn.ID, got.ID)

	require.NoError(t, repo.UpdateName(c, tn.ID, "renamed", editor, now))
	got, err = repo.GetByID(c, tn.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", got.Name)
	require.Equal(t, editor, *got.UpdatedBy)
	untouched, err := repo.GetByID(c, other.ID)
	require.NoError(t, err)
	require.Equal(t, other.Name, untouched.Name, "rename only touches the addressed tenant")

	require.ErrorIs(t, repo.UpdateName(c, uuid.New(), "x", editor, now), domainerrors.ErrTenantNotFound)

	require.NoError(t, db.Exec("UPDATE tenants SET deleted_at = NOW() WHERE id = ?", tn.ID).Error)
	require.ErrorIs(t, repo.UpdateName(c, tn.ID, "x", editor, now), domainerrors.ErrTenantNotFound)
	_, err = repo.LockByID(c, tn.ID)
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
}

func newDomain(tid uuid.UUID, domain string, actor uuid.UUID) *entity.TenantDomain {
	now := time.Now().UTC()
	return &entity.TenantDomain{ID: uuid.New(), TenantID: tid, Domain: domain, CreatedAt: now, UpdatedAt: now, CreatedBy: &actor, UpdatedBy: &actor}
}

func TestGormTenantDomainManager_UniquenessAndSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainManager(db)
	c := context.Background()
	actor := uuid.New()
	a := createTestTenant(t, db, "dom-a-"+uuid.NewString())
	b := createTestTenant(t, db, "dom-b-"+uuid.NewString())
	dom := "uniq-" + uuid.NewString()[:8] + ".example.com"

	d1 := newDomain(a.ID, dom, actor)
	require.NoError(t, repo.Create(c, d1))

	// Live duplicate: same tenant and another tenant both conflict.
	require.ErrorIs(t, repo.Create(c, newDomain(a.ID, dom, actor)), domainerrors.ErrDomainAlreadyExists)
	require.ErrorIs(t, repo.Create(c, newDomain(b.ID, dom, actor)), domainerrors.ErrDomainAlreadyExists)

	d2 := newDomain(a.ID, "second-"+dom, actor)
	require.NoError(t, repo.Create(c, d2))
	n, err := repo.CountByTenantID(c, a.ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
	list, err := repo.ListByTenantID(c, a.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, actor, *list[0].CreatedBy)
	otherList, err := repo.ListByTenantID(c, b.ID)
	require.NoError(t, err)
	require.Empty(t, otherList)

	// GetByID is tenant-scoped.
	_, err = repo.GetByID(c, b.ID, d1.ID)
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)
	got, err := repo.GetByID(c, a.ID, d1.ID)
	require.NoError(t, err)
	require.Equal(t, dom, got.Domain)

	// Soft delete: tenant-scoped, sets deleted_at/deleted_by, hides from reads.
	remover := uuid.New()
	require.ErrorIs(t, repo.SoftDelete(c, b.ID, d1.ID, remover, time.Now().UTC()), domainerrors.ErrDomainNotFound)
	require.NoError(t, repo.SoftDelete(c, a.ID, d1.ID, remover, time.Now().UTC()))
	var row struct {
		DeletedAt *time.Time
		DeletedBy *uuid.UUID
	}
	require.NoError(t, db.Raw("SELECT deleted_at, deleted_by FROM tenant_domains WHERE id = ?", d1.ID).Scan(&row).Error)
	require.NotNil(t, row.DeletedAt)
	require.Equal(t, remover, *row.DeletedBy)
	_, err = repo.GetByID(c, a.ID, d1.ID)
	require.ErrorIs(t, err, domainerrors.ErrDomainNotFound)
	require.ErrorIs(t, repo.SoftDelete(c, a.ID, d1.ID, remover, time.Now().UTC()), domainerrors.ErrDomainNotFound, "second delete")
	n, _ = repo.CountByTenantID(c, a.ID)
	require.EqualValues(t, 1, n)

	// Login resolution no longer finds the removed domain.
	_, err = NewGormTenantDomainRepository(db).FindTenantByDomain(c, dom)
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)

	// A removed domain can be registered again (by another tenant): the soft-deleted row is revived.
	d3 := newDomain(b.ID, dom, actor)
	require.NoError(t, repo.Create(c, d3))
	require.Equal(t, d1.ID, d3.ID, "soft-deleted row is revived")
	list, err = repo.ListByTenantID(c, b.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, dom, list[0].Domain)
	require.Nil(t, list[0].DeletedAt)
	got2, err := NewGormTenantDomainRepository(db).FindTenantByDomain(c, dom)
	require.NoError(t, err)
	require.Equal(t, b.ID, got2.ID)
	require.ErrorIs(t, repo.Create(c, newDomain(a.ID, dom, actor)), domainerrors.ErrDomainAlreadyExists)
}

func TestGormTenantDomainManager_CreateInsideRolledBackTransaction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainManager(db)
	trepo := NewGormTenantRepository(db)
	c := context.Background()
	actor := uuid.New()
	dom := "rb-" + uuid.NewString()[:8] + ".example.com"

	tn := &entity.Tenant{ID: uuid.New(), Name: "rb-" + uuid.NewString(), IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	tx := db.Begin()
	require.NoError(t, tx.Error)
	// Reuse the production tx-in-context mechanism.
	txCtx := withTestTx(c, tx)
	require.NoError(t, tx.Exec("INSERT INTO tenants (id, name, is_active) VALUES (?, ?, TRUE)", tn.ID, tn.Name).Error)
	require.NoError(t, repo.Create(txCtx, newDomain(tn.ID, dom, actor)))
	require.NoError(t, tx.Rollback().Error)

	_, err := trepo.GetByID(c, tn.ID)
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	_, err = NewGormTenantDomainRepository(db).FindTenantByDomain(c, dom)
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
}

func TestGormTenantDomainManager_UpdateDomain(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainManager(db)
	c := context.Background()
	actor := uuid.New()
	a := createTestTenant(t, db, "upd-a-"+uuid.NewString())
	b := createTestTenant(t, db, "upd-b-"+uuid.NewString())
	sfx := uuid.NewString()[:8] + ".example.com"

	d1 := newDomain(a.ID, "one-"+sfx, actor)
	d2 := newDomain(a.ID, "two-"+sfx, actor)
	db1 := newDomain(b.ID, "bee-"+sfx, actor)
	for _, d := range []*entity.TenantDomain{d1, d2, db1} {
		require.NoError(t, repo.Create(c, d))
	}
	editor := uuid.New()
	at := time.Now().UTC().Add(time.Second)

	// Success: value, updated_by/updated_at change; tenant and id stay.
	require.NoError(t, repo.UpdateDomain(c, a.ID, d1.ID, "new-"+sfx, editor, at))
	got, err := repo.GetByID(c, a.ID, d1.ID)
	require.NoError(t, err)
	require.Equal(t, "new-"+sfx, got.Domain)
	require.Equal(t, a.ID, got.TenantID)
	require.Equal(t, editor, *got.UpdatedBy)
	tn, err := NewGormTenantDomainRepository(db).FindTenantByDomain(c, "new-"+sfx)
	require.NoError(t, err)
	require.Equal(t, a.ID, tn.ID)
	_, err = NewGormTenantDomainRepository(db).FindTenantByDomain(c, "one-"+sfx)
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)

	// Live conflicts: same tenant and another tenant.
	require.ErrorIs(t, repo.UpdateDomain(c, a.ID, d1.ID, d2.Domain, editor, at), domainerrors.ErrDomainAlreadyExists)
	require.ErrorIs(t, repo.UpdateDomain(c, a.ID, d1.ID, db1.Domain, editor, at), domainerrors.ErrDomainAlreadyExists)
	got, _ = repo.GetByID(c, a.ID, d1.ID)
	require.Equal(t, "new-"+sfx, got.Domain, "failed update leaves the value untouched")

	// Cross-tenant isolation: tenant A cannot touch tenant B's domain id.
	require.ErrorIs(t, repo.UpdateDomain(c, a.ID, db1.ID, "hijack-"+sfx, editor, at), domainerrors.ErrDomainNotFound)
	gotB, err := repo.GetByID(c, b.ID, db1.ID)
	require.NoError(t, err)
	require.Equal(t, "bee-"+sfx, gotB.Domain)

	// Soft-deleted row cannot be updated and is reported as not found.
	require.NoError(t, repo.SoftDelete(c, a.ID, d2.ID, editor, at))
	require.ErrorIs(t, repo.UpdateDomain(c, a.ID, d2.ID, "zzz-"+sfx, editor, at), domainerrors.ErrDomainNotFound)

	// A tombstone holding the target value is renamed to a placeholder, not
	// destroyed (UNIQUE covers soft-deleted rows).
	require.NoError(t, repo.UpdateDomain(c, a.ID, d1.ID, d2.Domain, editor, at))
	got, err = repo.GetByID(c, a.ID, d1.ID)
	require.NoError(t, err)
	require.Equal(t, d2.Domain, got.Domain)
	var tomb struct {
		Domain    string
		DeletedAt *time.Time
	}
	require.NoError(t, db.Raw("SELECT domain, deleted_at FROM tenant_domains WHERE id = ?", d2.ID).Scan(&tomb).Error)
	require.Equal(t, d2.Domain+"#deleted-"+d2.ID.String(), tomb.Domain)
	require.NotNil(t, tomb.DeletedAt, "tombstone row keeps its soft-delete marker")
	// The placeholder is not matched by lookups.
	ok, err := NewGormTenantDomainRepository(db).DomainBelongsToTenant(c, a.ID, tomb.Domain)
	require.NoError(t, err)
	require.False(t, ok)
	_, err = NewGormTenantDomainRepository(db).FindTenantByDomain(c, tomb.Domain)
	require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	// Create's revive path still works for a real tombstone value.
	require.NoError(t, repo.SoftDelete(c, a.ID, d1.ID, editor, at))
	revived := newDomain(b.ID, d2.Domain, actor)
	require.NoError(t, repo.Create(c, revived))
	require.Equal(t, d1.ID, revived.ID)
	_, err = repo.GetByID(c, b.ID, d1.ID)
	require.NoError(t, err)
	require.ErrorIs(t, repo.UpdateDomain(c, a.ID, uuid.New(), "q-"+sfx, editor, at), domainerrors.ErrDomainNotFound)
}

func TestGormTenantDomainManager_CrossTenantSoftDeleteIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainManager(db)
	c := context.Background()
	actor := uuid.New()
	a := createTestTenant(t, db, "iso-a-"+uuid.NewString())
	b := createTestTenant(t, db, "iso-b-"+uuid.NewString())
	sfx := uuid.NewString()[:8] + ".example.com"
	db1 := newDomain(b.ID, "bee1-"+sfx, actor)
	db2 := newDomain(b.ID, "bee2-"+sfx, actor)
	require.NoError(t, repo.Create(c, db1))
	require.NoError(t, repo.Create(c, db2))

	require.ErrorIs(t, repo.SoftDelete(c, a.ID, db1.ID, actor, time.Now().UTC()), domainerrors.ErrDomainNotFound)
	n, err := repo.CountByTenantID(c, b.ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
	_, err = repo.GetByID(c, b.ID, db1.ID)
	require.NoError(t, err)
}

func TestGormTenantDomainManager_RenameThenReAddOldValue(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainManager(db)
	c := context.Background()
	actor := uuid.New()
	a := createTestTenant(t, db, "ra-"+uuid.NewString())
	sfx := uuid.NewString()[:8] + ".example.com"
	d := newDomain(a.ID, "old-"+sfx, actor)
	require.NoError(t, repo.Create(c, d))
	require.NoError(t, repo.UpdateDomain(c, a.ID, d.ID, "fresh-"+sfx, actor, time.Now().UTC()))
	// The old value is free again.
	d2 := newDomain(a.ID, "old-"+sfx, actor)
	require.NoError(t, repo.Create(c, d2))
	require.NotEqual(t, d.ID, d2.ID)
}

func TestGormTenantDomainManager_CountUsersOnDomain(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormTenantDomainManager(db)
	c := context.Background()
	a := createTestTenant(t, db, "cu-a-"+uuid.NewString())
	b := createTestTenant(t, db, "cu-b-"+uuid.NewString())
	sfx := uuid.NewString()[:8]
	dom := "acme-" + sfx + ".com"

	count := func(tid uuid.UUID, d string) int64 {
		n, err := repo.CountUsersOnDomain(c, tid, d)
		require.NoError(t, err)
		return n
	}
	require.EqualValues(t, 0, count(a.ID, dom))

	u1 := createTestUser(t, db, a.ID, "one@"+dom)
	createTestUser(t, db, a.ID, "Two@"+strings.ToUpper(dom)) // case-insensitive
	inactive := createTestUser(t, db, a.ID, "three@"+dom)
	require.NoError(t, db.Exec("UPDATE users SET is_active = FALSE WHERE id = ?", inactive.ID).Error)
	createTestUser(t, db, a.ID, "x@evil"+dom)        // longer domain, suffix lookalike
	createTestUser(t, db, a.ID, "x@sub."+dom)        // subdomain
	createTestUser(t, db, a.ID, "a@"+dom+".evil.io") // prefix lookalike
	createTestUser(t, db, b.ID, "other@"+dom)        // other tenant

	require.EqualValues(t, 3, count(a.ID, dom), "active + inactive + mixed case, no lookalikes")
	require.EqualValues(t, 3, count(a.ID, strings.ToUpper(dom)))
	require.EqualValues(t, 1, count(b.ID, dom), "tenant scoped")
	require.EqualValues(t, 0, count(uuid.New(), dom))

	require.NoError(t, db.Exec("UPDATE users SET deleted_at = NOW() WHERE id = ?", u1.ID).Error)
	require.EqualValues(t, 2, count(a.ID, dom), "soft-deleted users are ignored")
}
