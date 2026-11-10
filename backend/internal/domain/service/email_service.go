package service

import (
	"context"
	"errors"
)

// ErrInvalidEmailMessage marks a permanent, non-retryable problem with an
// EmailMessage (missing recipient/template, unrenderable template, header
// injection attempt). Retrying can never succeed, so workers cancel the job.
var ErrInvalidEmailMessage = errors.New("invalid email message")

// EmailTemplateName identifies transactional email templates.
type EmailTemplateName string

const (
	EmailTemplatePasswordReset  EmailTemplateName = "PasswordReset"
	EmailTemplateUserInvitation EmailTemplateName = "UserInvitation"
	EmailTemplateWelcome        EmailTemplateName = "Welcome"
	EmailTemplateLeaveRequested EmailTemplateName = "LeaveRequested"
	EmailTemplateLeaveApproved  EmailTemplateName = "LeaveApproved"
	EmailTemplateLeaveRejected  EmailTemplateName = "LeaveRejected"
)

// EmailMessage defines the data structure for sending transactional emails.
type EmailMessage struct {
	To           string                 `json:"to"`
	Subject      string                 `json:"subject"`
	TemplateName EmailTemplateName      `json:"template_name"`
	TemplateData map[string]interface{} `json:"template_data"`
}

// EmailService is the interface for sending transactional emails.
// It is called exclusively by background workers, never directly by usecases.
type EmailService interface {
	Send(ctx context.Context, msg EmailMessage) error
}
