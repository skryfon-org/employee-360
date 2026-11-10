package request

import (
	"context"

	"github.com/google/uuid"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/types/leave"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/request"
)

type listMyLeave struct {
	repo repository.LeaveRequestRepository
}

// NewListMyLeaveUseCase constructs ListMyLeaveUseCase.
func NewListMyLeaveUseCase(repo repository.LeaveRequestRepository) leaveuc.ListMyLeaveUseCase {
	return &listMyLeave{repo: repo}
}

func (uc *listMyLeave) Execute(ctx context.Context, tenantID, userID uuid.UUID, q leave.ListMyLeaveQuery) (*leave.LeaveRequestPage, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	st, err := parseStatusFilter(q.Status)
	if err != nil {
		return nil, err
	}
	from, to, err := parseDateRange(q.From, q.To)
	if err != nil {
		return nil, err
	}
	typeID, err := parseUUIDFilter(q.LeaveTypeID, "leave_type_id")
	if err != nil {
		return nil, err
	}
	page, size, offset := pagination.Normalize(q.Page, q.PageSize)
	items, total, err := uc.repo.ListByUser(ctx, tenantID, userID, repository.LeaveRequestFilter{Status: st, From: from, To: to, LeaveTypeID: typeID}, size, offset)
	if err != nil {
		return nil, err
	}
	return &leave.LeaveRequestPage{Items: items, Total: total, Page: page, PageSize: size}, nil
}
