package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

// UpdatePositionUseCase defines the contract for updating a position within the caller's tenant.
type UpdatePositionUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.UpdatePositionInput) (*entity.Position, error)
}
