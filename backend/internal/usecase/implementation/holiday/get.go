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

type getHolidayUseCase struct {
	holidayRepo repository.HolidayRepository
}

// NewGetHolidayUseCase creates a new GetHolidayUseCase.
func NewGetHolidayUseCase(holidayRepo repository.HolidayRepository) holidayuc.GetHolidayUseCase {
	return &getHolidayUseCase{
		holidayRepo: holidayRepo,
	}
}

func (uc *getHolidayUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, query holidaytypes.GetHolidayQuery) (*entity.Holiday, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if query.ID == uuid.Nil {
		return nil, domainerrors.ErrHolidayNotFound
	}

	return uc.holidayRepo.GetByID(ctx, tenantID, query.ID)
}
