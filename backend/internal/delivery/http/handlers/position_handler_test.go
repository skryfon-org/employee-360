package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
	posimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/position"
)

type mockCreatePositionUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.CreatePositionInput) (*entity.Position, error)
}

func (m *mockCreatePositionUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.CreatePositionInput) (*entity.Position, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockGetPositionUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID, input postypes.GetPositionQuery) (*entity.Position, error)
}

func (m *mockGetPositionUC) Execute(ctx context.Context, tenantID uuid.UUID, input postypes.GetPositionQuery) (*entity.Position, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, input)
	}
	return nil, nil
}

type mockListPositionsUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID, input postypes.ListPositionsQuery) (*postypes.ListPositionsResult, error)
}

func (m *mockListPositionsUC) Execute(ctx context.Context, tenantID uuid.UUID, input postypes.ListPositionsQuery) (*postypes.ListPositionsResult, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, input)
	}
	return nil, nil
}

type mockUpdatePositionUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.UpdatePositionInput) (*entity.Position, error)
}

func (m *mockUpdatePositionUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.UpdatePositionInput) (*entity.Position, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockDeletePositionUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.DeletePositionInput) error
}

func (m *mockDeletePositionUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.DeletePositionInput) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil
}

func setupPositionTestRouter(h *PositionHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/positions", withIdentity(testTenantID, testActorID))
	{
		group.POST("", h.Create)
		group.GET("", h.List)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
	return r
}

func TestPositionHandler_Create(t *testing.T) {
	t.Run("success 201", func(t *testing.T) {
		posID := uuid.New()
		createUC := &mockCreatePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.CreatePositionInput) (*entity.Position, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				return &entity.Position{
					ID:          posID,
					TenantID:    tenantID,
					Name:        input.Name,
					Description: input.Description,
					IsActive:    true,
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				}, nil
			},
		}

		h := NewPositionHandler(createUC, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		body, _ := json.Marshal(postypes.CreatePositionRequest{Name: "Software Engineer", Description: "Platform team"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/positions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp response.Envelope
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("invalid json 400", func(t *testing.T) {
		h := NewPositionHandler(nil, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/positions", bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation error from UC 400", func(t *testing.T) {
		createUC := &mockCreatePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.CreatePositionInput) (*entity.Position, error) {
				return nil, posimpl.ErrPositionNameRequired
			},
		}

		h := NewPositionHandler(createUC, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		body, _ := json.Marshal(postypes.CreatePositionRequest{Name: "Eng"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/positions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("duplicate name 409", func(t *testing.T) {
		createUC := &mockCreatePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.CreatePositionInput) (*entity.Position, error) {
				return nil, domainerrors.ErrPositionNameTaken
			},
		}

		h := NewPositionHandler(createUC, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		body, _ := json.Marshal(postypes.CreatePositionRequest{Name: "Existing"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/positions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestPositionHandler_GetByID(t *testing.T) {
	posID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		getUC := &mockGetPositionUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input postypes.GetPositionQuery) (*entity.Position, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, posID, input.ID)
				return &entity.Position{
					ID:        posID,
					TenantID:  tenantID,
					Name:      "Designer",
					IsActive:  true,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}, nil
			},
		}

		h := NewPositionHandler(nil, getUC, nil, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/positions/"+posID.String(), nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid uuid 400", func(t *testing.T) {
		h := NewPositionHandler(nil, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/positions/not-a-uuid", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		getUC := &mockGetPositionUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input postypes.GetPositionQuery) (*entity.Position, error) {
				return nil, domainerrors.ErrPositionNotFound
			},
		}

		h := NewPositionHandler(nil, getUC, nil, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/positions/"+posID.String(), nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestPositionHandler_List(t *testing.T) {
	t.Run("success 200", func(t *testing.T) {
		listUC := &mockListPositionsUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input postypes.ListPositionsQuery) (*postypes.ListPositionsResult, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, 1, input.Page)
				assert.Equal(t, 20, input.PageSize)
				return &postypes.ListPositionsResult{
					Positions: []*entity.Position{
						{ID: uuid.New(), TenantID: tenantID, Name: "A", IsActive: true},
					},
					Total:    1,
					Page:     1,
					PageSize: 20,
				}, nil
			},
		}

		h := NewPositionHandler(nil, nil, listUC, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/positions", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("is_active filter parsing", func(t *testing.T) {
		tr := true
		listUC := &mockListPositionsUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input postypes.ListPositionsQuery) (*postypes.ListPositionsResult, error) {
				assert.Equal(t, &tr, input.IsActive)
				return &postypes.ListPositionsResult{}, nil
			},
		}

		h := NewPositionHandler(nil, nil, listUC, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/positions?is_active=true", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Invalid is_active returns 400
		reqBad := httptest.NewRequest(http.MethodGet, "/api/v1/positions?is_active=maybe", nil)
		recBad := httptest.NewRecorder()
		r.ServeHTTP(recBad, reqBad)
		assert.Equal(t, http.StatusBadRequest, recBad.Code)
	})

	t.Run("invalid page or page_size 400", func(t *testing.T) {
		h := NewPositionHandler(nil, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/positions?page=abc", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/positions?page_size=xyz", nil)
		rec2 := httptest.NewRecorder()
		r.ServeHTTP(rec2, req2)
		assert.Equal(t, http.StatusBadRequest, rec2.Code)
	})
}

func TestPositionHandler_Update(t *testing.T) {
	posID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		updateUC := &mockUpdatePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.UpdatePositionInput) (*entity.Position, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				assert.Equal(t, posID, input.ID)
				return &entity.Position{
					ID:        posID,
					TenantID:  tenantID,
					Name:      input.Name,
					IsActive:  true,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}, nil
			},
		}

		h := NewPositionHandler(nil, nil, nil, updateUC, nil)
		r := setupPositionTestRouter(h)

		body, _ := json.Marshal(postypes.UpdatePositionRequest{Name: "Updated Name"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/positions/"+posID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid uuid 400", func(t *testing.T) {
		h := NewPositionHandler(nil, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/positions/bad-id", bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		updateUC := &mockUpdatePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.UpdatePositionInput) (*entity.Position, error) {
				return nil, domainerrors.ErrPositionNotFound
			},
		}

		h := NewPositionHandler(nil, nil, nil, updateUC, nil)
		r := setupPositionTestRouter(h)

		body, _ := json.Marshal(postypes.UpdatePositionRequest{Name: "Name"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/positions/"+posID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("name taken 409", func(t *testing.T) {
		updateUC := &mockUpdatePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.UpdatePositionInput) (*entity.Position, error) {
				return nil, domainerrors.ErrPositionNameTaken
			},
		}

		h := NewPositionHandler(nil, nil, nil, updateUC, nil)
		r := setupPositionTestRouter(h)

		body, _ := json.Marshal(postypes.UpdatePositionRequest{Name: "Taken"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/positions/"+posID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestPositionHandler_Delete(t *testing.T) {
	posID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		deleteUC := &mockDeletePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.DeletePositionInput) error {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				assert.Equal(t, posID, input.ID)
				return nil
			},
		}

		h := NewPositionHandler(nil, nil, nil, nil, deleteUC)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/positions/"+posID.String(), nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid uuid 400", func(t *testing.T) {
		h := NewPositionHandler(nil, nil, nil, nil, nil)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/positions/bad-uuid", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		deleteUC := &mockDeletePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.DeletePositionInput) error {
				return domainerrors.ErrPositionNotFound
			},
		}

		h := NewPositionHandler(nil, nil, nil, nil, deleteUC)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/positions/"+posID.String(), nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("in use 409", func(t *testing.T) {
		deleteUC := &mockDeletePositionUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input postypes.DeletePositionInput) error {
				return domainerrors.ErrPositionInUse
			},
		}

		h := NewPositionHandler(nil, nil, nil, nil, deleteUC)
		r := setupPositionTestRouter(h)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/positions/"+posID.String(), nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}
