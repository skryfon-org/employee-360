//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

func TestGormPositionRepository_CRUDAndTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPositionRepository(db)

	tenantA := createTestTenant(t, db, "pos-tenant-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "pos-tenant-b-"+uuid.NewString())

	actorA := uuid.New()
	actorB := uuid.New()

	ctxA := context.Background()
	ctxB := context.Background()

	// 1. Create position in Tenant A
	posA := &entity.Position{
		Name:        "Software Engineer",
		Description: "Core engineering role",
	}
	err := repo.Create(ctxA, tenantA.ID, actorA, posA)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, posA.ID)
	require.Equal(t, tenantA.ID, posA.TenantID)
	require.Equal(t, &actorA, posA.CreatedBy)
	require.Equal(t, &actorA, posA.UpdatedBy)

	// 2. Tenant B can create a position with the same name (tenant-scoped uniqueness)
	posB := &entity.Position{
		Name:        "Software Engineer",
		Description: "Tenant B engineering",
	}
	err = repo.Create(ctxB, tenantB.ID, actorB, posB)
	require.NoError(t, err)
	require.NotEqual(t, posA.ID, posB.ID)

	// 3. Duplicate position name within the same tenant fails (case-insensitive check)
	posADup := &entity.Position{
		Name: "software engineer",
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, posADup)
	require.ErrorIs(t, err, domainerrors.ErrPositionNameTaken)

	// 4. ExistsByName check
	exists, err := repo.ExistsByName(ctxA, tenantA.ID, "Software Engineer")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = repo.ExistsByName(ctxA, tenantA.ID, "Product Manager")
	require.NoError(t, err)
	require.False(t, exists)

	// 5. Tenant B cannot read Tenant A's position
	_, err = repo.GetByID(ctxB, tenantB.ID, posA.ID)
	require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)

	// 6. Tenant A reads own position
	foundA, err := repo.GetByID(ctxA, tenantA.ID, posA.ID)
	require.NoError(t, err)
	require.Equal(t, "Software Engineer", foundA.Name)
	require.Equal(t, "Core engineering role", foundA.Description)

	// 7. Update position in Tenant A
	foundA.Name = "Senior Software Engineer"
	foundA.Description = "Updated description"
	err = repo.Update(ctxA, tenantA.ID, actorB, foundA)
	require.NoError(t, err)

	updatedA, err := repo.GetByID(ctxA, tenantA.ID, posA.ID)
	require.NoError(t, err)
	require.Equal(t, "Senior Software Engineer", updatedA.Name)
	require.Equal(t, "Updated description", updatedA.Description)
	require.Equal(t, &actorA, updatedA.CreatedBy)
	require.Equal(t, &actorB, updatedA.UpdatedBy)

	// 8. Tenant B cannot update Tenant A's position
	err = repo.Update(ctxB, tenantB.ID, actorB, posA)
	require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)

	// 9. List is tenant-scoped
	listA, totalA, err := repo.List(ctxA, tenantA.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), totalA)
	require.Len(t, listA, 1)
	require.Equal(t, posA.ID, listA[0].ID)

	listB, totalB, err := repo.List(ctxB, tenantB.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), totalB)
	require.Len(t, listB, 1)
	require.Equal(t, posB.ID, listB[0].ID)

	// 10. Tenant B cannot delete Tenant A's position
	err = repo.Delete(ctxB, tenantB.ID, posA.ID, actorB)
	require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)

	// 11. Delete in Tenant A succeeds (soft delete)
	err = repo.Delete(ctxA, tenantA.ID, posA.ID, actorA)
	require.NoError(t, err)

	_, err = repo.GetByID(ctxA, tenantA.ID, posA.ID)
	require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)

	// Verify the position row is soft-deleted (deleted_at and deleted_by populated)
	var rawPos entity.Position
	err = db.Table("positions").Where("id = ?", posA.ID).First(&rawPos).Error
	require.NoError(t, err)
	require.NotNil(t, rawPos.DeletedAt)
	require.Equal(t, &actorA, rawPos.DeletedBy)

	// 12. Soft-deleted position name can be reused in Tenant A (partial unique index test)
	reusedPos := &entity.Position{
		Name:        "Senior Software Engineer", // same name as soft-deleted posA
		Description: "Recreated position",
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, reusedPos)
	require.NoError(t, err)
	require.NotEqual(t, posA.ID, reusedPos.ID)

	// 13. Case-insensitive duplicate name within same tenant fails
	dupCasePos := &entity.Position{
		Name: "senior software engineer", // same lower name
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, dupCasePos)
	require.ErrorIs(t, err, domainerrors.ErrPositionNameTaken)
}

