package request

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// ListApproversUseCase returns the users the caller may address a leave request to.
type ListApproversUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.LeaveApprover, error)
}
