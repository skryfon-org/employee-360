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

type listTeamLeave struct {
	repo repository.LeaveRequestRepository
}

// NewListTeamLeaveUseCase constructs ListTeamLeaveUseCase.
func NewListTeamLeaveUseCase(repo repository.LeaveRequestRepository) leaveuc.ListTeamLeaveUseCase {
	return &listTeamLeave{repo: repo}
}

func (uc *listTeamLeave) Execute(ctx context.Context, tenantID, approverID uuid.UUID, q leave.ListMyLeaveQuery) (*leave.LeaveRequestPage, error) {
	if tenantID == uuid.Nil || approverID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	st, err := parseStatusFilter(q.Status)
	if err != nil {
		return nil, err
	}
	// exclude_status is optional; when it equals status the result is
	// necessarily an empty page (no row can match both), which the repository
	// query yields naturally.
	ex, err := parseStatusFilter(q.ExcludeStatus)
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
	items, total, err := uc.repo.ListByApprover(ctx, tenantID, approverID, repository.LeaveRequestFilter{Status: st, ExcludeStatus: ex, From: from, To: to, LeaveTypeID: typeID}, size, offset)
	if err != nil {
		return nil, err
	}
	return &leave.LeaveRequestPage{Items: items, Total: total, Page: page, PageSize: size}, nil
}
