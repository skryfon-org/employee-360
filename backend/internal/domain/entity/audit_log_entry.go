package entity

import (
	"time"

	"github.com/google/uuid"
)

// AuditActor is the resolved identity of the user who performed an audited
// action, looked up within the audit entry's own tenant.
type AuditActor struct {
	ID        uuid.UUID
	Email     string
	FirstName string
	LastName  string
}

// DisplayName returns "First Last", falling back to whichever part is set.
func (a *AuditActor) DisplayName() string {
	switch {
	case a.FirstName == "":
		return a.LastName
	case a.LastName == "":
		return a.FirstName
	default:
		return a.FirstName + " " + a.LastName
	}
}

// AuditLogEntry is a read model: an AuditLog row plus its resolved actor.
// Actor is nil for system-initiated actions (no actor_user_id).
type AuditLogEntry struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	Actor      *AuditActor
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	// Metadata is the JSON blob as stored (usecases sanitize it before exposure).
	Metadata  string
	CreatedAt time.Time
}
