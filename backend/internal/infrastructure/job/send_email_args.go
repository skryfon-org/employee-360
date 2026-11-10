package job

import "github.com/skryfon-org/employee-360/backend/internal/domain/service"

// SendEmailKind is the River job kind for transactional email delivery.
const SendEmailKind = "send_email"

// SendEmailArgs is the single shared River job payload covering every
// email-triggering domain event. TenantID travels in the job args (river_job
// is infrastructure and has no tenant_id column).
type SendEmailArgs struct {
	TenantID     string                    `json:"tenant_id"`
	TemplateName service.EmailTemplateName `json:"template_name"`
	To           string                    `json:"to"`
	Subject      string                    `json:"subject"`
	TemplateData map[string]interface{}    `json:"template_data"`
}

// Kind implements river.JobArgs.
func (SendEmailArgs) Kind() string { return SendEmailKind }
