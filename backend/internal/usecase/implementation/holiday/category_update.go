package holiday

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

type updateCategoryUseCase struct {
	categoryRepo repository.HolidayCategoryRepository
	audit        domainservice.AuditRecorder
	transactor   ucshared.Transactor
}

// NewUpdateCategoryUseCase constructs an UpdateCategoryUseCase.
func NewUpdateCategoryUseCase(
	categoryRepo repository.HolidayCategoryRepository,
	audit domainservice.AuditRecorder,
	transactor ucshared.Transactor,
) holidayuc.UpdateCategoryUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateCategoryUseCase{
		categoryRepo: categoryRepo,
		audit:        audit,
		transactor:   transactor,
	}
}

func (uc *updateCategoryUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domainerrors.ErrHolidayCategoryNameRequired
	}
	if utf8.RuneCountInString(name) > 100 {
		return nil, domainerrors.ErrHolidayCategoryNameTooLong
	}

	description := strings.TrimSpace(input.Description)
	if utf8.RuneCountInString(description) > 500 {
		return nil, domainerrors.ErrHolidayCategoryDescTooLong
	}

	color := input.Color
	if !isValidHexColor(color) {
		return nil, domainerrors.ErrHolidayCategoryColorInvalid
	}

	var updated *entity.HolidayCategory

	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := uc.categoryRepo.GetByID(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}

		exists, err := uc.categoryRepo.ExistsByName(txCtx, tenantID, name, &input.ID)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrHolidayCategoryDuplicate
		}

		existing.Name = name
		existing.Description = description
		existing.Color = color
		if input.IsActive != nil {
			existing.IsActive = *input.IsActive
		}
		if input.IsPublic != nil {
			existing.IsPublic = *input.IsPublic
		}
		existing.UpdatedAt = time.Now().UTC()
		existing.UpdatedBy = &actorID

		if err := uc.categoryRepo.Update(txCtx, tenantID, actorID, existing); err != nil {
			return err
		}

		if uc.audit != nil {
			if err := uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionHolidayCategoryUpdate, domainaudit.EntityHolidayCategory, existing.ID, map[string]any{
				"name":      existing.Name,
				"color":     existing.Color,
				"is_active": existing.IsActive,
				"is_public": existing.IsPublic,
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
