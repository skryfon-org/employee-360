package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type listEmergencyContactsUseCase struct {
	users    repository.UserRepository
	contacts repository.EmergencyContactRepository
}

// NewListEmergencyContactsUseCase creates a new ListEmergencyContactsUseCase.
func NewListEmergencyContactsUseCase(
	users repository.UserRepository,
	contacts repository.EmergencyContactRepository,
) empuc.ListEmergencyContactsUseCase {
	return &listEmergencyContactsUseCase{users: users, contacts: contacts}
}

func (uc *listEmergencyContactsUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.EmergencyContact, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if _, err := uc.users.GetByID(ctx, tenantID, userID); err != nil {
		return nil, err
	}
	return uc.contacts.ListByUser(ctx, tenantID, userID)
}
