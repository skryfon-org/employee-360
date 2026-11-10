// Package holiday holds API DTOs and usecase input types for holiday operations.
package holiday

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// UpdateSettingsRequest contains fields for updating holiday calendar settings.
// Note: HTTP DTOs in this file are intentional pre-work for G5 (HTTP handlers).
// WeekStartDay is *int with binding:"required,min=0,max=6" so an omitted JSON field
// fails validation rather than silently defaulting to 0 (Sunday).
type UpdateSettingsRequest struct {
	DefaultView  string `json:"default_view" binding:"required,oneof=calendar list"`
	DefaultYear  *int   `json:"default_year"`
	WeekStartDay *int   `json:"week_start_day" binding:"required,min=0,max=6"`
}

// SettingsResponse represents the holiday calendar settings presentation model.
// Note: HTTP DTOs in this file are intentional pre-work for G5 (HTTP handlers).
type SettingsResponse struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	DefaultView  string    `json:"default_view"`
	DefaultYear  *int      `json:"default_year"`
	WeekStartDay int       `json:"week_start_day"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToSettingsResponse maps a domain HolidayCalendarSettings entity to SettingsResponse.
func ToSettingsResponse(s *entity.HolidayCalendarSettings) SettingsResponse {
	return SettingsResponse{
		TenantID:     s.TenantID,
		DefaultView:  s.DefaultView,
		DefaultYear:  s.DefaultYear,
		WeekStartDay: s.WeekStartDay,
		UpdatedAt:    s.UpdatedAt,
	}
}

// UpdateSettingsInput holds validated input to update holiday calendar settings.
type UpdateSettingsInput struct {
	DefaultView  string
	DefaultYear  *int
	WeekStartDay int
}
