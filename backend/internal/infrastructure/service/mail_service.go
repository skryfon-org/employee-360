package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"path"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/skryfon-org/employee-360/backend/config"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

//go:embed mail/templates/*
var defaultTemplateFS embed.FS

// SMTPDialer is the function signature for dialing a network connection with context.
type SMTPDialer func(ctx context.Context, network, addr string) (net.Conn, error)

// TLSDialer is the function signature for dialing a TLS network connection with context.
type TLSDialer func(ctx context.Context, network, addr string, tlsConfig *tls.Config) (net.Conn, error)

// SMTPMailService is a provider-agnostic SMTP implementation of the domain EmailService interface.
// It is called exclusively by background workers and never directly by usecases.
type SMTPMailService struct {
	cfg        config.SMTPConfig
	templateFS embed.FS
	dialer     SMTPDialer
	tlsDialer  TLSDialer
}

// Option allows configuring optional parameters of SMTPMailService.
type Option func(*SMTPMailService)

// WithTemplateFS overrides the template file system.
func WithTemplateFS(fs embed.FS) Option {
	return func(s *SMTPMailService) {
		s.templateFS = fs
	}
}

// WithCustomDialer overrides the default network dialer (useful for testing).
func WithCustomDialer(dialer SMTPDialer, tlsDialer TLSDialer) Option {
	return func(s *SMTPMailService) {
		s.dialer = dialer
		s.tlsDialer = tlsDialer
	}
}

// NewSMTPMailService creates a new provider-agnostic SMTP EmailService instance.
func NewSMTPMailService(cfg config.SMTPConfig, opts ...Option) *SMTPMailService {
	svc := &SMTPMailService{
		cfg:        cfg,
		templateFS: defaultTemplateFS,
		dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 15 * time.Second}
			return d.DialContext(ctx, network, addr)
		},
		tlsDialer: func(ctx context.Context, network, addr string, tlsConfig *tls.Config) (net.Conn, error) {
			d := &net.Dialer{Timeout: 15 * time.Second}
			return tls.DialWithDialer(d, network, addr, tlsConfig)
		},
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

// Send renders the appropriate email template and sends the transactional email via plain SMTP.
func (s *SMTPMailService) Send(ctx context.Context, msg domainservice.EmailMessage) error {
	if strings.TrimSpace(msg.To) == "" {
		return fmt.Errorf("%w: recipient (To) cannot be empty", domainservice.ErrInvalidEmailMessage)
	}

	if msg.TemplateName == "" {
		return fmt.Errorf("%w: template name cannot be empty", domainservice.ErrInvalidEmailMessage)
	}

	// Normalize data for consistent template variable access
	data := normalizeTemplateData(msg.TemplateData)

	// Render Subject
	subject := msg.Subject
	if subject == "" {
		renderedSubject, err := s.renderSubjectTemplate(msg.TemplateName, data)
		if err != nil {
			return fmt.Errorf("%w: failed to render email subject template %q: %w", domainservice.ErrInvalidEmailMessage, msg.TemplateName, err)
		}
		subject = renderedSubject
	} else {
		// If subject was provided, support optional template execution within subject string
		if strings.Contains(subject, "{{") {
			tmpl, err := texttemplate.New("subject").Parse(subject)
			if err != nil {
				return fmt.Errorf("%w: failed to parse custom email subject template: %w", domainservice.ErrInvalidEmailMessage, err)
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				return fmt.Errorf("%w: failed to execute custom email subject template: %w", domainservice.ErrInvalidEmailMessage, err)
			}
			subject = strings.TrimSpace(buf.String())
		}
	}

	// Render Plain Text Body
	textBody, err := s.renderTextTemplate(msg.TemplateName, data)
	if err != nil {
		return fmt.Errorf("%w: failed to render email text template %q: %w", domainservice.ErrInvalidEmailMessage, msg.TemplateName, err)
	}

	// Render HTML Body
	htmlBody, err := s.renderHTMLTemplate(msg.TemplateName, data)
	if err != nil {
		return fmt.Errorf("%w: failed to render email HTML template %q: %w", domainservice.ErrInvalidEmailMessage, msg.TemplateName, err)
	}

	// Build MIME message
	from := s.cfg.From
	if from == "" {
		from = "no-reply@employee360.local"
	}

	rawMessage, err := s.buildMIMEMessage(from, msg.To, subject, textBody, htmlBody)
	if err != nil {
		return fmt.Errorf("%w: failed to build MIME email message: %w", domainservice.ErrInvalidEmailMessage, err)
	}

	// Transmit message via SMTP
	fromAddress := extractEmailAddress(from)
	toAddress := extractEmailAddress(msg.To)
	if toAddress == "" {
		toAddress = msg.To
	}

	if err := s.sendViaSMTP(ctx, fromAddress, []string{toAddress}, rawMessage); err != nil {
		return fmt.Errorf("failed to deliver email via SMTP to %s: %w", msg.To, err)
	}

	return nil
}

