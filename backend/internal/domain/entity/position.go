package entity

import (
	"time"

	"github.com/google/uuid"
)

// Position represents a tenant-scoped job title or role grouping for users.
type Position struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   *uuid.UUID `json:"-"`
	UpdatedBy   *uuid.UUID `json:"-"`
	DeletedBy   *uuid.UUID `json:"-"`
	DeletedAt   *time.Time `json:"-"`
}
