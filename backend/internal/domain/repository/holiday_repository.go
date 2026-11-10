package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// HolidayRepository defines data access methods for tenant-scoped holidays.
// Every method takes tenantID explicitly (resolved by the handler from the auth
// context); implementations must not read the tenant from context.Context.
// Create, Update, and SoftDelete take actorID to record audit columns.
type HolidayRepository interface {
	Create(ctx context.Context, tenantID, actorID uuid.UUID, holiday *entity.Holiday) error
	Update(ctx context.Context, tenantID, actorID uuid.UUID, holiday *entity.Holiday) error
	SoftDelete(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Holiday, error)
	ListByYear(ctx context.Context, tenantID uuid.UUID, year int, categoryID *uuid.UUID, month *int) ([]*entity.Holiday, error)
	// ListByDateRange lists the tenant's holidays with from <= date < to
	// (inclusive from, exclusive to), ordered date ASC, name ASC.
	ListByDateRange(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]*entity.Holiday, error)
	ExistsDuplicate(ctx context.Context, tenantID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error)
}
