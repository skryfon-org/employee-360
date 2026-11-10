package eventing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/job"
)

func TestDispatcher_MapsAllEmailEvents(t *testing.T) {
	tenantID := uuid.New()
	exp := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	d := NewDispatcher()

	cases := []struct {
		name     string
		evt      event.Event
		template service.EmailTemplateName
		urlKey   string
		url      string
	}{
		{"invited", event.Event{EventType: event.EventTypeUserInvited, Payload: event.UserInvitedPayload{TenantID: tenantID, Email: "a@x.io", RoleName: "employee", InviteURL: "http://i", ExpiresAt: exp}}, service.EmailTemplateUserInvitation, "invite_url", "http://i"},
		{"resent (pointer payload)", event.Event{EventType: event.EventTypeInvitationResent, Payload: &event.InvitationResentPayload{TenantID: tenantID, Email: "a@x.io", InviteURL: "http://r", ExpiresAt: exp}}, service.EmailTemplateUserInvitation, "invite_url", "http://r"},
		{"reset", event.Event{EventType: event.EventTypePasswordResetRequested, Payload: event.PasswordResetRequestedPayload{TenantID: tenantID, UserID: uuid.New(), Email: "a@x.io", UserName: "Ann Lee", ResetURL: "http://p", ExpiresAt: exp}}, service.EmailTemplatePasswordReset, "reset_url", "http://p"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := d.Dispatch(tc.evt)
			require.NoError(t, err)
			got, ok := args.(job.SendEmailArgs)
			require.True(t, ok)
			assert.Equal(t, tc.template, got.TemplateName)
			assert.Equal(t, "a@x.io", got.To)
			assert.Equal(t, tenantID.String(), got.TenantID)
			assert.Equal(t, tc.url, got.TemplateData[tc.urlKey])
			assert.Equal(t, "2030-01-02T03:04:05Z", got.TemplateData["expires_at"])
			assert.NotContains(t, got.TemplateData, "plain_token", "plaintext tokens must not be persisted in river_job.args")
			if tc.template == service.EmailTemplatePasswordReset {
				assert.Equal(t, "Ann Lee", got.TemplateData["user_name"])
			}
		})
	}
}

func TestDispatcher_RejectsUnknownEventAndBadPayload(t *testing.T) {
	d := NewDispatcher()
	_, err := d.Dispatch(event.Event{EventType: "Nope"})
	assert.ErrorIs(t, err, ErrUnsupportedEvent)
	_, err = d.Dispatch(event.Event{EventType: event.EventTypeUserInvited, Payload: "wrong"})
	assert.ErrorIs(t, err, ErrUnsupportedEvent)
}

func TestDispatcher_MapsUserActivatedToWelcome(t *testing.T) {
	tenantID := uuid.New()
	args, err := NewDispatcher().Dispatch(event.Event{EventType: event.EventTypeUserActivated, Payload: event.UserActivatedPayload{
		UserID: uuid.New(), TenantID: tenantID, Email: "a@x.io", FirstName: "Ann", LastName: "Lee",
		TenantName: "Acme", RoleName: "employee", LoginURL: "http://app/login", WelcomeMessage: "Hi there",
	}})
	require.NoError(t, err)
	got, ok := args.(job.SendEmailArgs)
	require.True(t, ok)
	assert.Equal(t, service.EmailTemplateWelcome, got.TemplateName)
	assert.Equal(t, "a@x.io", got.To)
	assert.Equal(t, tenantID.String(), got.TenantID)
	assert.Equal(t, "Ann Lee", got.TemplateData["user_name"])
	assert.Equal(t, "Acme", got.TemplateData["tenant_name"])
	assert.Equal(t, "http://app/login", got.TemplateData["login_url"])
	assert.Equal(t, "Hi there", got.TemplateData["welcome_message"])
}

func TestDispatcher_LeaveEvents(t *testing.T) {
	tenantID := uuid.New()
	start := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)

	t.Run("LeaveRequested goes to the approver", func(t *testing.T) {
		args, err := NewDispatcher().Dispatch(event.Event{EventType: event.EventTypeLeaveRequested, Payload: event.LeaveRequestedPayload{
			TenantID: tenantID, RequestID: uuid.New(), EmployeeName: " Ann Lee ", ApproverEmail: "mgr@example.com", ApproverName: "Bob",
			LeaveType: "Casual", StartDate: start, EndDate: end, DayPart: "full", Days: 2, Reason: "Trip", ReviewURL: "https://x/leave",
		}})
		require.NoError(t, err)
		a := args.(job.SendEmailArgs)
		assert.Equal(t, service.EmailTemplateLeaveRequested, a.TemplateName)
		assert.Equal(t, "mgr@example.com", a.To)
		assert.Equal(t, tenantID.String(), a.TenantID)
		assert.Equal(t, "Ann Lee", a.TemplateData["employee_name"])
		assert.Equal(t, "Casual", a.TemplateData["leave_type"])
		assert.Equal(t, "Tue, 10 Nov 2026", a.TemplateData["start_date"])
		assert.Equal(t, "Wed, 11 Nov 2026", a.TemplateData["end_date"])
		assert.Equal(t, "2 days", a.TemplateData["duration"])
		assert.Equal(t, "Trip", a.TemplateData["reason"])
		assert.Equal(t, "https://x/leave", a.TemplateData["review_url"])
	})

	for _, tc := range []struct {
		typ  event.EventType
		tmpl service.EmailTemplateName
	}{
		{event.EventTypeLeaveApproved, service.EmailTemplateLeaveApproved},
		{event.EventTypeLeaveRejected, service.EmailTemplateLeaveRejected},
	} {
		t.Run(string(tc.typ)+" goes to the employee (pointer payload)", func(t *testing.T) {
			args, err := NewDispatcher().Dispatch(event.Event{EventType: tc.typ, Payload: &event.LeaveDecisionPayload{
				TenantID: tenantID, EmployeeEmail: "ann@example.com", EmployeeName: "Ann", ApproverName: "Bob",
				LeaveType: "Casual", StartDate: start, EndDate: start, DayPart: "first_half", Days: 0.5, Comment: "ok", LeaveURL: "https://x/leave",
			}})
			require.NoError(t, err)
			a := args.(job.SendEmailArgs)
			assert.Equal(t, tc.tmpl, a.TemplateName)
			assert.Equal(t, "ann@example.com", a.To)
			assert.Equal(t, "Bob", a.TemplateData["approver_name"])
			assert.Equal(t, "ok", a.TemplateData["comment"])
			assert.Equal(t, "Half day (first half)", a.TemplateData["duration"])
		})
	}

	t.Run("wrong payload type is rejected", func(t *testing.T) {
		_, err := NewDispatcher().Dispatch(event.Event{EventType: event.EventTypeLeaveApproved, Payload: "x"})
		assert.ErrorIs(t, err, ErrUnsupportedEvent)
	})
}
