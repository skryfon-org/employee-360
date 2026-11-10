package department

import (
	"context"

	"github.com/google/uuid"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

// ListDepartmentsUseCase defines the contract for listing departments within the caller's tenant.
type ListDepartmentsUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error)
}
