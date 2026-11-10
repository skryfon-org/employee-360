package entity

import (
	"time"

	"github.com/google/uuid"
)

// Department represents a tenant-scoped organizational grouping of users.
type Department struct {
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
