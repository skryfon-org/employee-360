package seeder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seeder must reject missing credentials before touching the DB, so a
// nil *gorm.DB / hasher is safe here (a panic would mean validation ran late).
func TestRun_MissingPasswordErrors(t *testing.T) {
	_, err := New(nil, nil).Run(context.Background(), Options{SuperAdminEmail: "root@example.com"})
	require.ErrorIs(t, err, ErrSuperAdminPasswordRequired)
}

func TestRun_MissingEmailErrors(t *testing.T) {
	_, err := New(nil, nil).Run(context.Background(), Options{SuperAdminPassword: "pw"})
	require.ErrorIs(t, err, ErrSuperAdminEmailRequired)
}

func TestOptionsValidate_Defaults(t *testing.T) {
	o := Options{SuperAdminEmail: "  Root@Example.com ", SuperAdminPassword: "pw"}
	require.NoError(t, o.Validate())
	assert.Equal(t, DefaultSystemTenantName, o.SystemTenantName)
	assert.Equal(t, "root@example.com", o.SuperAdminEmail)
}

func TestOptionsValidate_EmailShape(t *testing.T) {
	for _, email := range []string{"a@@b.com", "a@b@c.com", "nodomain", "user@", "@example.com", "user@  "} {
		o := Options{SuperAdminEmail: email, SuperAdminPassword: "pw"}
		assert.ErrorIs(t, o.Validate(), ErrSuperAdminEmailInvalid, email)
	}
}

func TestOptionsValidate_RejectsPasswordPlaceholder(t *testing.T) {
	for _, pw := range []string{PasswordPlaceholder, "  " + PasswordPlaceholder + " "} {
		o := Options{SuperAdminEmail: "root@example.com", SuperAdminPassword: pw}
		assert.ErrorIs(t, o.Validate(), ErrSuperAdminPasswordPlaceholder, pw)
	}
}
