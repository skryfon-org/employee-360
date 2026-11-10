// Package job contains River background workers.
package job

import (
	"context"
	"errors"

	"github.com/riverqueue/river"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

// EmailWorker delivers queued transactional emails. It is the only caller of
// EmailService.Send in the codebase.
type EmailWorker struct {
	river.WorkerDefaults[SendEmailArgs]
	emailService service.EmailService
}

// NewEmailWorker constructs an EmailWorker.
func NewEmailWorker(emailService service.EmailService) *EmailWorker {
	return &EmailWorker{emailService: emailService}
}

// Work sends the email. Permanent validation failures cancel the job (no
// retries); every other error is returned so River applies its default
// retry/backoff.
func (w *EmailWorker) Work(ctx context.Context, j *river.Job[SendEmailArgs]) error {
	args := j.Args
	err := w.emailService.Send(ctx, service.EmailMessage{
		To:           args.To,
		Subject:      args.Subject,
		TemplateName: args.TemplateName,
		TemplateData: args.TemplateData,
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, service.ErrInvalidEmailMessage) {
		return river.JobCancel(err)
	}
	return err
}
