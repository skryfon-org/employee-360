package entity

import (
	"time"

	"github.com/google/uuid"
)

// User represents an individual user within a tenant organization.
type User struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	DepartmentID    *uuid.UUID `json:"department_id,omitempty"`
	PositionID      *uuid.UUID `json:"position_id,omitempty"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	Email           string     `json:"email"`
	PasswordHash    *string    `json:"-"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *uuid.UUID `json:"-"`
	UpdatedBy       *uuid.UUID `json:"-"`
	DeletedBy       *uuid.UUID `json:"-"`
	DeletedAt       *time.Time `json:"-"`

	// Associations (populated when loaded).
	//
	// gorm:"-" is required, not cosmetic: without it GORM's default schema
	// parser tries to guess a many2many association for this field and
	// fails at Create/Update time ("invalid field found ... define a valid
	// foreign key for relations or implement the Valuer/Scanner interface"),
	// since there is no matching Role.Users/UserID field for it to infer a
	// relationship from. Roles is populated manually by the persistence
	// layer via an explicit join query instead (see
	// internal/infrastructure/persistence/user_repository.go's loadRoles),
	// so GORM must never treat it as a column or association on its own.
	Roles []Role `json:"roles,omitempty" gorm:"-"`
}

// FullName returns the concatenated first and last name.
func (u *User) FullName() string {
	if u.FirstName == "" {
		return u.LastName
	}
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}
