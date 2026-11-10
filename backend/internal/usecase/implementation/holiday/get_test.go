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
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

func TestGetHolidayUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	holidayID := uuid.New()
	ctx := context.Background()

	expectedHoliday := &entity.Holiday{
		ID:        holidayID,
		TenantID:  tenantID,
		Name:      "New Year's Day",
		Date:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success returns holiday", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, holidayID, id)
				return expectedHoliday, nil
			},
		}
		uc := NewGetHolidayUseCase(hRepo)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.GetHolidayQuery{ID: holidayID})
		require.NoError(t, err)
		assert.Equal(t, expectedHoliday, res)
	})

	t.Run("cross-tenant isolation: tenant B cannot get tenant A's holiday (returns ErrHolidayNotFound)", func(t *testing.T) {
		tenantB := uuid.New()
		actorB := uuid.New()

		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				assert.Equal(t, tenantB, tID)
				// Tenant B querying for holiday from Tenant A gets ErrHolidayNotFound
				return nil, domainerrors.ErrHolidayNotFound
			},
		}
		uc := NewGetHolidayUseCase(hRepo)

		_, err := uc.Execute(ctx, tenantB, actorB, holidaytypes.GetHolidayQuery{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
	})

	t.Run("fails when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewGetHolidayUseCase(&mockHolidayRepo{})

		_, err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.GetHolidayQuery{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.GetHolidayQuery{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("fails when holiday ID is nil", func(t *testing.T) {
		uc := NewGetHolidayUseCase(&mockHolidayRepo{})

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.GetHolidayQuery{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
	})

	t.Run("non-existent holiday returns ErrHolidayNotFound", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return nil, domainerrors.ErrHolidayNotFound
			},
		}
		uc := NewGetHolidayUseCase(hRepo)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.GetHolidayQuery{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db error")
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return nil, expectedErr
			},
		}
		uc := NewGetHolidayUseCase(hRepo)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.GetHolidayQuery{ID: holidayID})
		require.ErrorIs(t, err, expectedErr)
	})
}
