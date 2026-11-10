//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestGormIdentityReader_GetIdentityState(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormIdentityReader(db)
	c := context.Background()
	a := createTestTenant(t, db, "idr-a-"+uuid.NewString())
	b := createTestTenant(t, db, "idr-b-"+uuid.NewString())
	now := time.Now().UTC()

	userA := createTestUser(t, db, a.ID, "idr-a+"+uuid.NewString()+"@example.com")
	userB := createTestUser(t, db, b.ID, "idr-b+"+uuid.NewString()+"@example.com")
	noRoles := createTestUser(t, db, a.ID, "idr-none+"+uuid.NewString()+"@example.com")

	mkRole := func(tenantID uuid.UUID, name string) *entity.Role {
		r := &entity.Role{ID: uuid.New(), TenantID: tenantID, Name: name, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, db.Create(r).Error)
		return r
	}
	assign := func(tenantID, userID, roleID uuid.UUID) {
		require.NoError(t, db.Create(&entity.UserRole{ID: uuid.New(), TenantID: tenantID, UserID: userID, RoleID: roleID, CreatedAt: now, UpdatedAt: now}).Error)
	}
	roleAdmin, roleEmp, roleGone := mkRole(a.ID, "admin"), mkRole(a.ID, "employee"), mkRole(a.ID, "retired")
	assign(a.ID, userA.ID, roleEmp.ID)
	assign(a.ID, userA.ID, roleAdmin.ID)
	assign(a.ID, userA.ID, roleGone.ID)
	require.NoError(t, db.Exec("UPDATE roles SET deleted_at = NOW() WHERE id = ?", roleGone.ID).Error)

	t.Run("healthy identity with current non-deleted roles, ordered", func(t *testing.T) {
		st, err := repo.GetIdentityState(c, a.ID, userA.ID)
		require.NoError(t, err)
		require.True(t, st.TenantActive)
		require.True(t, st.UserFound)
		require.True(t, st.UserActive)
		require.Equal(t, []string{"admin", "employee"}, st.RoleNames)
	})

	t.Run("user with zero roles still resolves", func(t *testing.T) {
		st, err := repo.GetIdentityState(c, a.ID, noRoles.ID)
		require.NoError(t, err)
		require.True(t, st.UserFound)
		require.Empty(t, st.RoleNames)
	})

	t.Run("user from tenant B with tenant A id is not found", func(t *testing.T) {
		st, err := repo.GetIdentityState(c, a.ID, userB.ID)
		require.NoError(t, err)
		require.False(t, st.UserFound)
		require.Empty(t, st.RoleNames)
	})

	t.Run("unknown tenant", func(t *testing.T) {
		_, err := repo.GetIdentityState(c, uuid.New(), userA.ID)
		require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	})

	t.Run("inactive user and inactive tenant are reported", func(t *testing.T) {
		require.NoError(t, db.Exec("UPDATE users SET is_active = FALSE WHERE id = ?", noRoles.ID).Error)
		st, err := repo.GetIdentityState(c, a.ID, noRoles.ID)
		require.NoError(t, err)
		require.True(t, st.UserFound)
		require.False(t, st.UserActive)

		require.NoError(t, db.Exec("UPDATE tenants SET is_active = FALSE WHERE id = ?", b.ID).Error)
		st, err = repo.GetIdentityState(c, b.ID, userB.ID)
		require.NoError(t, err)
		require.False(t, st.TenantActive)
	})

	t.Run("soft-deleted user is not found", func(t *testing.T) {
		require.NoError(t, db.Exec("UPDATE users SET deleted_at = NOW() WHERE id = ?", userA.ID).Error)
		st, err := repo.GetIdentityState(c, a.ID, userA.ID)
		require.NoError(t, err)
		require.False(t, st.UserFound)
	})

	t.Run("soft-deleted tenant is not found", func(t *testing.T) {
		require.NoError(t, db.Exec("UPDATE tenants SET deleted_at = NOW() WHERE id = ?", a.ID).Error)
		_, err := repo.GetIdentityState(c, a.ID, noRoles.ID)
		require.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	})
}
