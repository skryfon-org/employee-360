//go:build integration

package integration

import (
	"context"
	"errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	invimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/invitation"
)

type stubPublisher struct{ err error }

func (p stubPublisher) Publish(context.Context, ...event.Event) error { return p.err }

// TestInviteUser_RollbackLeavesNothing proves AC-5 against a real database:
// when publishing fails after the user, user_role, invitation and audit rows
// were written, the real GormTransactor rolls all of them back.
func TestInviteUser_RollbackLeavesNothing(t *testing.T) {
	db, _, _ := setup(t)

	tenantID, adminID, roleID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Create(&entity.Tenant{ID: tenantID, Name: "invrb-" + uuid.NewString()[:8], IsActive: true}).Error)
	t.Cleanup(func() { db.Exec("DELETE FROM tenants WHERE id = ?", tenantID) })
	require.NoError(t, db.Exec("INSERT INTO tenant_domains (id, tenant_id, domain) VALUES (?, ?, ?)", uuid.New(), tenantID, "invrb.test").Error)
	require.NoError(t, db.Create(&entity.Role{ID: roleID, TenantID: tenantID, Name: entity.RoleEmployee}).Error)
	adminRole := uuid.New()
	require.NoError(t, db.Create(&entity.Role{ID: adminRole, TenantID: tenantID, Name: entity.RoleAdmin}).Error)
	require.NoError(t, db.Create(&entity.User{ID: adminID, TenantID: tenantID, FirstName: "A", LastName: "B",
		Email: "admin-" + uuid.NewString()[:8] + "@invrb.test", IsActive: true}).Error)

	build := func(pub stubPublisher) *invimpl.InviteUserUseCaseImpl {
		return invimpl.NewInviteUserUseCase(
			persistence.NewGormUserRepository(db), persistence.NewGormUserRoleRepository(db),
			persistence.NewGormRoleRepository(db), persistence.NewGormUserInvitationRepository(db),
			persistence.NewGormOrgReferenceRepository(db), persistence.NewGormTenantDomainRepository(db), persistence.NewGormTenantRepository(db), infraservice.NewAuditRecorder(persistence.NewGormAuditRepository(db)),
			infraservice.NewHashService(), pub, database.NewGormTransactor(db), invimpl.AppURLs{Default: "http://frontend.test"})
	}
	bg := context.Background()

	count := func(table, email string) int64 {
		var n int64
		switch table {
		case "users":
			require.NoError(t, db.Raw("SELECT count(*) FROM users WHERE tenant_id = ? AND email = ?", tenantID, email).Scan(&n).Error)
		case "user_invitations":
			require.NoError(t, db.Raw("SELECT count(*) FROM user_invitations WHERE tenant_id = ? AND email = ?", tenantID, email).Scan(&n).Error)
		case "user_roles":
			require.NoError(t, db.Raw("SELECT count(*) FROM user_roles ur JOIN users u ON u.id = ur.user_id WHERE ur.tenant_id = ? AND u.email = ?", tenantID, email).Scan(&n).Error)
		case "audit_logs":
			require.NoError(t, db.Raw("SELECT count(*) FROM audit_logs WHERE tenant_id = ? AND action = 'invitation.invite' AND metadata->>'email' = ?", tenantID, email).Scan(&n).Error)
		}
		return n
	}
	tables := []string{"users", "user_roles", "user_invitations", "audit_logs"}

	failEmail := "rollback-" + uuid.NewString()[:8] + "@invrb.test"
	_, err := build(stubPublisher{err: errors.New("publish failed")}).Execute(bg, emptypes.Actor{TenantID: tenantID, UserID: adminID, Roles: []string{"admin"}}, invtypes.InviteUserRequest{Email: failEmail, RoleID: roleID})
	require.Error(t, err)
	for _, tb := range tables {
		require.EqualValues(t, 0, count(tb, failEmail), "%s must be rolled back", tb)
	}

	// Control: the same setup commits when publishing succeeds.
	okEmail := "commit-" + uuid.NewString()[:8] + "@invrb.test"
	_, err = build(stubPublisher{}).Execute(bg, emptypes.Actor{TenantID: tenantID, UserID: adminID, Roles: []string{"admin"}}, invtypes.InviteUserRequest{Email: okEmail, RoleID: roleID})
	require.NoError(t, err)
	for _, tb := range tables {
		require.EqualValues(t, 1, count(tb, okEmail), "%s must be committed", tb)
	}
}
