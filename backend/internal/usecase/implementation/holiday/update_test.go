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

func TestUpdateHolidayUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	categoryID := uuid.New()
	holidayID := uuid.New()
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

	existingHoliday := &entity.Holiday{
		ID:          holidayID,
		TenantID:    tenantID,
		CategoryID:  categoryID,
		Name:        "New Year's Day",
		Description: "First day",
		Date:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	validDate := time.Date(2026, time.December, 25, 16, 30, 45, 0, time.UTC)
	expectedNormalizedDate := time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC)

	t.Run("success updates holiday and records audit log", func(t *testing.T) {
		var updatedHoliday *entity.Holiday
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, holidayID, id)
				return existingHoliday, nil
			},
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, expectedNormalizedDate, date)
				assert.Equal(t, "Christmas Day", name)
				require.NotNil(t, excludeID)
				assert.Equal(t, holidayID, *excludeID)
				return false, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, h *entity.Holiday) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				assert.Equal(t, expectedNormalizedDate, h.Date)
				updatedHoliday = h
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
		uc := NewUpdateHolidayUseCase(hRepo, cRepo, auditRecorder, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{
			ID:          holidayID,
			CategoryID:  categoryID,
			Name:        "  Christmas Day  ",
			Description: "Year end holiday",
			Date:        validDate,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "Christmas Day", res.Name)
		assert.Equal(t, "Year end holiday", res.Description)
		assert.Equal(t, expectedNormalizedDate, res.Date)
		assert.Equal(t, updatedHoliday, res)

		require.Len(t, auditRecorder.Logs, 1)
		log := auditRecorder.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityHoliday, log.EntityType)
		assert.Equal(t, domainaudit.ActionHolidayUpdate, log.Action)
		assert.Equal(t, holidayID, log.EntityID)
	})

	t.Run("keeping same name and date on same holiday succeeds without conflict", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return existingHoliday, nil
			},
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				// excludeID matches existing holiday, so no duplicate found
				return false, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, h *entity.Holiday) error {
				return nil
			},
		}
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		uc := NewUpdateHolidayUseCase(hRepo, cRepo, nil, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{
			ID:          holidayID,
			CategoryID:  categoryID,
			Name:        "New Year's Day",
			Description: "Updated desc",
			Date:        existingHoliday.Date,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated desc", res.Description)
	})

	t.Run("cross-tenant isolation: tenant B cannot update tenant A's holiday (returns ErrHolidayNotFound)", func(t *testing.T) {
		tenantB := uuid.New()
		actorB := uuid.New()

		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				assert.Equal(t, tenantB, tID)
				// Tenant B querying for holiday from Tenant A gets ErrHolidayNotFound
				return nil, domainerrors.ErrHolidayNotFound
			},
		}
		uc := NewUpdateHolidayUseCase(hRepo, &mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantB, actorB, holidaytypes.UpdateHolidayInput{
			ID:         holidayID,
			CategoryID: categoryID,
			Name:       "Attempted Update",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
	})

	t.Run("update moving holiday onto another existing holiday returns ErrHolidayDuplicate", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return existingHoliday, nil
			},
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				// Another holiday already exists with this name and date
				return true, nil
			},
		}
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		uc := NewUpdateHolidayUseCase(hRepo, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{
			ID:         holidayID,
			CategoryID: categoryID,
			Name:       "Existing Other Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayDuplicate)
	})

	t.Run("validation: missing or invalid fields", func(t *testing.T) {
		uc := NewUpdateHolidayUseCase(&mockHolidayRepo{}, &mockHolidayCategoryRepo{}, nil, nil)

		// Nil tenant
		_, err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: "A", Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		// Nil actor
		_, err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: "A", Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		// Nil holiday ID
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: uuid.Nil, CategoryID: categoryID, Name: "A", Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)

		// Nil category ID
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: uuid.Nil, Name: "A", Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)

		// Empty name
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: "  ", Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNameRequired)

		// Name too long
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: strings.Repeat("x", 151), Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrHolidayNameTooLong)

		// Zero date
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: "A", Date: time.Time{}})
		require.ErrorIs(t, err, domainerrors.ErrHolidayDateInvalid)

		// Invalid year
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: "A", Date: time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)})
		require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)
	})

	t.Run("category validation: inactive or other tenant rejected", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return existingHoliday, nil
			},
		}

		// Inactive category
		cRepoInactive := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return &entity.HolidayCategory{ID: id, TenantID: tenantID, IsActive: false}, nil
			},
		}
		ucInactive := NewUpdateHolidayUseCase(hRepo, cRepoInactive, nil, nil)
		_, err := ucInactive.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: "Valid", Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryInactive)

		// Category from other tenant
		cRepoOtherTenant := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return &entity.HolidayCategory{ID: id, TenantID: uuid.New(), IsActive: true}, nil
			},
		}
		ucOtherTenant := NewUpdateHolidayUseCase(hRepo, cRepoOtherTenant, nil, nil)
		_, err = ucOtherTenant.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{ID: holidayID, CategoryID: categoryID, Name: "Valid", Date: validDate})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("race condition unique violation on repo Update returns ErrHolidayDuplicate", func(t *testing.T) {
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return existingHoliday, nil
			},
			existsDuplicateFn: func(ctx context.Context, tID uuid.UUID, date time.Time, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, h *entity.Holiday) error {
				return domainerrors.ErrHolidayDuplicate
			},
		}
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return activeCategory, nil
			},
		}
		uc := NewUpdateHolidayUseCase(hRepo, cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{
			ID:         holidayID,
			CategoryID: categoryID,
			Name:       "Race Holiday",
			Date:       validDate,
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayDuplicate)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db error")
		hRepo := &mockHolidayRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Holiday, error) {
				return nil, expectedErr
			},
		}
		uc := NewUpdateHolidayUseCase(hRepo, &mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateHolidayInput{
			ID:         holidayID,
			CategoryID: categoryID,
			Name:       "Valid",
			Date:       validDate,
		})
		require.ErrorIs(t, err, expectedErr)
	})
}
