package department

import (
	"context"

	"github.com/google/uuid"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

// DeleteDepartmentUseCase defines the contract for deleting a department within the caller's tenant.
type DeleteDepartmentUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.DeleteDepartmentInput) error
}
