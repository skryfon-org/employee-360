package allocation

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"

	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
)

// ListMyProjectsUseCase returns the caller's allocations for a scope. The
// active scope returns date-active allocations on live projects (status active
// or on_hold); upcoming, past and all return every project status (each item
// carries project_status and counts_toward_total). TotalPercent and
// OverAllocated always describe today's load on live projects. Counts holds the
// caller's active/upcoming/past totals as of today regardless of scope.
type ListMyProjectsUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error)
}
