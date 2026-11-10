package entity

import (
	"time"

	"github.com/google/uuid"
)

// HolidayCalendarSettings holds per-tenant calendar preferences.
type HolidayCalendarSettings struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	DefaultView  string     `json:"default_view"`
	DefaultYear  *int       `json:"default_year,omitempty"`
	WeekStartDay int        `json:"week_start_day"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedBy    *uuid.UUID `json:"-"`
	UpdatedBy    *uuid.UUID `json:"-"`
}
