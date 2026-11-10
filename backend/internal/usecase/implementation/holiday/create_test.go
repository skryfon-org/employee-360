package holiday

import (
	"context"
	"errors"
	"strings"
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

func TestCreateHolidayUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	categoryID := uuid.New()
	ctx := context.Background()

	activeCategory := &entity.HolidayCategory{
		ID:        categoryID,
		TenantID:  tenantID,
		Name:      "Public Holiday",
		Color:     "#15803D",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	validDate := time.Date(2026, time.January, 1, 10, 30, 0, 0, time.UTC)
	expectedNormalizedDate := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("success creates holiday and records audit log", func(t *testing.T) {
		var createdHoliday *entity.Holiday
		hRepo := &mockHolidayRepo{
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, expectedNormalizedDate, date)
				assert.Equal(t, "New Year's Day", name)
				assert.Nil(t, excludeID)
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, h *entity.Holiday) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				assert.Equal(t, expectedNormalizedDate, h.Date)
				createdHoliday = h
				return nil
			},
		}
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, categoryID, id)
				return activeCategory, nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewCreateHolidayUseCase(hRepo, cRepo, auditRecorder, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID:  categoryID,
			Name:        "  New Year's Day  ",
			Description: "First day of the year",
			Date:        validDate,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "New Year's Day", res.Name)
		assert.Equal(t, "First day of the year", res.Description)
		assert.Equal(t, expectedNormalizedDate, res.Date)
		assert.Equal(t, categoryID, res.CategoryID)
		assert.Equal(t, tenantID, res.TenantID)
		assert.NotEqual(t, uuid.Nil, res.ID)
		assert.Equal(t, createdHoliday, res)

		require.Len(t, auditRecorder.Logs, 1)
		log := auditRecorder.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityHoliday, log.EntityType)
		assert.Equal(t, domainaudit.ActionHolidayCreate, log.Action)
		assert.Equal(t, res.ID, log.EntityID)
		assert.Contains(t, log.Metadata, "New Year's Day")
		assert.Contains(t, log.Metadata, "2026-01-01")
	})

	t.Run("fails when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, &mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("validation: name required and trimmed", func(t *testing.T) {
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, &mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNameRequired)

		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "   \t\n  ",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNameRequired)
	})

	t.Run("validation: name max 150 characters", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, cRepo, nil, nil)

		// 151 chars rejected
		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       strings.Repeat("a", 151),
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNameTooLong)

		// 151 unicode runes rejected
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       strings.Repeat("🎉", 151),
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNameTooLong)

		// 150 chars accepted
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       strings.Repeat("a", 150),
			Date:       validDate,
		})
		require.NoError(t, err)
	})

	t.Run("validation: date non-zero and within year 2000-2100", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, cRepo, nil, nil)

		// Zero date rejected
		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       time.Time{},
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayDateInvalid)

		// Year 1999 rejected
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)

		// Year 2101 rejected
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       time.Date(2101, 1, 1, 0, 0, 0, 0, time.UTC),
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)

		// Year 2000 accepted
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		})
		require.NoError(t, err)

		// Year 2100 accepted
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC),
		})
		require.NoError(t, err)
	})

	t.Run("validation: categoryID nil rejected", func(t *testing.T) {
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, &mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: uuid.Nil,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("validation: category not found", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("validation: category belongs to another tenant (cross-tenant rejected)", func(t *testing.T) {
		otherTenantID := uuid.New()
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				// Even if repo double-checks, return category from other tenant
				return &entity.HolidayCategory{
					ID:       id,
					TenantID: otherTenantID,
					IsActive: true,
				}, nil
			},
		}
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("validation: soft-deleted category rejected", func(t *testing.T) {
		deletedAt := time.Now().UTC()
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return &entity.HolidayCategory{
					ID:        id,
					TenantID:  tenantID,
					IsActive:  true,
					DeletedAt: &deletedAt,
				}, nil
			},
		}
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("validation: inactive category rejected", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return &entity.HolidayCategory{
					ID:       id,
					TenantID: tenantID,
					IsActive: false,
				}, nil
			},
		}
		uc := NewCreateHolidayUseCase(&mockHolidayRepo{}, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryInactive)
	})

	t.Run("duplicate holiday returns ErrHolidayDuplicate", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		hRepo := &mockHolidayRepo{
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		uc := NewCreateHolidayUseCase(hRepo, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "New Year's Day",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayDuplicate)
	})

	t.Run("unique violation from repo (race condition) mapped to ErrHolidayDuplicate", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		hRepo := &mockHolidayRepo{
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, h *entity.Holiday) error {
				return domainerrors.ErrHolidayDuplicate
			},
		}
		uc := NewCreateHolidayUseCase(hRepo, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Race Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayDuplicate)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db error")
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		hRepo := &mockHolidayRepo{
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, h *entity.Holiday) error {
				return expectedErr
			},
		}
		uc := NewCreateHolidayUseCase(hRepo, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateHolidayInput{
			CategoryID: categoryID,
			Name:       "Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, expectedErr)
	})
}
