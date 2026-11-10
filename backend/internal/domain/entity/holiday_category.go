package entity

import (
	"time"

	"github.com/google/uuid"
)

// HolidayCategory represents a tenant-owned label used to group and colour-code holidays.
type HolidayCategory struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	IsActive    bool      `json:"is_active"`
	// IsPublic marks the category as company days off: its holidays are excluded from leave day counts.
	IsPublic  bool       `json:"is_public"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}
