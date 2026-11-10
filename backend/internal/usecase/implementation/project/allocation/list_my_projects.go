package allocation

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	allocuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/allocation"
)

type listMyProjectsUseCase struct {
	users       repository.UserRepository
	allocations repository.ProjectAllocationRepository
}

// NewListMyProjectsUseCase creates a ListMyProjectsUseCase.
func NewListMyProjectsUseCase(users repository.UserRepository, allocations repository.ProjectAllocationRepository) allocuc.ListMyProjectsUseCase {
	return &listMyProjectsUseCase{users: users, allocations: allocations}
}

func (uc *listMyProjectsUseCase) Execute(c context.Context, tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if _, err := uc.users.GetByID(c, tenantID, userID); err != nil {
		return nil, err
	}
	if scope == "" {
		scope = entity.AllocationScopeActive
	}
	if _, ok := entity.ParseAllocationScope(string(scope)); !ok {
		return nil, ErrInvalidScope
	}
	ref := today()
	items, err := uc.allocations.ListByUserScope(c, tenantID, userID, scope, ref)
	if err != nil {
		return nil, err
	}
	if scope == entity.AllocationScopeActive {
		items = liveOnly(items)
	}
	// Total/over-allocated reflect today's load on live projects only.
	total := liveTotalOn(items, ref)
	counts, err := uc.allocations.CountByUserScopes(c, tenantID, userID, ref)
	if err != nil {
		return nil, err
	}
	return &prjtypes.UserAllocationsResult{Allocations: items, TotalPercent: total, OverAllocated: total > entity.MaxTotalAllocationPercent, Counts: &counts}, nil
}
