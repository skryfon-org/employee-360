package allocation

import (
	"context"

	"github.com/google/uuid"

	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

// UpdateAllocationUseCase patches an allocation (percent, role, dates).
type UpdateAllocationUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, in prjtypes.UpdateAllocationInput) (*prjtypes.AllocationResult, error)
}
