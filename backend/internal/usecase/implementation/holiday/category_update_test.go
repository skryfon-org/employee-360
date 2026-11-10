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

func TestUpdateCategoryUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	catID := uuid.New()
	ctx := context.Background()

	baseCategory := func() *entity.HolidayCategory {
		return &entity.HolidayCategory{
			ID:          catID,
			TenantID:    tenantID,
			Name:        "Old Name",
			Description: "Old Desc",
			Color:       "#15803D",
			IsActive:    true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
	}

	t.Run("success updates category and records audit log", func(t *testing.T) {
		cat := baseCategory()
		var updatedCat *entity.HolidayCategory

		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, catID, id)
				return cat, nil
			},
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, "New Name", name)
				require.NotNil(t, excludeID)
				assert.Equal(t, catID, *excludeID)
				return false, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, c *entity.HolidayCategory) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				updatedCat = c
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewUpdateCategoryUseCase(cRepo, auditRecorder, nil)

		newActive := false
		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:          catID,
			Name:        "  New Name  ",
			Description: "New Desc",
			Color:       "#3B82F6",
			IsActive:    &newActive,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "New Name", res.Name)
		assert.Equal(t, "New Desc", res.Description)
		assert.Equal(t, "#3B82F6", res.Color)
		assert.False(t, res.IsActive)
		assert.Equal(t, &actorID, res.UpdatedBy)
		assert.Equal(t, updatedCat, res)

		require.Len(t, auditRecorder.Logs, 1)
		log := auditRecorder.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityHolidayCategory, log.EntityType)
		assert.Equal(t, domainaudit.ActionHolidayCategoryUpdate, log.Action)
		assert.Equal(t, catID, log.EntityID)
		assert.Contains(t, log.Metadata, "New Name")
		assert.Contains(t, log.Metadata, "#3B82F6")
		assert.Contains(t, log.Metadata, "false")
	})

	t.Run("fails when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewUpdateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Name",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Name",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("fails when ID is nil", func(t *testing.T) {
		uc := NewUpdateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    uuid.Nil,
			Name:  "Name",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("fails when category does not exist", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}
		uc := NewUpdateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Name",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})

	t.Run("validation: name required and max 100", func(t *testing.T) {
		uc := NewUpdateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNameRequired)

		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  strings.Repeat("b", 101),
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNameTooLong)
	})

	t.Run("validation: color invalid rejected", func(t *testing.T) {
		uc := NewUpdateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Name",
			Color: "invalid",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryColorInvalid)
	})

	t.Run("duplicate name conflict with another category in tenant", func(t *testing.T) {
		cat := baseCategory()
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return cat, nil
			},
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		uc := NewUpdateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Existing Another Category",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryDuplicate)
	})

	t.Run("repo unique violation mapped to ErrHolidayCategoryDuplicate", func(t *testing.T) {
		cat := baseCategory()
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return cat, nil
			},
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, c *entity.HolidayCategory) error {
				return domainerrors.ErrHolidayCategoryDuplicate
			},
		}
		uc := NewUpdateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Name",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryDuplicate)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		cat := baseCategory()
		expectedErr := errors.New("update db error")
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				return cat, nil
			},
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, c *entity.HolidayCategory) error {
				return expectedErr
			},
		}
		uc := NewUpdateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Name",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("cross-tenant isolation: tenant B cannot update tenant A's category", func(t *testing.T) {
		tenantB := uuid.New()
		actorB := uuid.New()

		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantB, tID)
				assert.Equal(t, catID, id)
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}
		uc := NewUpdateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantB, actorB, holidaytypes.UpdateCategoryInput{
			ID:    catID,
			Name:  "Hacked Category",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
	})
}

func TestUpdateCategoryUseCase_IsPublic(t *testing.T) {
	tenantID, actorID, catID := uuid.New(), uuid.New(), uuid.New()
	run := func(t *testing.T, current bool, in *bool) *entity.HolidayCategory {
		cRepo := &mockHolidayCategoryRepo{
			getByIDFn: func(_ context.Context, tID, id uuid.UUID) (*entity.HolidayCategory, error) {
				assert.Equal(t, tenantID, tID)
				return &entity.HolidayCategory{ID: id, TenantID: tID, Name: "X", Color: "#15803D", IsActive: true, IsPublic: current}, nil
			},
			existsByNameFn: func(context.Context, uuid.UUID, string, *uuid.UUID) (bool, error) { return false, nil },
			updateFn:       func(context.Context, uuid.UUID, uuid.UUID, *entity.HolidayCategory) error { return nil },
		}
		res, err := NewUpdateCategoryUseCase(cRepo, &ucsharedtest.RecordingAuditRecorder{}, nil).Execute(context.Background(), tenantID, actorID,
			holidaytypes.UpdateCategoryInput{ID: catID, Name: "X", Color: "#15803D", IsPublic: in})
		require.NoError(t, err)
		return res
	}
	yes, no := true, false
	assert.True(t, run(t, false, &yes).IsPublic, "can be turned on")
	assert.False(t, run(t, true, &no).IsPublic, "can be turned off")
	assert.True(t, run(t, true, nil).IsPublic, "nil leaves it unchanged")
}
