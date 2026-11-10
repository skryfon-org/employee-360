package position

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

func TestListPositionsUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("success returns paginated positions", func(t *testing.T) {
		expectedItems := []*entity.Position{
			{ID: uuid.New(), TenantID: tenantID, Name: "Backend Engineer", IsActive: true},
			{ID: uuid.New(), TenantID: tenantID, Name: "Frontend Engineer", IsActive: true},
		}
		repo := &mockPositionRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error) {
				assert.Equal(t, tenantID, tID)
				assert.Nil(t, isActive)
				assert.Equal(t, 10, limit)
				assert.Equal(t, 20, offset)
				return expectedItems, 25, nil
			},
		}
		uc := NewListPositionsUseCase(repo)

		out, err := uc.Execute(ctx, tenantID, postypes.ListPositionsQuery{
			Page:     3,
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, expectedItems, out.Positions)
		assert.Equal(t, int64(25), out.Total)
		assert.Equal(t, 3, out.Page)
		assert.Equal(t, 10, out.PageSize)
	})

	t.Run("default pagination when invalid", func(t *testing.T) {
		repo := &mockPositionRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error) {
				assert.Contains(t, []int{20, 100}, limit)
				assert.Equal(t, 0, offset)
				return []*entity.Position{}, 0, nil
			},
		}
		uc := NewListPositionsUseCase(repo)

		out, err := uc.Execute(ctx, tenantID, postypes.ListPositionsQuery{
			Page:     0,
			PageSize: 0,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, out.Page)
		assert.Equal(t, 20, out.PageSize)

		// Page size > 100 clamped to 100
		out, err = uc.Execute(ctx, tenantID, postypes.ListPositionsQuery{
			Page:     -5,
			PageSize: 500,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, out.Page)
		assert.Equal(t, 100, out.PageSize)
	})

	t.Run("honors is_active filter", func(t *testing.T) {
		tr := true
		repo := &mockPositionRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error) {
				assert.Equal(t, &tr, isActive)
				return []*entity.Position{}, 0, nil
			},
		}
		uc := NewListPositionsUseCase(repo)

		_, err := uc.Execute(ctx, tenantID, postypes.ListPositionsQuery{
			IsActive: &tr,
		})
		require.NoError(t, err)
	})

	t.Run("unauthorized on nil tenantID", func(t *testing.T) {
		uc := NewListPositionsUseCase(&mockPositionRepo{})

		_, err := uc.Execute(ctx, uuid.Nil, postypes.ListPositionsQuery{})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("query timeout")
		repo := &mockPositionRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Position, int64, error) {
				return nil, 0, expectedErr
			},
		}
		uc := NewListPositionsUseCase(repo)

		_, err := uc.Execute(ctx, tenantID, postypes.ListPositionsQuery{})
		require.ErrorIs(t, err, expectedErr)
	})
}
