package entity

import (
	"time"

	"github.com/google/uuid"
)

// TenantDomain maps an email domain to a tenant for tenant resolution.
type TenantDomain struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Domain    string     `json:"domain"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}
