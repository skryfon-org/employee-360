package invitation

import (
	"context"
	"errors"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *fixture) invite() *InviteUserUseCaseImpl {
	return NewInviteUserUseCase(f.s, fakeUserRoleRepo{f.s}, fakeRoleRepo{f.s}, fakeInvRepo{f.s}, fakeOrgRepo{f.s}, fakeTenantDomainRepo{f.s}, fakeTenantRepo{f.s}, f.s.recorder(), f.hash, f.s, f.s, AppURLs{Default: "http://app/", Admin: "http://admin.app/"})
}
func (f *fixture) resend() *ResendInvitationUseCaseImpl {
	return NewResendInvitationUseCase(fakeRoleRepo{f.s}, fakeInvRepo{f.s}, f.s.recorder(), f.hash, f.s, f.s, AppURLs{Default: "http://app"})
}
func (f *fixture) revoke() *RevokeInvitationUseCaseImpl {
	return NewRevokeInvitationUseCase(fakeInvRepo{f.s}, f.s, fakeUserRoleRepo{f.s}, f.s.recorder(), f.s)
}
func (f *fixture) accept() *AcceptInvitationUseCaseImpl {
	return NewAcceptInvitationUseCase(f.s, fakeInvRepo{f.s}, fakeRoleRepo{f.s}, fakeTenantRepo{f.s}, fakeSettingsRepo{f.s}, f.hash, f.s, f.s, AppURLs{Default: "http://app/", Admin: "http://admin.app/"})
}
func (f *fixture) validate() *ValidateInvitationUseCaseImpl {
	return NewValidateInvitationUseCase(f.s, fakeInvRepo{f.s}, fakeRoleRepo{f.s}, f.hash)
}

func (f *fixture) doInvite(t *testing.T, email string) (*entity.UserInvitation, string) {
	t.Helper()
	inv, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: email, RoleID: f.employeeRl.ID})
	require.NoError(t, err)
	p := f.s.events[len(f.s.events)-1].Payload.(event.UserInvitedPayload)
	return inv, p.PlainToken
}

// AC1: tenant comes from the explicit param only; foreign-tenant roles are rejected.
func TestInvite_TenantFromParamOnly(t *testing.T) {
	f := newFixture()
	inv, _ := f.doInvite(t, "New@Acme.com")
	assert.Equal(t, f.tenantA, inv.TenantID)
	assert.Equal(t, "new@acme.com", inv.Email)
	u, err := f.s.GetByTenantAndEmail(bg, f.tenantA, "new@acme.com")
	require.NoError(t, err)
	assert.False(t, u.IsActive)
	assert.Nil(t, u.PasswordHash)
	assert.Equal(t, f.tenantA, u.TenantID)
	assert.Equal(t, f.tenantA, f.s.events[0].TenantID)

	// Admin of tenant B cannot use tenant A's role, so cannot target tenant A data.
	_, err = f.invite().Execute(bg, adminActor(f.tenantB, uuid.New()), invtypes.InviteUserRequest{Email: "x@b.com", RoleID: f.employeeRl.ID})
	assert.ErrorIs(t, err, domainerrors.ErrRoleNotFound)

	// A nil tenant is unauthorized (no fallback to payload).
	_, err = f.invite().Execute(bg, adminActor(uuid.Nil, f.adminID), invtypes.InviteUserRequest{Email: "x@b.com", RoleID: f.employeeRl.ID})
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

func TestInvite_RejectsSuperAdminRoleAndDuplicate(t *testing.T) {
	f := newFixture()
	sa := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "super_admin"}
	f.s.roles[sa.ID] = sa
	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "a@acme.com", RoleID: sa.ID})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidRole)

	f.doInvite(t, "dup@acme.com")
	du, _ := f.s.GetByTenantAndEmail(bg, f.tenantA, "dup@acme.com")
	du.IsActive = true
	_, err = f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "dup@acme.com", RoleID: f.employeeRl.ID})
	assert.ErrorIs(t, err, domainerrors.ErrEmailAlreadyExists)
}

