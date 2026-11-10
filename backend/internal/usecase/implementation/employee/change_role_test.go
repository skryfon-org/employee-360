package employee

import (
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeRole(t *testing.T) {
	f := newFixture()
	uc := f.w.newChangeEmployeeRole()

	// admin cannot change roles at all.
	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, entity.RoleAdmin)
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)

	// super_admin grants admin.
	d, err := uc.Execute(bg, f.superActor(), f.employee.ID, entity.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, []string{"admin"}, d.Summary.Roles)
	assert.Contains(t, f.w.auditActions(), "employee.role_change")

	// revoke.
	d, err = uc.Execute(bg, f.superActor(), f.employee.ID, entity.RoleEmployee)
	require.NoError(t, err)
	assert.Equal(t, []string{"employee"}, d.Summary.Roles)

	// own role, super_admin untouchable, invalid role.
	_, err = uc.Execute(bg, f.superActor(), f.super.ID, entity.RoleEmployee)
	assert.ErrorIs(t, err, domainerrors.ErrCannotActOnSelf)
	super2 := f.w.user(f.tenant, "root2@acme.test", true, entity.RoleSuperAdmin)
	_, err = uc.Execute(bg, f.superActor(), super2.ID, entity.RoleEmployee)
	assert.ErrorIs(t, err, domainerrors.ErrRoleChangeNotAllowed)
	_, err = uc.Execute(bg, f.superActor(), f.employee.ID, entity.RoleSuperAdmin)
	assert.ErrorIs(t, err, domainerrors.ErrRoleChangeNotAllowed)
}

func TestChangeRole_LastAdminAndCrossTenant(t *testing.T) {
	f := newFixture()
	uc := f.w.newChangeEmployeeRole()
	_, err := uc.Execute(bg, f.superActor(), f.admin.ID, entity.RoleEmployee)
	assert.ErrorIs(t, err, domainerrors.ErrLastActiveAdmin)
	assert.Equal(t, []string{"admin"}, f.w.roleNamesOf(f.tenant, f.admin.ID))

	other := f.w.tenant("Other")
	os := f.w.user(other, "s@other.test", true, entity.RoleSuperAdmin)
	_, err = uc.Execute(bg, f.w.actor(os, entity.RoleSuperAdmin), f.employee.ID, entity.RoleAdmin)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
}

// ---- emergency contacts ----
