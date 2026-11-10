package repository

import (
	"context"

	"github.com/google/uuid"
)

// IdentityState is a point-in-time snapshot of everything needed to decide
// whether an authenticated identity is still usable. It is produced by a
// single read so the auth middleware pays one query per request.
type IdentityState struct {
	// TenantActive is the tenants.is_active flag of the (non-deleted) tenant.
	TenantActive bool
	// UserFound is true when a non-deleted user with the requested id exists
	// IN the requested tenant.
	UserFound bool
	// UserActive is the users.is_active flag; meaningful only when UserFound.
	UserActive bool
	// RoleNames are the user's CURRENT (non-deleted) role names, ordered by
	// name. Empty when the user has no roles.
	RoleNames []string
}

// IdentityReader is the read-only port used to verify a token's tenant/user
// pair against the database in one round trip.
type IdentityReader interface {
	// GetIdentityState loads the identity snapshot scoped by BOTH tenantID and
	// userID: a user belonging to another tenant is reported as not found.
	// It returns domainerrors.ErrTenantNotFound when no non-deleted tenant has
	// the given id. Soft-deleted tenants, users and roles are excluded.
	GetIdentityState(ctx context.Context, tenantID, userID uuid.UUID) (*IdentityState, error)
}
