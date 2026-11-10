package holiday

import (
	"context"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

type deleteCategoryUseCase struct {
	categoryRepo repository.HolidayCategoryRepository
	audit        domainservice.AuditRecorder
	transactor   ucshared.Transactor
}

// NewDeleteCategoryUseCase constructs a DeleteCategoryUseCase.
func NewDeleteCategoryUseCase(
	categoryRepo repository.HolidayCategoryRepository,
	audit domainservice.AuditRecorder,
	transactor ucshared.Transactor,
) holidayuc.DeleteCategoryUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &deleteCategoryUseCase{
		categoryRepo: categoryRepo,
		audit:        audit,
		transactor:   transactor,
	}
}

func (uc *deleteCategoryUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	if input.ID == uuid.Nil {
		return domainerrors.ErrHolidayCategoryNotFound
	}

	return uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := uc.categoryRepo.GetByIDForUpdate(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}

		count, err := uc.categoryRepo.CountHolidaysByCategory(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return domainerrors.ErrHolidayCategoryInUse
		}

		if err := uc.categoryRepo.SoftDelete(txCtx, tenantID, input.ID, actorID); err != nil {
			return err
		}

		// Re-count after soft-delete inside the same transaction to guarantee no concurrent holiday was linked
		countAfter, err := uc.categoryRepo.CountHolidaysByCategory(txCtx, tenantID, input.ID)
		if err != nil {
			return err
		}
		if countAfter > 0 {
			return domainerrors.ErrHolidayCategoryInUse
		}

		if uc.audit != nil {
			return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionHolidayCategoryDelete, domainaudit.EntityHolidayCategory, input.ID, map[string]any{
				"name": existing.Name,
			})
		}
		return nil
	})
}
