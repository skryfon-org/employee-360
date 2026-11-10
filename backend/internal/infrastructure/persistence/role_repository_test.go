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

func TestGormRoleRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRoleRepository(db)
	a := createTestTenant(t, db, "role-a-"+uuid.NewString())
	b := createTestTenant(t, db, "role-b-"+uuid.NewString())
	c := context.Background()

	now := time.Now().UTC()
	role := &entity.Role{ID: uuid.New(), TenantID: a.ID, Name: "role-" + uuid.NewString(), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repo.Create(c, role))

	// Happy path in own tenant.
	got, err := repo.GetByID(c, a.ID, role.ID)
	require.NoError(t, err)
	require.Equal(t, role.Name, got.Name)
	got, err = repo.GetByName(c, a.ID, role.Name)
	require.NoError(t, err)
	require.Equal(t, role.ID, got.ID)
	list, err := repo.List(c, a.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// Other tenant IDs see nothing (nil tenant is rejected; see identity_test.go).
	for name, tid := range map[string]uuid.UUID{"tenantB": b.ID} {
		t.Run(name, func(t *testing.T) {
			_, err := repo.GetByID(c, tid, role.ID)
			require.ErrorIs(t, err, domainerrors.ErrRoleNotFound)
			_, err = repo.GetByName(c, tid, role.Name)
			require.ErrorIs(t, err, domainerrors.ErrRoleNotFound)
			l, err := repo.List(c, tid)
			require.NoError(t, err)
			require.Empty(t, l)
			upd := &entity.Role{ID: role.ID, Name: "hijacked", UpdatedAt: time.Now().UTC()}
			require.ErrorIs(t, repo.Update(c, tid, upd), domainerrors.ErrRoleNotFound)
			require.ErrorIs(t, repo.Delete(c, tid, role.ID), domainerrors.ErrRoleNotFound)
		})
	}

	// Row untouched.
	got, err = repo.GetByID(c, a.ID, role.ID)
	require.NoError(t, err)
	require.Equal(t, role.Name, got.Name)

	// Own-tenant update and delete work.
	role.Name = "renamed-" + uuid.NewString()
	role.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.Update(c, a.ID, role))
	got, err = repo.GetByID(c, a.ID, role.ID)
	require.NoError(t, err)
	require.Equal(t, role.Name, got.Name)
	require.NoError(t, repo.Delete(c, a.ID, role.ID))
	_, err = repo.GetByID(c, a.ID, role.ID)
	require.ErrorIs(t, err, domainerrors.ErrRoleNotFound)
}
