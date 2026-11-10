package request

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

// ApplyLeaveUseCase creates a pending leave request for the caller.
type ApplyLeaveUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID, in leavetypes.ApplyLeaveInput) (*entity.LeaveRequestDetail, error)
}
