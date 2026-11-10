package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// HolidayCalendarSettingsRepository defines data access methods for tenant-scoped calendar settings.
// Every method takes tenantID explicitly (resolved by the handler from the auth
// context); implementations must not read the tenant from context.Context.
type HolidayCalendarSettingsRepository interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error)
	Upsert(ctx context.Context, tenantID, actorID uuid.UUID, settings *entity.HolidayCalendarSettings) error
}
