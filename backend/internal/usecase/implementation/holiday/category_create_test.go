package holiday

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
)

func TestCreateCategoryUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	ctx := context.Background()

	t.Run("success creates category with default isActive and records audit log", func(t *testing.T) {
		var createdCat *entity.HolidayCategory
		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, "Public Holiday", name)
				assert.Nil(t, excludeID)
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, cat *entity.HolidayCategory) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				createdCat = cat
				return nil
			},
		}
		auditRecorder := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewCreateCategoryUseCase(cRepo, auditRecorder, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:        "  Public Holiday  ",
			Description: "Official public holidays",
			Color:       "#15803D",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "Public Holiday", res.Name)
		assert.Equal(t, "Official public holidays", res.Description)
		assert.Equal(t, "#15803D", res.Color)
		assert.True(t, res.IsActive)
		assert.False(t, res.IsPublic, "is_public defaults to false")
		assert.Equal(t, tenantID, res.TenantID)
		assert.NotEqual(t, uuid.Nil, res.ID)
		assert.Equal(t, createdCat, res)

		require.Len(t, auditRecorder.Logs, 1)
		log := auditRecorder.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityHolidayCategory, log.EntityType)
		assert.Equal(t, domainaudit.ActionHolidayCategoryCreate, log.Action)
		assert.Equal(t, res.ID, log.EntityID)
		assert.Contains(t, log.Metadata, "Public Holiday")
		assert.Contains(t, log.Metadata, "#15803D")
		assert.Contains(t, log.Metadata, "true")
	})

	t.Run("success creates category with explicit isActive false", func(t *testing.T) {
		inactive := false
		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, cat *entity.HolidayCategory) error {
				return nil
			},
		}
		uc := NewCreateCategoryUseCase(cRepo, nil, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:     "Optional",
			Color:    "#3B82F6",
			IsActive: &inactive,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.IsActive)
	})

	t.Run("fails when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewCreateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, uuid.Nil, actorID, holidaytypes.CreateCategoryInput{
			Name:  "Public",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(ctx, tenantID, uuid.Nil, holidaytypes.CreateCategoryInput{
			Name:  "Public",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("validation: name required", func(t *testing.T) {
		uc := NewCreateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  "",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNameRequired)

		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  "   \t\n ",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNameRequired)
	})

	t.Run("validation: name max 100 characters", func(t *testing.T) {
		uc := NewCreateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  strings.Repeat("a", 101),
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNameTooLong)

		// 101 unicode runes rejected
		_, err = uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  strings.Repeat("🎉", 101),
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNameTooLong)

		// 100 characters accepted
		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, cat *entity.HolidayCategory) error {
				return nil
			},
		}
		ucOk := NewCreateCategoryUseCase(cRepo, nil, nil)
		_, err = ucOk.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  strings.Repeat("a", 100),
			Color: "#15803D",
		})
		require.NoError(t, err)
	})

	t.Run("validation: description max 500 characters", func(t *testing.T) {
		uc := NewCreateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:        "Public",
			Description: strings.Repeat("d", 501),
			Color:       "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryDescTooLong)
	})

	t.Run("validation: color must match hex format ^#[0-9A-Fa-f]{6}$", func(t *testing.T) {
		uc := NewCreateCategoryUseCase(&mockHolidayCategoryRepo{}, nil, nil)

		invalidColors := []string{
			"",
			"red",
			"15803D",    // missing #
			"#FFF",      // 3 hex digits
			"#15803DG",  // invalid char G
			"#1234567",  // 7 hex digits
			"#12345",    // 5 hex digits
			" #15803D ", // with spaces
		}

		for _, color := range invalidColors {
			_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
				Name:  "Test Category",
				Color: color,
			})
			require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryColorInvalid, "expected invalid color for: %s", color)
		}

		validColors := []string{
			"#15803D",
			"#ffffff",
			"#000000",
			"#AaBbCc",
			"#123456",
		}

		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, cat *entity.HolidayCategory) error {
				return nil
			},
		}
		ucValid := NewCreateCategoryUseCase(cRepo, nil, nil)
		for _, color := range validColors {
			res, err := ucValid.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
				Name:  "Valid Color " + color,
				Color: color,
			})
			require.NoError(t, err, "expected valid color for: %s", color)
			assert.Equal(t, color, res.Color)
		}
	})

	t.Run("duplicate name in tenant returns conflict error", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				assert.Equal(t, "Public", name)
				return true, nil
			},
		}
		uc := NewCreateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  "Public",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryDuplicate)
	})

	t.Run("unique violation from repo (race condition) returns ErrHolidayCategoryDuplicate", func(t *testing.T) {
		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, cat *entity.HolidayCategory) error {
				return domainerrors.ErrHolidayCategoryDuplicate
			},
		}
		uc := NewCreateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  "Public",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryDuplicate)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db connection failure")
		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, cat *entity.HolidayCategory) error {
				return expectedErr
			},
		}
		uc := NewCreateCategoryUseCase(cRepo, nil, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, holidaytypes.CreateCategoryInput{
			Name:  "Public",
			Color: "#15803D",
		})
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("tenant isolation: passes tenantID to repo for existsByName and create", func(t *testing.T) {
		tenantB := uuid.New()
		actorB := uuid.New()

		checkedTenant := false
		createdTenant := false

		cRepo := &mockHolidayCategoryRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
				assert.Equal(t, tenantB, tID)
				checkedTenant = true
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, cat *entity.HolidayCategory) error {
				assert.Equal(t, tenantB, tID)
				assert.Equal(t, tenantB, cat.TenantID)
				assert.Equal(t, actorB, aID)
				createdTenant = true
				return nil
			},
		}
		uc := NewCreateCategoryUseCase(cRepo, nil, nil)

		res, err := uc.Execute(ctx, tenantB, actorB, holidaytypes.CreateCategoryInput{
			Name:  "Tenant B Category",
			Color: "#15803D",
		})
		require.NoError(t, err)
		assert.True(t, checkedTenant)
		assert.True(t, createdTenant)
		assert.Equal(t, tenantB, res.TenantID)
	})
}

func TestCreateCategoryUseCase_IsPublic(t *testing.T) {
	tenantID, actorID := uuid.New(), uuid.New()
	yes := true
	var created *entity.HolidayCategory
	cRepo := &mockHolidayCategoryRepo{
		existsByNameFn: func(context.Context, uuid.UUID, string, *uuid.UUID) (bool, error) { return false, nil },
		createFn: func(_ context.Context, tID, _ uuid.UUID, cat *entity.HolidayCategory) error {
			assert.Equal(t, tenantID, tID)
			created = cat
			return nil
		},
	}
	rec := &ucsharedtest.RecordingAuditRecorder{}
	res, err := NewCreateCategoryUseCase(cRepo, rec, nil).Execute(context.Background(), tenantID, actorID, holidaytypes.CreateCategoryInput{
		Name: "Public Holiday", Color: "#15803D", IsPublic: &yes,
	})
	require.NoError(t, err)
	assert.True(t, res.IsPublic)
	assert.True(t, created.IsPublic)
	assert.Equal(t, tenantID, created.TenantID)
	require.Len(t, rec.Logs, 1)
}
