//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestLeaveApproverRepository_ActiveAdminsAndTenantScope(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormLeaveApproverRepository(e.db)
	now := time.Now().UTC()

	mkRole := func(tenant uuid.UUID, name string) uuid.UUID {
		r := &entity.Role{ID: uuid.New(), TenantID: tenant, Name: name, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, e.db.Create(r).Error)
		return r.ID
	}
	assign := func(tenant, user, role uuid.UUID) {
		require.NoError(t, e.db.Create(&entity.UserRole{ID: uuid.New(), TenantID: tenant, UserID: user, RoleID: role, CreatedAt: now, UpdatedAt: now}).Error)
	}
	adminRole, empRole := mkRole(e.tA.ID, "admin"), mkRole(e.tA.ID, "employee")
	adminB := mkRole(e.tB.ID, "admin")

	adm := createTestUser(t, e.db, e.tA.ID, "adm-"+uuid.NewString()+"@a.test")
	inactive := createTestUser(t, e.db, e.tA.ID, "ina-"+uuid.NewString()+"@a.test")
	plain := createTestUser(t, e.db, e.tA.ID, "pl-"+uuid.NewString()+"@a.test")
	assign(e.tA.ID, adm.ID, adminRole)
	assign(e.tA.ID, inactive.ID, adminRole)
	assign(e.tA.ID, plain.ID, empRole)
	assign(e.tB.ID, e.uB.ID, adminB)
	require.NoError(t, e.db.Exec("UPDATE users SET is_active = FALSE WHERE id = ?", inactive.ID).Error)

	admins, err := repo.ListActiveAdmins(c, e.tA.ID)
	require.NoError(t, err)
	require.Len(t, admins, 1)
	assert.Equal(t, adm.ID, admins[0].UserID)

	_, ok, err := repo.GetActiveUser(c, e.tA.ID, plain.ID)
	require.NoError(t, err)
	assert.True(t, ok)
	_, ok, _ = repo.GetActiveUser(c, e.tA.ID, inactive.ID)
	assert.False(t, ok, "inactive")
	_, ok, _ = repo.GetActiveUser(c, e.tB.ID, plain.ID)
	assert.False(t, ok, "other tenant")
}
