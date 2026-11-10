package employee

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMe_FirstLoginAndOrganization(t *testing.T) {
	f := newFixture()
	me := f.w.newGetMe()

	m, err := me.Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme", m.OrganizationName)
	assert.True(t, m.FirstLogin)
	assert.False(t, m.ProfileCompleted)
	assert.True(t, m.ProfileRequired)

	// No profile row is created by reading /me; first_login stays true until completed.
	m, err = me.Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.True(t, m.FirstLogin)
	assert.Empty(t, f.w.profiles, "GetMe must not create a profile row")

	am, err := me.Execute(bg, f.tenant, f.admin.ID)
	require.NoError(t, err)
	assert.False(t, am.ProfileRequired, "admins are not subject to profile completion")
	assert.False(t, am.FirstLogin, "admins never get first_login")

	sm, err := me.Execute(bg, f.tenant, f.super.ID)
	require.NoError(t, err)
	assert.False(t, sm.ProfileRequired, "super_admins are not subject to profile completion")
	assert.False(t, sm.FirstLogin, "super_admins never get first_login")

	_, err = me.Execute(bg, uuid.New(), f.employee.ID)
	assert.Error(t, err, "wrong tenant cannot resolve the user")
}