func TestInvite_EmailDomainMustBelongToTenant(t *testing.T) {
	f := newFixture()
	req := func(email string) invtypes.InviteUserRequest {
		return invtypes.InviteUserRequest{Email: email, RoleID: f.employeeRl.ID}
	}
	// Allowed domain succeeds.
	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), req("ok@acme.com"))
	require.NoError(t, err)
	// Case-insensitive.
	_, err = f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), req("  Mixed@ACME.Com "))
	require.NoError(t, err)
	// Unregistered domain rejected; nothing persisted.
	_, err = f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), req("x@other.com"))
	assert.ErrorIs(t, err, domainerrors.ErrEmailDomainNotAllowed)
	// Domain registered to a different tenant rejected.
	_, err = f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), req("x@b.com"))
	assert.ErrorIs(t, err, domainerrors.ErrEmailDomainNotAllowed)
	assert.Len(t, f.s.invitations, 2)
}

// The tenant row lock must be taken before the domain check so the check
// serialises with tenant domain removal/update.
func TestInvite_LocksTenantBeforeDomainCheck(t *testing.T) {
	f := newFixture()
	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "l@acme.com", RoleID: f.employeeRl.ID})
	require.NoError(t, err)
	assert.Equal(t, []string{"lock", "domain_check"}, f.s.calls)

	// A rejected domain is still checked after the lock, and nothing persists.
	f.s.calls = nil
	_, err = f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "x@gone.com", RoleID: f.employeeRl.ID})
	assert.ErrorIs(t, err, domainerrors.ErrEmailDomainNotAllowed)
	assert.Equal(t, []string{"lock", "domain_check"}, f.s.calls)
	assert.Len(t, f.s.invitations, 1)
	assert.Len(t, f.s.users, 1)
}

func TestInvite_MissingTenantSurfacesNotFound(t *testing.T) {
	f := newFixture()
	delete(f.s.tenants, f.tenantA)
	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "m@acme.com", RoleID: f.employeeRl.ID})
	assert.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	assert.Zero(t, f.s.domainChecks)
	assert.Empty(t, f.s.users)
}

// AC2: only hashes stored.
func TestTokensStoredOnlyAsHashes(t *testing.T) {
	f := newFixture()
	inv, plain := f.doInvite(t, "h@acme.com")
	assert.NotEmpty(t, plain)
	assert.NotEqual(t, plain, inv.TokenHash)
	assert.Equal(t, f.hash.HashToken(plain), f.s.invitations[inv.ID].TokenHash)

	_, err := f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), inv.ID)
	require.NoError(t, err)
	p2 := f.s.events[len(f.s.events)-1].Payload.(event.InvitationResentPayload).PlainToken
	assert.NotEqual(t, plain, p2)
	assert.Equal(t, f.hash.HashToken(p2), f.s.invitations[inv.ID].TokenHash)
	assert.Contains(t, f.s.events[len(f.s.events)-1].Payload.(event.InvitationResentPayload).InviteURL, "http://app/accept-invitation?token="+p2)
}

// AC3: resend/revoke reject accepted or revoked; resend works while pending.
func TestResendRevoke_RejectNonPending(t *testing.T) {
	f := newFixture()
	accepted, tok := f.doInvite(t, "acc@acme.com")
	require.NoError(t, f.accept().Execute(context.Background(), invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"}))
	revoked, _ := f.doInvite(t, "rev@acme.com")
	require.NoError(t, f.revoke().Execute(bg, f.tenantA, f.adminID, revoked.ID))

	for _, id := range []uuid.UUID{accepted.ID, revoked.ID} {
		_, err := f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), id)
		assert.ErrorIs(t, err, domainerrors.ErrInvitationNotPending)
		assert.ErrorIs(t, f.revoke().Execute(bg, f.tenantA, f.adminID, id), domainerrors.ErrInvitationNotPending)
	}

	// Cross-tenant lookups look like not-found.
	assert.ErrorIs(t, f.revoke().Execute(bg, f.tenantB, uuid.New(), revoked.ID), domainerrors.ErrInvitationNotFound)
	_, err := f.resend().Execute(bg, adminActor(f.tenantB, uuid.New()), accepted.ID)
	assert.ErrorIs(t, err, domainerrors.ErrInvitationNotFound)
}

