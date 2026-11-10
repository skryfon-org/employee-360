// Package organization implements the organization settings usecases.
package organization

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	orguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/organization"
)

type getSettingsUseCase struct {
	repo repository.OrganizationSettingsRepository
}

// NewGetSettingsUseCase constructs a GetSettingsUseCase.
func NewGetSettingsUseCase(repo repository.OrganizationSettingsRepository) orguc.GetSettingsUseCase {
	return &getSettingsUseCase{repo: repo}
}

func (uc *getSettingsUseCase) Execute(ctx context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	s, err := uc.repo.GetByTenantID(ctx, tenantID)
	if errors.Is(err, domainerrors.ErrOrganizationSettingsNotFound) {
		return entity.DefaultOrganizationSettings(tenantID), nil
	}
	return s, err
}
