package allocation

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

// ListUserAllocationsUseCase returns a user's allocations for a scope (active
// default, upcoming, past, all; each with its project status, any project
// status) for the admin employee view. total_percent and over_allocated count
// only allocations active today on live projects (active or on_hold), whatever
// the scope. An invalid scope yields ErrInvalidScope.
type ListUserAllocationsUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error)
}