func TestGormPositionRepository_IsReferenced(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPositionRepository(db)

	tenant := createTestTenant(t, db, "pos-ref-tenant-"+uuid.NewString())
	actor := uuid.New()
	c := context.Background()

	pos := &entity.Position{
		Name: "Architect",
	}
	require.NoError(t, repo.Create(c, tenant.ID, actor, pos))

	// Initially not referenced
	referenced, err := repo.IsReferenced(c, tenant.ID, pos.ID)
	require.NoError(t, err)
	require.False(t, referenced)

	// Reference in users table
	user := &entity.User{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		PositionID: &pos.ID,
		Email:      "pos-user-" + uuid.NewString() + "@example.com",
		FirstName:  "John",
		LastName:   "Doe",
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	require.NoError(t, db.Create(user).Error)

	referenced, err = repo.IsReferenced(c, tenant.ID, pos.ID)
	require.NoError(t, err)
	require.True(t, referenced)

	// Remove user reference
	require.NoError(t, db.Delete(user).Error)

	referenced, err = repo.IsReferenced(c, tenant.ID, pos.ID)
	require.NoError(t, err)
	require.False(t, referenced)

	// Reference in user_invitations table
	role := &entity.Role{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		Name:      "role-" + uuid.NewString(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, db.Create(role).Error)

	inviter := &entity.User{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		Email:     "inviter-" + uuid.NewString() + "@example.com",
		FirstName: "Admin",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, db.Create(inviter).Error)

	inv := &entity.UserInvitation{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		Email:      "invited-" + uuid.NewString() + "@example.com",
		RoleID:     role.ID,
		PositionID: &pos.ID,
		InvitedBy:  inviter.ID,
		TokenHash:  "mock-token-hash-" + uuid.NewString(),
		ExpiresAt:  time.Now().UTC().Add(48 * time.Hour),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	require.NoError(t, db.Create(inv).Error)

	referenced, err = repo.IsReferenced(c, tenant.ID, pos.ID)
	require.NoError(t, err)
	require.True(t, referenced)
}

func TestGormPositionRepository_SoftDeletePreservesSoftDeletedUserReferences(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPositionRepository(db)

	tenant := createTestTenant(t, db, "pos-soft-pres-tenant-"+uuid.NewString())
	actor := uuid.New()
	c := context.Background()

	pos := &entity.Position{
		Name: "Director",
	}
	require.NoError(t, repo.Create(c, tenant.ID, actor, pos))

	now := time.Now().UTC()
	// Soft-deleted user referencing this position
	user := &entity.User{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		PositionID: &pos.ID,
		Email:      "soft-deleted-pos-user-" + uuid.NewString() + "@example.com",
		FirstName:  "Jane",
		LastName:   "Doe",
		IsActive:   false,
		DeletedAt:  &now,
		DeletedBy:  &actor,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, db.Create(user).Error)

	// Soft-deleted user does not block deletion because they are deleted_at IS NOT NULL
	referenced, err := repo.IsReferenced(c, tenant.ID, pos.ID)
	require.NoError(t, err)
	require.False(t, referenced)

	// Delete position (soft-delete)
	require.NoError(t, repo.Delete(c, tenant.ID, pos.ID, actor))

	// Verify user still references position_id (was NOT nulled out by FK cascade because it was not a hard delete)
	var reloadedUser entity.User
	require.NoError(t, db.Table("users").Where("id = ?", user.ID).First(&reloadedUser).Error)
	require.NotNil(t, reloadedUser.PositionID)
	require.Equal(t, pos.ID, *reloadedUser.PositionID)
}

func TestGormPositionRepository_IsReferenced_InvitationStates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPositionRepository(db)

	tenant := createTestTenant(t, db, "pos-inv-tenant-"+uuid.NewString())
	actor := uuid.New()
	c := context.Background()
	now := time.Now().UTC()

	role := &entity.Role{ID: uuid.New(), TenantID: tenant.ID, Name: "role-" + uuid.NewString(), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(role).Error)
	inviter := &entity.User{
		ID: uuid.New(), TenantID: tenant.ID, Email: "inviter-" + uuid.NewString() + "@example.com",
		FirstName: "Admin", LastName: "User", IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, db.Create(inviter).Error)

	past := now.Add(-time.Hour)
	cases := []struct {
		name     string
		mutate   func(*entity.UserInvitation)
		blocking bool
	}{
		{"pending blocks", func(i *entity.UserInvitation) {}, true},
		{"revoked does not block", func(i *entity.UserInvitation) { i.RevokedAt = &past }, false},
		{"expired does not block", func(i *entity.UserInvitation) { i.ExpiresAt = past }, false},
		{"accepted does not block", func(i *entity.UserInvitation) { i.AcceptedAt = &past }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := &entity.Position{Name: "Pos " + uuid.NewString()}
			require.NoError(t, repo.Create(c, tenant.ID, actor, pos))

			inv := &entity.UserInvitation{
				ID: uuid.New(), TenantID: tenant.ID, Email: "inv-" + uuid.NewString() + "@example.com",
				RoleID: role.ID, PositionID: &pos.ID, InvitedBy: inviter.ID,
				TokenHash: "hash-" + uuid.NewString(), ExpiresAt: now.Add(48 * time.Hour),
				CreatedAt: now, UpdatedAt: now,
			}
			tc.mutate(inv)
			require.NoError(t, db.Create(inv).Error)

			referenced, err := repo.IsReferenced(c, tenant.ID, pos.ID)
			require.NoError(t, err)
			require.Equal(t, tc.blocking, referenced)
		})
	}
}

func TestGormPositionRepository_RowLocks(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPositionRepository(db)

	tenant := createTestTenant(t, db, "pos-lock-tenant-"+uuid.NewString())
	other := createTestTenant(t, db, "pos-lock-other-"+uuid.NewString())
	actor := uuid.New()
	pos := &entity.Position{Name: "Locked"}
	require.NoError(t, repo.Create(context.Background(), tenant.ID, actor, pos))

	err := db.Transaction(func(tx *gorm.DB) error {
		c := database.WithTx(context.Background(), tx)

		got, err := repo.GetByIDForUpdate(c, tenant.ID, pos.ID)
		require.NoError(t, err)
		require.Equal(t, pos.ID, got.ID)

		// Cross-tenant lookups never see (or lock) the row.
		_, err = repo.GetByIDForUpdate(c, other.ID, pos.ID)
		require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)
		return nil
	})
	require.NoError(t, err)
}

