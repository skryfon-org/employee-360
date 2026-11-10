package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// ListAssignmentsUseCase returns department/position history, newest first (admin).
type ListAssignmentsUseCase interface {
	Execute(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.UserAssignment, error)
}
