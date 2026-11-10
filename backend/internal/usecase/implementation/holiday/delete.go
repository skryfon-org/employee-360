package holiday

import (
	"context"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

type deleteHolidayUseCase struct {
	holidayRepo repository.HolidayRepository
	audit       service.AuditRecorder
	transactor  ucshared.Transactor
}

// NewDeleteHolidayUseCase creates a new DeleteHolidayUseCase.
func NewDeleteHolidayUseCase(
	holidayRepo repository.HolidayRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) holidayuc.DeleteHolidayUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &deleteHolidayUseCase{
		holidayRepo: holidayRepo,
		audit:       audit,
		transactor:  transactor,
	}
}

func (uc *deleteHolidayUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteHolidayInput) error {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return domainerrors.ErrHolidayNotFound
	}

	return uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := uc.holidayRepo.GetByID(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}

		if err := uc.holidayRepo.SoftDelete(txCtx, tenantID, input.ID, actorID); err != nil {
			return err
		}

		if uc.audit != nil {
			return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionHolidayDelete, domainaudit.EntityHoliday, input.ID, map[string]any{
				"name": existing.Name,
				"date": existing.Date.Format("2006-01-02"),
			})
		}
		return nil
	})
}
