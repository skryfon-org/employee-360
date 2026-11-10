package request

import (
	"context"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/types/leave"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/request"
)

type listAllLeave struct {
	repo repository.LeaveRequestRepository
}

// NewListAllLeaveUseCase constructs ListAllLeaveUseCase.
func NewListAllLeaveUseCase(repo repository.LeaveRequestRepository) leaveuc.ListAllLeaveUseCase {
	return &listAllLeave{repo: repo}
}

func (uc *listAllLeave) Execute(ctx context.Context, tenantID uuid.UUID, q leave.ListLeaveRequestsQuery) (*leave.LeaveRequestPage, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	var f repository.LeaveRequestFilter
	var err error
	if f.Status, err = parseStatusFilter(q.Status); err != nil {
		return nil, err
	}
	if f.UserID, err = parseUUIDFilter(q.EmployeeID, "employee"); err != nil {
		return nil, err
	}
	if f.LeaveTypeID, err = parseUUIDFilter(q.LeaveTypeID, "type"); err != nil {
		return nil, err
	}
	if f.From, err = parseDateFilter(q.From, "from"); err != nil {
		return nil, err
	}
	if f.To, err = parseDateFilter(q.To, "to"); err != nil {
		return nil, err
	}
	if f.From != nil && f.To != nil && f.To.Before(*f.From) {
		return nil, leavecommon.Invalid("to must not be before from")
	}
	page, size, offset := pagination.Normalize(q.Page, q.PageSize)
	items, total, err := uc.repo.List(ctx, tenantID, f, size, offset)
	if err != nil {
		return nil, err
	}
	return &leave.LeaveRequestPage{Items: items, Total: total, Page: page, PageSize: size}, nil
}
