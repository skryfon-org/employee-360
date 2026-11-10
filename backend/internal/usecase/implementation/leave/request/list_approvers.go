package request

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/request"
)

type listApprovers struct {
	profiles  repository.EmployeeProfileRepository
	approvers repository.LeaveApproverRepository
}

// NewListApproversUseCase constructs ListApproversUseCase.
func NewListApproversUseCase(profiles repository.EmployeeProfileRepository, approvers repository.LeaveApproverRepository) leaveuc.ListApproversUseCase {
	return &listApprovers{profiles: profiles, approvers: approvers}
}

func (uc *listApprovers) Execute(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.LeaveApprover, error) {
	return resolveApprovers(ctx, uc.profiles, uc.approvers, tenantID, userID)
}
