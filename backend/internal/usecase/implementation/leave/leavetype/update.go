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

type updateLeaveType struct {
	repo       repository.LeaveTypeRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewUpdateLeaveTypeUseCase constructs UpdateLeaveTypeUseCase.
func NewUpdateLeaveTypeUseCase(repo repository.LeaveTypeRepository, audit service.AuditRecorder, transactor ucshared.Transactor) leaveuc.UpdateLeaveTypeUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateLeaveType{repo: repo, audit: audit, transactor: transactor}
}

func (uc *updateLeaveType) Execute(ctx context.Context, tenantID, actorID, id uuid.UUID, in leavetypes.UpdateLeaveTypeInput) (*entity.LeaveType, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	var out *entity.LeaveType
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		lt, err := uc.repo.GetByID(txCtx, tenantID, id)
		if err != nil {
			return err
		}
		changes := map[string]any{}
		if in.Name != nil {
			if lt.Name, err = normalizeName(*in.Name); err != nil {
				return err
			}
			changes["name"] = lt.Name
		}
		if in.Code != nil {
			code, err := normalizeCode(*in.Code)
			if err != nil {
				return err
			}
			if code != lt.Code {
				taken, err := uc.repo.ExistsByCode(txCtx, tenantID, code, lt.ID)
				if err != nil {
					return err
				}
				if taken {
					return domainerrors.ErrLeaveTypeCodeTaken
				}
			}
			lt.Code = code
			changes["code"] = code
		}
		if in.Description != nil {
			if lt.Description, err = leavecommon.NormalizeText(*in.Description, "description", 500); err != nil {
				return err
			}
			changes["description"] = lt.Description
		}
		if in.IsActive != nil {
			lt.IsActive = *in.IsActive
			changes["is_active"] = lt.IsActive
		}
		if len(changes) == 0 {
			out = lt
			return nil
		}
		if err := uc.repo.Update(txCtx, tenantID, actorID, lt); err != nil {
			return err
		}
		out = lt
		return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionLeaveTypeUpdate, domainaudit.EntityLeaveType, lt.ID, changes)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
