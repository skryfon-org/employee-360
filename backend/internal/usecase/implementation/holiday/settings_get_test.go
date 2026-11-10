package holiday

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestGetSettingsUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("returns existing settings when present", func(t *testing.T) {
		year := 2026
		existing := &entity.HolidayCalendarSettings{
			ID:           uuid.New(),
			TenantID:     tenantID,
			DefaultView:  "list",
			DefaultYear:  &year,
			WeekStartDay: 0,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		sRepo := &mockHolidayCalendarSettingsRepo{
			getByTenantIDFn: func(ctx context.Context, tID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
				assert.Equal(t, tenantID, tID)
				return existing, nil
			},
		}
		uc := NewGetSettingsUseCase(sRepo)

		res, err := uc.Execute(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, existing, res)
		assert.Equal(t, "list", res.DefaultView)
		require.NotNil(t, res.DefaultYear)
		assert.Equal(t, 2026, *res.DefaultYear)
		assert.Equal(t, 0, res.WeekStartDay)
	})

	t.Run("returns default settings without writing a row on read when none exists", func(t *testing.T) {
		upsertCalled := false
		sRepo := &mockHolidayCalendarSettingsRepo{
			getByTenantIDFn: func(ctx context.Context, tID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
				assert.Equal(t, tenantID, tID)
				return nil, domainerrors.ErrHolidayCalendarSettingsNotFound
			},
			upsertFn: func(ctx context.Context, tID, aID uuid.UUID, settings *entity.HolidayCalendarSettings) error {
				upsertCalled = true
				t.Fatal("upsert should NOT be called on read")
				return nil
			},
		}
		uc := NewGetSettingsUseCase(sRepo)

		res, err := uc.Execute(ctx, tenantID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, tenantID, res.TenantID)
		assert.Equal(t, "calendar", res.DefaultView)
		assert.Nil(t, res.DefaultYear)
		assert.Equal(t, 1, res.WeekStartDay)
		assert.False(t, upsertCalled, "Get must not write a row to the database on read")
	})

	t.Run("fails when tenantID is nil", func(t *testing.T) {
		uc := NewGetSettingsUseCase(&mockHolidayCalendarSettingsRepo{})

		_, err := uc.Execute(ctx, uuid.Nil)
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db error")
		sRepo := &mockHolidayCalendarSettingsRepo{
			getByTenantIDFn: func(ctx context.Context, tID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
				return nil, expectedErr
			},
		}
		uc := NewGetSettingsUseCase(sRepo)

		_, err := uc.Execute(ctx, tenantID)
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("cross-tenant isolation: tenant B gets own defaults when tenant A has settings", func(t *testing.T) {
		tenantB := uuid.New()

		sRepo := &mockHolidayCalendarSettingsRepo{
			getByTenantIDFn: func(ctx context.Context, tID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
				assert.Equal(t, tenantB, tID)
				return nil, domainerrors.ErrHolidayCalendarSettingsNotFound
			},
		}
		uc := NewGetSettingsUseCase(sRepo)

		res, err := uc.Execute(ctx, tenantB)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, tenantB, res.TenantID)
		assert.Equal(t, "calendar", res.DefaultView)
		assert.Equal(t, 1, res.WeekStartDay)
	})
}
