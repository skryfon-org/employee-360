package invitation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
)

// AcceptInvitationUseCaseImpl implements invusecase.AcceptInvitationUseCase.
type AcceptInvitationUseCaseImpl struct {
	userRepo       repository.UserRepository
	invitationRepo repository.UserInvitationRepository
	roleRepo       repository.RoleRepository
	tenantReader   repository.TenantReader
	settingsRepo   repository.OrganizationSettingsRepository
	hashService    service.HashService
	eventPublisher service.EventPublisher
	transactor     ucshared.Transactor
	appURLs        AppURLs
}

var _ invusecase.AcceptInvitationUseCase = (*AcceptInvitationUseCaseImpl)(nil)

// NewAcceptInvitationUseCase constructs an AcceptInvitationUseCaseImpl.
func NewAcceptInvitationUseCase(
	userRepo repository.UserRepository,
	invitationRepo repository.UserInvitationRepository,
	roleRepo repository.RoleRepository,
	tenantReader repository.TenantReader,
	settingsRepo repository.OrganizationSettingsRepository,
	hashService service.HashService,
	eventPublisher service.EventPublisher,
	transactor ucshared.Transactor,
	appURLs AppURLs,
) *AcceptInvitationUseCaseImpl {
	return &AcceptInvitationUseCaseImpl{
		userRepo: userRepo, invitationRepo: invitationRepo, roleRepo: roleRepo, tenantReader: tenantReader,
		settingsRepo: settingsRepo, hashService: hashService, eventPublisher: eventPublisher,
		transactor: transactor, appURLs: appURLs,
	}
}

// Execute consumes the token, sets the password (mandatory for every role) and activates the user.
// The tenant is derived from the invitation row found by token hash, never from input.
func (u *AcceptInvitationUseCaseImpl) Execute(c context.Context, req invtypes.AcceptInvitationRequest) error {
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return domainerrors.ErrInvalidToken
	}
	if len(req.Password) < minPasswordLength || len(req.Password) > service.MaxPasswordBytes {
		return domainerrors.ErrInvalidPassword
	}

	hashed, err := u.hashService.HashPassword(req.Password)
	if err != nil {
		return err
	}

	// Lookup, usability check and the conditional MarkAccepted all run in one
	// transaction, so concurrent accepts (or an accept racing a revoke) cannot
	// both succeed.
	return u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		now := time.Now().UTC()
		inv, user, err := lookupUsableInvitation(txCtx, u.invitationRepo, u.userRepo, u.hashService, token, now)
		if err != nil {
			return err
		}
		if err := u.invitationRepo.MarkAccepted(txCtx, inv.TenantID, inv.ID, now); err != nil {
			if errors.Is(err, domainerrors.ErrInvitationNotPending) {
				// Lost a race with a concurrent accept or revoke.
				return domainerrors.ErrInvitationAccepted
			}
			return err
		}
		user.PasswordHash = &hashed
		user.IsActive = true
		user.EmailVerifiedAt = &now
		user.UpdatedAt = now
		if err := u.userRepo.Update(txCtx, inv.TenantID, user); err != nil {
			return err
		}
		return u.publishUserActivated(txCtx, inv, user, now)
	})
}

// publishUserActivated enqueues the Welcome email in the accept transaction. The
// tenant comes from the invitation row; the custom welcome message (if any) is
// resolved here so the event dispatcher stays pure.
func (u *AcceptInvitationUseCaseImpl) publishUserActivated(c context.Context, inv *entity.UserInvitation, user *entity.User, now time.Time) error {
	tenant, err := u.tenantReader.GetByID(c, inv.TenantID)
	if err != nil {
		return err
	}
	role, err := u.roleRepo.GetByID(c, inv.TenantID, inv.RoleID)
	if err != nil {
		return err
	}
	welcome := ""
	settings, err := u.settingsRepo.GetByTenantID(c, inv.TenantID)
	switch {
	case err == nil:
		if settings.WelcomeMessage != nil {
			welcome = strings.TrimSpace(*settings.WelcomeMessage)
		}
	case errors.Is(err, domainerrors.ErrOrganizationSettingsNotFound):
		// No row: the built-in default message applies.
	default:
		return err
	}
	return u.eventPublisher.Publish(c, event.Event{
		ID:            uuid.New(),
		TenantID:      inv.TenantID,
		EventType:     event.EventTypeUserActivated,
		AggregateType: "user",
		AggregateID:   user.ID.String(),
		Payload: event.UserActivatedPayload{
			UserID: user.ID, TenantID: inv.TenantID, Email: user.Email,
			FirstName: user.FirstName, LastName: user.LastName,
			TenantName: tenant.Name, RoleName: role.Name,
			LoginURL:       u.appURLs.ForRole(role.Name) + "/login",
			WelcomeMessage: welcome,
		},
		OccurredAt: now,
	})
}
