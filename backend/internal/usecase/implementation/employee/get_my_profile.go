package employee

import (
	"context"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type getMyProfileUseCase struct {
	users    repository.UserRepository
	orgNames repository.OrgNameRepository
	profiles repository.EmployeeProfileRepository
	contacts repository.EmergencyContactRepository
}

// NewGetMyProfileUseCase creates a new GetMyProfileUseCase.
func NewGetMyProfileUseCase(
	users repository.UserRepository,
	orgNames repository.OrgNameRepository,
	profiles repository.EmployeeProfileRepository,
	contacts repository.EmergencyContactRepository,
) empuc.GetMyProfileUseCase {
	return &getMyProfileUseCase{users: users, orgNames: orgNames, profiles: profiles, contacts: contacts}
}

func (uc *getMyProfileUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID) (*emptypes.MyProfile, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	return myProfile(ctx, uc.users, uc.orgNames, uc.profiles, uc.contacts, tenantID, userID)
}
