package invitation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
)

// InviteUserUseCaseImpl implements invusecase.InviteUserUseCase.
type InviteUserUseCaseImpl struct {
	userRepo       repository.UserRepository
	userRoleRepo   repository.UserRoleRepository
	roleRepo       repository.RoleRepository
	invitationRepo repository.UserInvitationRepository
	orgRefRepo     repository.OrgReferenceRepository
	tenantDomains  repository.TenantDomainRepository
	tenantRepo     repository.TenantRepository
	audit          service.AuditRecorder
	hashService    service.HashService
	eventPublisher service.EventPublisher
	transactor     ucshared.Transactor
	expiry         time.Duration
	appURLs        AppURLs
}

var _ invusecase.InviteUserUseCase = (*InviteUserUseCaseImpl)(nil)

// NewInviteUserUseCase constructs an InviteUserUseCaseImpl.
func NewInviteUserUseCase(
	userRepo repository.UserRepository,
	userRoleRepo repository.UserRoleRepository,
	roleRepo repository.RoleRepository,
	invitationRepo repository.UserInvitationRepository,
	orgRefRepo repository.OrgReferenceRepository,
	tenantDomains repository.TenantDomainRepository,
	tenantRepo repository.TenantRepository,
	audit service.AuditRecorder,
	hashService service.HashService,
	eventPublisher service.EventPublisher,
	transactor ucshared.Transactor,
	appURLs AppURLs,
) *InviteUserUseCaseImpl {
	return &InviteUserUseCaseImpl{
		userRepo: userRepo, userRoleRepo: userRoleRepo, roleRepo: roleRepo,
		invitationRepo: invitationRepo, orgRefRepo: orgRefRepo, tenantDomains: tenantDomains, tenantRepo: tenantRepo, audit: audit, hashService: hashService,
		eventPublisher: eventPublisher, transactor: transactor,
		expiry:  defaultInvitationExpiry,
		appURLs: appURLs,
	}
}

