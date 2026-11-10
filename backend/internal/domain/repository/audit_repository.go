package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// AuditRepository defines the data access methods for audit logs. Reads take
// tenantID explicitly; Create persists log.TenantID set by the usecase.
type AuditRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*entity.AuditLog, int64, error)
	ListByEntity(ctx context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID, limit, offset int) ([]*entity.AuditLog, int64, error)
}
