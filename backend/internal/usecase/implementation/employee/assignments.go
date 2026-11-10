package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

type listAssignmentsUseCase struct {
	users       repository.UserRepository
	assignments repository.UserAssignmentRepository
}

// NewListAssignmentsUseCase creates a new ListAssignmentsUseCase.
func NewListAssignmentsUseCase(
	users repository.UserRepository,
	assignments repository.UserAssignmentRepository,
) empuc.ListAssignmentsUseCase {
	return &listAssignmentsUseCase{users: users, assignments: assignments}
}

func (uc *listAssignmentsUseCase) Execute(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.UserAssignment, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	// 404 for users outside the tenant (or soft-deleted).
	if _, err := uc.users.GetByID(ctx, tenantID, userID); err != nil {
		return nil, err
	}
	return uc.assignments.ListByUser(ctx, tenantID, userID)
}
