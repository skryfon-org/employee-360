package request

import (
	"context"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/request"
)

type cancelLeave struct {
	repo       repository.LeaveRequestRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
	clock      leavecommon.Clock
}

// NewCancelLeaveUseCase constructs CancelLeaveUseCase.
func NewCancelLeaveUseCase(repo repository.LeaveRequestRepository, audit service.AuditRecorder, transactor ucshared.Transactor, clock leavecommon.Clock) leaveuc.CancelLeaveUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	if clock == nil {
		clock = leavecommon.RealClock
	}
	return &cancelLeave{repo: repo, audit: audit, transactor: transactor, clock: clock}
}

func (uc *cancelLeave) Execute(ctx context.Context, tenantID, userID, id uuid.UUID) (*entity.LeaveRequestDetail, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		r, err := uc.repo.GetByIDForUpdate(txCtx, tenantID, id)
		if err != nil {
			return err
		}
		// Somebody else's request is indistinguishable from a missing one.
		if r.UserID != userID {
			return domainerrors.ErrLeaveRequestNotFound
		}
		today := leavecommon.DateOnly(uc.clock())
		switch {
		case r.Status == entity.LeaveStatusPending:
		case r.Status == entity.LeaveStatusApproved && !r.StartDate.Before(today):
		default:
			return domainerrors.ErrLeaveNotCancellable
		}
		previous := r.Status
		// Only the status changes: the approver's decision trail stays intact.
		r.Status = entity.LeaveStatusCancelled
		if err := uc.repo.UpdateStatus(txCtx, tenantID, userID, r); err != nil {
			return err
		}
		return uc.audit.Record(txCtx, tenantID, userID, domainaudit.ActionLeaveRequestCancel, domainaudit.EntityLeaveRequest, r.ID, map[string]any{
			"previous_status": previous, "status": r.Status, "start_date": r.StartDate.Format("2006-01-02"),
		})
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.GetDetail(ctx, tenantID, id)
}
