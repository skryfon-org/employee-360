package request

import (
	"context"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/types/leave"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/request"
)

type applyLeave struct {
	requests   repository.LeaveRequestRepository
	types      repository.LeaveTypeRepository
	profiles   repository.EmployeeProfileRepository
	approvers  repository.LeaveApproverRepository
	offDays    repository.WeeklyOffDaysReader
	holidays   repository.HolidayRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
	clock      leavecommon.Clock
	events     service.EventPublisher
	urls       leavecommon.AppURLs
}

// NewApplyLeaveUseCase constructs ApplyLeaveUseCase.
func NewApplyLeaveUseCase(
	requests repository.LeaveRequestRepository,
	types repository.LeaveTypeRepository,
	profiles repository.EmployeeProfileRepository,
	approvers repository.LeaveApproverRepository,
	offDays repository.WeeklyOffDaysReader,
	holidays repository.HolidayRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
	clock leavecommon.Clock,
	events service.EventPublisher,
	urls leavecommon.AppURLs,
) leaveuc.ApplyLeaveUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	if clock == nil {
		clock = leavecommon.RealClock
	}
	return &applyLeave{requests: requests, types: types, profiles: profiles, approvers: approvers, offDays: offDays, holidays: holidays, audit: audit, transactor: transactor, clock: clock, events: events, urls: urls}
}

func (uc *applyLeave) Execute(ctx context.Context, tenantID, userID uuid.UUID, in leave.ApplyLeaveInput) (*entity.LeaveRequestDetail, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if in.StartDate.IsZero() || in.EndDate.IsZero() {
		return nil, leavecommon.Invalid("start_date and end_date are required")
	}
	start, end := leavecommon.DateOnly(in.StartDate), leavecommon.DateOnly(in.EndDate)
	if end.Before(start) {
		return nil, leavecommon.Invalid("end_date must not be before start_date")
	}
	dayPart, err := ValidateDayPart(in.DayPart, start, end)
	if err != nil {
		return nil, err
	}
	if err := ValidateStartWindow(start, uc.clock()); err != nil {
		return nil, err
	}
	reason, err := leavecommon.NormalizeText(in.Reason, "reason", 1000)
	if err != nil {
		return nil, err
	}
	if in.ApproverID == userID {
		return nil, domainerrors.ErrInvalidLeaveApprover
	}

	lt, err := uc.types.GetByID(ctx, tenantID, in.LeaveTypeID)
	if err != nil {
		return nil, err
	}
	if !lt.IsActive {
		return nil, domainerrors.ErrLeaveTypeInactive
	}

	allowed, err := resolveApprovers(ctx, uc.profiles, uc.approvers, tenantID, userID)
	if err != nil {
		return nil, err
	}
	var approver *entity.LeaveApprover
	for _, a := range allowed {
		if a.UserID == in.ApproverID {
			approver = a
			break
		}
	}
	if approver == nil {
		return nil, domainerrors.ErrInvalidLeaveApprover
	}

	offDays, err := uc.offDays.WeeklyOffDays(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	// Only public-category holidays are company days off; optional ones are working days.
	holidayDates, err := ucshared.PublicHolidayDates(ctx, uc.holidays, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	days := entity.CountLeaveDays(start, end, dayPart, offDays, holidayDates)
	if days == 0 {
		return nil, leavecommon.Invalid("selected dates fall only on weekly off days or public holidays; choose at least one working day")
	}

	req := &entity.LeaveRequest{
		ID: uuid.New(), TenantID: tenantID, UserID: userID, LeaveTypeID: lt.ID,
		StartDate: start, EndDate: end, DayPart: dayPart, Reason: reason,
		ApproverID: in.ApproverID, Status: entity.LeaveStatusPending, Days: days,
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.requests.LockUser(txCtx, tenantID, userID); err != nil {
			return err
		}
		overlap, err := uc.requests.HasOverlap(txCtx, tenantID, userID, start, end, dayPart, uuid.Nil)
		if err != nil {
			return err
		}
		if overlap {
			return domainerrors.ErrLeaveOverlap
		}
		if err := uc.requests.Create(txCtx, tenantID, userID, req); err != nil {
			return err
		}
		if err := uc.audit.Record(txCtx, tenantID, userID, domainaudit.ActionLeaveRequestApply, domainaudit.EntityLeaveRequest, req.ID, map[string]any{
			"leave_type_id": req.LeaveTypeID, "start_date": start.Format("2006-01-02"), "end_date": end.Format("2006-01-02"),
			"day_part": dayPart, "days": days, "approver_id": req.ApproverID,
		}); err != nil {
			return err
		}
		// Published in the same transaction (outbox): the email is queued iff the request commits.
		// An enqueue failure intentionally rolls back the action; sending itself is async and retried by River.
		detail, err := uc.requests.GetDetail(txCtx, tenantID, req.ID)
		if err != nil {
			return err
		}
		return uc.events.Publish(txCtx, event.Event{
			ID: uuid.New(), TenantID: tenantID, EventType: event.EventTypeLeaveRequested,
			AggregateType: "leave_request", AggregateID: req.ID.String(),
			Payload: event.LeaveRequestedPayload{
				RequestID: req.ID, TenantID: tenantID, EmployeeID: userID, EmployeeName: detail.UserName,
				ApproverID: approver.UserID, ApproverEmail: approver.Email, ApproverName: approver.Name,
				LeaveType: lt.Name, StartDate: start, EndDate: end, DayPart: dayPart, Days: days,
				Reason: reason, ReviewURL: uc.urls.ReviewLink(approver.Kind),
			},
			OccurredAt: uc.clock(),
		})
	})
	if err != nil {
		return nil, err
	}
	return uc.requests.GetDetail(ctx, tenantID, req.ID)
}
