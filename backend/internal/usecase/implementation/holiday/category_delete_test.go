package holiday

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
)

func TestDeleteCategoryUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	catID := uuid.New()
	ctx := context.Background()

	activeCategory := &entity.HolidayCategory{
		ID:        catID,
		TenantID:  tenantID,
		Name:      "Public Holiday",
		Color:     "#15803D",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success deletes unused category and records audit log", func(t *testing.T) {
		softDeleted := false
		locked := false
		cRepo := &mockHolidayCategoryRepo{
			getByIDForUpdateFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, catID, id)
				locked = true
				return activeCategory, nil
			},
			countHolidaysByCategoryFn: func(ctx context.Context, tID, cID uuid.UUID) (int64, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, catID, cID)
				return 0, nil
			},
			softDeleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, catID, id)
				assert.Equal(t, actorID, aID)
				softDeleted = true
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewDeleteCategoryUseCase(cRepo, auditRecorder, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: catID})
		require.NoError(t, err)
		assert.True(t, locked)
		assert.True(t, softDeleted)

		require.Len(t, auditRecorder.Logs, 1)
		log := auditRecorder.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityHolidayCategory, log.EntityType)
		assert.Equal(t, domainaudit.ActionHolidayCategoryDelete, log.Action)
		assert.Equal(t, catID, log.EntityID)
		assert.Contains(t, log.Metadata, "Public Holiday")
	})

	t.Run("rejected with ErrHolidayCategoryInUse when holidays reference category", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDForUpdateFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
			countHolidaysByCategoryFn: func(ctx context.Context, tID, cID uuid.UUID) (int64, error) {
				return 3, nil // 3 non-deleted holidays referencing this category
			},
			softDeleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				t.Fatal("SoftDelete should not be called when category is in use")
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewDeleteCategoryUseCase(cRepo, auditRecorder, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryInUse)
		assert.Empty(t, auditRecorder.Logs)
	})

	t.Run("rejected with ErrHolidayCategoryInUse when holiday added concurrently before commit", func(t *testing.T) {
		callCount := 0
		cRepo := &mockHolidayCategoryRepo{
			getByIDForUpdateFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
			countHolidaysByCategoryFn: func(ctx context.Context, tID, cID uuid.UUID) (int64, error) {
				callCount++
				if callCount == 1 {
					return 0, nil // Initial count before soft-delete
				}
				return 1, nil // Concurrent holiday created, caught by second count
			},
			softDeleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewDeleteCategoryUseCase(cRepo, auditRecorder, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryInUse)
		assert.Empty(t, auditRecorder.Logs)
	})

	t.Run("cross-tenant isolation: tenant B cannot delete tenant A's category", func(t *testing.T) {
		tenantB := uuid.New()
		actorB := uuid.New()

		cRepo := &mockHolidayCategoryRepo{
			getByIDForUpdateFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantB, tID)
				assert.Equal(t, catID, id)
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}
		uc := NewDeleteCategoryUseCase(cRepo, nil, nil)

		err := uc.Execute(ctx, tenantB, actorB, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("fails when category not found", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDForUpdateFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}
		uc := NewDeleteCategoryUseCase(cRepo, nil, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("fails when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewDeleteCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("fails when input ID is nil", func(t *testing.T) {
		uc := NewDeleteCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("count error propagates", func(t *testing.T) {
		expectedErr := errors.New("db count error")
		cRepo := &mockHolidayCategoryRepo{
			getByIDForUpdateFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
			countHolidaysByCategoryFn: func(ctx context.Context, tID, cID uuid.UUID) (int64, error) {
				return 0, expectedErr
			},
		}
		uc := NewDeleteCategoryUseCase(cRepo, nil, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("soft-delete error propagates", func(t *testing.T) {
		expectedErr := errors.New("db delete error")
		cRepo := &mockHolidayCategoryRepo{
			getByIDForUpdateFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
			countHolidaysByCategoryFn: func(ctx context.Context, tID, cID uuid.UUID) (int64, error) {
				return 0, nil
			},
			softDeleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				return expectedErr
			},
		}
		uc := NewDeleteCategoryUseCase(cRepo, nil, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: catID})
		require.ErrorIs(t, err, expectedErr)
	})
}
