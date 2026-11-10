package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	altypes "github.com/skryfon-org/employee-360/backend/internal/types/auditlog"
)

type mockListAuditUC struct {
	fn func(ctx context.Context, tenantID uuid.UUID, in altypes.ListAuditLogsQuery) (*altypes.ListAuditLogsResult, error)
}

func (m *mockListAuditUC) Execute(ctx context.Context, tenantID uuid.UUID, in altypes.ListAuditLogsQuery) (*altypes.ListAuditLogsResult, error) {
	return m.fn(ctx, tenantID, in)
}

type mockGetAuditUC struct {
	fn func(ctx context.Context, tenantID uuid.UUID, in altypes.GetAuditLogQuery) (*entity.AuditLogEntry, error)
}

func (m *mockGetAuditUC) Execute(ctx context.Context, tenantID uuid.UUID, in altypes.GetAuditLogQuery) (*entity.AuditLogEntry, error) {
	return m.fn(ctx, tenantID, in)
}

func setupAuditRouter(h *AuditLogHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1/audit-logs", withIdentity(testTenantID, testActorID))
	g.GET("", h.List)
	g.GET("/:id", h.GetByID)
	return r
}

func auditGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestAuditLogHandler_List(t *testing.T) {
	t.Run("200 maps filters and response shape", func(t *testing.T) {
		actorID := uuid.New()
		entityID := uuid.New()
		var got altypes.ListAuditLogsQuery
		h := NewAuditLogHandler(&mockListAuditUC{fn: func(_ context.Context, tenantID uuid.UUID, in altypes.ListAuditLogsQuery) (*altypes.ListAuditLogsResult, error) {
			assert.Equal(t, testTenantID, tenantID)
			got = in
			return &altypes.ListAuditLogsResult{
				Entries: []*entity.AuditLogEntry{
					{ID: uuid.New(), Action: "department.create", EntityType: "department", EntityID: &entityID,
						Actor:    &entity.AuditActor{ID: actorID, Email: "a@x.com", FirstName: "Ada", LastName: "L"},
						Metadata: `{"name":"HR"}`, CreatedAt: time.Now().UTC()},
					{ID: uuid.New(), Action: "system.job", EntityType: "job", Metadata: `{}`, CreatedAt: time.Now().UTC()},
				},
				Total: 41, Page: 2, PageSize: 20,
			}, nil
		}}, nil)

		q := url.Values{
			"page": {"2"}, "page_size": {"20"}, "action": {"department."}, "actor_user_id": {actorID.String()},
			"entity_type": {"department"}, "from": {"2026-01-01T00:00:00Z"}, "to": {"2026-02-01T00:00:00+02:00"},
		}
		rec := auditGet(setupAuditRouter(h), "/api/v1/audit-logs?"+q.Encode())
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, 2, got.Page)
		assert.Equal(t, "department.", got.Action)
		assert.Equal(t, &actorID, got.ActorUserID)
		require.NotNil(t, got.From)
		require.NotNil(t, got.To)

		var body struct {
			Data []map[string]any `json:"data"`
			Meta map[string]any   `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Data, 2)
		actor := body.Data[0]["actor"].(map[string]any)
		assert.Equal(t, "Ada L", actor["name"])
		assert.Equal(t, "a@x.com", actor["email"])
		assert.Equal(t, map[string]any{"name": "HR"}, body.Data[0]["metadata"])
		assert.Nil(t, body.Data[1]["actor"])
		assert.Nil(t, body.Data[1]["entity_id"])
		assert.EqualValues(t, 41, body.Meta["total_items"])
		assert.EqualValues(t, 3, body.Meta["total_pages"])
	})

	invalid := []string{
		"page=x", "page_size=x", "actor_user_id=nope", "from=yesterday", "to=2026-01-01",
	}
	for _, q := range invalid {
		t.Run("400 "+q, func(t *testing.T) {
			h := NewAuditLogHandler(&mockListAuditUC{fn: func(context.Context, uuid.UUID, altypes.ListAuditLogsQuery) (*altypes.ListAuditLogsResult, error) {
				t.Fatal("usecase must not be called")
				return nil, nil
			}}, nil)
			assert.Equal(t, http.StatusBadRequest, auditGet(setupAuditRouter(h), "/api/v1/audit-logs?"+q).Code)
		})
	}

	t.Run("400 on invalid filter from usecase", func(t *testing.T) {
		h := NewAuditLogHandler(&mockListAuditUC{fn: func(context.Context, uuid.UUID, altypes.ListAuditLogsQuery) (*altypes.ListAuditLogsResult, error) {
			return nil, domainerrors.ErrInvalidAuditLogFilter
		}}, nil)
		assert.Equal(t, http.StatusBadRequest, auditGet(setupAuditRouter(h), "/api/v1/audit-logs").Code)
	})

	t.Run("500 on unexpected error", func(t *testing.T) {
		h := NewAuditLogHandler(&mockListAuditUC{fn: func(context.Context, uuid.UUID, altypes.ListAuditLogsQuery) (*altypes.ListAuditLogsResult, error) {
			return nil, assert.AnError
		}}, nil)
		assert.Equal(t, http.StatusInternalServerError, auditGet(setupAuditRouter(h), "/api/v1/audit-logs").Code)
	})
}

func TestAuditLogHandler_GetByID(t *testing.T) {
	id := uuid.New()
	h := NewAuditLogHandler(nil, &mockGetAuditUC{fn: func(_ context.Context, tenantID uuid.UUID, in altypes.GetAuditLogQuery) (*entity.AuditLogEntry, error) {
		assert.Equal(t, testTenantID, tenantID)
		if in.ID != id {
			return nil, domainerrors.ErrAuditLogNotFound
		}
		return &entity.AuditLogEntry{ID: id, Action: "position.create", EntityType: "position", Metadata: `{}`, CreatedAt: time.Now()}, nil
	}})
	r := setupAuditRouter(h)

	assert.Equal(t, http.StatusOK, auditGet(r, "/api/v1/audit-logs/"+id.String()).Code)
	assert.Equal(t, http.StatusNotFound, auditGet(r, "/api/v1/audit-logs/"+uuid.NewString()).Code)
	assert.Equal(t, http.StatusBadRequest, auditGet(r, "/api/v1/audit-logs/not-a-uuid").Code)
}