func TestResend_PendingPublishesEventAndInvalidatesOldToken(t *testing.T) {
	f := newFixture()
	inv, old := f.doInvite(t, "r@acme.com")
	_, err := f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), inv.ID)
	require.NoError(t, err)
	assert.Equal(t, event.EventTypeInvitationResent, f.s.events[len(f.s.events)-1].EventType)
	err = f.accept().Execute(context.Background(), invtypes.AcceptInvitationRequest{Token: old, Password: "password123"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)
}

func TestResend_ExpiredInvitationReissuesTokenAndExpiry(t *testing.T) {
	f := newFixture()
	inv, old := f.doInvite(t, "exp@acme.com")
	f.s.invitations[inv.ID].ExpiresAt = time.Now().Add(-time.Hour)
	oldHash := f.s.invitations[inv.ID].TokenHash
	// Expired: the old token is rejected before resend.
	require.ErrorIs(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: old, Password: "password123"}), domainerrors.ErrInvitationExpired)

	_, err := f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), inv.ID)
	require.NoError(t, err)

	stored := f.s.invitations[inv.ID]
	assert.NotEqual(t, oldHash, stored.TokenHash)
	assert.True(t, stored.ExpiresAt.After(time.Now()), "expiry must be in the future")

	last := f.s.events[len(f.s.events)-1]
	require.Equal(t, event.EventTypeInvitationResent, last.EventType)
	newTok := last.Payload.(event.InvitationResentPayload).PlainToken
	assert.NotEqual(t, old, newTok)

	// Old token no longer works; the new one does.
	assert.ErrorIs(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: old, Password: "password123"}), domainerrors.ErrInvalidToken)
	assert.NoError(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: newTok, Password: "password123"}))
}

// AC4: every role must set a password.
func TestAccept_RequiresPasswordForEveryRole(t *testing.T) {
	f := newFixture()
	adminRole := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "admin"}
	f.s.roles[adminRole.ID] = adminRole

	for _, role := range []*entity.Role{f.employeeRl, adminRole} {
		inv, err := f.invite().Execute(bg, superActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: role.Name + "@acme.com", RoleID: role.ID})
		require.NoError(t, err)
		tok := f.s.events[len(f.s.events)-1].Payload.(event.UserInvitedPayload).PlainToken

		for _, pw := range []string{"", "short", strings.Repeat("a", 73)} {
			err := f.accept().Execute(context.Background(), invtypes.AcceptInvitationRequest{Token: tok, Password: pw})
			assert.ErrorIs(t, err, domainerrors.ErrInvalidPassword, role.Name)
		}
		u, _ := f.s.GetByTenantAndEmail(context.Background(), f.tenantA, inv.Email)
		assert.False(t, u.IsActive)

		require.NoError(t, f.accept().Execute(context.Background(), invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"}))
		u, _ = f.s.GetByTenantAndEmail(context.Background(), f.tenantA, inv.Email)
		assert.True(t, u.IsActive)
		require.NotNil(t, u.PasswordHash)
		assert.NoError(t, f.hash.ComparePassword(*u.PasswordHash, "password123"))
		assert.NotNil(t, f.s.invitations[inv.ID].AcceptedAt)

		// Token is single-use.
		err = f.accept().Execute(context.Background(), invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"})
		assert.ErrorIs(t, err, domainerrors.ErrInvitationAccepted)
	}
}

func TestAccept_DistinctFailureStates(t *testing.T) {
	f := newFixture()
	bg := context.Background()
	pw := "password123"

	inv, tok := f.doInvite(t, "e@acme.com")
	f.s.invitations[inv.ID].ExpiresAt = time.Now().Add(-time.Minute)
	assert.ErrorIs(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: pw}), domainerrors.ErrInvitationExpired)
	assert.ErrorIs(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: "nope", Password: pw}), domainerrors.ErrInvalidToken)

	f.s.invitations[inv.ID].ExpiresAt = time.Now().Add(time.Hour)
	require.NoError(t, f.revoke().Execute(bg, f.tenantA, f.adminID, inv.ID))
	assert.ErrorIs(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: pw}), domainerrors.ErrInvitationRevoked)

	_, tok2 := f.doInvite(t, "a@acme.com")
	require.NoError(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok2, Password: pw}))
	assert.ErrorIs(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok2, Password: pw}), domainerrors.ErrInvitationAccepted)
}