// renderSubjectTemplate loads and renders the subject template file for the given template name.
func (s *SMTPMailService) renderSubjectTemplate(templateName domainservice.EmailTemplateName, data map[string]interface{}) (string, error) {
	filename := path.Join("mail/templates", fmt.Sprintf("%s.subject", templateName))
	content, err := s.templateFS.ReadFile(filename)
	if err != nil {
		// Fallback default subject based on template name if .subject file is missing
		switch templateName {
		case domainservice.EmailTemplatePasswordReset:
			return "Reset your Employee360 password", nil
		case domainservice.EmailTemplateUserInvitation:
			return "You have been invited to Employee360", nil
		case domainservice.EmailTemplateWelcome:
			return "Welcome to Employee360", nil
		case domainservice.EmailTemplateLeaveRequested:
			return "New leave request awaiting your review", nil
		case domainservice.EmailTemplateLeaveApproved:
			return "Your leave request was approved", nil
		case domainservice.EmailTemplateLeaveRejected:
			return "Your leave request was rejected", nil
		default:
			return string(templateName), nil
		}
	}

	tmpl, err := texttemplate.New(string(templateName) + "_subject").Parse(string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

// renderTextTemplate loads and renders the plain text template for the given template name.
func (s *SMTPMailService) renderTextTemplate(templateName domainservice.EmailTemplateName, data map[string]interface{}) (string, error) {
	filename := path.Join("mail/templates", fmt.Sprintf("%s.txt", templateName))
	content, err := s.templateFS.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("plain text template file %s not found: %w", filename, err)
	}

	tmpl, err := texttemplate.New(string(templateName) + "_text").Parse(string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderHTMLTemplate loads and renders the HTML template for the given template name.
func (s *SMTPMailService) renderHTMLTemplate(templateName domainservice.EmailTemplateName, data map[string]interface{}) (string, error) {
	filename := path.Join("mail/templates", fmt.Sprintf("%s.html", templateName))
	content, err := s.templateFS.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("HTML template file %s not found: %w", filename, err)
	}

	tmpl, err := htmltemplate.New(string(templateName) + "_html").Parse(string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// validateNoCRLF rejects header values containing embedded CR/LF (or other control
// characters) to prevent email header injection (CWE-93). net/smtp only validates
// SMTP *command* lines (MAIL FROM/RCPT TO) against embedded CRLF — it does not
// sanitize the raw DATA payload headers built by buildMIMEMessage, so any value
// that ends up in a header (From, To, Subject) must be checked here first.
func validateNoCRLF(fieldName, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%w: invalid %s header value: contains carriage return or line feed characters", domainservice.ErrInvalidEmailMessage, fieldName)
	}
	return nil
}

// buildMIMEMessage constructs a multipart/alternative RFC 2822 email message with text and HTML parts.
func (s *SMTPMailService) buildMIMEMessage(from, to, subject, textBody, htmlBody string) ([]byte, error) {
	if err := validateNoCRLF("From", from); err != nil {
		return nil, err
	}
	if err := validateNoCRLF("To", to); err != nil {
		return nil, err
	}
	if err := validateNoCRLF("Subject", subject); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	// Top-level headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))

	// Plain text part
	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/plain; charset=\"utf-8\"")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return nil, err
	}
	qpText := quotedprintable.NewWriter(textPart)
	if _, err := qpText.Write([]byte(textBody)); err != nil {
		return nil, err
	}
	if err := qpText.Close(); err != nil {
		return nil, err
	}

	// HTML part
	htmlHeader := make(textproto.MIMEHeader)
	htmlHeader.Set("Content-Type", "text/html; charset=\"utf-8\"")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return nil, err
	}
	qpHtml := quotedprintable.NewWriter(htmlPart)
	if _, err := qpHtml.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}
	if err := qpHtml.Close(); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// starttlsPorts are the conventional SMTP submission/relay ports that speak plaintext
// initially and then upgrade via STARTTLS: 587 (the modern submission port), 25
// (classic SMTP relay, which may or may not offer STARTTLS), and 2525 (a common
// alternate submission port used by some PaaS/relay providers). Port 465 is the
// IANA-registered implicit-TLS (SMTPS) port and is handled separately below.
var starttlsPorts = map[int]bool{
	587:  true,
	25:   true,
	2525: true,
}

