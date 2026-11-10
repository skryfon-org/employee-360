package dashboard

import (
	"context"

	"github.com/google/uuid"
	dashtypes "github.com/skryfon-org/employee-360/backend/internal/types/dashboard"
)

// AdminDashboardUseCase builds the tenant admin dashboard for the caller's tenant.
type AdminDashboardUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID) (*dashtypes.AdminDashboardResult, error)
}

// SuperAdminDashboardUseCase builds the super admin overview of the caller's
// tenant (admin dashboard plus tenant info and users by role). Callers must
// already have passed the super_admin role guard.
type SuperAdminDashboardUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID) (*dashtypes.SuperAdminDashboardResult, error)
}

// EmployeeDashboardUseCase builds the read-only home dashboard of one employee
// within their tenant. Both IDs are resolved by the handler from the auth context.
type EmployeeDashboardUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) (*dashtypes.EmployeeDashboardResponse, error)
}