// AC5: rollback leaves neither rows nor queued events; usecases depend only on the EventPublisher port.
func TestInviteAndResend_RollbackLeavesNothing(t *testing.T) {
	f := newFixture()
	f.s.publishErr = errors.New("enqueue failed")
	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "rb@acme.com", RoleID: f.employeeRl.ID})
	require.Error(t, err)
	assert.Empty(t, f.s.invitations)
	assert.Empty(t, f.s.users)
	assert.Empty(t, f.s.userRoles)
	assert.Empty(t, f.s.events)

	f.s.publishErr = nil
	inv, _ := f.doInvite(t, "rb2@acme.com")
	nEvents := len(f.s.events)
	f.s.publishErr = errors.New("enqueue failed")
	_, err = f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), inv.ID)
	require.Error(t, err)
	assert.Equal(t, nEvents, len(f.s.events))
}

func TestList_TenantScoped(t *testing.T) {
	f := newFixture()
	f.doInvite(t, "l1@acme.com")
	f.doInvite(t, "l2@acme.com")
	f.s.invitations[uuid.New()] = &entity.UserInvitation{ID: uuid.New(), TenantID: f.tenantB}
	uc := NewListInvitationsUseCase(fakeInvRepo{f.s})
	res, err := uc.Execute(bg, f.tenantA, invtypes.ListInvitationsQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)
	assert.Equal(t, 1, res.TotalPages)
	for _, i := range res.Items {
		assert.Equal(t, f.tenantA, i.TenantID)
	}
}

func TestInvite_MalformedEmailIsValidationError(t *testing.T) {
	f := newFixture()
	for _, e := range []string{"", "nope", "@x.com", "a@"} {
		_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: e, RoleID: f.employeeRl.ID})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidEmail, e)
	}
}

func TestInvite_DepartmentPositionMustBelongToTenant(t *testing.T) {
	f := newFixture()
	ownDept, ownPos := uuid.New(), uuid.New()
	foreignDept, foreignPos := uuid.New(), uuid.New()
	f.s.depts[ownDept], f.s.positions[ownPos] = f.tenantA, f.tenantA
	f.s.depts[foreignDept], f.s.positions[foreignPos] = f.tenantB, f.tenantB

	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "d@acme.com", RoleID: f.employeeRl.ID, DepartmentID: &foreignDept})
	assert.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	_, err = f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "p@acme.com", RoleID: f.employeeRl.ID, PositionID: &foreignPos})
	assert.ErrorIs(t, err, domainerrors.ErrPositionNotFound)
	assert.Empty(t, f.s.users)
	assert.Empty(t, f.s.invitations)
	assert.Empty(t, f.s.audits)

	inv, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "ok@acme.com", RoleID: f.employeeRl.ID, DepartmentID: &ownDept, PositionID: &ownPos})
	require.NoError(t, err)
	assert.Equal(t, &ownDept, inv.DepartmentID)
}

func TestInvite_RejectsInactiveDepartment(t *testing.T) {
	f := newFixture()
	dept := uuid.New()
	f.s.depts[dept] = f.tenantA
	f.s.inactiveDepts[dept] = true

	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "i@acme.com", RoleID: f.employeeRl.ID, DepartmentID: &dept})
	assert.ErrorIs(t, err, domainerrors.ErrDepartmentInactive)
	assert.Empty(t, f.s.users)
	assert.Empty(t, f.s.invitations)
	assert.Empty(t, f.s.audits)
}

func TestInvite_RejectsInactivePosition(t *testing.T) {
	f := newFixture()
	pos := uuid.New()
	f.s.positions[pos] = f.tenantA
	f.s.inactivePositions[pos] = true

	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "i@acme.com", RoleID: f.employeeRl.ID, PositionID: &pos})
	assert.ErrorIs(t, err, domainerrors.ErrPositionInactive)
	assert.Empty(t, f.s.users)
	assert.Empty(t, f.s.invitations)
	assert.Empty(t, f.s.audits)
}

