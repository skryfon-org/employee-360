package leave

import (
	"time"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// ApplyLeaveRequest is the body of POST /me/leave-requests.
type ApplyLeaveRequest struct {
	LeaveTypeID uuid.UUID     `json:"leave_type_id" binding:"required"`
	StartDate   emptypes.Date `json:"start_date" binding:"required" swaggertype:"string" format:"date"`
	EndDate     emptypes.Date `json:"end_date" binding:"required" swaggertype:"string" format:"date"`
	// DayPart defaults to full; first_half/second_half require start_date = end_date.
	DayPart    string    `json:"day_part" enums:"full,first_half,second_half"`
	Reason     string    `json:"reason"`
	ApproverID uuid.UUID `json:"approver_id" binding:"required"`
}

// ApplyLeaveInput is the usecase input for applying for leave.
type ApplyLeaveInput struct {
	LeaveTypeID uuid.UUID
	StartDate   time.Time
	EndDate     time.Time
	DayPart     string
	Reason      string
	ApproverID  uuid.UUID
}

// DecisionRequest is the body of POST /leave-requests/:id/approve|reject.
type DecisionRequest struct {
	Comment string `json:"comment"`
}

// ListMyLeaveQuery holds the paging and status filter of the caller-scoped
// lists (GET /me/leave-requests, GET /leave-requests/team).
type ListMyLeaveQuery struct {
	Page, PageSize int
	Status         string
	// ExcludeStatus omits requests with this status (team listing only).
	ExcludeStatus string
	// From/To are raw YYYY-MM-DD strings; requests overlapping [From, To] match.
	From, To string
	// LeaveTypeID is a raw UUID string; empty means no filter.
	LeaveTypeID string
}

// ListLeaveRequestsQuery holds the admin listing filters (raw query strings).
type ListLeaveRequestsQuery struct {
	Page, PageSize int
	Status         string
	EmployeeID     string
	LeaveTypeID    string
	From, To       string
}

// LeaveRequestPage is a page of leave requests.
type LeaveRequestPage struct {
	Items    []*entity.LeaveRequestDetail
	Total    int64
	Page     int
	PageSize int
}

// LeaveRequestResponse is a leave request on the wire.
type LeaveRequestResponse struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	UserName      string    `json:"user_name"`
	UserEmail     string    `json:"user_email"`
	LeaveTypeID   uuid.UUID `json:"leave_type_id"`
	LeaveTypeName string    `json:"leave_type_name"`
	LeaveTypeCode string    `json:"leave_type_code"`
	StartDate     string    `json:"start_date"`
	EndDate       string    `json:"end_date"`
	DayPart       string    `json:"day_part" enums:"full,first_half,second_half"`
	// Days is the working-day count (tenant weekly off days excluded); a half day is 0.5.
	Days         float64    `json:"days"`
	Reason       string     `json:"reason"`
	ApproverID   uuid.UUID  `json:"approver_id"`
	ApproverName string     `json:"approver_name"`
	Status       string     `json:"status" enums:"pending,approved,rejected,cancelled"`
	DecidedBy    *uuid.UUID `json:"decided_by,omitempty"`
	// DecidedByName is the decider's display name; null when undecided.
	DecidedByName   *string    `json:"decided_by_name"`
	DecidedAt       *time.Time `json:"decided_at,omitempty"`
	DecisionComment string     `json:"decision_comment"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ToLeaveRequestResponse maps a detail row to its response.
func ToLeaveRequestResponse(d *entity.LeaveRequestDetail) LeaveRequestResponse {
	var decidedByName *string
	if d.DecidedBy != nil {
		n := d.DecidedByName
		decidedByName = &n
	}
	return LeaveRequestResponse{
		DecidedByName: decidedByName,
		ID:            d.ID, UserID: d.UserID, UserName: d.UserName, UserEmail: d.UserEmail,
		LeaveTypeID: d.LeaveTypeID, LeaveTypeName: d.LeaveTypeName, LeaveTypeCode: d.LeaveTypeCode,
		StartDate: d.StartDate.UTC().Format(emptypes.DateLayout), EndDate: d.EndDate.UTC().Format(emptypes.DateLayout),
		DayPart: d.DayPart, Days: d.Days, Reason: d.Reason, ApproverID: d.ApproverID, ApproverName: d.ApproverName,
		Status: d.Status, DecidedBy: d.DecidedBy, DecidedAt: d.DecidedAt, DecisionComment: d.DecisionComment,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// ToLeaveRequestResponses maps a slice of detail rows.
func ToLeaveRequestResponses(in []*entity.LeaveRequestDetail) []LeaveRequestResponse {
	out := make([]LeaveRequestResponse, 0, len(in))
	for _, d := range in {
		out = append(out, ToLeaveRequestResponse(d))
	}
	return out
}

// LeaveApproverResponse is a selectable approver.
type LeaveApproverResponse struct {
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	Kind   string    `json:"kind" enums:"manager,admin"`
}

// ToLeaveApproverResponses maps approvers to responses.
func ToLeaveApproverResponses(in []*entity.LeaveApprover) []LeaveApproverResponse {
	out := make([]LeaveApproverResponse, 0, len(in))
	for _, a := range in {
		out = append(out, LeaveApproverResponse{UserID: a.UserID, Name: a.Name, Email: a.Email, Kind: a.Kind})
	}
	return out
}
