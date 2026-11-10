package position

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
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
)

func TestDeletePositionUseCase_Execute(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	posID := uuid.New()
	ctx := context.Background()

	t.Run("success soft-deletes and writes audit log", func(t *testing.T) {
		pos := &entity.Position{
			ID:       posID,
			TenantID: tenantID,
			Name:     "Lead Engineer",
		}
		var deletedID uuid.UUID
		repo := &mockPositionRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Position, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, posID, id)
				return pos, nil
			},
			isReferencedFn: func(ctx context.Context, tID, id uuid.UUID) (bool, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, posID, id)
				return false, nil
			},
			deleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, posID, id)
				assert.Equal(t, actorID, aID)
				deletedID = id
				return nil
			},
		}
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		uc := NewDeletePositionUseCase(repo, auditRepo, nil)

		err := uc.Execute(ctx, tenantID, actorID, postypes.DeletePositionInput{ID: posID})
		require.NoError(t, err)
		assert.Equal(t, posID, deletedID)
		assert.Equal(t, 1, repo.forUpdateCalls)

		require.Len(t, auditRepo.Logs, 1)
		log := auditRepo.Logs[0]
		assert.Equal(t, domainaudit.ActionPositionDelete, log.Action)
		assert.Equal(t, domainaudit.EntityPosition, log.EntityType)
		assert.Equal(t, posID, log.EntityID)
		assert.Contains(t, log.Metadata, "Lead Engineer")
	})

	t.Run("unauthorized on nil tenantID or actorID", func(t *testing.T) {
		uc := NewDeletePositionUseCase(&mockPositionRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)

		err := uc.Execute(ctx, uuid.Nil, actorID, postypes.DeletePositionInput{ID: posID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		err = uc.Execute(ctx, tenantID, uuid.Nil, postypes.DeletePositionInput{ID: posID})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("nil uuid returns ErrPositionNotFound", func(t *testing.T) {
		uc := NewDeletePositionUseCase(&mockPositionRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)

		err := uc.Execute(ctx, tenantID, actorID, postypes.DeletePositionInput{ID: uuid.Nil})
		require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)
	})

	t.Run("position not found returns ErrPositionNotFound", func(t *testing.T) {
		repo := &mockPositionRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Position, error) {
				return nil, domainerrors.ErrPositionNotFound
			},
		}
		uc := NewDeletePositionUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)

		err := uc.Execute(ctx, tenantID, actorID, postypes.DeletePositionInput{ID: posID})
		require.ErrorIs(t, err, domainerrors.ErrPositionNotFound)
	})

	t.Run("position in use returns ErrPositionInUse", func(t *testing.T) {
		repo := &mockPositionRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Position, error) {
				return &entity.Position{ID: posID, TenantID: tenantID, Name: "In Use Role"}, nil
			},
			isReferencedFn: func(ctx context.Context, tID, id uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		uc := NewDeletePositionUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)

		err := uc.Execute(ctx, tenantID, actorID, postypes.DeletePositionInput{ID: posID})
		require.ErrorIs(t, err, domainerrors.ErrPositionInUse)
	})

	t.Run("repo delete error propagates", func(t *testing.T) {
		expectedErr := errors.New("delete failed")
		repo := &mockPositionRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Position, error) {
				return &entity.Position{ID: posID, TenantID: tenantID, Name: "Test"}, nil
			},
			isReferencedFn: func(ctx context.Context, tID, id uuid.UUID) (bool, error) {
				return false, nil
			},
			deleteFn: func(ctx context.Context, tID, id, aID uuid.UUID) error {
				return expectedErr
			},
		}
		uc := NewDeletePositionUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)

		err := uc.Execute(ctx, tenantID, actorID, postypes.DeletePositionInput{ID: posID})
		require.ErrorIs(t, err, expectedErr)
	})
}
