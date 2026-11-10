package employee

import (
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivate_RevokesTokensAndBlocksSelfAndLastAdmin(t *testing.T) {
	f := newFixture()
	deact := f.w.newDeactivateEmployee()

	_, err := deact.Execute(bg, f.adminActor(), f.admin.ID)
	assert.ErrorIs(t, err, domainerrors.ErrCannotActOnSelf)

	d, err := deact.Execute(bg, f.adminActor(), f.employee.ID)
	require.NoError(t, err)
	assert.False(t, d.Summary.IsActive)
	assert.True(t, f.w.revoked[f.employee.ID], "refresh tokens revoked")
	assert.Contains(t, f.w.auditActions(), "employee.deactivate")

	// Admin cannot touch super_admin.
	_, err = deact.Execute(bg, f.adminActor(), f.super.ID)
	assert.ErrorIs(t, err, domainerrors.ErrCannotActOnSuperAdmin)

	// Last active admin: super_admin tries to deactivate the only admin.
	_, err = deact.Execute(bg, f.superActor(), f.admin.ID)
	assert.ErrorIs(t, err, domainerrors.ErrLastActiveAdmin)
	assert.True(t, f.w.users[f.admin.ID].IsActive)

	// With a second active admin it works.
	f.w.user(f.tenant, "admin2@acme.test", true, entity.RoleAdmin)
	_, err = deact.Execute(bg, f.superActor(), f.admin.ID)
	assert.NoError(t, err)
}

func TestDeactivate_CrossTenant404(t *testing.T) {
	f := newFixture()
	other := f.w.tenant("Other")
	oa := f.w.user(other, "a@other.test", true, entity.RoleAdmin)
	_, err := f.w.newDeactivateEmployee().Execute(bg, f.w.actor(oa, entity.RoleAdmin), f.employee.ID)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	assert.True(t, f.w.users[f.employee.ID].IsActive)
}
