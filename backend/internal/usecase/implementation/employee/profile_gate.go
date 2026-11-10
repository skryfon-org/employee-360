package employee

import (
	"context"
	"errors"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type profileGateUseCase struct {
	profiles repository.EmployeeProfileRepository
}

// NewProfileGateUseCase creates a new ProfileGateUseCase (behind the profile gating middleware).
func NewProfileGateUseCase(profiles repository.EmployeeProfileRepository) empuc.ProfileGateUseCase {
	return &profileGateUseCase{profiles: profiles}
}

func (uc *profileGateUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return false, domainerrors.ErrUnauthorized
	}
	p, err := uc.profiles.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrEmployeeProfileNotFound) {
			return false, nil
		}
		return false, err
	}
	return p.ProfileCompletedAt != nil, nil
}
