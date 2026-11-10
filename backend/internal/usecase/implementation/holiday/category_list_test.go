package holiday

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

func TestListCategoriesUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	ctx := context.Background()

	cats := []*entity.HolidayCategory{
		{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Name:      "Optional",
			Color:     "#3B82F6",
			IsActive:  true,
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Name:      "Public",
			Color:     "#15803D",
			IsActive:  false,
			CreatedAt: time.Now().UTC(),
		},
	}

	t.Run("success returns all categories", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error) {
				assert.Equal(t, tenantID, tID)
				assert.Nil(t, isActive)
				return cats, 2, nil
			},
		}
		uc := NewListCategoriesUseCase(cRepo)

		res, err := uc.Execute(ctx, tenantID, holidaytypes.ListCategoriesQuery{})
		require.NoError(t, err)
		assert.Equal(t, cats, res.Categories)
		assert.Equal(t, int64(2), res.Total)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize)
	})

	t.Run("success returns filtered categories", func(t *testing.T) {
		activeOnly := true
		cRepo := &mockHolidayCategoryRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error) {
				assert.Equal(t, tenantID, tID)
				require.NotNil(t, isActive)
				assert.True(t, *isActive)
				return cats[:1], 1, nil
			},
		}
		uc := NewListCategoriesUseCase(cRepo)

		res, err := uc.Execute(ctx, tenantID, holidaytypes.ListCategoriesQuery{IsActive: &activeOnly})
		require.NoError(t, err)
		assert.Len(t, res.Categories, 1)
		assert.Equal(t, "Optional", res.Categories[0].Name)
	})

	t.Run("fails when tenantID is nil", func(t *testing.T) {
		uc := NewListCategoriesUseCase(&mockHolidayCategoryRepo{})

		_, err := uc.Execute(ctx, uuid.Nil, holidaytypes.ListCategoriesQuery{})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("tenant isolation: passes tenantID to listFn", func(t *testing.T) {
		tenantB := uuid.New()
		catsB := []*entity.HolidayCategory{
			{
				ID:       uuid.New(),
				TenantID: tenantB,
				Name:     "Tenant B Unique Category",
			},
		}

		cRepo := &mockHolidayCategoryRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error) {
				assert.Equal(t, tenantB, tID)
				return catsB, 1, nil
			},
		}
		uc := NewListCategoriesUseCase(cRepo)

		res, err := uc.Execute(ctx, tenantB, holidaytypes.ListCategoriesQuery{})
		require.NoError(t, err)
		assert.Equal(t, catsB, res.Categories)
	})

	t.Run("computes limit/offset and clamps page size", func(t *testing.T) {
		cases := []struct {
			name                   string
			page, size             int
			wantLimit, wantOff     int
			wantPage, wantPageSize int
		}{
			{"defaults", 0, 0, 20, 0, 1, 20},
			{"page 3 size 10", 3, 10, 10, 20, 3, 10},
			{"size clamped to max", 1, 500, 100, 0, 1, 100},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				cRepo := &mockHolidayCategoryRepo{
					listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.HolidayCategory, int64, error) {
						assert.Equal(t, tc.wantLimit, limit)
						assert.Equal(t, tc.wantOff, offset)
						return nil, 0, nil
					},
				}
				res, err := NewListCategoriesUseCase(cRepo).Execute(ctx, tenantID, holidaytypes.ListCategoriesQuery{Page: tc.page, PageSize: tc.size})
				require.NoError(t, err)
				assert.Equal(t, tc.wantPage, res.Page)
				assert.Equal(t, tc.wantPageSize, res.PageSize)
			})
		}
	})
}
