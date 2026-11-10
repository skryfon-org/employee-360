package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// UpdateHolidayUseCase defines the contract for updating a holiday within the caller's tenant.
type UpdateHolidayUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error)
}
