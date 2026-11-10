package allocation

import (
	"context"

	"github.com/google/uuid"

	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

// CreateAllocationUseCase allocates a user to a project (warn-only over-allocation).
type CreateAllocationUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, in prjtypes.CreateAllocationInput) (*prjtypes.AllocationResult, error)
}
