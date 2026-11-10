package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// CreateCategoryUseCase defines the contract for creating a holiday category.
type CreateCategoryUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error)
}

// UpdateCategoryUseCase defines the contract for updating a holiday category.
type UpdateCategoryUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error)
}

// DeleteCategoryUseCase defines the contract for removing a holiday category.
type DeleteCategoryUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error
}

// GetCategoryUseCase defines the contract for looking up a holiday category by ID.
type GetCategoryUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error)
}

// ListCategoriesUseCase defines the contract for listing holiday categories.
type ListCategoriesUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error)
}
