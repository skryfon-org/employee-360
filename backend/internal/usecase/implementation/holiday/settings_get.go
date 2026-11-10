package holiday

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

type getSettingsUseCase struct {
	settingsRepo repository.HolidayCalendarSettingsRepository
}

// NewGetSettingsUseCase constructs a GetSettingsUseCase.
func NewGetSettingsUseCase(settingsRepo repository.HolidayCalendarSettingsRepository) holidayuc.GetSettingsUseCase {
	return &getSettingsUseCase{
		settingsRepo: settingsRepo,
	}
}

func (uc *getSettingsUseCase) Execute(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	settings, err := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrHolidayCalendarSettingsNotFound) {
			// Return default settings without writing a row to the database.
			return &entity.HolidayCalendarSettings{
				TenantID:     tenantID,
				DefaultView:  "calendar",
				DefaultYear:  nil,
				WeekStartDay: 1,
			}, nil
		}
		return nil, err
	}

	return settings, nil
}
