package entity

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Employee code format defaults and limits. A tenant without an
// organization_settings row uses the defaults.
const (
	DefaultEmployeeCodePrefix      = "EMP"
	DefaultEmployeeCodePadding     = 4
	DefaultEmployeeCodeStartNumber = int64(1)

	MaxEmployeeCodePrefixLen   = 10
	MaxEmployeeCodePadding     = 10
	MaxEmployeeCodeStartNumber = int64(9999999999) // 10 digits
	MaxEmployeeCodeLen         = 20

	// MaxWelcomeMessageLen is the maximum length (in characters) of the welcome message.
	MaxWelcomeMessageLen = 2000
)

// OrganizationSettings holds per-tenant organization preferences (one row per tenant).
type OrganizationSettings struct {
	ID                      uuid.UUID `json:"id"`
	TenantID                uuid.UUID `json:"tenant_id"`
	EmployeeCodePrefix      string    `json:"employee_code_prefix"`
	EmployeeCodePadding     int       `json:"employee_code_padding"`
	EmployeeCodeStartNumber int64     `json:"employee_code_start_number"`
	// EmployeeCodeNextNumber is the server-managed counter: the number the next
	// generated employee code will use.
	EmployeeCodeNextNumber int64 `json:"employee_code_next_number"`
	// EmployeeCodeLastIssuedNumber is the highest number ever generated (0 = none).
	EmployeeCodeLastIssuedNumber int64 `json:"employee_code_last_issued_number"`
	// WelcomeMessage is the optional plain-text message included in the Welcome
	// email; nil means the built-in default is used.
	WelcomeMessage *string `json:"welcome_message,omitempty"`
	// WeeklyOffDays lists the tenant's weekly off days as weekday numbers
	// (0 = Sunday .. 6 = Saturday). Stored as smallint[]; the persistence adapter
	// reads/writes it explicitly, hence gorm:"-".
	WeeklyOffDays []int      `json:"weekly_off_days" gorm:"-"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CreatedBy     *uuid.UUID `json:"-"`
	UpdatedBy     *uuid.UUID `json:"-"`
}

// DefaultWeeklyOffDays returns a fresh copy of the default off days (Sunday, Saturday).
func DefaultWeeklyOffDays() []int { return []int{0, 6} }

// DefaultOrganizationSettings returns the in-memory defaults for a tenant that
// has no settings row yet.
func DefaultOrganizationSettings(tenantID uuid.UUID) *OrganizationSettings {
	return &OrganizationSettings{
		TenantID:                tenantID,
		EmployeeCodePrefix:      DefaultEmployeeCodePrefix,
		EmployeeCodePadding:     DefaultEmployeeCodePadding,
		EmployeeCodeStartNumber: DefaultEmployeeCodeStartNumber,
		EmployeeCodeNextNumber:  DefaultEmployeeCodeStartNumber,
		WeeklyOffDays:           DefaultWeeklyOffDays(),
	}
}

// CodesIssued reports whether any employee code has been generated.
func (s *OrganizationSettings) CodesIssued() bool {
	return s.EmployeeCodeLastIssuedNumber > 0
}

// NextEmployeeCode previews the code the next generated employee will receive.
func (s *OrganizationSettings) NextEmployeeCode() string {
	return FormatEmployeeCode(s.EmployeeCodePrefix, s.EmployeeCodePadding, s.EmployeeCodeNextNumber)
}

// FormatEmployeeCode renders prefix + number left-padded with zeros to at least
// padding digits. Padding is a minimum width: longer numbers are not truncated.
func FormatEmployeeCode(prefix string, padding int, number int64) string {
	return prefix + fmt.Sprintf("%0*d", padding, number)
}

// EmployeeCodePattern returns the regexp a manually entered employee code must
// match for the given prefix: the prefix followed by one or more digits.
func EmployeeCodePattern(prefix string) *regexp.Regexp {
	return regexp.MustCompile("^" + regexp.QuoteMeta(prefix) + `[0-9]+$`)
}

// NormalizeEmployeeCodePrefix trims and uppercases a prefix.
func NormalizeEmployeeCodePrefix(p string) string {
	return strings.ToUpper(strings.TrimSpace(p))
}

// EmployeeCodeCounter is one value taken from a tenant's employee code counter,
// together with the format in force when it was taken.
type EmployeeCodeCounter struct {
	Prefix  string
	Padding int
	Number  int64
}

// Code renders the counter value with FormatEmployeeCode.
func (c EmployeeCodeCounter) Code() string {
	return FormatEmployeeCode(c.Prefix, c.Padding, c.Number)
}
