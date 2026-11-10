package entity

import (
	"time"

	"github.com/google/uuid"
)

// Leave request statuses.
const (
	LeaveStatusPending   = "pending"
	LeaveStatusApproved  = "approved"
	LeaveStatusRejected  = "rejected"
	LeaveStatusCancelled = "cancelled"
)

// LeaveStatuses lists the allowed leave request statuses.
var LeaveStatuses = []string{LeaveStatusPending, LeaveStatusApproved, LeaveStatusRejected, LeaveStatusCancelled}

// Leave day parts. Half-day parts are only valid when start_date = end_date.
const (
	LeaveDayPartFull       = "full"
	LeaveDayPartFirstHalf  = "first_half"
	LeaveDayPartSecondHalf = "second_half"
)

// LeaveDayParts lists the allowed day parts.
var LeaveDayParts = []string{LeaveDayPartFull, LeaveDayPartFirstHalf, LeaveDayPartSecondHalf}

// LeaveRequest is an employee's request for leave, decided by its approver.
// Date fields are calendar dates (UTC midnight).
type LeaveRequest struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	UserID          uuid.UUID  `json:"user_id"`
	LeaveTypeID     uuid.UUID  `json:"leave_type_id"`
	StartDate       time.Time  `json:"start_date"`
	EndDate         time.Time  `json:"end_date"`
	DayPart         string     `json:"day_part"`
	Reason          string     `json:"reason"`
	ApproverID      uuid.UUID  `json:"approver_id"`
	Status          string     `json:"status"`
	DecidedBy       *uuid.UUID `json:"decided_by,omitempty"`
	DecidedAt       *time.Time `json:"decided_at,omitempty"`
	DecisionComment string     `json:"decision_comment"`
	// Days is the working-day count (tenant weekly off days and holidays excluded) fixed at
	// apply time; a half day is 0.5.
	Days float64 `json:"days"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// CountLeaveDays returns the working days a request covers: weekly off days
// (0 = Sunday .. 6 = Saturday) inside [start, end] are not counted. A half day
// counts 0.5 when it falls on a working day, else 0. Holiday dates (any time of
// day; only the calendar date matters) are non-working like weekly off days.
// This is the single leave-day counting rule.
func CountLeaveDays(start, end time.Time, dayPart string, offDays []int, holidays []time.Time) float64 {
	off := make(map[time.Weekday]bool, len(offDays))
	for _, d := range offDays {
		off[time.Weekday(d)] = true
	}
	hol := make(map[string]bool, len(holidays))
	for _, h := range holidays {
		hol[h.Format("2006-01-02")] = true
	}
	s := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	e := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	half := dayPart != LeaveDayPartFull && dayPart != ""
	n := 0.0
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		if off[d.Weekday()] || hol[d.Format("2006-01-02")] {
			continue
		}
		if half {
			return 0.5
		}
		n++
	}
	return n
}

// LeaveRequestDetail joins a request with display names of its applicant,
// approver and leave type.
type LeaveRequestDetail struct {
	LeaveRequest
	UserName     string `json:"user_name"`
	UserEmail    string `json:"user_email"`
	ApproverName string `json:"approver_name"`
	// DecidedByName is the decider's display name; empty when undecided.
	DecidedByName string `json:"decided_by_name"`
	LeaveTypeName string `json:"leave_type_name"`
	LeaveTypeCode string `json:"leave_type_code"`
}

// LeaveApprover is a user a leave request can be addressed to.
type LeaveApprover struct {
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	// Kind is "manager" or "admin".
	Kind string `json:"kind"`
}

// Leave approver kinds.
const (
	LeaveApproverManager = "manager"
	LeaveApproverAdmin   = "admin"
)
