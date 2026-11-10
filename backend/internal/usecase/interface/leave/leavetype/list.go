package leavetype

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// ListLeaveTypesUseCase lists leave types; inactive ones only when includeInactive.
type ListLeaveTypesUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]*entity.LeaveType, error)
}
