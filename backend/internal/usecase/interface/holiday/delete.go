package holiday

import (
	"context"

	"github.com/google/uuid"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// DeleteHolidayUseCase defines the contract for soft-deleting a holiday within the caller's tenant.
type DeleteHolidayUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteHolidayInput) error
}