func TestRevoke_RemovesPendingUserAndAllowsReinvite(t *testing.T) {
	f := newFixture()
	inv, _ := f.doInvite(t, "again@acme.com")
	require.NoError(t, f.revoke().Execute(bg, f.tenantA, f.adminID, inv.ID))
	assert.Empty(t, f.s.users)
	assert.Empty(t, f.s.userRoles)

	inv2, _ := f.doInvite(t, "again@acme.com")
	assert.NotEqual(t, inv.ID, inv2.ID)
}

func TestRevoke_KeepsActivatedUser(t *testing.T) {
	f := newFixture()
	inv, _ := f.doInvite(t, "act@acme.com")
	u, _ := f.s.GetByTenantAndEmail(bg, f.tenantA, "act@acme.com")
	u.IsActive = true // activated out-of-band while invitation remained pending
	require.NoError(t, f.revoke().Execute(bg, f.tenantA, f.adminID, inv.ID))
	assert.Len(t, f.s.users, 1)
}

// racyInvRepo simulates another request accepting the invitation between this
// request's lookup and its MarkAccepted.
type racyInvRepo struct{ fakeInvRepo }

func (r racyInvRepo) GetByTokenHash(c context.Context, h string) (*entity.UserInvitation, error) {
	inv, err := r.fakeInvRepo.GetByTokenHash(c, h)
	if err == nil {
		now := time.Now()
		r.s.invitations[inv.ID].AcceptedAt = &now
	}
	return inv, err
}

func TestAccept_AlreadyAcceptedBetweenCheckAndMark(t *testing.T) {
	f := newFixture()
	inv, tok := f.doInvite(t, "race@acme.com")
	uc := NewAcceptInvitationUseCase(f.s, racyInvRepo{fakeInvRepo{f.s}}, fakeRoleRepo{f.s}, fakeTenantRepo{f.s}, fakeSettingsRepo{f.s}, f.hash, f.s, f.s, AppURLs{})
	err := uc.Execute(context.Background(), invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"})
	assert.ErrorIs(t, err, domainerrors.ErrInvitationAccepted)
	u, _ := f.s.GetByTenantAndEmail(context.Background(), f.tenantA, inv.Email)
	assert.False(t, u.IsActive)
}

func TestAudit_InviteResendRevoke(t *testing.T) {
	f := newFixture()
	inv, _ := f.doInvite(t, "aud@acme.com")
	_, err := f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), inv.ID)
	require.NoError(t, err)
	require.NoError(t, f.revoke().Execute(bg, f.tenantA, f.adminID, inv.ID))

	require.Len(t, f.s.audits, 3)
	want := []string{domainaudit.ActionInvitationInvite, domainaudit.ActionInvitationResend, domainaudit.ActionInvitationRevoke}
	for i, a := range f.s.audits {
		assert.Equal(t, want[i], a.Action)
		assert.Equal(t, f.tenantA, a.TenantID)
		assert.Equal(t, inv.ID, a.EntityID)
		require.NotNil(t, a.ActorUserID)
		assert.Equal(t, f.adminID, *a.ActorUserID)
	}
}

func TestAudit_RolledBackWithTransaction(t *testing.T) {
	f := newFixture()
	f.s.publishErr = errors.New("enqueue failed")
	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "rb@acme.com", RoleID: f.employeeRl.ID})
	require.Error(t, err)
	assert.Empty(t, f.s.audits)
}

func TestValidate_TokenStatus(t *testing.T) {
	f := newFixture()
	bg := context.Background()
	inv, tok := f.doInvite(t, "val@acme.com")

	res, err := f.validate().Execute(bg, tok)
	require.NoError(t, err)
	assert.Equal(t, "val@acme.com", res.Email)

	_, err = f.validate().Execute(bg, "")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)
	_, err = f.validate().Execute(bg, "unknown-token")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)

	f.s.invitations[inv.ID].ExpiresAt = time.Now().Add(-time.Minute)
	_, err = f.validate().Execute(bg, tok)
	assert.ErrorIs(t, err, domainerrors.ErrInvitationExpired)

	f.s.invitations[inv.ID].ExpiresAt = time.Now().Add(time.Hour)
	require.NoError(t, f.revoke().Execute(bg, f.tenantA, f.adminID, inv.ID))
	_, err = f.validate().Execute(bg, tok)
	assert.ErrorIs(t, err, domainerrors.ErrInvitationRevoked)

	_, tok2 := f.doInvite(t, "done@acme.com")
	require.NoError(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok2, Password: "password123"}))
	_, err = f.validate().Execute(bg, tok2)
	assert.ErrorIs(t, err, domainerrors.ErrInvitationAccepted)
}

