package allocation

import (
	"context"

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

type updateAllocationUseCase struct {
	projects    repository.ProjectRepository
	allocations repository.ProjectAllocationRepository
	audit       service.AuditRecorder
	transactor  ucshared.Transactor
}

// NewUpdateAllocationUseCase creates an UpdateAllocationUseCase.
func NewUpdateAllocationUseCase(
	projects repository.ProjectRepository,
	allocations repository.ProjectAllocationRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) allocuc.UpdateAllocationUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateAllocationUseCase{projects: projects, allocations: allocations, audit: audit, transactor: transactor}
}

func (uc *updateAllocationUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, in prjtypes.UpdateAllocationInput) (*prjtypes.AllocationResult, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if in.ID == uuid.Nil {
		return nil, domainerrors.ErrAllocationNotFound
	}
	var (
		out   *entity.ProjectAllocation
		total int
	)
	err := uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		a, err := uc.allocations.GetByID(txCtx, tenantID, in.ID)
		if err != nil {
			return err
		}
		project, err := uc.projects.GetByIDForUpdate(txCtx, tenantID, a.ProjectID)
		if err != nil {
			return err
		}
		// Re-read under the project lock so the overlap check sees current dates.
		if a, err = uc.allocations.GetByID(txCtx, tenantID, in.ID); err != nil {
			return err
		}

		if in.AllocationPercent.Set {
			if in.AllocationPercent.Null || in.AllocationPercent.Value < 1 || in.AllocationPercent.Value > 100 {
				return ErrPercentOutOfRange
			}
			a.AllocationPercent = in.AllocationPercent.Value
		}
		if in.RoleOnProject.Set {
			if in.RoleOnProject.Null {
				a.RoleOnProject = nil
			} else {
				v := in.RoleOnProject.Value
				if a.RoleOnProject, err = projectcommon.NormalizeOptional(&v, 100, ErrRoleTooLong); err != nil {
					return err
				}
			}
		}
		if in.StartDate.Set {
			if in.StartDate.Null || in.StartDate.Value.IsZero() {
				return projectcommon.ErrStartDateRequired
			}
			a.StartDate = projectcommon.DateOnly(in.StartDate.Value.Time)
		}
		if in.EndDate.Set {
			if in.EndDate.Null {
				a.EndDate = nil
			} else {
				e := projectcommon.DateOnly(in.EndDate.Value.Time)
				a.EndDate = &e
			}
		}
		if a.EndDate != nil && a.EndDate.Before(a.StartDate) {
			return ErrAllocationDateOrder
		}
		// Only re-check the window when dates changed, so unrelated edits (percent,
		// role) and ending an allocation early still work on legacy out-of-window rows.
		if in.StartDate.Set || in.EndDate.Set {
			if err := validateWithinProject(project, a.StartDate, a.EndDate); err != nil {
				return err
			}
		}

		overlap, err := uc.allocations.HasOverlap(txCtx, tenantID, a.ProjectID, a.UserID, a.StartDate, a.EndDate, a.ID)
		if err != nil {
			return err
		}
		if overlap {
			return domainerrors.ErrAllocationOverlap
		}
		if err := uc.allocations.Update(txCtx, tenantID, actorID, a); err != nil {
			return err
		}
		if total, err = totalFor(txCtx, uc.allocations, tenantID, a); err != nil {
			return err
		}
		out = a
		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionProjectAllocationUpdate, domainaudit.EntityProjectAllocation, a.ID, map[string]any{
			"project_id": a.ProjectID, "user_id": a.UserID, "allocation_percent": a.AllocationPercent,
		})
	})
	if err != nil {
		return nil, err
	}
	return &prjtypes.AllocationResult{Allocation: out, UserTotalPercent: total, OverAllocated: total > entity.MaxTotalAllocationPercent}, nil
}
