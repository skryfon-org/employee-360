package leavetype

import (
	"context"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	leaveuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/leavetype"
)

type createLeaveType struct {
	repo       repository.LeaveTypeRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewCreateLeaveTypeUseCase constructs CreateLeaveTypeUseCase.
func NewCreateLeaveTypeUseCase(repo repository.LeaveTypeRepository, audit service.AuditRecorder, transactor ucshared.Transactor) leaveuc.CreateLeaveTypeUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createLeaveType{repo: repo, audit: audit, transactor: transactor}
}

func (uc *createLeaveType) Execute(ctx context.Context, tenantID, actorID uuid.UUID, in leavetypes.CreateLeaveTypeInput) (*entity.LeaveType, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	name, err := normalizeName(in.Name)
	if err != nil {
		return nil, err
	}
	code, err := normalizeCode(in.Code)
	if err != nil {
		return nil, err
	}
	desc, err := leavecommon.NormalizeText(in.Description, "description", 500)
	if err != nil {
		return nil, err
	}
	active := in.IsActive == nil || *in.IsActive
	lt := &entity.LeaveType{ID: uuid.New(), TenantID: tenantID, Name: name, Code: code, Description: desc, IsActive: active}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		taken, err := uc.repo.ExistsByCode(txCtx, tenantID, code, uuid.Nil)
		if err != nil {
			return err
		}
		if taken {
			return domainerrors.ErrLeaveTypeCodeTaken
		}
		if err := uc.repo.Create(txCtx, tenantID, actorID, lt); err != nil {
			return err
		}
		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionLeaveTypeCreate, domainaudit.EntityLeaveType, lt.ID, map[string]any{
			"name": lt.Name, "code": lt.Code, "is_active": lt.IsActive,
		})
	})
	if err != nil {
		return nil, err
	}
	return lt, nil
}