// plaintextDevPorts are additional non-standard ports used by local dev SMTP relays
// that speak plaintext-only with no STARTTLS support (e.g. Mailpit's default port
// 1025). They are excluded from the implicit-TLS heuristic below so that a
// UseTLS=false dev config is never misclassified as requiring an SMTPS handshake.
var plaintextDevPorts = map[int]bool{
	1025: true, // Mailpit default
}

// sendViaSMTP establishes an SMTP connection, handles TLS / STARTTLS, authenticates, and sends the raw message.
func (s *SMTPMailService) sendViaSMTP(ctx context.Context, from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	host := s.cfg.Host

	var conn net.Conn
	var err error

	// Determine if implicit TLS (SMTPS, usually port 465) or plain/STARTTLS is used
	isImplicitTLS := s.cfg.Port == 465 || (s.cfg.UseTLS && !starttlsPorts[s.cfg.Port] && !plaintextDevPorts[s.cfg.Port])

	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}

	if isImplicitTLS {
		conn, err = s.tlsDialer(ctx, "tcp", addr, tlsConfig)
	} else {
		conn, err = s.dialer(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP host %s: %w", addr, err)
	}
	defer conn.Close()

	// Ensure connection closes on context cancellation
	doneCh := make(chan struct{})
	defer close(doneCh)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-doneCh:
		}
	}()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() {
		_ = client.Quit()
	}()

	// Handle STARTTLS for port 587 or explicit TLS request on non-implicit ports
	if !isImplicitTLS {
		hasStartTLS, _ := client.Extension("STARTTLS")
		switch {
		case hasStartTLS && (s.cfg.UseTLS || starttlsPorts[s.cfg.Port]):
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to initiate STARTTLS: %w", err)
			}
		case s.cfg.UseTLS:
			// The operator explicitly requested TLS (SMTP_USE_TLS=true) but the server
			// does not advertise STARTTLS support on this non-implicit-TLS port. Fail
			// closed rather than silently transmitting the message (including
			// password-reset URLs/tokens) in plaintext.
			return fmt.Errorf("SMTP server at %s does not support STARTTLS but TLS was explicitly requested (SMTP_USE_TLS=true); refusing to send in plaintext", addr)
		}
	}

	// Authenticate if credentials are provided
	if s.cfg.Username != "" && s.cfg.Password != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP command MAIL FROM failed: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP command RCPT TO <%s> failed: %w", recipient, err)
		}
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP command DATA failed: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("failed to stream email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to complete DATA command: %w", err)
	}

	return nil
}

// extractEmailAddress parses an email address like "Display Name <user@example.com>" to "user@example.com".
func extractEmailAddress(addr string) string {
	start := strings.Index(addr, "<")
	end := strings.Index(addr, ">")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(addr[start+1 : end])
	}
	return strings.TrimSpace(addr)
}

// normalizeTemplateData creates a normalized map supporting both snake_case and CamelCase keys.
func normalizeTemplateData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}

	// Ensure common aliases are mapped seamlessly.
	// NOTE: this table is hand-maintained — adding a new template variable that needs
	// to be accessible under both snake_case (Go map key convention used by callers)
	// and CamelCase (Go template field convention used inside .html/.txt/.subject
	// files) requires adding a corresponding snake_case -> CamelCase entry here, or
	// the variable will only be available under whichever casing the caller supplied.
	mappings := map[string]string{
		"reset_url":       "ResetURL",
		"invite_url":      "InviteURL",
		"tenant_name":     "TenantName",
		"role_name":       "RoleName",
		"plain_token":     "PlainToken",
		"expires_at":      "ExpiresAt",
		"expires_in":      "ExpiresIn",
		"email":           "Email",
		"user_name":       "UserName",
		"user_id":         "UserID",
		"tenant_id":       "TenantID",
		"invited_by":      "InvitedBy",
		"login_url":       "LoginURL",
		"welcome_message": "WelcomeMessage",
		"employee_name":   "EmployeeName",
		"approver_name":   "ApproverName",
		"leave_type":      "LeaveType",
		"start_date":      "StartDate",
		"end_date":        "EndDate",
		"duration":        "Duration",
		"reason":          "Reason",
		"comment":         "Comment",
		"review_url":      "ReviewURL",
		"leave_url":       "LeaveURL",
	}

	for snake, camel := range mappings {
		if val, ok := result[snake]; ok {
			if _, exists := result[camel]; !exists {
				result[camel] = val
			}
		}
		if val, ok := result[camel]; ok {
			if _, exists := result[snake]; !exists {
				result[snake] = val
			}
		}
	}

	return result
}
