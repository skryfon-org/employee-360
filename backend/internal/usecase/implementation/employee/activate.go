package employee

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type activateEmployeeUseCase struct {
	users      repository.UserRepository
	profiles   repository.EmployeeProfileRepository
	contacts   repository.EmergencyContactRepository
	directory  repository.EmployeeDirectoryRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewActivateEmployeeUseCase creates a new ActivateEmployeeUseCase.
func NewActivateEmployeeUseCase(
	users repository.UserRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
	directory repository.EmployeeDirectoryRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.ActivateEmployeeUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &activateEmployeeUseCase{
		users: users, profiles: profiles, contacts: contacts, directory: directory,
		audit: audit, transactor: transactor,
	}
}

func (uc *activateEmployeeUseCase) Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID) (*emptypes.EmployeeDetail, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	err := uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		user, err := loadTarget(tx, uc.users, actor.TenantID, userID)
		if err != nil {
			return err
		}
		if err := guardSuperAdminTarget(actor, user); err != nil {
			return err
		}
		if user.IsActive {
			return nil // idempotent
		}
		// A pending invitee has no credentials yet; activation is via accepting the invitation.
		if user.PasswordHash == nil || *user.PasswordHash == "" {
			return domainerrors.ErrUserNotOnboarded
		}
		user.IsActive = true
		user.UpdatedAt = time.Now().UTC()
		user.UpdatedBy = &actor.UserID
		if err := uc.users.Update(tx, actor.TenantID, user); err != nil {
			return err
		}
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeActivate, userID, map[string]any{"email": user.Email})
	})
	if err != nil {
		return nil, err
	}
	return detail(ctx, uc.directory, uc.profiles, uc.contacts, actor.TenantID, userID)
}
