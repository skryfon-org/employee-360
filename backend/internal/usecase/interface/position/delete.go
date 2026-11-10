package position

import (
	"context"

	"github.com/google/uuid"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

// DeletePositionUseCase defines the contract for deleting a position within the caller's tenant.
type DeletePositionUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.DeletePositionInput) error
}
