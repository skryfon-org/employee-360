package employee

import (
	"context"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type deleteEmergencyContactUseCase struct {
	users      repository.UserRepository
	profiles   repository.EmployeeProfileRepository
	contacts   repository.EmergencyContactRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewDeleteEmergencyContactUseCase creates a new DeleteEmergencyContactUseCase.
func NewDeleteEmergencyContactUseCase(
	users repository.UserRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.DeleteEmergencyContactUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &deleteEmergencyContactUseCase{users: users, profiles: profiles, contacts: contacts, audit: audit, transactor: transactor}
}

func (uc *deleteEmergencyContactUseCase) Execute(ctx context.Context, actor emptypes.Actor, userID, contactID uuid.UUID) error {
	if err := requireActor(actor); err != nil {
		return err
	}
	return uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		if err := authorizeTarget(tx, uc.users, actor, userID); err != nil {
			return err
		}
		c, err := uc.contacts.GetByID(tx, actor.TenantID, userID, contactID)
		if err != nil {
			return err
		}
		if err := uc.contacts.Delete(tx, actor.TenantID, userID, contactID, actor.UserID); err != nil {
			return err
		}
		// Keep a primary: promote the oldest remaining contact if the primary went away.
		if c.IsPrimary {
			rest, err := uc.contacts.ListByUser(tx, actor.TenantID, userID)
			if err != nil {
				return err
			}
			if len(rest) > 0 {
				rest[0].IsPrimary = true
				if err := uc.contacts.Update(tx, actor.TenantID, actor.UserID, rest[0]); err != nil {
					return err
				}
			}
		}
		if _, err := syncCompletion(tx, uc.profiles, uc.contacts, actor.TenantID, userID, actor.UserID); err != nil {
			return err
		}
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeContactDelete, userID, map[string]any{"contact_id": contactID})
	})
}
