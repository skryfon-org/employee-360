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

func TestListHolidaysByYearUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	categoryID := uuid.New()
	ctx := context.Background()

	holiday1 := &entity.Holiday{
		ID:         uuid.New(),
		TenantID:   tenantID,
		CategoryID: categoryID,
		Name:       "New Year's Day",
		Date:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	holiday2 := &entity.Holiday{
		ID:         uuid.New(),
		TenantID:   tenantID,
		CategoryID: categoryID,
		Name:       "Labor Day",
		Date:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	expectedList := []*entity.Holiday{holiday1, holiday2}

	t.Run("success lists holidays by year", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			listByYearFn: func(ctx context.Context, tID uuid.UUID, year int, catID *uuid.UUID, month *int) ([]*entity.Holiday, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, 2026, year)
				assert.Nil(t, catID)
				assert.Nil(t, month)
				return expectedList, nil
			},
		}
		uc := NewListHolidaysByYearUseCase(hRepo)

		res, err := uc.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{
			Year: 2026,
		})
		require.NoError(t, err)
		assert.Equal(t, expectedList, res)
	})

	t.Run("with categoryID and month filters", func(t *testing.T) {
		month := 5
		hRepo := &mockHolidayRepo{
			listByYearFn: func(ctx context.Context, tID uuid.UUID, year int, catID *uuid.UUID, m *int) ([]*entity.Holiday, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, 2026, year)
				require.NotNil(t, catID)
				assert.Equal(t, categoryID, *catID)
				require.NotNil(t, m)
				assert.Equal(t, 5, *m)
				return []*entity.Holiday{holiday2}, nil
			},
		}
		uc := NewListHolidaysByYearUseCase(hRepo)

		res, err := uc.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{
			Year:       2026,
			CategoryID: &categoryID,
			Month:      &month,
		})
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, holiday2, res[0])
	})

	t.Run("cross-tenant isolation: tenant B listing holidays only queries tenant B", func(t *testing.T) {
		tenantB := uuid.New()
		hRepo := &mockHolidayRepo{
			listByYearFn: func(ctx context.Context, tID uuid.UUID, year int, catID *uuid.UUID, month *int) ([]*entity.Holiday, error) {
				assert.Equal(t, tenantB, tID)
				// Tenant B has no holidays
				return []*entity.Holiday{}, nil
			},
		}
		uc := NewListHolidaysByYearUseCase(hRepo)

		res, err := uc.Execute(ctx, tenantB, holidaytypes.ListHolidaysByYearQuery{Year: 2026})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("fails when tenantID is nil", func(t *testing.T) {
		uc := NewListHolidaysByYearUseCase(&mockHolidayRepo{})

		_, err := uc.Execute(ctx, uuid.Nil, holidaytypes.ListHolidaysByYearQuery{Year: 2026})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("validation: year range 2000-2100", func(t *testing.T) {
		uc := NewListHolidaysByYearUseCase(&mockHolidayRepo{})

		// Year 1999 rejected
		_, err := uc.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 1999})
		require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)

		// Year 2101 rejected
		_, err = uc.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2101})
		require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)

		// Year 2000 accepted
		hRepo := &mockHolidayRepo{
			listByYearFn: func(ctx context.Context, tID uuid.UUID, year int, catID *uuid.UUID, month *int) ([]*entity.Holiday, error) {
				return []*entity.Holiday{}, nil
			},
		}
		ucValid := NewListHolidaysByYearUseCase(hRepo)
		_, err = ucValid.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2000})
		require.NoError(t, err)

		// Year 2100 accepted
		_, err = ucValid.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2100})
		require.NoError(t, err)
	})

	t.Run("validation: month range 1-12", func(t *testing.T) {
		uc := NewListHolidaysByYearUseCase(&mockHolidayRepo{})

		invalidMonth0 := 0
		_, err := uc.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2026, Month: &invalidMonth0})
		require.ErrorIs(t, err, domainerrors.ErrHolidayMonthInvalid)

		invalidMonth13 := 13
		_, err = uc.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2026, Month: &invalidMonth13})
		require.ErrorIs(t, err, domainerrors.ErrHolidayMonthInvalid)

		// Month 1 and 12 accepted
		hRepo := &mockHolidayRepo{
			listByYearFn: func(ctx context.Context, tID uuid.UUID, year int, catID *uuid.UUID, month *int) ([]*entity.Holiday, error) {
				return []*entity.Holiday{}, nil
			},
		}
		ucValid := NewListHolidaysByYearUseCase(hRepo)

		m1 := 1
		_, err = ucValid.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2026, Month: &m1})
		require.NoError(t, err)

		m12 := 12
		_, err = ucValid.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2026, Month: &m12})
		require.NoError(t, err)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db error")
		hRepo := &mockHolidayRepo{
			listByYearFn: func(ctx context.Context, tID uuid.UUID, year int, catID *uuid.UUID, month *int) ([]*entity.Holiday, error) {
				return nil, expectedErr
			},
		}
		uc := NewListHolidaysByYearUseCase(hRepo)

		_, err := uc.Execute(ctx, tenantID, holidaytypes.ListHolidaysByYearQuery{Year: 2026})
		require.ErrorIs(t, err, expectedErr)
	})
}
