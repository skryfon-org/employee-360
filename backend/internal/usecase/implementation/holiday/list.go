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

type listHolidaysByYearUseCase struct {
	holidayRepo repository.HolidayRepository
}

// NewListHolidaysByYearUseCase creates a new ListHolidaysByYearUseCase.
func NewListHolidaysByYearUseCase(holidayRepo repository.HolidayRepository) holidayuc.ListHolidaysByYearUseCase {
	return &listHolidaysByYearUseCase{
		holidayRepo: holidayRepo,
	}
}

func (uc *listHolidaysByYearUseCase) Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	if err := validateYear(query.Year); err != nil {
		return nil, err
	}

	if err := validateMonth(query.Month); err != nil {
		return nil, err
	}

	return uc.holidayRepo.ListByYear(ctx, tenantID, query.Year, query.CategoryID, query.Month)
}
