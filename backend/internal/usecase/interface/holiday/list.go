package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// ListHolidaysByYearUseCase defines the contract for listing holidays for a given year within the caller's tenant.
type ListHolidaysByYearUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error)
}
