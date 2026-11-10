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

type listUserAllocationsUseCase struct {
	users       repository.UserRepository
	allocations repository.ProjectAllocationRepository
}

// NewListUserAllocationsUseCase creates a ListUserAllocationsUseCase.
func NewListUserAllocationsUseCase(users repository.UserRepository, allocations repository.ProjectAllocationRepository) allocuc.ListUserAllocationsUseCase {
	return &listUserAllocationsUseCase{users: users, allocations: allocations}
}

func (uc *listUserAllocationsUseCase) Execute(c context.Context, tenantID, userID uuid.UUID, scope entity.AllocationScope) (*prjtypes.UserAllocationsResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	// 404 for users outside the tenant (or soft-deleted).
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
	// Admin view: no project-status filtering; each item carries
	// project_status/counts_toward_total instead. Ordering is start_date, project
	// name, id for every scope.
	items, err := uc.allocations.ListByUserScope(c, tenantID, userID, scope, ref)
	if err != nil {
		return nil, err
	}
	// Total/over-allocated reflect today's load on live projects only, so
	// upcoming/past items are listed but never counted.
	total := liveTotalOn(items, ref)
	return &prjtypes.UserAllocationsResult{Allocations: items, TotalPercent: total, OverAllocated: total > entity.MaxTotalAllocationPercent}, nil
}
