package service

import (
	"context"

	"github.com/google/uuid"
)

// AuditRecorder writes an audit-log entry synchronously. Implementations must
// participate in any transaction carried by ctx so the entry commits or rolls
// back with the caller's mutation. Constants for action/entityType live in
// internal/domain/audit.
type AuditRecorder interface {
	Record(ctx context.Context, tenantID, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, meta map[string]any) error
}
