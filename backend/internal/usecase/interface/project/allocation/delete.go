package allocation

import (
	"context"

	"github.com/google/uuid"
)

// DeleteAllocationUseCase soft-deletes an allocation.
type DeleteAllocationUseCase interface {
	Execute(ctx context.Context, tenantID, actorID, id uuid.UUID) error
}