func TestValidate_ReturnsInviteeRole(t *testing.T) {
	f := newFixture()
	adminRole := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "admin"}
	f.s.roles[adminRole.ID] = adminRole

	_, empTok := f.doInvite(t, "emp@acme.com")
	res, err := f.validate().Execute(context.Background(), empTok)
	require.NoError(t, err)
	assert.Equal(t, "employee", res.Role)

	_, err = f.invite().Execute(bg, superActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "adm@acme.com", RoleID: adminRole.ID})
	require.NoError(t, err)
	admTok := f.s.events[len(f.s.events)-1].Payload.(event.UserInvitedPayload).PlainToken
	res, err = f.validate().Execute(context.Background(), admTok)
	require.NoError(t, err)
	assert.Equal(t, "admin", res.Role)
}

func TestInviteLink_TargetsRoleApp(t *testing.T) {
	f := newFixture()
	adminRole := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "admin"}
	f.s.roles[adminRole.ID] = adminRole

	_, err := f.invite().Execute(bg, superActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "adm@acme.com", RoleID: adminRole.ID})
	require.NoError(t, err)
	p := f.s.events[len(f.s.events)-1].Payload.(event.UserInvitedPayload)
	assert.Equal(t, "http://admin.app/accept-invitation?token="+p.PlainToken, p.InviteURL)

	// Employee has no override configured: falls back to the default URL.
	_, tok := f.doInvite(t, "emp@acme.com")
	p = f.s.events[len(f.s.events)-1].Payload.(event.UserInvitedPayload)
	assert.Equal(t, "http://app/accept-invitation?token="+tok, p.InviteURL)
}

func TestInvite_MultipleOrMisplacedAtRejectedBeforeAnyWork(t *testing.T) {
	f := newFixture()
	for _, e := range []string{"x@evil.com@acme.com", "a@@acme.com", "@acme.com", "a@"} {
		_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: e, RoleID: f.employeeRl.ID})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidEmail, e)
	}
	assert.Zero(t, f.s.domainChecks)
	assert.Empty(t, f.s.invitations)
	assert.Empty(t, f.s.events)
}

func activatedEvents(f *fixture) []event.Event {
	var out []event.Event
	for _, e := range f.s.events {
		if e.EventType == event.EventTypeUserActivated {
			out = append(out, e)
		}
	}
	return out
}

func TestAccept_PublishesUserActivated_DefaultMessage(t *testing.T) {
	f := newFixture()
	inv, tok := f.doInvite(t, "w@acme.com")
	require.NoError(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"}))

	evs := activatedEvents(f)
	require.Len(t, evs, 1)
	assert.Equal(t, f.tenantA, evs[0].TenantID, "tenant comes from the invitation")
	p := evs[0].Payload.(event.UserActivatedPayload)
	assert.Equal(t, f.tenantA, p.TenantID)
	assert.Equal(t, inv.Email, p.Email)
	assert.Equal(t, "Acme Corp", p.TenantName)
	assert.Equal(t, "employee", p.RoleName)
	assert.Equal(t, "http://app/login", p.LoginURL)
	assert.Empty(t, p.WelcomeMessage, "no settings row: built-in default is used downstream")
}

func TestAccept_PublishesUserActivated_CustomMessageAndAdminURL(t *testing.T) {
	f := newFixture()
	msg := "  Glad you joined!\nSee HR for your laptop.  "
	f.s.settings[f.tenantA] = &entity.OrganizationSettings{TenantID: f.tenantA, WelcomeMessage: &msg}
	// A message configured for another tenant must never leak.
	other := "other tenant"
	f.s.settings[f.tenantB] = &entity.OrganizationSettings{TenantID: f.tenantB, WelcomeMessage: &other}
	adminRole := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "admin"}
	f.s.roles[adminRole.ID] = adminRole
	_, err := f.invite().Execute(bg, superActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "adm2@acme.com", RoleID: adminRole.ID})
	require.NoError(t, err)
	tok := f.s.events[len(f.s.events)-1].Payload.(event.UserInvitedPayload).PlainToken
	require.NoError(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"}))

	evs := activatedEvents(f)
	require.Len(t, evs, 1)
	p := evs[0].Payload.(event.UserActivatedPayload)
	assert.Equal(t, "Glad you joined!\nSee HR for your laptop.", p.WelcomeMessage)
	assert.Equal(t, "http://admin.app/login", p.LoginURL)
}

