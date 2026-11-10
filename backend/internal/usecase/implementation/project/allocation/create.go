package allocation

import (
	"context"
	"time"

	projectcommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/project"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	allocuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/allocation"
)

type createAllocationUseCase struct {
	projects    repository.ProjectRepository
	users       repository.UserRepository
	allocations repository.ProjectAllocationRepository
	audit       service.AuditRecorder
	transactor  ucshared.Transactor
}

// NewCreateAllocationUseCase creates a CreateAllocationUseCase.
func NewCreateAllocationUseCase(
	projects repository.ProjectRepository,
	users repository.UserRepository,
	allocations repository.ProjectAllocationRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) allocuc.CreateAllocationUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createAllocationUseCase{projects: projects, users: users, allocations: allocations, audit: audit, transactor: transactor}
}

func (uc *createAllocationUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, in prjtypes.CreateAllocationInput) (*prjtypes.AllocationResult, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if in.UserID == uuid.Nil {
		return nil, ErrUserIDRequired
	}
	if in.AllocationPercent < 1 || in.AllocationPercent > 100 {
		return nil, ErrPercentOutOfRange
	}
	role, err := projectcommon.NormalizeOptional(in.RoleOnProject, 100, ErrRoleTooLong)
	if err != nil {
		return nil, err
	}
	if in.StartDate.IsZero() {
		return nil, projectcommon.ErrStartDateRequired
	}
	start := projectcommon.DateOnly(in.StartDate)
	var end *time.Time
	if in.EndDate != nil {
		e := projectcommon.DateOnly(*in.EndDate)
		if e.Before(start) {
			return nil, ErrAllocationDateOrder
		}
		end = &e
	}

	a := &entity.ProjectAllocation{
		ID: uuid.New(), TenantID: tenantID, ProjectID: in.ProjectID, UserID: in.UserID,
		AllocationPercent: in.AllocationPercent, RoleOnProject: role, StartDate: start, EndDate: end,
	}
	var total int
	err = uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		// Lock the project row to serialise overlap checks for its allocations.
		project, err := uc.projects.GetByIDForUpdate(txCtx, tenantID, in.ProjectID)
		if err != nil {
			return err
		}
		if project.Status == entity.ProjectStatusArchived {
			return domainerrors.ErrProjectArchived
		}
		if err := validateWithinProject(project, start, end); err != nil {
			return err
		}
		user, err := uc.users.GetByID(txCtx, tenantID, in.UserID)
		if err != nil {
			return err
		}
		if !user.IsActive {
			return domainerrors.ErrAllocationUserInactive
		}
		overlap, err := uc.allocations.HasOverlap(txCtx, tenantID, in.ProjectID, in.UserID, start, end, uuid.Nil)
		if err != nil {
			return err
		}
		if overlap {
			return domainerrors.ErrAllocationOverlap
		}
		if err := uc.allocations.Create(txCtx, tenantID, actorID, a); err != nil {
			return err
		}
		if total, err = totalFor(txCtx, uc.allocations, tenantID, a); err != nil {
			return err
		}
		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionProjectAllocationCreate, domainaudit.EntityProjectAllocation, a.ID, map[string]any{
			"project_id": a.ProjectID, "user_id": a.UserID, "allocation_percent": a.AllocationPercent,
		})
	})
	if err != nil {
		return nil, err
	}
	return &prjtypes.AllocationResult{Allocation: a, UserTotalPercent: total, OverAllocated: total > entity.MaxTotalAllocationPercent}, nil
}
