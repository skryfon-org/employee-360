package holiday

import (
	"context"
	"strings"

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

type updateSettingsUseCase struct {
	settingsRepo repository.HolidayCalendarSettingsRepository
	audit        domainservice.AuditRecorder
	transactor   ucshared.Transactor
}

// NewUpdateSettingsUseCase constructs an UpdateSettingsUseCase.
func NewUpdateSettingsUseCase(
	settingsRepo repository.HolidayCalendarSettingsRepository,
	audit domainservice.AuditRecorder,
	transactor ucshared.Transactor,
) holidayuc.UpdateSettingsUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateSettingsUseCase{
		settingsRepo: settingsRepo,
		audit:        audit,
		transactor:   transactor,
	}
}

func (uc *updateSettingsUseCase) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateSettingsInput) (*entity.HolidayCalendarSettings, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	view := strings.ToLower(strings.TrimSpace(input.DefaultView))
	if view != "calendar" && view != "list" {
		return nil, domainerrors.ErrHolidayCalendarSettingsInvalidDefaultView
	}

	if input.DefaultYear != nil {
		if *input.DefaultYear < 2000 || *input.DefaultYear > 2100 {
			return nil, domainerrors.ErrHolidayCalendarSettingsInvalidDefaultYear
		}
	}

	if input.WeekStartDay < 0 || input.WeekStartDay > 6 {
		return nil, domainerrors.ErrHolidayCalendarSettingsInvalidWeekStartDay
	}

	settings := &entity.HolidayCalendarSettings{
		TenantID:     tenantID,
		DefaultView:  view,
		DefaultYear:  input.DefaultYear,
		WeekStartDay: input.WeekStartDay,
	}

	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.settingsRepo.Upsert(txCtx, tenantID, actorID, settings); err != nil {
			return err
		}

		if uc.audit != nil {
			auditData := map[string]any{
				"default_view":   settings.DefaultView,
				"week_start_day": settings.WeekStartDay,
			}
			if settings.DefaultYear != nil {
				auditData["default_year"] = *settings.DefaultYear
			} else {
				auditData["default_year"] = nil
			}

			return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionHolidayCalendarSettingsUpdate, domainaudit.EntityHolidayCalendarSettings, settings.ID, auditData)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return settings, nil
}
