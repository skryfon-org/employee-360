package position

import (
	"context"

	"github.com/google/uuid"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

// ListPositionsUseCase defines the contract for listing positions within the caller's tenant.
type ListPositionsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, input postypes.ListPositionsQuery) (*postypes.ListPositionsResult, error)
}
