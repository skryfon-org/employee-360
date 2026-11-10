package allocation

import (
	"context"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	allocuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/allocation"
)

type deleteAllocationUseCase struct {
	allocations repository.ProjectAllocationRepository
	audit       service.AuditRecorder
	transactor  ucshared.Transactor
}

// NewDeleteAllocationUseCase creates a DeleteAllocationUseCase.
func NewDeleteAllocationUseCase(allocations repository.ProjectAllocationRepository, audit service.AuditRecorder, transactor ucshared.Transactor) allocuc.DeleteAllocationUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &deleteAllocationUseCase{allocations: allocations, audit: audit, transactor: transactor}
}

func (uc *deleteAllocationUseCase) Execute(c context.Context, tenantID, actorID, id uuid.UUID) error {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	return uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		a, err := uc.allocations.GetByID(txCtx, tenantID, id)
		if err != nil {
			return err
		}
		if err := uc.allocations.SoftDelete(txCtx, tenantID, id, actorID); err != nil {
			return err
		}
		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionProjectAllocationDelete, domainaudit.EntityProjectAllocation, a.ID, map[string]any{
			"project_id": a.ProjectID, "user_id": a.UserID,
		})
	})
}
