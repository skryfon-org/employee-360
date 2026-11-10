package department

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

// GetDepartmentUseCase defines the contract for retrieving a department by ID within the caller's tenant.
type GetDepartmentUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, input depttypes.GetDepartmentQuery) (*entity.Department, error)
}
