package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

type mockCreateCategoryUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error)
}

func (m *mockCreateCategoryUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockGetCategoryUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error)
}

func (m *mockGetCategoryUC) Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, query)
	}
	return nil, nil
}

type mockListCategoriesUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error)
}

func (m *mockListCategoriesUC) Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, query)
	}
	return &holidaytypes.ListCategoriesResult{Page: 1, PageSize: 20}, nil
}

type mockUpdateCategoryUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error)
}

func (m *mockUpdateCategoryUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockDeleteCategoryUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error
}

func (m *mockDeleteCategoryUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil
}

// ============================================================================
// Holiday Categories Tests
// ============================================================================

func TestHolidayHandler_CreateCategory(t *testing.T) {
	t.Run("success 201", func(t *testing.T) {
		catID := uuid.New()
		h := defaultHolidayHandler()
		h.createCategoryUC = &mockCreateCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, "National Holiday", input.Name)
				assert.Equal(t, "#15803D", input.Color)
				return &entity.HolidayCategory{
					ID:        catID,
					TenantID:  tenantID,
					Name:      input.Name,
					Color:     input.Color,
					IsActive:  true,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.CreateCategoryRequest{
			Name:  "National Holiday",
			Color: "#15803D",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holiday-categories", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		var resp response.Envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
	})

	t.Run("conflict 409 on duplicate name", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.createCategoryUC = &mockCreateCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryDuplicate
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.CreateCategoryRequest{
			Name:  "Existing",
			Color: "#15803D",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holiday-categories", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("bad request 400 on invalid color code", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.createCategoryUC = &mockCreateCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryColorInvalid
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.CreateCategoryRequest{
			Name:  "Invalid Color",
			Color: "not-a-color",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holiday-categories", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHolidayHandler_GetCategoryByID(t *testing.T) {
	catID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.getCategoryUC = &mockGetCategoryUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, catID, query.ID)
				return &entity.HolidayCategory{
					ID:       catID,
					TenantID: tenantID,
					Name:     "Public",
					Color:    "#15803D",
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holiday-categories/"+catID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.getCategoryUC = &mockGetCategoryUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holiday-categories/"+catID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHolidayHandler_ListCategories(t *testing.T) {
	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.listCategoriesUC = &mockListCategoriesUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, 2, query.Page)
				assert.Equal(t, 10, query.PageSize)
				return &holidaytypes.ListCategoriesResult{
					Categories: []*entity.HolidayCategory{
						{ID: uuid.New(), TenantID: tenantID, Name: "Public", Color: "#15803D"},
					},
					Total: 11, Page: 2, PageSize: 10,
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holiday-categories?page=2&page_size=10", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Data []map[string]any `json:"data"`
			Meta map[string]any   `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Len(t, body.Data, 1)
		assert.EqualValues(t, 2, body.Meta["page"])
		assert.EqualValues(t, 10, body.Meta["page_size"])
		assert.EqualValues(t, 11, body.Meta["total_items"])
		assert.EqualValues(t, 2, body.Meta["total_pages"])
	})

	t.Run("non-integer pagination returns 400", func(t *testing.T) {
		for _, q := range []string{"?page=abc", "?page=1.5", "?page_size=x"} {
			r := setupHolidayTestRouter(defaultHolidayHandler())
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/holiday-categories"+q, nil))
			assert.Equal(t, http.StatusBadRequest, rec.Code, q)
		}
	})

	t.Run("is_active filter supported", func(t *testing.T) {
		for q, want := range map[string]*bool{"": nil, "?is_active=true": boolP(true), "?is_active=false": boolP(false)} {
			var got *bool
			h := defaultHolidayHandler()
			h.listCategoriesUC = &mockListCategoriesUC{
				executeFn: func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error) {
					got = query.IsActive
					return &holidaytypes.ListCategoriesResult{Page: 1, PageSize: 20}, nil
				},
			}
			r := setupHolidayTestRouter(h)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/holiday-categories"+q, nil))
			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, want, got, q)
		}
	})

	t.Run("invalid is_active returns 400", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/holiday-categories?is_active=invalid", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHolidayHandler_UpdateCategory(t *testing.T) {
	catID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.updateCategoryUC = &mockUpdateCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, catID, input.ID)
				return &entity.HolidayCategory{
					ID:       catID,
					TenantID: tenantID,
					Name:     input.Name,
					Color:    input.Color,
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.UpdateCategoryRequest{
			Name:  "Updated Category",
			Color: "#3B82F6",
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holiday-categories/"+catID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("conflict 409", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.updateCategoryUC = &mockUpdateCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error) {
				return nil, domainerrors.ErrHolidayCategoryDuplicate
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.UpdateCategoryRequest{
			Name:  "Duplicate",
			Color: "#3B82F6",
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holiday-categories/"+catID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestHolidayHandler_DeleteCategory(t *testing.T) {
	catID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.deleteCategoryUC = &mockDeleteCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, catID, input.ID)
				return nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/holiday-categories/"+catID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("conflict 409 when category is in use", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.deleteCategoryUC = &mockDeleteCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error {
				return domainerrors.ErrHolidayCategoryInUse
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/holiday-categories/"+catID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
		require.Contains(t, rec.Body.String(), `"code":"CATEGORY_IN_USE"`)
		require.Contains(t, rec.Body.String(), "holiday category is in use by holidays")
	})

	t.Run("not found 404", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.deleteCategoryUC = &mockDeleteCategoryUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error {
				return domainerrors.ErrHolidayCategoryNotFound
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/holiday-categories/"+catID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHolidayHandler_ListCategories_OutOfRangePassedToUsecaseForClamping(t *testing.T) {
	r := setupHolidayTestRouter(defaultHolidayHandler())
	for _, q := range []string{"?page=0", "?page=-1", "?page_size=0", "?page_size=101"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/holiday-categories"+q, nil))
		assert.Equal(t, http.StatusOK, rec.Code, q)
	}
}
