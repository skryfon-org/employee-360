package holiday

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

type updateHolidayUseCase struct {
	holidayRepo  repository.HolidayRepository
	categoryRepo repository.HolidayCategoryRepository
	audit        service.AuditRecorder
	transactor   ucshared.Transactor
}

// NewUpdateHolidayUseCase creates a new UpdateHolidayUseCase.
func NewUpdateHolidayUseCase(
	holidayRepo repository.HolidayRepository,
	categoryRepo repository.HolidayCategoryRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) holidayuc.UpdateHolidayUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateHolidayUseCase{
		holidayRepo:  holidayRepo,
		categoryRepo: categoryRepo,
		audit:        audit,
		transactor:   transactor,
	}
}

func (uc *updateHolidayUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return nil, domainerrors.ErrHolidayNotFound
	}
	if input.CategoryID == uuid.Nil {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}

	name, err := validateName(input.Name)
	if err != nil {
		return nil, err
	}

	date, err := NormalizeHolidayDate(input.Date)
	if err != nil {
		return nil, err
	}

	var updated *entity.Holiday
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := uc.holidayRepo.GetByID(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}

		// Category must exist in the same tenant and not be soft-deleted.
		cat, err := uc.categoryRepo.GetByIDForUpdate(txCtx, tenantID, input.CategoryID)
		if err != nil {
			return err
		}
		if cat == nil || cat.TenantID != tenantID || cat.DeletedAt != nil {
			return domainerrors.ErrHolidayCategoryNotFound
		}
		if !cat.IsActive {
			return domainerrors.ErrHolidayCategoryInactive
		}

		// Check for duplicate (tenant, date, lower(name)) excluding current holiday
		exists, err := uc.holidayRepo.ExistsDuplicate(txCtx, tenantID, date, name, &existing.ID)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrHolidayDuplicate
		}

		existing.CategoryID = input.CategoryID
		existing.CategoryIsPublic = cat.IsPublic
		existing.Name = name
		existing.Description = strings.TrimSpace(input.Description)
		existing.Date = date
		existing.UpdatedAt = time.Now().UTC()

		if err := uc.holidayRepo.Update(txCtx, tenantID, actorID, existing); err != nil {
			return err
		}

		if uc.audit != nil {
			if err := uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionHolidayUpdate, domainaudit.EntityHoliday, existing.ID, map[string]any{
				"name":        existing.Name,
				"date":        existing.Date.Format("2006-01-02"),
				"category_id": existing.CategoryID,
			}); err != nil {
				return err
			}
		}

		updated = existing
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}
