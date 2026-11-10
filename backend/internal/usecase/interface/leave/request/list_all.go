package request

import (
	"context"

	"github.com/google/uuid"

	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

// ListAllLeaveUseCase lists every tenant request with filters (admin).
type ListAllLeaveUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, q leavetypes.ListLeaveRequestsQuery) (*leavetypes.LeaveRequestPage, error)
}