func TestGormPositionRepository_IsActivePersistenceAndFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormPositionRepository(db)
	tenant := createTestTenant(t, db, "pos-active-a-"+uuid.NewString())
	other := createTestTenant(t, db, "pos-active-b-"+uuid.NewString())
	actor := uuid.New()
	c := context.Background()

	active := &entity.Position{Name: "Active", IsActive: true}
	inactive := &entity.Position{Name: "Inactive", IsActive: false}
	foreign := &entity.Position{Name: "Foreign Inactive", IsActive: false}
	require.NoError(t, repo.Create(c, tenant.ID, actor, active))
	require.NoError(t, repo.Create(c, tenant.ID, actor, inactive))
	require.NoError(t, repo.Create(c, other.ID, actor, foreign))

	got, err := repo.GetByID(c, tenant.ID, inactive.ID)
	require.NoError(t, err)
	require.False(t, got.IsActive)

	tr, fl := true, false
	all, total, err := repo.List(c, tenant.ID, nil, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, all, 2)

	onlyActive, total, err := repo.List(c, tenant.ID, &tr, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, active.ID, onlyActive[0].ID)

	onlyInactive, total, err := repo.List(c, tenant.ID, &fl, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, inactive.ID, onlyInactive[0].ID)

	// Toggle via Update.
	got.IsActive = true
	require.NoError(t, repo.Update(c, tenant.ID, actor, got))
	got, err = repo.GetByID(c, tenant.ID, inactive.ID)
	require.NoError(t, err)
	require.True(t, got.IsActive)
}
