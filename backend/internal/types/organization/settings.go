// Package organization holds API DTOs and usecase input types for organization settings.
package organization

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// UpdateSettingsRequest is the PUT /organization/settings body. The code counter
// (employee_code_next_number) is server-managed and not accepted: unknown fields
// are ignored. Pointer fields with binding:"required" make an omitted field a 400
// instead of silently defaulting to zero.
type UpdateSettingsRequest struct {
	EmployeeCodePrefix      string `json:"employee_code_prefix" binding:"required"`
	EmployeeCodePadding     *int   `json:"employee_code_padding" binding:"required"`
	EmployeeCodeStartNumber *int64 `json:"employee_code_start_number" binding:"required"`
	// WelcomeMessage is optional plain text (max 2000 chars). Omitted/null leaves the
	// stored message unchanged; an empty (or blank) string clears it.
	WelcomeMessage *string `json:"welcome_message,omitempty"`
	// WeeklyOffDays optionally replaces the tenant's weekly off days (0 = Sunday ..
	// 6 = Saturday; unique, 1-6 entries). Omitted/null leaves them unchanged.
	WeeklyOffDays []int `json:"weekly_off_days,omitempty" example:"0,6"`
}

// SettingsResponse is the organization settings presentation model.
type SettingsResponse struct {
	TenantID                uuid.UUID `json:"tenant_id"`
	EmployeeCodePrefix      string    `json:"employee_code_prefix"`
	EmployeeCodePadding     int       `json:"employee_code_padding"`
	EmployeeCodeStartNumber int64     `json:"employee_code_start_number"`
	// EmployeeCodeNextNumber is the read-only server-managed counter.
	EmployeeCodeNextNumber int64 `json:"employee_code_next_number"`
	// EmployeeCodesIssued is true once at least one code has been generated; the
	// start number can then only be raised above the counter.
	EmployeeCodesIssued bool `json:"employee_codes_issued"`
	// EmployeeCodePreview is the code the next created employee will receive.
	EmployeeCodePreview string `json:"employee_code_preview"`
	// UpdatedAt is absent while the tenant still uses the defaults (no row yet).
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	// WelcomeMessage is the custom welcome email message; null when unset (the
	// built-in default is used).
	WelcomeMessage *string `json:"welcome_message"`
	// WeeklyOffDays are the weekly off days, 0 = Sunday .. 6 = Saturday (default [0,6]).
	WeeklyOffDays []int `json:"weekly_off_days" example:"0,6"`
}

// ToSettingsResponse maps the entity to its response model.
func ToSettingsResponse(s *entity.OrganizationSettings) SettingsResponse {
	r := SettingsResponse{
		TenantID:                s.TenantID,
		EmployeeCodePrefix:      s.EmployeeCodePrefix,
		EmployeeCodePadding:     s.EmployeeCodePadding,
		EmployeeCodeStartNumber: s.EmployeeCodeStartNumber,
		EmployeeCodeNextNumber:  s.EmployeeCodeNextNumber,
		EmployeeCodesIssued:     s.CodesIssued(),
		EmployeeCodePreview:     s.NextEmployeeCode(),
		WelcomeMessage:          s.WelcomeMessage,
		WeeklyOffDays:           offDaysOrDefault(s.WeeklyOffDays),
	}
	if !s.UpdatedAt.IsZero() {
		t := s.UpdatedAt
		r.UpdatedAt = &t
	}
	return r
}

func offDaysOrDefault(d []int) []int {
	if len(d) == 0 {
		return entity.DefaultWeeklyOffDays()
	}
	return d
}

// UpdateSettingsInput holds the raw input of the update usecase (validated there).
type UpdateSettingsInput struct {
	EmployeeCodePrefix      string
	EmployeeCodePadding     int
	EmployeeCodeStartNumber int64
	// WelcomeMessage nil = leave unchanged; blank = clear.
	WelcomeMessage *string
	// WeeklyOffDays nil = leave unchanged; otherwise validated and replaced.
	WeeklyOffDays []int
}

// WeeklyOffDaysResponse is the employee-readable weekly off days model
// (0 = Sunday .. 6 = Saturday). It deliberately exposes nothing else from the
// organization settings.
type WeeklyOffDaysResponse struct {
	WeeklyOffDays []int `json:"weekly_off_days" example:"0,6"`
}
