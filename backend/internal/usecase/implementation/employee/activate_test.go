package employee

import (
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivate(t *testing.T) {
	f := newFixture()
	act := f.w.newActivateEmployee()
	off := f.w.user(f.tenant, "off@acme.test", false, entity.RoleEmployee)
	d, err := act.Execute(bg, f.adminActor(), off.ID)
	require.NoError(t, err)
	assert.True(t, d.Summary.IsActive)

	pending := f.w.user(f.tenant, "pending@acme.test", false, entity.RoleEmployee)
	pending.PasswordHash = nil
	f.w.users[pending.ID] = pending
	_, err = act.Execute(bg, f.adminActor(), pending.ID)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotOnboarded)

	offSuper := f.w.user(f.tenant, "os@acme.test", false, entity.RoleSuperAdmin)
	_, err = act.Execute(bg, f.adminActor(), offSuper.ID)
	assert.ErrorIs(t, err, domainerrors.ErrCannotActOnSuperAdmin)
}

// ---- role change ----
