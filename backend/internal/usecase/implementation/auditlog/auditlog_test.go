package auditlog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	altypes "github.com/skryfon-org/employee-360/backend/internal/types/auditlog"
)

type mockRepo struct {
	listFn func(ctx context.Context, tenantID uuid.UUID, f repository.AuditLogFilter, limit, offset int) ([]*entity.AuditLogEntry, int64, error)
	getFn  func(ctx context.Context, tenantID, id uuid.UUID) (*entity.AuditLogEntry, error)
}

func (m *mockRepo) List(ctx context.Context, tenantID uuid.UUID, f repository.AuditLogFilter, limit, offset int) ([]*entity.AuditLogEntry, int64, error) {
	return m.listFn(ctx, tenantID, f, limit, offset)
}

func (m *mockRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.AuditLogEntry, error) {
	return m.getFn(ctx, tenantID, id)
}

func TestListAuditLogs(t *testing.T) {
	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("pagination, filters and metadata redaction", func(t *testing.T) {
		actor := uuid.New()
		from := time.Now().Add(-time.Hour)
		to := time.Now()
		repo := &mockRepo{listFn: func(_ context.Context, tID uuid.UUID, f repository.AuditLogFilter, limit, offset int) ([]*entity.AuditLogEntry, int64, error) {
			assert.Equal(t, tenantID, tID)
			assert.Equal(t, "department.", f.Action)
			assert.True(t, f.ActionPrefix)
			assert.Equal(t, &actor, f.ActorUserID)
			assert.Equal(t, "department", f.EntityType)
			assert.Equal(t, &from, f.From)
			assert.Equal(t, &to, f.To)
			assert.Equal(t, 10, limit)
			assert.Equal(t, 20, offset)
			return []*entity.AuditLogEntry{{ID: uuid.New(), Metadata: `{"name":"HR","token":"abc","nested":{"password_hash":"x","ok":1}}`}}, 25, nil
		}}
		out, err := NewListAuditLogsUseCase(repo).Execute(ctx, tenantID, altypes.ListAuditLogsQuery{
			Page: 3, PageSize: 10, Action: " department. ", ActorUserID: &actor, EntityType: "department", From: &from, To: &to,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(25), out.Total)
		assert.Equal(t, 3, out.Page)
		assert.Equal(t, 10, out.PageSize)
		var meta map[string]any
		require.NoError(t, json.Unmarshal([]byte(out.Entries[0].Metadata), &meta))
		assert.Equal(t, map[string]any{"name": "HR", "nested": map[string]any{"ok": float64(1)}}, meta)
	})

	t.Run("exact action is not prefix; defaults applied", func(t *testing.T) {
		repo := &mockRepo{listFn: func(_ context.Context, _ uuid.UUID, f repository.AuditLogFilter, limit, offset int) ([]*entity.AuditLogEntry, int64, error) {
			assert.False(t, f.ActionPrefix)
			assert.Equal(t, 20, limit)
			assert.Equal(t, 0, offset)
			return nil, 0, nil
		}}
		out, err := NewListAuditLogsUseCase(repo).Execute(ctx, tenantID, altypes.ListAuditLogsQuery{Page: -1, PageSize: 0, Action: "position.create"})
		require.NoError(t, err)
		assert.Equal(t, 1, out.Page)
		assert.Equal(t, 20, out.PageSize)
	})

	t.Run("page size above max is clamped to 100", func(t *testing.T) {
		repo := &mockRepo{listFn: func(_ context.Context, _ uuid.UUID, _ repository.AuditLogFilter, limit, offset int) ([]*entity.AuditLogEntry, int64, error) {
			assert.Equal(t, 100, limit)
			assert.Equal(t, 0, offset)
			return nil, 0, nil
		}}
		out, err := NewListAuditLogsUseCase(repo).Execute(ctx, tenantID, altypes.ListAuditLogsQuery{PageSize: 500})
		require.NoError(t, err)
		assert.Equal(t, 100, out.PageSize)
	})

	t.Run("from after to is invalid", func(t *testing.T) {
		from := time.Now()
		to := from.Add(-time.Minute)
		_, err := NewListAuditLogsUseCase(&mockRepo{}).Execute(ctx, tenantID, altypes.ListAuditLogsQuery{From: &from, To: &to})
		require.ErrorIs(t, err, domainerrors.ErrInvalidAuditLogFilter)
	})

	t.Run("overlong filter is invalid", func(t *testing.T) {
		long := make([]byte, 101)
		for i := range long {
			long[i] = 'a'
		}
		_, err := NewListAuditLogsUseCase(&mockRepo{}).Execute(ctx, tenantID, altypes.ListAuditLogsQuery{Action: string(long)})
		require.ErrorIs(t, err, domainerrors.ErrInvalidAuditLogFilter)
	})

	t.Run("nil tenant unauthorized", func(t *testing.T) {
		_, err := NewListAuditLogsUseCase(&mockRepo{}).Execute(ctx, uuid.Nil, altypes.ListAuditLogsQuery{})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		boom := errors.New("boom")
		repo := &mockRepo{listFn: func(context.Context, uuid.UUID, repository.AuditLogFilter, int, int) ([]*entity.AuditLogEntry, int64, error) {
			return nil, 0, boom
		}}
		_, err := NewListAuditLogsUseCase(repo).Execute(ctx, tenantID, altypes.ListAuditLogsQuery{})
		require.ErrorIs(t, err, boom)
	})
}

func TestGetAuditLog(t *testing.T) {
	tenantID := uuid.New()
	id := uuid.New()
	ctx := context.Background()

	t.Run("success redacts metadata", func(t *testing.T) {
		repo := &mockRepo{getFn: func(_ context.Context, tID, i uuid.UUID) (*entity.AuditLogEntry, error) {
			assert.Equal(t, tenantID, tID)
			assert.Equal(t, id, i)
			return &entity.AuditLogEntry{ID: id, Metadata: `{"email":"a@b.c","reset_token":"zzz"}`}, nil
		}}
		e, err := NewGetAuditLogUseCase(repo).Execute(ctx, tenantID, altypes.GetAuditLogQuery{ID: id})
		require.NoError(t, err)
		assert.JSONEq(t, `{"email":"a@b.c"}`, e.Metadata)
	})

	t.Run("not found propagates", func(t *testing.T) {
		repo := &mockRepo{getFn: func(context.Context, uuid.UUID, uuid.UUID) (*entity.AuditLogEntry, error) {
			return nil, domainerrors.ErrAuditLogNotFound
		}}
		_, err := NewGetAuditLogUseCase(repo).Execute(ctx, tenantID, altypes.GetAuditLogQuery{ID: id})
		require.ErrorIs(t, err, domainerrors.ErrAuditLogNotFound)
	})

	t.Run("nil id is not found, nil tenant unauthorized", func(t *testing.T) {
		uc := NewGetAuditLogUseCase(&mockRepo{})
		_, err := uc.Execute(ctx, tenantID, altypes.GetAuditLogQuery{})
		require.ErrorIs(t, err, domainerrors.ErrAuditLogNotFound)
		_, err = uc.Execute(ctx, uuid.Nil, altypes.GetAuditLogQuery{ID: id})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})
}

func TestSanitizeMetadata(t *testing.T) {
	assert.Equal(t, "{}", sanitizeMetadata(""))
	assert.Equal(t, "{}", sanitizeMetadata("not json"))
	assert.JSONEq(t, `{"a":[{"b":1}]}`, sanitizeMetadata(`{"a":[{"b":1,"Password":"x","apiKey":"y"}]}`))
}
