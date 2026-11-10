package department

import (
	"context"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	deptuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/department"
)

type deleteDepartmentUseCase struct {
	repo       repository.DepartmentRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewDeleteDepartmentUseCase creates a new DeleteDepartmentUseCase.
func NewDeleteDepartmentUseCase(
	repo repository.DepartmentRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) deptuc.DeleteDepartmentUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &deleteDepartmentUseCase{
		repo:       repo,
		audit:      audit,
		transactor: transactor,
	}
}

func (uc *deleteDepartmentUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, input depttypes.DeleteDepartmentInput) error {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return domainerrors.ErrDepartmentNotFound
	}

	return uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		// Lock the department row so a concurrent invite cannot attach it
		// between the reference check and the soft delete.
		dept, err := uc.repo.GetByIDForUpdate(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}

		// Verify department is not referenced by active users or invitations
		isReferenced, err := uc.repo.IsReferenced(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}
		if isReferenced {
			return domainerrors.ErrDepartmentInUse
		}

		if err := uc.repo.Delete(txCtx, tenantID, input.ID, actorID); err != nil {
			return err
		}

		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionDepartmentDelete, domainaudit.EntityDepartment, dept.ID, map[string]any{
			"name": dept.Name,
		})
	})
}
