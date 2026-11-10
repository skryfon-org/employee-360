package request

import (
	"context"

	"github.com/google/uuid"

	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

// ListTeamLeaveUseCase lists requests addressed to the caller as approver.
type ListTeamLeaveUseCase interface {
	Execute(ctx context.Context, tenantID, approverID uuid.UUID, q leavetypes.ListMyLeaveQuery) (*leavetypes.LeaveRequestPage, error)
}
