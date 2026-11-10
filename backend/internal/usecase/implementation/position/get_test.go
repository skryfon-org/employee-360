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

func TestGetPositionUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	posID := uuid.New()
	ctx := context.Background()

	t.Run("success returns position", func(t *testing.T) {
		expected := &entity.Position{
			ID:          posID,
			TenantID:    tenantID,
			Name:        "Product Designer",
			Description: "Design team",
			IsActive:    true,
		}
		repo := &mockPositionRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Position, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, posID, id)
				return expected, nil
			},
		}
		uc := NewGetPositionUseCase(repo)

		res, err := uc.Execute(ctx, tenantID, postypes.GetPositionQuery{ID: posID})
		require.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("nil tenantID returns ErrUnauthorized", func(t *testing.T) {
		uc := NewGetPositionUseCase(&mockPositionRepo{})

		_, err := uc.Execute(ctx, uuid.Nil, postypes.GetPositionQuery{ID: posID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("nil uuid returns ErrPositionNotFound", func(t *testing.T) {
		uc := NewGetPositionUseCase(&mockPositionRepo{})

		_, err := uc.Execute(ctx, tenantID, postypes.GetPositionQuery{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)
	})

	t.Run("not found returns ErrPositionNotFound", func(t *testing.T) {
		repo := &mockPositionRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Position, error) {
				return nil, domainerrors.ErrPositionNotFound
			},
		}
		uc := NewGetPositionUseCase(repo)

		_, err := uc.Execute(ctx, tenantID, postypes.GetPositionQuery{ID: posID})
		require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db connection lost")
		repo := &mockPositionRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Position, error) {
				return nil, expectedErr
			},
		}
		uc := NewGetPositionUseCase(repo)

		_, err := uc.Execute(ctx, tenantID, postypes.GetPositionQuery{ID: posID})
		require.ErrorIs(t, err, expectedErr)
	})
}
