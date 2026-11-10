package invitation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
)

// RevokeInvitationUseCaseImpl implements invusecase.RevokeInvitationUseCase.
type RevokeInvitationUseCaseImpl struct {
	invitationRepo repository.UserInvitationRepository
	userRepo       repository.UserRepository
	userRoleRepo   repository.UserRoleRepository
	audit          service.AuditRecorder
	transactor     ucshared.Transactor
}

var _ invusecase.RevokeInvitationUseCase = (*RevokeInvitationUseCaseImpl)(nil)

// NewRevokeInvitationUseCase constructs a RevokeInvitationUseCaseImpl.
func NewRevokeInvitationUseCase(
	invitationRepo repository.UserInvitationRepository,
	userRepo repository.UserRepository,
	userRoleRepo repository.UserRoleRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) *RevokeInvitationUseCaseImpl {
	return &RevokeInvitationUseCaseImpl{
		invitationRepo: invitationRepo, userRepo: userRepo, userRoleRepo: userRoleRepo,
		audit: audit, transactor: transactor,
	}
}

// Execute revokes a pending invitation in the caller's tenant and, in the same
// transaction, removes the still-inactive pending user (and its role
// assignment) so the email can be invited again.
func (u *RevokeInvitationUseCaseImpl) Execute(c context.Context, tenantID, actorID, id uuid.UUID) error {
	if err := requireIdentity(tenantID, actorID); err != nil {
		return err
	}
	return u.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		inv, err := u.invitationRepo.GetByID(txCtx, tenantID, id)
		if err != nil || inv == nil {
			return domainerrors.ErrInvitationNotFound
		}
		if !inv.IsPending() {
			return domainerrors.ErrInvitationNotPending
		}
		if err := u.invitationRepo.MarkRevoked(txCtx, tenantID, inv.ID, actorID, time.Now().UTC()); err != nil {
			return err
		}

		user, err := u.userRepo.GetByTenantAndEmail(txCtx, tenantID, inv.Email)
		if err != nil && !errors.Is(err, domainerrors.ErrNotFound) && !errors.Is(err, domainerrors.ErrUserNotFound) {
			return err
		}
		// Only a never-activated pending user is removed; a real account is left alone.
		removed := false
		if user != nil && !user.IsActive && user.PasswordHash == nil && user.EmailVerifiedAt == nil && user.LastLoginAt == nil {
			if err := u.userRoleRepo.DeleteByUserID(txCtx, tenantID, user.ID); err != nil {
				return err
			}
			if err := u.userRepo.Delete(txCtx, tenantID, user.ID, actorID); err != nil {
				return err
			}
			removed = true
		}
		return u.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionInvitationRevoke, domainaudit.EntityInvitation, inv.ID,
			map[string]any{"email": inv.Email, "pending_user_removed": removed})
	})
}
