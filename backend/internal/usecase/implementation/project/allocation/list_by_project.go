package allocation

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	allocuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/allocation"
)

type listProjectAllocationsUseCase struct {
	projects    repository.ProjectRepository
	allocations repository.ProjectAllocationRepository
}

// NewListProjectAllocationsUseCase creates a ListProjectAllocationsUseCase.
func NewListProjectAllocationsUseCase(projects repository.ProjectRepository, allocations repository.ProjectAllocationRepository) allocuc.ListProjectAllocationsUseCase {
	return &listProjectAllocationsUseCase{projects: projects, allocations: allocations}
}

func (uc *listProjectAllocationsUseCase) Execute(c context.Context, tenantID, projectID uuid.UUID, scope entity.AllocationScope) ([]*entity.AllocationWithUser, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if _, err := uc.projects.GetByID(c, tenantID, projectID); err != nil {
		return nil, err
	}
	// Empty means "all" here (ParseAllocationScope("") would map to active).
	if scope == "" {
		scope = entity.AllocationScopeAll
	} else if _, ok := entity.ParseAllocationScope(string(scope)); !ok {
		return nil, ErrInvalidScope
	}
	return uc.allocations.ListByProject(c, tenantID, projectID, scope, today())
}
