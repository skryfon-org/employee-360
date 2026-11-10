package request

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// CancelLeaveUseCase cancels the caller's own request.
type CancelLeaveUseCase interface {
	Execute(ctx context.Context, tenantID, userID, id uuid.UUID) (*entity.LeaveRequestDetail, error)
}
