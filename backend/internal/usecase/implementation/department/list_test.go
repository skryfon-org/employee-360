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

func TestListDepartmentsUseCase(t *testing.T) {
	tenantID := uuid.New()
	bg := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, 20, limit)
				assert.Equal(t, 0, offset)
				return []*entity.Department{
					{ID: uuid.New(), TenantID: tenantID, Name: "Engineering"},
				}, 1, nil
			},
		}

		uc := NewListDepartmentsUseCase(repo)
		out, err := uc.Execute(bg, tenantID, depttypes.ListDepartmentsQuery{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), out.Total)
		assert.Equal(t, 1, out.Page)
		assert.Equal(t, 20, out.PageSize)
		assert.Len(t, out.Departments, 1)
	})

	t.Run("is_active filter is passed to the repository", func(t *testing.T) {
		f := false
		repo := &mockDepartmentRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error) {
				require.NotNil(t, isActive)
				assert.False(t, *isActive)
				return []*entity.Department{}, 0, nil
			},
		}
		_, err := NewListDepartmentsUseCase(repo).Execute(bg, tenantID, depttypes.ListDepartmentsQuery{IsActive: &f})
		require.NoError(t, err)
	})

	t.Run("custom pagination", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, 10, limit)
				assert.Equal(t, 20, offset) // page 3 with size 10 -> offset 20
				return []*entity.Department{}, 25, nil
			},
		}

		uc := NewListDepartmentsUseCase(repo)
		out, err := uc.Execute(bg, tenantID, depttypes.ListDepartmentsQuery{
			Page:     3,
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(25), out.Total)
		assert.Equal(t, 3, out.Page)
		assert.Equal(t, 10, out.PageSize)
	})

	t.Run("page size above max is clamped to 100", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error) {
				assert.Equal(t, 100, limit)
				return []*entity.Department{}, 0, nil
			},
		}
		out, err := NewListDepartmentsUseCase(repo).Execute(bg, tenantID, depttypes.ListDepartmentsQuery{PageSize: 500})
		require.NoError(t, err)
		assert.Equal(t, 100, out.PageSize)
	})

	t.Run("nil tenant returns unauthorized", func(t *testing.T) {
		uc := NewListDepartmentsUseCase(&mockDepartmentRepo{})
		_, err := uc.Execute(bg, uuid.Nil, depttypes.ListDepartmentsQuery{})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		boom := errors.New("database error")
		repo := &mockDepartmentRepo{
			listFn: func(ctx context.Context, tID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error) {
				return nil, 0, boom
			},
		}

		uc := NewListDepartmentsUseCase(repo)
		_, err := uc.Execute(bg, tenantID, depttypes.ListDepartmentsQuery{})
		require.ErrorIs(t, err, boom)
	})
}
