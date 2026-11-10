package auth

import "github.com/google/uuid"

// VerifiedIdentity is the database-confirmed identity of an authenticated
// caller: the tenant and user exist and are usable, and Roles are the user's
// CURRENT role names (not the possibly stale ones carried in the JWT).
type VerifiedIdentity struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Roles    []string
}
