package holiday

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
)

func TestUpdateSettingsUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	ctx := context.Background()

	t.Run("success updates settings and records audit log", func(t *testing.T) {
		var upsertedSettings *entity.HolidayCalendarSettings
		sRepo := &mockHolidayCalendarSettingsRepo{
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, s *entity.HolidayCalendarSettings) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				upsertedSettings = s
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewUpdateSettingsUseCase(sRepo, auditRecorder, nil)

		year := 2026
		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "CALENDAR",
			DefaultYear:  &year,
			WeekStartDay: 1,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, tenantID, res.TenantID)
		assert.Equal(t, "calendar", res.DefaultView)
		require.NotNil(t, res.DefaultYear)
		assert.Equal(t, 2026, *res.DefaultYear)
		assert.Equal(t, 1, res.WeekStartDay)
		assert.Equal(t, upsertedSettings, res)

		require.Len(t, auditRecorder.Logs, 1)
		log := auditRecorder.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityHolidayCalendarSettings, log.EntityType)
		assert.Equal(t, domainaudit.ActionHolidayCalendarSettingsUpdate, log.Action)
		assert.Contains(t, log.Metadata, "calendar")
		assert.Contains(t, log.Metadata, "2026")
		assert.Contains(t, log.Metadata, "1")
	})

	t.Run("success clearing default_year (PUT is full replace)", func(t *testing.T) {
		var upsertedSettings *entity.HolidayCalendarSettings
		sRepo := &mockHolidayCalendarSettingsRepo{
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, s *entity.HolidayCalendarSettings) error {
				upsertedSettings = s
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewUpdateSettingsUseCase(sRepo, auditRecorder, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "list",
			DefaultYear:  nil, // clearing year
			WeekStartDay: 0,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "list", res.DefaultView)
		assert.Nil(t, res.DefaultYear)
		assert.Equal(t, 0, res.WeekStartDay)
		assert.Nil(t, upsertedSettings.DefaultYear)

		require.Len(t, auditRecorder.Logs, 1)
		assert.Contains(t, auditRecorder.Logs[0].Metadata, `"default_year":null`)
	})

	t.Run("fails when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewUpdateSettingsUseCase(&mockHolidayCalendarSettingsRepo{}, nil, nil)

		_, err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			WeekStartDay: 1,
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			WeekStartDay: 1,
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("validation: default_view must be calendar or list", func(t *testing.T) {
		uc := NewUpdateSettingsUseCase(&mockHolidayCalendarSettingsRepo{}, nil, nil)

		invalidViews := []string{"", "grid", "day", "month", "123"}
		for _, v := range invalidViews {
			_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
				DefaultView:  v,
				WeekStartDay: 1,
			})
			require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsInvalidDefaultView, "expected error for view: %s", v)
		}
	})

	t.Run("validation: default_year range 2000 to 2100", func(t *testing.T) {
		uc := NewUpdateSettingsUseCase(&mockHolidayCalendarSettingsRepo{}, nil, nil)

		year1999 := 1999
		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			DefaultYear:  &year1999,
			WeekStartDay: 1,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsInvalidDefaultYear)

		year2101 := 2101
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			DefaultYear:  &year2101,
			WeekStartDay: 1,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsInvalidDefaultYear)

		sRepo := &mockHolidayCalendarSettingsRepo{
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, s *entity.HolidayCalendarSettings) error {
				return nil
			},
		}
		ucOk := NewUpdateSettingsUseCase(sRepo, nil, nil)

		year2000 := 2000
		_, err = ucOk.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			DefaultYear:  &year2000,
			WeekStartDay: 1,
		})
		require.NoError(t, err)

		year2100 := 2100
		_, err = ucOk.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			DefaultYear:  &year2100,
			WeekStartDay: 1,
		})
		require.NoError(t, err)
	})

	t.Run("validation: week_start_day range 0 to 6", func(t *testing.T) {
		uc := NewUpdateSettingsUseCase(&mockHolidayCalendarSettingsRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			WeekStartDay: -1,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsInvalidWeekStartDay)

		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			WeekStartDay: 7,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsInvalidWeekStartDay)

		sRepo := &mockHolidayCalendarSettingsRepo{
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, s *entity.HolidayCalendarSettings) error {
				return nil
			},
		}
		ucOk := NewUpdateSettingsUseCase(sRepo, nil, nil)

		for day := 0; day <= 6; day++ {
			_, err = ucOk.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
				DefaultView:  "calendar",
				WeekStartDay: day,
			})
			require.NoError(t, err, "expected valid weekStartDay: %d", day)
		}
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("upsert failure")
		sRepo := &mockHolidayCalendarSettingsRepo{
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, s *entity.HolidayCalendarSettings) error {
				return expectedErr
			},
		}
		uc := NewUpdateSettingsUseCase(sRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			WeekStartDay: 1,
		})
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("preserves stored row ID and timestamps on update for return entity and audit log", func(t *testing.T) {
		existingID := uuid.New()
		existingCreatedAt := time.Now().UTC().Add(-48 * time.Hour)

		sRepo := &mockHolidayCalendarSettingsRepo{
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, s *entity.HolidayCalendarSettings) error {
				// Simulates database Upsert re-reading stored row with original ID and CreatedAt
				s.ID = existingID
				s.CreatedAt = existingCreatedAt
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewUpdateSettingsUseCase(sRepo, auditRecorder, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			WeekStartDay: 1,
		})
		require.NoError(t, err)
		assert.Equal(t, existingID, res.ID)
		assert.Equal(t, existingCreatedAt, res.CreatedAt)

		require.Len(t, auditRecorder.Logs, 1)
		assert.Equal(t, existingID, auditRecorder.Logs[0].EntityID)
	})

	t.Run("cross-tenant isolation: passes tenantID to upsert and audit log", func(t *testing.T) {
		tenantB := uuid.New()
		actorB := uuid.New()

		sRepo := &mockHolidayCalendarSettingsRepo{
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, s *entity.HolidayCalendarSettings) error {
				assert.Equal(t, tenantB, tID)
				assert.Equal(t, actorB, aID)
				assert.Equal(t, tenantB, s.TenantID)
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewUpdateSettingsUseCase(sRepo, auditRecorder, nil)

		res, err := uc.Execute(ctx, tenantB, actorB, holidaytypes.UpdateSettingsInput{
			DefaultView:  "calendar",
			WeekStartDay: 1,
		})
		require.NoError(t, err)
		assert.Equal(t, tenantB, res.TenantID)

		require.Len(t, auditRecorder.Logs, 1)
		assert.Equal(t, tenantB, auditRecorder.Logs[0].TenantID)
		assert.Equal(t, &actorB, auditRecorder.Logs[0].ActorUserID)
	})
}
