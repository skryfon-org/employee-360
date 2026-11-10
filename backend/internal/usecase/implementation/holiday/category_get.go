package holiday

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

type getCategoryUseCase struct {
	categoryRepo repository.HolidayCategoryRepository
}

// NewGetCategoryUseCase constructs a GetCategoryUseCase.
func NewGetCategoryUseCase(categoryRepo repository.HolidayCategoryRepository) holidayuc.GetCategoryUseCase {
	return &getCategoryUseCase{
		categoryRepo: categoryRepo,
	}
}

func (uc *getCategoryUseCase) Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if query.ID == uuid.Nil {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}

	return uc.categoryRepo.GetByID(ctx, tenantID, query.ID)
}
