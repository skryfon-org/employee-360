package entity

import (
	"time"

	"github.com/google/uuid"
)

// Holiday represents a tenant-owned holiday calendar entry.
type Holiday struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	CategoryID  uuid.UUID `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	// CategoryIsPublic is the holiday category's is_public flag, populated by read queries
	// (read-only; not a column of holidays).
	CategoryIsPublic bool       `json:"-" gorm:"->"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CreatedBy        *uuid.UUID `json:"-"`
	UpdatedBy        *uuid.UUID `json:"-"`
	DeletedBy        *uuid.UUID `json:"-"`
	DeletedAt        *time.Time `json:"-"`
}
