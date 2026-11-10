package employee

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type deactivateEmployeeUseCase struct {
	users         repository.UserRepository
	profiles      repository.EmployeeProfileRepository
	contacts      repository.EmergencyContactRepository
	directory     repository.EmployeeDirectoryRepository
	refreshTokens repository.RefreshTokenRepository
	audit         service.AuditRecorder
	transactor    ucshared.Transactor
}

// NewDeactivateEmployeeUseCase creates a new DeactivateEmployeeUseCase.
func NewDeactivateEmployeeUseCase(
	users repository.UserRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
	directory repository.EmployeeDirectoryRepository,
	refreshTokens repository.RefreshTokenRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.DeactivateEmployeeUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &deactivateEmployeeUseCase{
		users: users, profiles: profiles, contacts: contacts, directory: directory,
		refreshTokens: refreshTokens, audit: audit, transactor: transactor,
	}
}

func (uc *deactivateEmployeeUseCase) Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID) (*emptypes.EmployeeDetail, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	if userID == actor.UserID {
		return nil, domainerrors.ErrCannotActOnSelf
	}
	err := uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		user, err := loadTarget(tx, uc.users, actor.TenantID, userID)
		if err != nil {
			return err
		}
		if err := guardSuperAdminTarget(actor, user); err != nil {
			return err
		}
		if !user.IsActive {
			return nil // idempotent
		}
		if hasRole(user.Roles, entity.RoleAdmin) {
			ok, err := othersHoldingRole(tx, uc.directory, actor.TenantID, entity.RoleAdmin, userID)
			if err != nil {
				return err
			}
			if !ok {
				return domainerrors.ErrLastActiveAdmin
			}
		}
		user.IsActive = false
		user.UpdatedAt = time.Now().UTC()
		user.UpdatedBy = &actor.UserID
		if err := uc.users.Update(tx, actor.TenantID, user); err != nil {
			return err
		}
		if err := uc.refreshTokens.RevokeAllForUser(tx, userID); err != nil {
			return err
		}
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeDeactivate, userID, map[string]any{"email": user.Email})
	})
	if err != nil {
		return nil, err
	}
	return detail(ctx, uc.directory, uc.profiles, uc.contacts, actor.TenantID, userID)
}
