// Package auditlog declares the read-only audit-log viewer usecase contracts.
package auditlog

import (
	"context"

	"github.com/google/uuid"
	altypes "github.com/skryfon-org/employee-360/backend/internal/types/auditlog"
)

// ListAuditLogsUseCase lists the caller tenant's audit entries.
type ListAuditLogsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, input altypes.ListAuditLogsQuery) (*altypes.ListAuditLogsResult, error)
}
