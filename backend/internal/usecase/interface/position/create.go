package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

// CreatePositionUseCase defines the contract for creating a position within the caller's tenant.
type CreatePositionUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.CreatePositionInput) (*entity.Position, error)
}
