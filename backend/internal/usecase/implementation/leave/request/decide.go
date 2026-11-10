package request

import (
	"context"
	"strings"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/request"
)

type decideLeave struct {
	repo       repository.LeaveRequestRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
	clock      leavecommon.Clock
	status     string
	action     string
	events     service.EventPublisher
	urls       leavecommon.AppURLs
}

// NewApproveLeaveUseCase constructs the approve DecideLeaveUseCase.
func NewApproveLeaveUseCase(repo repository.LeaveRequestRepository, audit service.AuditRecorder, transactor ucshared.Transactor, clock leavecommon.Clock, events service.EventPublisher, urls leavecommon.AppURLs) leaveuc.DecideLeaveUseCase {
	return newDecide(repo, audit, transactor, clock, events, urls, entity.LeaveStatusApproved, domainaudit.ActionLeaveRequestApprove)
}

// NewRejectLeaveUseCase constructs the reject DecideLeaveUseCase.
func NewRejectLeaveUseCase(repo repository.LeaveRequestRepository, audit service.AuditRecorder, transactor ucshared.Transactor, clock leavecommon.Clock, events service.EventPublisher, urls leavecommon.AppURLs) leaveuc.DecideLeaveUseCase {
	return newDecide(repo, audit, transactor, clock, events, urls, entity.LeaveStatusRejected, domainaudit.ActionLeaveRequestReject)
}

func newDecide(repo repository.LeaveRequestRepository, audit service.AuditRecorder, transactor ucshared.Transactor, clock leavecommon.Clock, events service.EventPublisher, urls leavecommon.AppURLs, status, action string) leaveuc.DecideLeaveUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	if clock == nil {
		clock = leavecommon.RealClock
	}
	return &decideLeave{repo: repo, audit: audit, transactor: transactor, clock: clock, status: status, action: action, events: events, urls: urls}
}

func (uc *decideLeave) Execute(ctx context.Context, tenantID, actorID uuid.UUID, isAdmin bool, id uuid.UUID, comment string) (*entity.LeaveRequestDetail, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	comment = strings.TrimSpace(comment)
	if len([]rune(comment)) > 500 {
		return nil, leavecommon.Invalid("comment must not exceed 500 characters")
	}
	evtType := event.EventTypeLeaveApproved
	if uc.status == entity.LeaveStatusRejected {
		evtType = event.EventTypeLeaveRejected
	}
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		r, err := uc.repo.GetByIDForUpdate(txCtx, tenantID, id)
		if err != nil {
			return err
		}
		if r.UserID == actorID {
			return domainerrors.ErrLeaveSelfDecision
		}
		if r.ApproverID != actorID && !isAdmin {
			return domainerrors.ErrLeaveRequestNotFound
		}
		if r.Status != entity.LeaveStatusPending {
			return domainerrors.ErrLeaveNotPending
		}
		now := uc.clock()
		r.Status = uc.status
		r.DecidedBy, r.DecidedAt = &actorID, &now
		r.DecisionComment = comment
		if err := uc.repo.RecordDecision(txCtx, tenantID, actorID, r); err != nil {
			return err
		}
		if err := uc.audit.Record(txCtx, tenantID, actorID, uc.action, domainaudit.EntityLeaveRequest, r.ID, map[string]any{
			"user_id": r.UserID, "status": r.Status, "comment": comment,
		}); err != nil {
			return err
		}
		// Published in the same transaction (outbox): the email is queued iff the decision commits.
		// An enqueue failure intentionally rolls back the action; sending itself is async and retried by River.
		d, err := uc.repo.GetDetail(txCtx, tenantID, id)
		if err != nil {
			return err
		}
		return uc.events.Publish(txCtx, event.Event{
			ID: uuid.New(), TenantID: tenantID, EventType: evtType,
			AggregateType: "leave_request", AggregateID: r.ID.String(),
			Payload: event.LeaveDecisionPayload{
				RequestID: r.ID, TenantID: tenantID, EmployeeID: r.UserID, EmployeeEmail: d.UserEmail,
				EmployeeName: d.UserName, ApproverName: d.DecidedByName, LeaveType: d.LeaveTypeName,
				StartDate: r.StartDate, EndDate: r.EndDate, DayPart: r.DayPart, Days: r.Days,
				Comment: comment, LeaveURL: uc.urls.EmployeeLeaveLink(),
			},
			OccurredAt: now,
		})
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.GetDetail(ctx, tenantID, id)
}
