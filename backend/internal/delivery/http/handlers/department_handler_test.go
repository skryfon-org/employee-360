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
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

type mockCreateDepartmentUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.CreateDepartmentInput) (*entity.Department, error)
}

func (m *mockCreateDepartmentUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.CreateDepartmentInput) (*entity.Department, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockGetDepartmentUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID, input depttypes.GetDepartmentQuery) (*entity.Department, error)
}

func (m *mockGetDepartmentUC) Execute(ctx context.Context, tenantID uuid.UUID, input depttypes.GetDepartmentQuery) (*entity.Department, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, input)
	}
	return nil, nil
}

type mockListDepartmentsUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error)
}

func (m *mockListDepartmentsUC) Execute(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, input)
	}
	return nil, nil
}

type mockUpdateDepartmentUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.UpdateDepartmentInput) (*entity.Department, error)
}

func (m *mockUpdateDepartmentUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.UpdateDepartmentInput) (*entity.Department, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockDeleteDepartmentUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.DeleteDepartmentInput) error
}

func (m *mockDeleteDepartmentUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.DeleteDepartmentInput) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil
}

func setupDepartmentTestRouter(h *DepartmentHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/departments", withIdentity(testTenantID, testActorID))
	{
		group.POST("", h.Create)
		group.GET("", h.List)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
	return r
}

func TestDepartmentHandler_Create(t *testing.T) {
	t.Run("success 201", func(t *testing.T) {
		deptID := uuid.New()
		createUC := &mockCreateDepartmentUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.CreateDepartmentInput) (*entity.Department, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				return &entity.Department{
					ID:          deptID,
					TenantID:    tenantID,
					Name:        input.Name,
					Description: input.Description,
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				}, nil
			},
		}

		h := NewDepartmentHandler(createUC, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		body, _ := json.Marshal(depttypes.CreateDepartmentRequest{
			Name:        "Engineering",
			Description: "Platform team",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		var resp response.Envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
	})

	t.Run("conflict 409 when name taken", func(t *testing.T) {
		createUC := &mockCreateDepartmentUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.CreateDepartmentInput) (*entity.Department, error) {
				return nil, domainerrors.ErrDepartmentNameTaken
			},
		}

		h := NewDepartmentHandler(createUC, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		body, _ := json.Marshal(depttypes.CreateDepartmentRequest{Name: "Existing"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("bad request 400 when missing name", func(t *testing.T) {
		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		gin.SetMode(gin.TestMode)
		unauthRouter := gin.New()
		unauthRouter.POST("/api/v1/departments", h.Create)

		body, _ := json.Marshal(depttypes.CreateDepartmentRequest{Name: "Engineering"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		unauthRouter.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestDepartmentHandler_GetByID(t *testing.T) {
	deptID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		getUC := &mockGetDepartmentUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input depttypes.GetDepartmentQuery) (*entity.Department, error) {
				assert.Equal(t, testTenantID, tenantID)
				return &entity.Department{
					ID:   deptID,
					Name: "Operations",
				}, nil
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, getUC, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/departments/"+deptID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		getUC := &mockGetDepartmentUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input depttypes.GetDepartmentQuery) (*entity.Department, error) {
				return nil, domainerrors.ErrDepartmentNotFound
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, getUC, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/departments/"+deptID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("bad request 400 on invalid uuid", func(t *testing.T) {
		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/departments/invalid-uuid", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDepartmentHandler_List(t *testing.T) {
	t.Run("success 200 paginated", func(t *testing.T) {
		listUC := &mockListDepartmentsUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error) {
				assert.Equal(t, testTenantID, tenantID)
				return &depttypes.ListDepartmentsResult{
					Departments: []*entity.Department{
						{ID: uuid.New(), Name: "HR"},
					},
					Total:    1,
					Page:     1,
					PageSize: 20,
				}, nil
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, listUC, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/departments?page=1&page_size=20", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp response.Envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Meta)
	})

	t.Run("non-numeric page or page_size returns 400", func(t *testing.T) {
		called := false
		listUC := &mockListDepartmentsUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error) {
				called = true
				return &depttypes.ListDepartmentsResult{}, nil
			},
		}
		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, listUC, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		for _, q := range []string{"page=abc", "page_size=xyz", "page=1.5"} {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/departments?"+q, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code, q)
		}
		assert.False(t, called)
	})

	t.Run("is_active filter parsed", func(t *testing.T) {
		for q, want := range map[string]*bool{"": nil, "?is_active=true": boolP(true), "?is_active=false": boolP(false)} {
			var got *bool
			listUC := &mockListDepartmentsUC{
				executeFn: func(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error) {
					got = input.IsActive
					return &depttypes.ListDepartmentsResult{Page: 1, PageSize: 20}, nil
				},
			}
			h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, listUC, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
			r := setupDepartmentTestRouter(h)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/departments"+q, nil))
			require.Equal(t, http.StatusOK, rec.Code, q)
			assert.Equal(t, want, got, q)
		}
	})

	t.Run("invalid is_active returns 400", func(t *testing.T) {
		called := false
		listUC := &mockListDepartmentsUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error) {
				called = true
				return &depttypes.ListDepartmentsResult{}, nil
			},
		}
		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, listUC, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)
		for _, v := range []string{"maybe", "1", "T", "yes"} {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/departments?is_active="+v, nil))
			require.Equal(t, http.StatusBadRequest, rec.Code, v)
		}
		assert.False(t, called)
	})

	t.Run("empty page params use defaults", func(t *testing.T) {
		listUC := &mockListDepartmentsUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, input depttypes.ListDepartmentsQuery) (*depttypes.ListDepartmentsResult, error) {
				assert.Equal(t, 1, input.Page)
				assert.Equal(t, 20, input.PageSize)
				return &depttypes.ListDepartmentsResult{Page: 1, PageSize: 20}, nil
			},
		}
		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, listUC, &mockUpdateDepartmentUC{}, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/departments?page=&page_size=", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestDepartmentHandler_Update(t *testing.T) {
	deptID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		updateUC := &mockUpdateDepartmentUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.UpdateDepartmentInput) (*entity.Department, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				return &entity.Department{
					ID:   deptID,
					Name: input.Name,
				}, nil
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, updateUC, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		body, _ := json.Marshal(depttypes.UpdateDepartmentRequest{Name: "New HR"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/departments/"+deptID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("conflict 409", func(t *testing.T) {
		updateUC := &mockUpdateDepartmentUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.UpdateDepartmentInput) (*entity.Department, error) {
				return nil, domainerrors.ErrDepartmentNameTaken
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, updateUC, &mockDeleteDepartmentUC{})
		r := setupDepartmentTestRouter(h)

		body, _ := json.Marshal(depttypes.UpdateDepartmentRequest{Name: "Duplicate"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/departments/"+deptID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestDepartmentHandler_Delete(t *testing.T) {
	deptID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		deleteUC := &mockDeleteDepartmentUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.DeleteDepartmentInput) error {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				return nil
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, deleteUC)
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/"+deptID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("conflict 409 when department in use", func(t *testing.T) {
		deleteUC := &mockDeleteDepartmentUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.DeleteDepartmentInput) error {
				return domainerrors.ErrDepartmentInUse
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, deleteUC)
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/"+deptID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		deleteUC := &mockDeleteDepartmentUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input depttypes.DeleteDepartmentInput) error {
				return domainerrors.ErrDepartmentNotFound
			},
		}

		h := NewDepartmentHandler(&mockCreateDepartmentUC{}, &mockGetDepartmentUC{}, &mockListDepartmentsUC{}, &mockUpdateDepartmentUC{}, deleteUC)
		r := setupDepartmentTestRouter(h)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/"+deptID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func boolP(b bool) *bool { return &b }
