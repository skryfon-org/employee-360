package employee

import (
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContacts_SinglePrimary(t *testing.T) {
	f := newFixture()
	create, upd, del := f.w.newCreateEmergencyContact(), f.w.newUpdateEmergencyContact(), f.w.newDeleteEmergencyContact()
	list := f.w.newListEmergencyContacts()

	c1, err := create.Execute(bg, f.adminActor(), f.employee.ID, contact("A", false))
	require.NoError(t, err)
	assert.True(t, c1.IsPrimary, "first contact becomes primary")
	c2, err := create.Execute(bg, f.adminActor(), f.employee.ID, contact("B", true))
	require.NoError(t, err)
	assert.True(t, c2.IsPrimary)

	primaries := func() int {
		cs, _ := list.Execute(bg, f.tenant, f.employee.ID)
		n := 0
		for _, c := range cs {
			if c.IsPrimary {
				n++
			}
		}
		return n
	}
	assert.Equal(t, 1, primaries())

	_, err = upd.Execute(bg, f.adminActor(), f.employee.ID, c1.ID, contact("A2", true))
	require.NoError(t, err)
	assert.Equal(t, 1, primaries())

	require.NoError(t, del.Execute(bg, f.adminActor(), f.employee.ID, c1.ID))
	assert.Equal(t, 1, primaries(), "a primary remains after deleting the primary")
}

func TestContacts_ValidationTenantAndSuperAdminGuards(t *testing.T) {
	f := newFixture()
	create := f.w.newCreateEmergencyContact()
	_, err := create.Execute(bg, f.adminActor(), f.employee.ID, emptypes.EmergencyContactInput{Name: "", Relationship: "x", Phone: "12345"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData)
	bad := contact("A", false)
	bad.Phone = "zzz"
	_, err = create.Execute(bg, f.adminActor(), f.employee.ID, bad)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData)

	_, err = create.Execute(bg, f.adminActor(), f.super.ID, contact("A", false))
	assert.ErrorIs(t, err, domainerrors.ErrCannotActOnSuperAdmin)

	other := f.w.tenant("Other")
	oa := f.w.user(other, "a@other.test", true, entity.RoleAdmin)
	_, err = create.Execute(bg, f.w.actor(oa, entity.RoleAdmin), f.employee.ID, contact("A", false))
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)

	// Deleting a contact of another employee through the wrong user id is 404.
	c, err := create.Execute(bg, f.adminActor(), f.employee.ID, contact("A", false))
	require.NoError(t, err)
	e2 := f.w.user(f.tenant, "e2@acme.test", true, entity.RoleEmployee)
	err = f.w.newDeleteEmergencyContact().Execute(bg, f.adminActor(), e2.ID, c.ID)
	assert.ErrorIs(t, err, domainerrors.ErrEmergencyContactNotFound)
}

// ---- self service, first login and completion ----
