package department

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

func TestGetDepartmentUseCase(t *testing.T) {
	tenantID := uuid.New()
	deptID := uuid.New()
	bg := context.Background()

	t.Run("successful lookup", func(t *testing.T) {
		expected := &entity.Department{
			ID:          deptID,
			TenantID:    tenantID,
			Name:        "Engineering",
			Description: "Platform team",
		}
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, deptID, id)
				return expected, nil
			},
		}

		uc := NewGetDepartmentUseCase(repo)
		dept, err := uc.Execute(bg, tenantID, depttypes.GetDepartmentQuery{ID: deptID})
		require.NoError(t, err)
		assert.Equal(t, expected, dept)
	})

	t.Run("nil uuid returns ErrDepartmentNotFound", func(t *testing.T) {
		uc := NewGetDepartmentUseCase(&mockDepartmentRepo{})
		_, err := uc.Execute(bg, tenantID, depttypes.GetDepartmentQuery{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	})

	t.Run("not found returns ErrDepartmentNotFound", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return nil, domainerrors.ErrDepartmentNotFound
			},
		}
		uc := NewGetDepartmentUseCase(repo)
		_, err := uc.Execute(bg, tenantID, depttypes.GetDepartmentQuery{ID: deptID})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	})

	t.Run("nil tenant returns unauthorized", func(t *testing.T) {
		uc := NewGetDepartmentUseCase(&mockDepartmentRepo{})
		_, err := uc.Execute(bg, uuid.Nil, depttypes.GetDepartmentQuery{ID: deptID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		boom := errors.New("database failure")
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return nil, boom
			},
		}
		uc := NewGetDepartmentUseCase(repo)
		_, err := uc.Execute(bg, tenantID, depttypes.GetDepartmentQuery{ID: deptID})
		require.ErrorIs(t, err, boom)
	})
}
