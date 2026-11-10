package entity

import (
	"time"

	"github.com/google/uuid"
)

// Platform role names.
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleEmployee   = "employee"
)

// Role defines an authorization role within a tenant.
type Role struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}
