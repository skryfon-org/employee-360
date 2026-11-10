package entity

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the assignment of a Role to a User within a tenant.
type UserRole struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`

	// Associations (populated when loaded)
	User *User `json:"user,omitempty"`
	Role *Role `json:"role,omitempty"`
}
