//go:build integration

package integration

import (
	"context"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	invimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/invitation"
	tenantimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/tenant"
)

// TestInviteVsDomainRemoval covers the invite / tenant-domain-removal race.
// Sequential behaviour is asserted exactly; the concurrent case holds the tenant
// row lock the way an in-flight removal does and proves the invite blocks on it
// and then re-checks the domain after the removal commits.
func TestInviteVsDomainRemoval(t *testing.T) {
	db, _, _ := setup(t)
	bg := context.Background()

	tenantID, adminID, roleID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Create(&entity.Tenant{ID: tenantID, Name: "invlock-" + uuid.NewString()[:8], IsActive: true}).Error)
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_roles WHERE tenant_id = ?", tenantID)
		db.Exec("DELETE FROM user_invitations WHERE tenant_id = ?", tenantID)
		db.Exec("DELETE FROM audit_logs WHERE tenant_id = ?", tenantID)
		db.Exec("DELETE FROM users WHERE tenant_id = ?", tenantID)
		db.Exec("DELETE FROM roles WHERE tenant_id = ?", tenantID)
		db.Exec("DELETE FROM tenant_domains WHERE tenant_id = ?", tenantID)
		db.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	})
	suffix := uuid.NewString()[:8]
	keep, gone, gone2 := "keep-"+suffix+".test", "gone-"+suffix+".test", "gone2-"+suffix+".test"
	domainIDs := map[string]uuid.UUID{}
	for _, d := range []string{keep, gone, gone2} {
		id := uuid.New()
		domainIDs[d] = id
		require.NoError(t, db.Exec("INSERT INTO tenant_domains (id, tenant_id, domain) VALUES (?, ?, ?)", id, tenantID, d).Error)
	}
	require.NoError(t, db.Create(&entity.Role{ID: roleID, TenantID: tenantID, Name: entity.RoleEmployee}).Error)
	require.NoError(t, db.Create(&entity.User{ID: adminID, TenantID: tenantID, FirstName: "A", LastName: "B",
		Email: "admin@" + keep, IsActive: true}).Error)

	transactor := database.NewGormTransactor(db)
	tenantRepo := persistence.NewGormTenantRepository(db)
	invite := invimpl.NewInviteUserUseCase(
		persistence.NewGormUserRepository(db), persistence.NewGormUserRoleRepository(db),
		persistence.NewGormRoleRepository(db), persistence.NewGormUserInvitationRepository(db),
		persistence.NewGormOrgReferenceRepository(db), persistence.NewGormTenantDomainRepository(db), tenantRepo,
		infraservice.NewAuditRecorder(persistence.NewGormAuditRepository(db)), infraservice.NewHashService(), stubPublisher{}, transactor,
		invimpl.AppURLs{Default: "http://frontend.test"})
	remove := tenantimpl.NewRemoveTenantDomainUseCase(tenantRepo, persistence.NewGormTenantDomainManager(db),
		infraservice.NewAuditRecorder(persistence.NewGormAuditRepository(db)), transactor)
	req := func(d string) invtypes.InviteUserRequest {
		return invtypes.InviteUserRequest{Email: "u-" + uuid.NewString()[:6] + "@" + d, RoleID: roleID}
	}

	// Removal after invite -> DOMAIN_IN_USE.
	_, err := invite.Execute(bg, emptypes.Actor{TenantID: tenantID, UserID: adminID, Roles: []string{"admin"}}, req(gone))
	require.NoError(t, err)
	require.ErrorIs(t, remove.Execute(bg, adminID, tenantID, domainIDs[gone]), domainerrors.ErrDomainInUse)

	// Invite after removal -> EMAIL_DOMAIN_NOT_ALLOWED.
	require.NoError(t, remove.Execute(bg, adminID, tenantID, domainIDs[gone2]))
	_, err = invite.Execute(bg, emptypes.Actor{TenantID: tenantID, UserID: adminID, Roles: []string{"admin"}}, req(gone2))
	require.ErrorIs(t, err, domainerrors.ErrEmailDomainNotAllowed)

	// Concurrent: an in-flight removal holds the tenant lock; the invite must
	// wait for it and then see the removed domain. Without the lock the invite
	// would validate the domain, create the user and commit.
	// A further domain with no users, soft-deleted inside the lock-holding tx.
	extra := "extra-" + suffix + ".test"
	extraID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO tenant_domains (id, tenant_id, domain) VALUES (?, ?, ?)", extraID, tenantID, extra).Error)

	tx := db.Begin()
	require.NoError(t, tx.Error)
	txCtx := database.WithTx(bg, tx)
	_, err = tenantRepo.LockByID(txCtx, tenantID)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, e := invite.Execute(bg, emptypes.Actor{TenantID: tenantID, UserID: adminID, Roles: []string{"admin"}}, req(extra))
		done <- e
	}()
	select {
	case e := <-done:
		_ = tx.Rollback()
		t.Fatalf("invite must block on the tenant lock, returned %v", e)
	case <-time.After(500 * time.Millisecond):
	}
	require.NoError(t, tx.Exec("UPDATE tenant_domains SET deleted_at = now() WHERE id = ?", extraID).Error)
	require.NoError(t, tx.Commit().Error)
	select {
	case e := <-done:
		require.ErrorIs(t, e, domainerrors.ErrEmailDomainNotAllowed)
	case <-time.After(10 * time.Second):
		t.Fatal("invite did not resume after the lock was released")
	}
	var n int64
	require.NoError(t, db.Raw("SELECT count(*) FROM users WHERE tenant_id = ? AND email LIKE ?", tenantID, "%@"+extra).Scan(&n).Error)
	require.EqualValues(t, 0, n)
}
