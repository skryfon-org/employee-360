package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// DashboardRepository provides read-only, tenant-scoped aggregate queries for
// the dashboards. All methods exclude soft-deleted rows. tenantID always comes
// from the verified request context via the handler, never from client input.
// Invitation states are derived against now.
type DashboardRepository interface {
	// TenantCounts aggregates users, invitations, departments and positions.
	TenantCounts(ctx context.Context, tenantID uuid.UUID, now time.Time) (*entity.TenantDashboardCounts, error)
	// UsersByRole returns per-role user counts (ordered by role name).
	UsersByRole(ctx context.Context, tenantID uuid.UUID) ([]entity.RoleUserCount, error)
}
