package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// CreateHolidayUseCase defines the contract for creating a holiday within the caller's tenant.
type CreateHolidayUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error)
}
