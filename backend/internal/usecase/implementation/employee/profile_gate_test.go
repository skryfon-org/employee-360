package employee

import (
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileCompletion_FlipsAndGate(t *testing.T) {
	f := newFixture()
	upd := f.w.newUpdateMyProfile()
	gate := f.w.newProfileGate()
	create := f.w.newCreateEmergencyContact()
	del := f.w.newDeleteEmergencyContact()
	self := f.w.actor(f.employee, entity.RoleEmployee)

	ok, err := gate.Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.False(t, ok, "no profile row -> incomplete")

	m := fillAll(t, upd, f.tenant, f.employee.ID)
	assert.False(t, m.Completed)
	assert.Equal(t, []string{"emergency_contact"}, m.MissingFields)
	ok, _ = gate.Execute(bg, f.tenant, f.employee.ID)
	assert.False(t, ok, "mandatory fields without an emergency contact is still incomplete")

	c, err := create.Execute(bg, self, f.employee.ID, contact("Spouse", true))
	require.NoError(t, err)
	ok, _ = gate.Execute(bg, f.tenant, f.employee.ID)
	assert.True(t, ok, "flips to complete once fields + contact exist")
	me, _ := f.w.newGetMe().Execute(bg, f.tenant, f.employee.ID)
	assert.True(t, me.ProfileCompleted)
	assert.Contains(t, f.w.auditActions(), "employee.profile_self_update")

	// Removing the only contact makes it incomplete again.
	require.NoError(t, del.Execute(bg, self, f.employee.ID, c.ID))
	ok, _ = gate.Execute(bg, f.tenant, f.employee.ID)
	assert.False(t, ok)
}
