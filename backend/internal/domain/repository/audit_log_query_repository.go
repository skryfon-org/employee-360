package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// AuditLogFilter narrows an audit-log listing. Zero values mean "no filter".
type AuditLogFilter struct {
	// Action matches audit_logs.action exactly, or by prefix when ActionPrefix is true.
	Action       string
	ActionPrefix bool
	ActorUserID  *uuid.UUID
	EntityType   string
	// From is inclusive, To is inclusive.
	From *time.Time
	To   *time.Time
}

// AuditLogQueryRepository is the read-side port for the admin audit-log
// viewer. Every method is scoped by the explicit tenantID parameter.
type AuditLogQueryRepository interface {
	// List returns entries newest first (created_at DESC, id DESC) plus the
	// total number of matches.
	List(ctx context.Context, tenantID uuid.UUID, filter AuditLogFilter, limit, offset int) ([]*entity.AuditLogEntry, int64, error)
	// GetByID returns domainerrors.ErrAuditLogNotFound when the entry does not
	// exist in the tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.AuditLogEntry, error)
}
