package department

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

// CreateDepartmentUseCase defines the contract for creating a department within the caller's tenant.
type CreateDepartmentUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.CreateDepartmentInput) (*entity.Department, error)
}
