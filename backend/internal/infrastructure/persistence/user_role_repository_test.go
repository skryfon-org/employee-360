//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestGormUserRoleRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormUserRoleRepository(db)
	a := createTestTenant(t, db, "ur-a-"+uuid.NewString())
	b := createTestTenant(t, db, "ur-b-"+uuid.NewString())
	c := context.Background()

	now := time.Now().UTC()
	role := &entity.Role{ID: uuid.New(), TenantID: a.ID, Name: "r-" + uuid.NewString(), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(role).Error)
	user := createTestUser(t, db, a.ID, "ur+"+uuid.NewString()+"@example.com")
	ur := &entity.UserRole{ID: uuid.New(), TenantID: a.ID, UserID: user.ID, RoleID: role.ID, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repo.AssignRole(c, ur))

	// Happy path.
	roles, err := repo.GetRolesByUserID(c, a.ID, user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	require.Equal(t, role.ID, roles[0].ID)
	urs, err := repo.GetUserRolesByUserID(c, a.ID, user.ID)
	require.NoError(t, err)
	require.Len(t, urs, 1)

	for name, tid := range map[string]uuid.UUID{"tenantB": b.ID} {
		t.Run(name, func(t *testing.T) {
			roles, err := repo.GetRolesByUserID(c, tid, user.ID)
			require.NoError(t, err)
			require.Empty(t, roles)
			urs, err := repo.GetUserRolesByUserID(c, tid, user.ID)
			require.NoError(t, err)
			require.Empty(t, urs)
			require.NoError(t, repo.RemoveRole(c, tid, user.ID, role.ID))
			require.NoError(t, repo.DeleteByUserID(c, tid, user.ID))
		})
	}

	// Cross-tenant deletes were no-ops.
	urs, err = repo.GetUserRolesByUserID(c, a.ID, user.ID)
	require.NoError(t, err)
	require.Len(t, urs, 1)

	// Own-tenant removal works.
	require.NoError(t, repo.RemoveRole(c, a.ID, user.ID, role.ID))
	urs, err = repo.GetUserRolesByUserID(c, a.ID, user.ID)
	require.NoError(t, err)
	require.Empty(t, urs)

	require.NoError(t, repo.AssignRole(c, &entity.UserRole{ID: uuid.New(), TenantID: a.ID, UserID: user.ID, RoleID: role.ID, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, repo.DeleteByUserID(c, a.ID, user.ID))
	urs, err = repo.GetUserRolesByUserID(c, a.ID, user.ID)
	require.NoError(t, err)
	require.Empty(t, urs)
}
