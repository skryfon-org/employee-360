package request

import (
	"context"

	"github.com/google/uuid"

	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

// ListMyLeaveUseCase lists the caller's own requests, optionally by status.
type ListMyLeaveUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, q leavetypes.ListMyLeaveQuery) (*leavetypes.LeaveRequestPage, error)
}
