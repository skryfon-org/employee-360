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

func TestGetCategoryUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	catID := uuid.New()
	ctx := context.Background()

	expectedCat := &entity.HolidayCategory{
		ID:        catID,
		TenantID:  tenantID,
		Name:      "Public",
		Color:     "#15803D",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success returns category", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, catID, id)
				return expectedCat, nil
			},
		}
		uc := NewGetCategoryUseCase(cRepo)

		cat, err := uc.Execute(ctx, tenantID, holidaytypes.GetCategoryQuery{ID: catID})
		require.NoError(t, err)
		assert.Equal(t, expectedCat, cat)
	})

	t.Run("fails when category not found", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}
		uc := NewGetCategoryUseCase(cRepo)

		_, err := uc.Execute(ctx, tenantID, holidaytypes.GetCategoryQuery{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("fails when tenantID is nil", func(t *testing.T) {
		uc := NewGetCategoryUseCase(&mockHolidayCategoryRepo{})

		_, err := uc.Execute(ctx, uuid.Nil, holidaytypes.GetCategoryQuery{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("fails when ID is nil", func(t *testing.T) {
		uc := NewGetCategoryUseCase(&mockHolidayCategoryRepo{})

		_, err := uc.Execute(ctx, tenantID, holidaytypes.GetCategoryQuery{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("cross-tenant isolation: tenant B cannot get tenant A's category", func(t *testing.T) {
		tenantB := uuid.New()

		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantB, tID)
				assert.Equal(t, catID, id)
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}
		uc := NewGetCategoryUseCase(cRepo)

		_, err := uc.Execute(ctx, tenantB, holidaytypes.GetCategoryQuery{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})
}
