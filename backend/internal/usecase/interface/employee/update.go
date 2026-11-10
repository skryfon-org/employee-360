package employee

import (
	"context"

	"github.com/google/uuid"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// UpdateEmployeeUseCase updates user fields and upserts the profile, recording
// assignment history when department/position change (admin).
type UpdateEmployeeUseCase interface {
	Execute(ctx context.Context, actor emptypes.Actor, userID uuid.UUID, in emptypes.UpdateEmployeeInput) (*emptypes.EmployeeDetail, error)
}
