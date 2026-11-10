package position

import (
	"context"
	"errors"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

func TestCreatePositionUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	ctx := context.Background()

	t.Run("success creates position and audit log", func(t *testing.T) {
		var createdPos *entity.Position
		repo := &mockPositionRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, "Software Engineer", name)
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, p *entity.Position) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				createdPos = p
				return nil
			},
		}
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewCreatePositionUseCase(repo, auditRepo, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{
			Name:        "Software Engineer",
			Description: "Core engineering role",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "Software Engineer", res.Name)
		assert.Equal(t, "Core engineering role", res.Description)
		assert.True(t, res.IsActive)
		assert.Equal(t, tenantID, res.TenantID)
		assert.NotEqual(t, uuid.Nil, res.ID)
		assert.Equal(t, createdPos, res)

		require.Len(t, auditRepo.Logs, 1)
		log := auditRepo.Logs[0]
		assert.Equal(t, tenantID, log.TenantID)
		assert.Equal(t, &actorID, log.ActorUserID)
		assert.Equal(t, domainaudit.EntityPosition, log.EntityType)
		assert.Equal(t, domainaudit.ActionPositionCreate, log.Action)
		assert.Equal(t, res.ID, log.EntityID)
		assert.Contains(t, log.Metadata, "Software Engineer")
		assert.Contains(t, log.Metadata, `"is_active":true`)
	})

	t.Run("defaults is_active to true when nil", func(t *testing.T) {
		repo := &mockPositionRepo{}
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewCreatePositionUseCase(repo, auditRepo, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{
			Name:     "Product Manager",
			IsActive: nil,
		})
		require.NoError(t, err)
		assert.True(t, res.IsActive)
	})

	t.Run("honors is_active = false", func(t *testing.T) {
		repo := &mockPositionRepo{}
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewCreatePositionUseCase(repo, auditRepo, nil)

		fl := false
		res, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{
			Name:     "Legacy Role",
			IsActive: &fl,
		})
		require.NoError(t, err)
		assert.False(t, res.IsActive)
		require.Len(t, auditRepo.Logs, 1)
		assert.Contains(t, auditRepo.Logs[0].Metadata, `"is_active":false`)
	})

	t.Run("trims whitespace from name and description", func(t *testing.T) {
		repo := &mockPositionRepo{}
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewCreatePositionUseCase(repo, auditRepo, nil)

		res, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{
			Name:        "   Designer   ",
			Description: "   UI/UX design team   ",
		})
		require.NoError(t, err)
		assert.Equal(t, "Designer", res.Name)
		assert.Equal(t, "UI/UX design team", res.Description)
	})

	t.Run("unauthorized when tenantID or actorID is nil", func(t *testing.T) {
		uc := NewCreatePositionUseCase(&mockPositionRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)

		_, err := uc.Execute(ctx, uuid.Nil, actorID, postypes.CreatePositionInput{Name: "Engineering"})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(ctx, tenantID, uuid.Nil, postypes.CreatePositionInput{Name: "Engineering"})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("validation: name required", func(t *testing.T) {
		uc := NewCreatePositionUseCase(&mockPositionRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: ""})
		require.ErrorIs(t, err, ErrPositionNameRequired)

		_, err = uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: "   "})
		require.ErrorIs(t, err, ErrPositionNameRequired)
	})

	t.Run("validation: name too long", func(t *testing.T) {
		uc := NewCreatePositionUseCase(&mockPositionRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)

		longName := strings.Repeat("a", 101)
		_, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: longName})
		require.ErrorIs(t, err, ErrPositionNameTooLong)

		// 100 characters is allowed
		validName := strings.Repeat("a", 100)
		_, err = uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: validName})
		require.NoError(t, err)

		// Test multi-byte runes
		rune101 := strings.Repeat("🔥", 101)
		_, err = uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: rune101})
		require.ErrorIs(t, err, ErrPositionNameTooLong)
	})

	t.Run("validation: description too long", func(t *testing.T) {
		uc := NewCreatePositionUseCase(&mockPositionRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)

		longDesc := strings.Repeat("a", 501)
		_, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: "Valid", Description: longDesc})
		require.ErrorIs(t, err, ErrDescriptionTooLong)

		// 500 characters is allowed
		validDesc := strings.Repeat("a", 500)
		_, err = uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: "Valid", Description: validDesc})
		require.NoError(t, err)
	})

	t.Run("duplicate name returns ErrPositionNameTaken", func(t *testing.T) {
		repo := &mockPositionRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				return true, nil
			},
		}
		uc := NewCreatePositionUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: "Existing"})
		require.ErrorIs(t, err, domainerrors.ErrPositionNameTaken)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		expectedErr := errors.New("db error")
		repo := &mockPositionRepo{
			createFn: func(ctx context.Context, tID, aID uuid.UUID, p *entity.Position) error {
				return expectedErr
			},
		}
		uc := NewCreatePositionUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)

		_, err := uc.Execute(ctx, tenantID, actorID, postypes.CreatePositionInput{Name: "Valid"})
		require.ErrorIs(t, err, expectedErr)
	})
}