func TestAccept_NoActivatedEventOnFailure(t *testing.T) {
	f := newFixture()
	_, tok := f.doInvite(t, "nf@acme.com")
	// Invalid token.
	assert.Error(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: "nope", Password: "password123"}))
	// Weak password.
	assert.Error(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: "x"}))
	assert.Empty(t, activatedEvents(f))

	// Used token: only the first accept publishes.
	require.NoError(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"}))
	assert.Error(t, f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"}))
	assert.Len(t, activatedEvents(f), 1)
}

func TestAccept_LostRacePublishesNothing(t *testing.T) {
	f := newFixture()
	_, tok := f.doInvite(t, "lost@acme.com")
	uc := NewAcceptInvitationUseCase(f.s, racyInvRepo{fakeInvRepo{f.s}}, fakeRoleRepo{f.s}, fakeTenantRepo{f.s}, fakeSettingsRepo{f.s}, f.hash, f.s, f.s, AppURLs{})
	assert.ErrorIs(t, uc.Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"}), domainerrors.ErrInvitationAccepted)
	assert.Empty(t, activatedEvents(f))
}

func TestAccept_PublishErrorRollsBackAccept(t *testing.T) {
	f := newFixture()
	inv, tok := f.doInvite(t, "pub@acme.com")
	f.s.publishErr = errors.New("outbox down")
	err := f.accept().Execute(bg, invtypes.AcceptInvitationRequest{Token: tok, Password: "password123"})
	require.Error(t, err)
	f.s.publishErr = nil
	assert.Nil(t, f.s.invitations[inv.ID].AcceptedAt, "accept must roll back")
	u, _ := f.s.GetByTenantAndEmail(bg, f.tenantA, inv.Email)
	assert.False(t, u.IsActive)
	assert.Empty(t, activatedEvents(f))
}

// F-1: only a super_admin may invite (or resend an invite for) the admin role.
func TestInvite_AdminRoleRequiresSuperAdmin(t *testing.T) {
	f := newFixture()
	adminRole := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "admin"}
	f.s.roles[adminRole.ID] = adminRole
	req := invtypes.InviteUserRequest{Email: "adm@acme.com", RoleID: adminRole.ID}

	_, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), req)
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
	_, err = f.invite().Execute(bg, emptypes.Actor{TenantID: f.tenantA, UserID: f.adminID}, req)
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
	_, err = f.s.GetByTenantAndEmail(bg, f.tenantA, "adm@acme.com")
	assert.Error(t, err, "no user may be created on a forbidden invite")

	// An admin can still invite employees.
	_, err = f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "emp@acme.com", RoleID: f.employeeRl.ID})
	require.NoError(t, err)

	// super_admin can invite admin.
	_, err = f.invite().Execute(bg, superActor(f.tenantA, f.adminID), req)
	require.NoError(t, err)
}

func TestResend_AdminRoleRequiresSuperAdmin(t *testing.T) {
	f := newFixture()
	adminRole := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "admin"}
	f.s.roles[adminRole.ID] = adminRole
	inv, err := f.invite().Execute(bg, superActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "adm@acme.com", RoleID: adminRole.ID})
	require.NoError(t, err)
	events := len(f.s.events)

	_, err = f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), inv.ID)
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
	assert.Len(t, f.s.events, events, "no event on a forbidden resend")

	_, err = f.resend().Execute(bg, superActor(f.tenantA, f.adminID), inv.ID)
	require.NoError(t, err)

	// Employee invitations remain resendable by an admin.
	empInv, err := f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{Email: "emp@acme.com", RoleID: f.employeeRl.ID})
	require.NoError(t, err)
	_, err = f.resend().Execute(bg, adminActor(f.tenantA, f.adminID), empInv.ID)
	require.NoError(t, err)
}

