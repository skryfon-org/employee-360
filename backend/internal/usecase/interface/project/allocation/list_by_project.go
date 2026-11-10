package allocation

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// ListProjectAllocationsUseCase lists a project's allocations filtered by state
// (scope) relative to today: active (start <= today and end null or >= today),
// upcoming (start > today), past (end < today) or all (empty defaults to all).
type ListProjectAllocationsUseCase interface {
	Execute(ctx context.Context, tenantID, projectID uuid.UUID, scope entity.AllocationScope) ([]*entity.AllocationWithUser, error)
}
