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

type createHolidayUseCase struct {
	holidayRepo  repository.HolidayRepository
	categoryRepo repository.HolidayCategoryRepository
	audit        service.AuditRecorder
	transactor   ucshared.Transactor
}

// NewCreateHolidayUseCase creates a new CreateHolidayUseCase.
func NewCreateHolidayUseCase(
	holidayRepo repository.HolidayRepository,
	categoryRepo repository.HolidayCategoryRepository,
	audit service.AuditRecorder,
	transactor ucshared.Transactor,
) holidayuc.CreateHolidayUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createHolidayUseCase{
		holidayRepo:  holidayRepo,
		categoryRepo: categoryRepo,
		audit:        audit,
		transactor:   transactor,
	}
}

func (uc *createHolidayUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	name, err := validateName(input.Name)
	if err != nil {
		return nil, err
	}

	date, err := NormalizeHolidayDate(input.Date)
	if err != nil {
		return nil, err
	}

	if input.CategoryID == uuid.Nil {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}

	now := time.Now().UTC()
	holiday := &entity.Holiday{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CategoryID:  input.CategoryID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Date:        date,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
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

		holiday.CategoryIsPublic = cat.IsPublic

		// Check for duplicate (tenant, date, lower(name))
		exists, err := uc.holidayRepo.ExistsDuplicate(txCtx, tenantID, date, name, nil)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrHolidayDuplicate
		}

		if err := uc.holidayRepo.Create(txCtx, tenantID, actorID, holiday); err != nil {
			return err
		}

		if uc.audit != nil {
			return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionHolidayCreate, domainaudit.EntityHoliday, holiday.ID, map[string]any{
				"name":        holiday.Name,
				"date":        holiday.Date.Format("2006-01-02"),
				"category_id": holiday.CategoryID,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return holiday, nil
}
