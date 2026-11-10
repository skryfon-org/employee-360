package invitation

import (
	"context"
	"time"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
)

// ResendInvitationUseCaseImpl implements invusecase.ResendInvitationUseCase.
type ResendInvitationUseCaseImpl struct {
	roleRepo       repository.RoleRepository
	invitationRepo repository.UserInvitationRepository
	audit          service.AuditRecorder
	hashService    service.HashService
	eventPublisher service.EventPublisher
	transactor     ucshared.Transactor
	expiry         time.Duration
	appURLs        AppURLs
}

var _ invusecase.ResendInvitationUseCase = (*ResendInvitationUseCaseImpl)(nil)

// NewResendInvitationUseCase constructs a ResendInvitationUseCaseImpl.
func NewResendInvitationUseCase(
	roleRepo repository.RoleRepository,
	invitationRepo repository.UserInvitationRepository,
	audit service.AuditRecorder,
	hashService service.HashService,
	eventPublisher service.EventPublisher,
	transactor ucshared.Transactor,
	appURLs AppURLs,
) *ResendInvitationUseCaseImpl {
	return &ResendInvitationUseCaseImpl{
		roleRepo: roleRepo, invitationRepo: invitationRepo, audit: audit, hashService: hashService,
		eventPublisher: eventPublisher, transactor: transactor,
		expiry: defaultInvitationExpiry, appURLs: appURLs,
	}
}

// Execute reissues the token and publishes InvitationResent, only while pending.
func (u *ResendInvitationUseCaseImpl) Execute(c context.Context, actor emptypes.Actor, id uuid.UUID) (*entity.UserInvitation, error) {
	tenantID, actorID := actor.TenantID, actor.UserID
	if err := requireIdentity(tenantID, actorID); err != nil {
		return nil, err
	}
	inv, err := u.invitationRepo.GetByID(c, tenantID, id)
	if err != nil || inv == nil {
		return nil, domainerrors.ErrInvitationNotFound
	}
	if !inv.IsPending() {
		return nil, domainerrors.ErrInvitationNotPending
	}
	role, err := u.roleRepo.GetByID(c, tenantID, inv.RoleID)
	if err != nil || role == nil {
		return nil, domainerrors.ErrRoleNotFound
	}
	if err := requireCanGrantRole(actor, role.Name); err != nil {
		return nil, err
	}

	plainToken, err := newPlainToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	inv.TokenHash = u.hashService.HashToken(plainToken)
	inv.ExpiresAt = now.Add(u.expiry)
	inv.UpdatedAt = now

	evt := event.Event{
		ID:            uuid.New(),
		TenantID:      tenantID,
		EventType:     event.EventTypeInvitationResent,
		AggregateType: "user_invitation",
		AggregateID:   inv.ID.String(),
		Payload: event.InvitationResentPayload{
			InvitationID: inv.ID, TenantID: tenantID, Email: inv.Email,
			RoleName: role.Name, PlainToken: plainToken,
			InviteURL: u.appURLs.acceptLink(role.Name, plainToken),
			ExpiresAt: inv.ExpiresAt,
		},
		OccurredAt: now,
	}

	if err := u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		if err := u.invitationRepo.UpdateToken(txCtx, tenantID, inv.ID, actorID, inv.TokenHash, inv.ExpiresAt); err != nil {
			return err
		}
		if err := u.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionInvitationResend, domainaudit.EntityInvitation, inv.ID,
			map[string]any{"email": inv.Email}); err != nil {
			return err
		}
		return u.eventPublisher.Publish(txCtx, evt)
	}); err != nil {
		return nil, err
	}
	return inv, nil
}
