package leavetype

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

// UpdateLeaveTypeUseCase patches or (de)activates a leave type (admin).
type UpdateLeaveTypeUseCase interface {
	Execute(ctx context.Context, tenantID, actorID, id uuid.UUID, in leavetypes.UpdateLeaveTypeInput) (*entity.LeaveType, error)
}
