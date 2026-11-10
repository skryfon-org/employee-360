package entity

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog tracks actions and mutations performed within a tenant.
type AuditLog struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ActorUserID *uuid.UUID `json:"actor_user_id,omitempty"`
	Action      string     `json:"action"`
	EntityType  string     `json:"entity_type"`
	EntityID    uuid.UUID  `json:"entity_id"`
	// Metadata holds a JSON-encoded blob of additional context for the action
	// (e.g. changed fields, request metadata). It is persisted as-is into the
	// audit_logs.metadata jsonb column.
	Metadata  string    `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
