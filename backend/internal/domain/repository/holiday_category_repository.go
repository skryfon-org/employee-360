package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// HolidayCategoryRepository defines data access methods for tenant-scoped holiday categories.
// Every method takes tenantID explicitly (resolved by the handler from the auth
// context); implementations must not read the tenant from context.Context.
type HolidayCategoryRepository interface {
	Create(ctx context.Context, tenantID, actorID uuid.UUID, category *entity.HolidayCategory) error
	Update(ctx context.Context, tenantID, actorID uuid.UUID, category *entity.HolidayCategory) error
	SoftDelete(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error)
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*entity.HolidayCategory, error)
	List(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error)
	ExistsByName(ctx context.Context, tenantID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
	CountHolidaysByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (int64, error)
}
