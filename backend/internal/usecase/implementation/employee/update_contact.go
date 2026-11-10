package employee

import (
	"context"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type updateEmergencyContactUseCase struct {
	users      repository.UserRepository
	contacts   repository.EmergencyContactRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewUpdateEmergencyContactUseCase creates a new UpdateEmergencyContactUseCase.
func NewUpdateEmergencyContactUseCase(
	users repository.UserRepository,
	contacts repository.EmergencyContactRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) empuc.UpdateEmergencyContactUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateEmergencyContactUseCase{users: users, contacts: contacts, audit: audit, transactor: transactor}
}

func (uc *updateEmergencyContactUseCase) Execute(ctx context.Context, actor emptypes.Actor, userID, contactID uuid.UUID, in emptypes.EmergencyContactInput) (*entity.EmergencyContact, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	var out *entity.EmergencyContact
	err := uc.transactor.WithinTransaction(ctx, func(tx context.Context) error {
		if err := authorizeTarget(tx, uc.users, actor, userID); err != nil {
			return err
		}
		c, err := uc.contacts.GetByID(tx, actor.TenantID, userID, contactID)
		if err != nil {
			return err
		}
		wasPrimary := c.IsPrimary
		if err := normalizeContact(in, c); err != nil {
			return err
		}
		// A sole/primary contact cannot be demoted by omitting is_primary:
		// promote another contact instead.
		if wasPrimary && !c.IsPrimary {
			c.IsPrimary = true
		}
		if c.IsPrimary {
			if err := uc.contacts.ClearPrimary(tx, actor.TenantID, userID, c.ID, actor.UserID); err != nil {
				return err
			}
		}
		if err := uc.contacts.Update(tx, actor.TenantID, actor.UserID, c); err != nil {
			return err
		}
		out = c
		return record(tx, uc.audit, actor, domainaudit.ActionEmployeeContactUpdate, userID, map[string]any{"contact_id": c.ID, "is_primary": c.IsPrimary})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