// Execute creates a pending user (or reuses an inactive, never-accepted one) + invitation and publishes UserInvited in one transaction.
func (u *InviteUserUseCaseImpl) Execute(c context.Context, actor emptypes.Actor, req invtypes.InviteUserRequest) (*entity.UserInvitation, error) {
	tenantID, inviterID := actor.TenantID, actor.UserID
	if err := requireIdentity(tenantID, inviterID); err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	at := strings.Index(email, "@")
	if strings.Count(email, "@") != 1 || at <= 0 || at == len(email)-1 {
		return nil, domainerrors.ErrInvalidEmail
	}
	// The role must belong to the caller's tenant; super_admin is never invitable.
	role, err := u.roleRepo.GetByID(c, tenantID, req.RoleID)
	if err != nil || role == nil || role.TenantID != tenantID {
		return nil, domainerrors.ErrRoleNotFound
	}
	if role.Name == roleSuperAdmin {
		return nil, domainerrors.ErrInvalidRole
	}
	if err := requireCanGrantRole(actor, role.Name); err != nil {
		return nil, err
	}

	// Department and position are validated (and share-locked) inside the
	// transaction below.
	plainToken, err := newPlainToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	inv := &entity.UserInvitation{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        email,
		RoleID:       role.ID,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		InvitedBy:    inviterID,
		CreatedBy:    &inviterID,
		UpdatedBy:    &inviterID,
		TokenHash:    u.hashService.HashToken(plainToken),
		ExpiresAt:    now.Add(u.expiry),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	user := &entity.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Email:        email,
		IsActive:     false,
		CreatedBy:    &inviterID,
		UpdatedBy:    &inviterID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	evt := event.Event{
		ID:            uuid.New(),
		TenantID:      tenantID,
		EventType:     event.EventTypeUserInvited,
		AggregateType: "user_invitation",
		AggregateID:   inv.ID.String(),
		Payload: event.UserInvitedPayload{
			InvitationID: inv.ID, TenantID: tenantID, Email: email,
			RoleName: role.Name, PlainToken: plainToken,
			InviteURL: u.appURLs.acceptLink(role.Name, plainToken),
			InvitedBy: inviterID, ExpiresAt: inv.ExpiresAt,
		},
		OccurredAt: now,
	}

	if err := u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		// Serialise with tenant domain removal/update, which lock the same row
		// before checking for users on the domain. The domain check must come
		// after the lock so it cannot validate a domain that is concurrently
		// being removed.
		if _, err := u.tenantRepo.LockByID(txCtx, tenantID); err != nil {
			return err
		}
		// The email's domain must be registered to the caller's tenant (never a client-supplied tenant).
		ok, err := u.tenantDomains.DomainBelongsToTenant(txCtx, tenantID, email[at+1:])
		if err != nil {
			return err
		}
		if !ok {
			return domainerrors.ErrEmailDomainNotAllowed
		}
		// Department must belong to the caller's tenant (FKs only check existence).
		// The shared lock serialises with department deletion, which locks the
		// row FOR UPDATE before checking references.
		if req.DepartmentID != nil {
			found, active, err := u.orgRefRepo.LockDepartmentShared(txCtx, tenantID, *req.DepartmentID)
			if err != nil {
				return err
			}
			if !found {
				return domainerrors.ErrDepartmentNotFound
			}
			if !active {
				return domainerrors.ErrDepartmentInactive
			}
		}
		// Same for the position: must belong to the tenant and be active.
		if req.PositionID != nil {
			found, active, err := u.orgRefRepo.LockPositionShared(txCtx, tenantID, *req.PositionID)
			if err != nil {
				return err
			}
			if !found {
				return domainerrors.ErrPositionNotFound
			}
			if !active {
				return domainerrors.ErrPositionInactive
			}
		}
		// Existing user: the tenant row lock above serialises concurrent
		// invites, so this lookup cannot race another invite for the email.
		existing, err := u.userRepo.GetByTenantAndEmail(txCtx, tenantID, email)
		if err != nil && !errors.Is(err, domainerrors.ErrNotFound) && !errors.Is(err, domainerrors.ErrUserNotFound) {
			return err
		}
		reinvite := false
		if existing != nil {
			// The password / verified-email / last-login checks are deliberate:
			// a re-invite replaces the user's role and profile, so any sign the
			// account was ever really used means it must not be hijacked by a
			// new invitation, even if IsActive was later flipped off.
			// Only a never-activated pending user (e.g. whose invitation expired)
			// may be re-invited; a real account or one with an accepted
			// invitation is a conflict.
			if existing.IsActive || existing.PasswordHash != nil || existing.EmailVerifiedAt != nil || existing.LastLoginAt != nil {
				return domainerrors.ErrEmailAlreadyExists
			}
			accepted, err := u.invitationRepo.HasAcceptedForEmail(txCtx, tenantID, email)
			if err != nil {
				return err
			}
			if accepted {
				return domainerrors.ErrEmailAlreadyExists
			}
			reinvite = true
			// Supersede older unaccepted invitations so only the new token works.
			if _, err := u.invitationRepo.RevokeUnacceptedForEmail(txCtx, tenantID, email, inviterID, now); err != nil {
				return err
			}
			existing.DepartmentID = req.DepartmentID
			existing.PositionID = req.PositionID
			existing.FirstName = user.FirstName
			existing.LastName = user.LastName
			existing.UpdatedBy = &inviterID
			existing.UpdatedAt = now
			if err := u.userRepo.Update(txCtx, tenantID, existing); err != nil {
				return err
			}
			user = existing
			// Replace the role assignment with the one in this request.
			if err := u.userRoleRepo.DeleteByUserID(txCtx, tenantID, user.ID); err != nil {
				return err
			}
		} else if err := u.userRepo.Create(txCtx, user); err != nil {
			return err
		}
		if err := u.userRoleRepo.AssignRole(txCtx, &entity.UserRole{
			ID: uuid.New(), TenantID: tenantID, UserID: user.ID, RoleID: role.ID,
			CreatedAt: now, UpdatedAt: now, CreatedBy: &inviterID, UpdatedBy: &inviterID,
		}); err != nil {
			return err
		}
		if err := u.invitationRepo.Create(txCtx, inv); err != nil {
			return err
		}
		if err := u.audit.Record(txCtx, tenantID, inviterID, domainaudit.ActionInvitationInvite, domainaudit.EntityInvitation, inv.ID,
			map[string]any{"email": email, "role_id": role.ID, "user_id": user.ID, "reinvite": reinvite}); err != nil {
			return err
		}
		return u.eventPublisher.Publish(txCtx, evt)
	}); err != nil {
		return nil, err
	}
	return inv, nil
}
