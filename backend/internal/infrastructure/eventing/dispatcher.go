// Package eventing bridges domain events to the River-backed transactional outbox.
package eventing

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/riverqueue/river"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/job"
)

// ErrUnsupportedEvent is returned for events the dispatcher has no mapping for.
var ErrUnsupportedEvent = errors.New("eventing: unsupported event type or payload")

// Dispatcher maps domain events to River job args.
type Dispatcher struct{}

// NewDispatcher constructs a Dispatcher.
func NewDispatcher() *Dispatcher { return &Dispatcher{} }

// Dispatch converts an event into the River job args that deliver it.
func (d *Dispatcher) Dispatch(evt event.Event) (river.JobArgs, error) {
	switch evt.EventType {
	case event.EventTypeUserInvited:
		p, ok := payloadAs[event.UserInvitedPayload](evt.Payload)
		if !ok {
			return nil, fmt.Errorf("%w: %s payload %T", ErrUnsupportedEvent, evt.EventType, evt.Payload)
		}
		return invitationArgs(p.TenantID.String(), p.Email, p.RoleName, p.InviteURL, p.ExpiresAt), nil
	case event.EventTypeInvitationResent:
		p, ok := payloadAs[event.InvitationResentPayload](evt.Payload)
		if !ok {
			return nil, fmt.Errorf("%w: %s payload %T", ErrUnsupportedEvent, evt.EventType, evt.Payload)
		}
		return invitationArgs(p.TenantID.String(), p.Email, p.RoleName, p.InviteURL, p.ExpiresAt), nil
	case event.EventTypePasswordResetRequested:
		p, ok := payloadAs[event.PasswordResetRequestedPayload](evt.Payload)
		if !ok {
			return nil, fmt.Errorf("%w: %s payload %T", ErrUnsupportedEvent, evt.EventType, evt.Payload)
		}
		return job.SendEmailArgs{
			TenantID:     p.TenantID.String(),
			TemplateName: service.EmailTemplatePasswordReset,
			To:           p.Email,
			TemplateData: map[string]interface{}{
				"email":      p.Email,
				"user_name":  strings.TrimSpace(p.UserName),
				"user_id":    p.UserID.String(),
				"tenant_id":  p.TenantID.String(),
				"reset_url":  p.ResetURL,
				"expires_at": p.ExpiresAt.UTC().Format(time.RFC3339),
			},
		}, nil
	case event.EventTypeUserActivated:
		p, ok := payloadAs[event.UserActivatedPayload](evt.Payload)
		if !ok {
			return nil, fmt.Errorf("%w: %s payload %T", ErrUnsupportedEvent, evt.EventType, evt.Payload)
		}
		return job.SendEmailArgs{
			TenantID:     p.TenantID.String(),
			TemplateName: service.EmailTemplateWelcome,
			To:           p.Email,
			TemplateData: map[string]interface{}{
				"email":           p.Email,
				"user_id":         p.UserID.String(),
				"tenant_id":       p.TenantID.String(),
				"user_name":       strings.TrimSpace(p.FirstName + " " + p.LastName),
				"tenant_name":     p.TenantName,
				"role_name":       p.RoleName,
				"login_url":       p.LoginURL,
				"welcome_message": p.WelcomeMessage,
			},
		}, nil
	case event.EventTypeLeaveRequested:
		p, ok := payloadAs[event.LeaveRequestedPayload](evt.Payload)
		if !ok {
			return nil, fmt.Errorf("%w: %s payload %T", ErrUnsupportedEvent, evt.EventType, evt.Payload)
		}
		return job.SendEmailArgs{
			TenantID:     p.TenantID.String(),
			TemplateName: service.EmailTemplateLeaveRequested,
			To:           p.ApproverEmail,
			TemplateData: map[string]interface{}{
				"email":         p.ApproverEmail,
				"tenant_id":     p.TenantID.String(),
				"request_id":    p.RequestID.String(),
				"approver_name": strings.TrimSpace(p.ApproverName),
				"employee_name": strings.TrimSpace(p.EmployeeName),
				"leave_type":    p.LeaveType,
				"start_date":    formatLeaveDate(p.StartDate),
				"end_date":      formatLeaveDate(p.EndDate),
				"duration":      formatLeaveDuration(p.Days, p.DayPart),
				"reason":        p.Reason,
				"review_url":    p.ReviewURL,
			},
		}, nil
	case event.EventTypeLeaveApproved, event.EventTypeLeaveRejected:
		p, ok := payloadAs[event.LeaveDecisionPayload](evt.Payload)
		if !ok {
			return nil, fmt.Errorf("%w: %s payload %T", ErrUnsupportedEvent, evt.EventType, evt.Payload)
		}
		tmpl := service.EmailTemplateLeaveApproved
		if evt.EventType == event.EventTypeLeaveRejected {
			tmpl = service.EmailTemplateLeaveRejected
		}
		return job.SendEmailArgs{
			TenantID:     p.TenantID.String(),
			TemplateName: tmpl,
			To:           p.EmployeeEmail,
			TemplateData: map[string]interface{}{
				"email":         p.EmployeeEmail,
				"tenant_id":     p.TenantID.String(),
				"request_id":    p.RequestID.String(),
				"employee_name": strings.TrimSpace(p.EmployeeName),
				"approver_name": strings.TrimSpace(p.ApproverName),
				"leave_type":    p.LeaveType,
				"start_date":    formatLeaveDate(p.StartDate),
				"end_date":      formatLeaveDate(p.EndDate),
				"duration":      formatLeaveDuration(p.Days, p.DayPart),
				"comment":       strings.TrimSpace(p.Comment),
				"leave_url":     p.LeaveURL,
			},
		}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedEvent, evt.EventType)
	}
}

func invitationArgs(tenantID, email, role, inviteURL string, expiresAt time.Time) job.SendEmailArgs {
	return job.SendEmailArgs{
		TenantID:     tenantID,
		TemplateName: service.EmailTemplateUserInvitation,
		To:           email,
		TemplateData: map[string]interface{}{
			"email":      email,
			"tenant_id":  tenantID,
			"role_name":  role,
			"invite_url": inviteURL,
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
		},
	}
}

func formatLeaveDate(t time.Time) string { return t.UTC().Format("Mon, 02 Jan 2006") }

// formatLeaveDuration renders a human duration such as "2 days" or "Half day (first half)".
func formatLeaveDuration(days float64, dayPart string) string {
	switch dayPart {
	case entity.LeaveDayPartFirstHalf:
		return "Half day (first half)"
	case entity.LeaveDayPartSecondHalf:
		return "Half day (second half)"
	}
	n := strconv.FormatFloat(days, 'f', -1, 64)
	if days == 1 {
		return n + " day"
	}
	return n + " days"
}

// payloadAs accepts a payload as either a value or a pointer of type T.
func payloadAs[T any](v interface{}) (T, bool) {
	switch p := v.(type) {
	case T:
		return p, true
	case *T:
		if p != nil {
			return *p, true
		}
	}
	var zero T
	return zero, false
}
