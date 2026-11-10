package event

import (
	"time"

	"github.com/google/uuid"
)

// EventType identifies the kind of domain event.
type EventType string

const (
	EventTypeUserInvited            EventType = "UserInvited"
	EventTypeInvitationResent       EventType = "InvitationResent"
	EventTypePasswordResetRequested EventType = "PasswordResetRequested"
	EventTypeUserActivated          EventType = "UserActivated"
	EventTypeLeaveRequested         EventType = "LeaveRequested"
	EventTypeLeaveApproved          EventType = "LeaveApproved"
	EventTypeLeaveRejected          EventType = "LeaveRejected"
)

// Event represents an immutable domain event occurring within a tenant.
type Event struct {
	ID            uuid.UUID   `json:"id"`
	TenantID      uuid.UUID   `json:"tenant_id"`
	EventType     EventType   `json:"event_type"`
	AggregateType string      `json:"aggregate_type"`
	AggregateID   string      `json:"aggregate_id"`
	Payload       interface{} `json:"payload"`
	OccurredAt    time.Time   `json:"occurred_at"`
}

// UserInvitedPayload holds the data carried by a UserInvited domain event.
type UserInvitedPayload struct {
	InvitationID uuid.UUID `json:"invitation_id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Email        string    `json:"email"`
	RoleName     string    `json:"role_name"`
	PlainToken   string    `json:"plain_token"`
	InviteURL    string    `json:"invite_url"`
	InvitedBy    uuid.UUID `json:"invited_by"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// InvitationResentPayload holds the data carried by an InvitationResent domain event.
type InvitationResentPayload struct {
	InvitationID uuid.UUID `json:"invitation_id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Email        string    `json:"email"`
	RoleName     string    `json:"role_name"`
	PlainToken   string    `json:"plain_token"`
	InviteURL    string    `json:"invite_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// PasswordResetRequestedPayload holds the data carried by a PasswordResetRequested domain event.
type PasswordResetRequestedPayload struct {
	UserID     uuid.UUID `json:"user_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Email      string    `json:"email"`
	UserName   string    `json:"user_name,omitempty"`
	PlainToken string    `json:"plain_token"`
	ResetURL   string    `json:"reset_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// UserActivatedPayload holds the data carried by a UserActivated domain event
// (an invited user accepted their invitation). WelcomeMessage is the tenant's
// custom message, empty when the built-in default should be used.
type UserActivatedPayload struct {
	UserID         uuid.UUID `json:"user_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	TenantName     string    `json:"tenant_name"`
	RoleName       string    `json:"role_name"`
	LoginURL       string    `json:"login_url"`
	WelcomeMessage string    `json:"welcome_message,omitempty"`
}

// LeaveRequestedPayload holds the data carried by a LeaveRequested domain event.
// The recipient is the request's approver; ReviewURL links to the review page.
type LeaveRequestedPayload struct {
	RequestID     uuid.UUID `json:"request_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	EmployeeID    uuid.UUID `json:"employee_id"`
	EmployeeName  string    `json:"employee_name"`
	ApproverID    uuid.UUID `json:"approver_id"`
	ApproverEmail string    `json:"approver_email"`
	ApproverName  string    `json:"approver_name"`
	LeaveType     string    `json:"leave_type"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	DayPart       string    `json:"day_part"`
	Days          float64   `json:"days"`
	Reason        string    `json:"reason"`
	ReviewURL     string    `json:"review_url"`
}

// LeaveDecisionPayload holds the data carried by LeaveApproved and LeaveRejected
// domain events. The recipient is the requesting employee.
type LeaveDecisionPayload struct {
	RequestID     uuid.UUID `json:"request_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	EmployeeID    uuid.UUID `json:"employee_id"`
	EmployeeEmail string    `json:"employee_email"`
	EmployeeName  string    `json:"employee_name"`
	ApproverName  string    `json:"approver_name"`
	LeaveType     string    `json:"leave_type"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	DayPart       string    `json:"day_part"`
	Days          float64   `json:"days"`
	Comment       string    `json:"comment,omitempty"`
	LeaveURL      string    `json:"leave_url"`
}
