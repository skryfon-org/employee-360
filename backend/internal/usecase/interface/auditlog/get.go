package auditlog

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	altypes "github.com/skryfon-org/employee-360/backend/internal/types/auditlog"
)

// GetAuditLogUseCase retrieves one audit entry within the caller's tenant.
type GetAuditLogUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, input altypes.GetAuditLogQuery) (*entity.AuditLogEntry, error)
}
