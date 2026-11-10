package leavetype

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/leavetype"
)

type listLeaveTypes struct {
	repo repository.LeaveTypeRepository
}

// NewListLeaveTypesUseCase constructs ListLeaveTypesUseCase.
func NewListLeaveTypesUseCase(repo repository.LeaveTypeRepository) leaveuc.ListLeaveTypesUseCase {
	return &listLeaveTypes{repo: repo}
}

func (uc *listLeaveTypes) Execute(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]*entity.LeaveType, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	return uc.repo.List(ctx, tenantID, !includeInactive)
}
