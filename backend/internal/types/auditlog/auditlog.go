// Package auditlog holds API DTOs for the audit-log viewer.
package auditlog

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// AuditActorResponse identifies who performed an audited action.
type AuditActorResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	// Name is the actor's display name ("First Last"); may be empty.
	Name string `json:"name"`
}

// AuditLogResponse is the audit-log API presentation model.
type AuditLogResponse struct {
	ID         uuid.UUID `json:"id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	// EntityID is null when the entry has no target entity.
	EntityID *uuid.UUID `json:"entity_id"`
	// Actor is null for system-initiated actions.
	Actor *AuditActorResponse `json:"actor"`
	// Metadata is the stored JSON object with sensitive keys removed.
	Metadata  json.RawMessage `json:"metadata" swaggertype:"object"`
	CreatedAt time.Time       `json:"created_at"`
}

// ToAuditLogResponse maps a domain AuditLogEntry to AuditLogResponse.
func ToAuditLogResponse(e *entity.AuditLogEntry) AuditLogResponse {
	var actor *AuditActorResponse
	if e.Actor != nil {
		actor = &AuditActorResponse{ID: e.Actor.ID, Email: e.Actor.Email, Name: e.Actor.DisplayName()}
	}
	meta := json.RawMessage(e.Metadata)
	if len(meta) == 0 || !json.Valid(meta) {
		meta = json.RawMessage("{}")
	}
	var entityID *uuid.UUID
	if e.EntityID != nil && *e.EntityID != uuid.Nil {
		entityID = e.EntityID
	}
	return AuditLogResponse{
		ID: e.ID, Action: e.Action, EntityType: e.EntityType, EntityID: entityID,
		Actor: actor, Metadata: meta, CreatedAt: e.CreatedAt,
	}
}

// GetAuditLogQuery specifies the audit entry to look up.
type GetAuditLogQuery struct {
	ID uuid.UUID
}

// ListAuditLogsQuery holds pagination and filter parameters.
type ListAuditLogsQuery struct {
	Page     int
	PageSize int
	// Action is an exact action ("department.create") or, when it ends in ".",
	// a prefix ("department." matches every department action).
	Action      string
	ActorUserID *uuid.UUID
	EntityType  string
	From        *time.Time
	To          *time.Time
}

// ListAuditLogsResult contains one page of entries (newest first) and metadata.
type ListAuditLogsResult struct {
	Entries  []*entity.AuditLogEntry
	Total    int64
	Page     int
	PageSize int
}
