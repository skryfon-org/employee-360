package department

import (
	"context"
	"errors"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

func TestDeleteDepartmentUseCase(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	deptID := uuid.New()
	bg := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				assert.Equal(t, tenantID, tID)
				return &entity.Department{ID: deptID, TenantID: tenantID, Name: "Engineering"}, nil
			},
			isReferencedFn: func(ctx context.Context, tID, id uuid.UUID) (bool, error) {
				assert.Equal(t, tenantID, tID)
				return false, nil
			},
			deleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, deptID, id)
				assert.Equal(t, actorID, aID)
				return nil
			},
		}

		uc := NewDeleteDepartmentUseCase(repo, auditRepo, nil)
		err := uc.Execute(bg, tenantID, actorID, depttypes.DeleteDepartmentInput{ID: deptID})
		require.NoError(t, err)
		assert.Equal(t, 1, repo.forUpdateCalls, "department row must be locked before the reference check")

		require.Len(t, auditRepo.Logs, 1)
		assert.Equal(t, domainaudit.ActionDepartmentDelete, auditRepo.Logs[0].Action)
		assert.Equal(t, domainaudit.EntityDepartment, auditRepo.Logs[0].EntityType)
		assert.Equal(t, deptID, auditRepo.Logs[0].EntityID)
		assert.Equal(t, &actorID, auditRepo.Logs[0].ActorUserID)
		assert.Equal(t, tenantID, auditRepo.Logs[0].TenantID)
	})

	t.Run("nil uuid returns ErrDepartmentNotFound", func(t *testing.T) {
		uc := NewDeleteDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		err := uc.Execute(bg, tenantID, actorID, depttypes.DeleteDepartmentInput{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	})

	t.Run("department not found returns ErrDepartmentNotFound", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return nil, domainerrors.ErrDepartmentNotFound
			},
		}
		uc := NewDeleteDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		err := uc.Execute(bg, tenantID, actorID, depttypes.DeleteDepartmentInput{ID: deptID})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	})

	t.Run("department in use returns ErrDepartmentInUse", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return &entity.Department{ID: deptID, TenantID: tenantID}, nil
			},
			isReferencedFn: func(ctx context.Context, tID, id uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		uc := NewDeleteDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		err := uc.Execute(bg, tenantID, actorID, depttypes.DeleteDepartmentInput{ID: deptID})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentInUse)
	})

	t.Run("nil tenant or actor returns unauthorized", func(t *testing.T) {
		uc := NewDeleteDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		err := uc.Execute(bg, uuid.Nil, actorID, depttypes.DeleteDepartmentInput{ID: deptID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		err = uc.Execute(bg, tenantID, uuid.Nil, depttypes.DeleteDepartmentInput{ID: deptID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("repo error on delete propagated", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return &entity.Department{ID: deptID, TenantID: tenantID}, nil
			},
			isReferencedFn: func(ctx context.Context, tID, id uuid.UUID) (bool, error) {
				return false, nil
			},
			deleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				return boom
			},
		}
		uc := NewDeleteDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		err := uc.Execute(bg, tenantID, actorID, depttypes.DeleteDepartmentInput{ID: deptID})
		require.ErrorIs(t, err, boom)
	})
}