func (f *fixture) reinvite(t *testing.T, email, first string, role uuid.UUID) (*entity.UserInvitation, error) {
	return f.invite().Execute(bg, adminActor(f.tenantA, f.adminID), invtypes.InviteUserRequest{
		Email: email, FirstName: first, LastName: "New", RoleID: role,
	})
}

func TestInvite_ReinviteAfterExpiryReusesInactiveUser(t *testing.T) {
	f := newFixture()
	old, _ := f.doInvite(t, "exp@acme.com")
	f.s.invitations[old.ID].ExpiresAt = time.Now().Add(-time.Hour)
	u, _ := f.s.GetByTenantAndEmail(bg, f.tenantA, "exp@acme.com")
	userID := u.ID
	adminRole := &entity.Role{ID: uuid.New(), TenantID: f.tenantA, Name: "manager"}
	f.s.roles[adminRole.ID] = adminRole
	events := len(f.s.events)

	inv, err := f.reinvite(t, "exp@acme.com", "Renamed", adminRole.ID)
	require.NoError(t, err)
	assert.NotEqual(t, old.ID, inv.ID)
	assert.NotEqual(t, old.TokenHash, inv.TokenHash)
	assert.Len(t, f.s.users, 1)
	assert.Equal(t, userID, f.s.users[userID].ID)
	assert.Equal(t, "Renamed", f.s.users[userID].FirstName)
	assert.False(t, f.s.users[userID].IsActive)
	assert.NotNil(t, f.s.invitations[old.ID].RevokedAt)
	assert.Nil(t, f.s.invitations[inv.ID].RevokedAt)
	assert.Len(t, f.s.events, events+1)
	require.Len(t, f.s.userRoles, 1)
	assert.Equal(t, adminRole.ID, f.s.userRoles[0].RoleID)
}

func TestInvite_ReinvitePendingSupersedesOldInvitation(t *testing.T) {
	f := newFixture()
	old, _ := f.doInvite(t, "pend@acme.com")
	inv, err := f.reinvite(t, "pend@acme.com", "P", f.employeeRl.ID)
	require.NoError(t, err)
	assert.NotNil(t, f.s.invitations[old.ID].RevokedAt)
	assert.Len(t, f.s.users, 1)
	assert.NotEqual(t, old.ID, inv.ID)
}

func TestInvite_ReinviteConflicts(t *testing.T) {
	t.Run("active user", func(t *testing.T) {
		f := newFixture()
		f.doInvite(t, "a@acme.com")
		u, _ := f.s.GetByTenantAndEmail(bg, f.tenantA, "a@acme.com")
		u.IsActive = true
		_, err := f.reinvite(t, "a@acme.com", "X", f.employeeRl.ID)
		assert.ErrorIs(t, err, domainerrors.ErrEmailAlreadyExists)
	})
	t.Run("accepted invitation", func(t *testing.T) {
		f := newFixture()
		inv, _ := f.doInvite(t, "b@acme.com")
		now := time.Now()
		f.s.invitations[inv.ID].AcceptedAt = &now // user still inactive in store
		_, err := f.reinvite(t, "b@acme.com", "X", f.employeeRl.ID)
		assert.ErrorIs(t, err, domainerrors.ErrEmailAlreadyExists)
		assert.Len(t, f.s.invitations, 1)
	})
}

func TestInvite_ReinviteRollsBackOnPublishFailure(t *testing.T) {
	f := newFixture()
	old, _ := f.doInvite(t, "rb@acme.com")
	f.s.publishErr = errors.New("boom")
	_, err := f.reinvite(t, "rb@acme.com", "X", f.employeeRl.ID)
	require.Error(t, err)
	assert.Nil(t, f.s.invitations[old.ID].RevokedAt)
	assert.Len(t, f.s.invitations, 1)
}
