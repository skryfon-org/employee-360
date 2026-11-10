package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// GetSettingsUseCase defines the contract for retrieving calendar settings for a tenant.
// If no row exists, returns default settings without writing a row to the database.
type GetSettingsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error)
}

// UpdateSettingsUseCase defines the contract for updating calendar settings for a tenant.
type UpdateSettingsUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateSettingsInput) (*entity.HolidayCalendarSettings, error)
}

// Aliases for HolidayCalendarSettings naming
type GetHolidayCalendarSettingsUseCase = GetSettingsUseCase
type UpdateHolidayCalendarSettingsUseCase = UpdateSettingsUseCase
