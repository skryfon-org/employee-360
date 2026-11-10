package leavetype

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

// CreateLeaveTypeUseCase creates a leave type (admin).
type CreateLeaveTypeUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, in leavetypes.CreateLeaveTypeInput) (*entity.LeaveType, error)
}
