package employee

import (
	"context"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type getEmployeeUseCase struct {
	directory repository.EmployeeDirectoryRepository
	profiles  repository.EmployeeProfileRepository
	contacts  repository.EmergencyContactRepository
}

// NewGetEmployeeUseCase creates a new GetEmployeeUseCase.
func NewGetEmployeeUseCase(
	directory repository.EmployeeDirectoryRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
) empuc.GetEmployeeUseCase {
	return &getEmployeeUseCase{directory: directory, profiles: profiles, contacts: contacts}
}

func (uc *getEmployeeUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID) (*emptypes.EmployeeDetail, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	return detail(ctx, uc.directory, uc.profiles, uc.contacts, tenantID, userID)
}
