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

type createCategoryUseCase struct {
	categoryRepo repository.HolidayCategoryRepository
	audit        domainservice.AuditRecorder
	transactor   ucshared.Transactor
}

// NewCreateCategoryUseCase constructs a CreateCategoryUseCase.
func NewCreateCategoryUseCase(
	categoryRepo repository.HolidayCategoryRepository,
	audit domainservice.AuditRecorder,
	transactor ucshared.Transactor,
) holidayuc.CreateCategoryUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createCategoryUseCase{
		categoryRepo: categoryRepo,
		audit:        audit,
		transactor:   transactor,
	}
}

func (uc *createCategoryUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
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

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	isPublic := input.IsPublic != nil && *input.IsPublic

	now := time.Now().UTC()
	cat := &entity.HolidayCategory{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Color:       color,
		IsActive:    isActive,
		IsPublic:    isPublic,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   &actorID,
		UpdatedBy:   &actorID,
	}

	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		exists, err := uc.categoryRepo.ExistsByName(txCtx, tenantID, name, nil)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrHolidayCategoryDuplicate
		}

		if err := uc.categoryRepo.Create(txCtx, tenantID, actorID, cat); err != nil {
			return err
		}

		if uc.audit != nil {
			return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionHolidayCategoryCreate, domainaudit.EntityHolidayCategory, cat.ID, map[string]any{
				"name":      cat.Name,
				"color":     cat.Color,
				"is_active": cat.IsActive,
				"is_public": cat.IsPublic,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cat, nil
}
