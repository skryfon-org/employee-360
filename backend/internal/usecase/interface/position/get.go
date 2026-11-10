package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

// GetPositionUseCase defines the contract for retrieving a position by ID within the caller's tenant.
type GetPositionUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, input postypes.GetPositionQuery) (*entity.Position, error)
}
