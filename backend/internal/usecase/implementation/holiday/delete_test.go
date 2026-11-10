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

func TestDeleteHolidayUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	holidayID := uuid.New()
	ctx := context.Background()

	existingHoliday := &entity.Holiday{
		ID:        holidayID,
		TenantID:  tenantID,
		Name:      "New Year's Day",
		Date:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success soft-deletes holiday and records audit log", func(t *testing.T) {
		deletedID := uuid.Nil
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, holidayID, id)
				return existingHoliday, nil
			},
			softDeleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, holidayID, id)
				assert.Equal(t, actorID, aID)
				deletedID = id
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewDeleteHolidayUseCase(hRepo, auditRecorder, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteHolidayInput{ID: holidayID})
		require.NoError(t, err)
		assert.Equal(t, holidayID, deletedID)

		require.Len(t, auditRecorder.Logs, 1)
		log := auditRecorder.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityHoliday, log.EntityType)
		assert.Equal(t, domainaudit.ActionHolidayDelete, log.Action)
		assert.Equal(t, holidayID, log.EntityID)
		assert.Contains(t, log.Metadata, "New Year's Day")
	})

	t.Run("cross-tenant isolation: tenant B cannot delete tenant A's holiday (returns ErrHolidayNotFound)", func(t *testing.T) {
		tenantB := uuid.New()
		actorB := uuid.New()

		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				assert.Equal(t, tenantB, tID)
				// Tenant B querying for holiday from Tenant A gets ErrHolidayNotFound
				return nil, domainerrors.ErrHolidayNotFound
			},
		}
		uc := NewDeleteHolidayUseCase(hRepo, nil, nil)

		err := uc.Execute(ctx, tenantB, actorB, holidaytypes.DeleteHolidayInput{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
	})

	t.Run("fails when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewDeleteHolidayUseCase(&mockHolidayRepo{}, nil, nil)

		err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.DeleteHolidayInput{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.DeleteHolidayInput{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("fails when holiday ID is nil", func(t *testing.T) {
		uc := NewDeleteHolidayUseCase(&mockHolidayRepo{}, nil, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteHolidayInput{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
	})

	t.Run("non-existent holiday returns ErrHolidayNotFound", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return nil, domainerrors.ErrHolidayNotFound
			},
		}
		uc := NewDeleteHolidayUseCase(hRepo, nil, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteHolidayInput{ID: holidayID})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db error")
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return existingHoliday, nil
			},
			softDeleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				return expectedErr
			},
		}
		uc := NewDeleteHolidayUseCase(hRepo, nil, nil)

		err := uc.Execute(ctx, tenantID, actorID, holidaytypes.DeleteHolidayInput{ID: holidayID})
		require.ErrorIs(t, err, expectedErr)
	})
}
