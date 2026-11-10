package entity

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents an organization or company tenant in the platform.
type Tenant struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}
