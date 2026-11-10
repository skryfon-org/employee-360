// Package organization declares the organization settings usecase contracts.
package organization

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
)

// GetSettingsUseCase returns the tenant's settings, or the defaults (without
// writing a row) when none exist yet.
type GetSettingsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID) (*entity.OrganizationSettings, error)
}

// UpdateSettingsUseCase validates and saves the employee code format.
type UpdateSettingsUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, in orgtypes.UpdateSettingsInput) (*entity.OrganizationSettings, error)
}

// GetWeeklyOffDaysUseCase returns only the tenant's weekly off days
// (0 = Sunday .. 6 = Saturday), the default {0,6} when none are saved. It is the
// employee-readable slice of the organization settings.
type GetWeeklyOffDaysUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID) ([]int, error)
}
