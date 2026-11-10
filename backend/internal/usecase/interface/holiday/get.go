package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// GetHolidayUseCase defines the contract for retrieving a holiday by ID within the caller's tenant.
type GetHolidayUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, query holidaytypes.GetHolidayQuery) (*entity.Holiday, error)
}
