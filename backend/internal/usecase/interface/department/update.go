package department

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

// UpdateDepartmentUseCase defines the contract for updating a department within the caller's tenant.
type UpdateDepartmentUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.UpdateDepartmentInput) (*entity.Department, error)
}
