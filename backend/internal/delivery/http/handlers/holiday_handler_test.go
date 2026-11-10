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
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

type mockCreateHolidayUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error)
}

func (m *mockCreateHolidayUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockGetHolidayUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, query holidaytypes.GetHolidayQuery) (*entity.Holiday, error)
}

func (m *mockGetHolidayUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, query holidaytypes.GetHolidayQuery) (*entity.Holiday, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, query)
	}
	return nil, nil
}

type mockListHolidaysUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error)
}

func (m *mockListHolidaysUC) Execute(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, query)
	}
	return nil, nil
}

type mockUpdateHolidayUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error)
}

func (m *mockUpdateHolidayUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

type mockDeleteHolidayUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteHolidayInput) error
}

func (m *mockDeleteHolidayUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteHolidayInput) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil
}

type mockGetSettingsUC struct {
	executeFn func(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error)
}

func (m *mockGetSettingsUC) Execute(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID)
	}
	return nil, nil
}

type mockUpdateSettingsUC struct {
	executeFn func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateSettingsInput) (*entity.HolidayCalendarSettings, error)
}

func (m *mockUpdateSettingsUC) Execute(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateSettingsInput) (*entity.HolidayCalendarSettings, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, actorID, input)
	}
	return nil, nil
}

func defaultHolidayHandler() *HolidayHandler {
	return NewHolidayHandler(
		&mockCreateHolidayUC{},
		&mockGetHolidayUC{},
		&mockListHolidaysUC{},
		&mockUpdateHolidayUC{},
		&mockDeleteHolidayUC{},
		&mockCreateCategoryUC{},
		&mockGetCategoryUC{},
		&mockListCategoriesUC{},
		&mockUpdateCategoryUC{},
		&mockDeleteCategoryUC{},
		&mockGetSettingsUC{},
		&mockUpdateSettingsUC{},
	)
}

func setupHolidayTestRouter(h *HolidayHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	holGroup := r.Group("/api/v1/holidays", withIdentity(testTenantID, testActorID))
	{
		holGroup.POST("", h.CreateHoliday)
		holGroup.GET("", h.ListHolidays)
		holGroup.GET("/:id", h.GetHolidayByID)
		holGroup.PUT("/:id", h.UpdateHoliday)
		holGroup.DELETE("/:id", h.DeleteHoliday)
	}
	catGroup := r.Group("/api/v1/holiday-categories", withIdentity(testTenantID, testActorID))
	{
		catGroup.POST("", h.CreateCategory)
		catGroup.GET("", h.ListCategories)
		catGroup.GET("/:id", h.GetCategoryByID)
		catGroup.PUT("/:id", h.UpdateCategory)
		catGroup.DELETE("/:id", h.DeleteCategory)
	}
	settingsGroup := r.Group("/api/v1/holiday-calendar/settings", withIdentity(testTenantID, testActorID))
	{
		settingsGroup.GET("", h.GetSettings)
		settingsGroup.PUT("", h.UpdateSettings)
	}
	return r
}

// ============================================================================
// Holiday Handler Tests
// ============================================================================

func TestHolidayHandler_CreateHoliday(t *testing.T) {
	catID := uuid.New()
	dateStr := "2026-12-25"
	parsedDate, _ := time.Parse("2006-01-02", dateStr)

	t.Run("success 201 returns created holiday", func(t *testing.T) {
		holID := uuid.New()
		h := defaultHolidayHandler()
		h.createHolidayUC = &mockCreateHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				assert.Equal(t, "Christmas", input.Name)
				assert.Equal(t, parsedDate, input.Date)
				return &entity.Holiday{
					ID:          holID,
					TenantID:    tenantID,
					CategoryID:  catID,
					Name:        input.Name,
					Description: input.Description,
					Date:        parsedDate,
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.CreateHolidayRequest{
			CategoryID:  catID,
			Name:        "Christmas",
			Description: "Public holiday",
			Date:        dateStr,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holidays", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		var resp response.Envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
	})

	t.Run("conflict 409 on duplicate holiday", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.createHolidayUC = &mockCreateHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error) {
				return nil, domainerrors.ErrHolidayDuplicate
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.CreateHolidayRequest{
			CategoryID: catID,
			Name:       "Christmas",
			Date:       dateStr,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holidays", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("bad request 400 on invalid date format", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		body, _ := json.Marshal(holidaytypes.CreateHolidayRequest{
			CategoryID: catID,
			Name:       "Christmas",
			Date:       "not-a-date",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holidays", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		var resp response.Envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotNil(t, resp.Error)
		assert.Equal(t, domainerrors.ErrHolidayDateInvalid.Error(), resp.Error.Message)
	})

	t.Run("bad request 400 on missing required fields", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/holidays", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("not found 404 when category does not exist", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.createHolidayUC = &mockCreateHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error) {
				return nil, domainerrors.ErrHolidayCategoryNotFound
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.CreateHolidayRequest{
			CategoryID: catID,
			Name:       "Christmas",
			Date:       dateStr,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holidays", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		h := defaultHolidayHandler()
		gin.SetMode(gin.TestMode)
		unauth := gin.New()
		unauth.POST("/api/v1/holidays", h.CreateHoliday)

		body, _ := json.Marshal(holidaytypes.CreateHolidayRequest{
			CategoryID: catID,
			Name:       "Christmas",
			Date:       dateStr,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holidays", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		unauth.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHolidayHandler_GetHolidayByID(t *testing.T) {
	holID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.getHolidayUC = &mockGetHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, query holidaytypes.GetHolidayQuery) (*entity.Holiday, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, testActorID, actorID)
				assert.Equal(t, holID, query.ID)
				return &entity.Holiday{
					ID:       holID,
					TenantID: tenantID,
					Name:     "New Year",
					Date:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays/"+holID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.getHolidayUC = &mockGetHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, query holidaytypes.GetHolidayQuery) (*entity.Holiday, error) {
				return nil, domainerrors.ErrHolidayNotFound
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays/"+holID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("bad request 400 on invalid uuid", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays/not-a-uuid", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHolidayHandler_ListHolidays(t *testing.T) {
	t.Run("success 200 with required year", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.listHolidaysUC = &mockListHolidaysUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, 2026, query.Year)
				assert.Nil(t, query.CategoryID)
				assert.Nil(t, query.Month)
				return []*entity.Holiday{
					{
						ID:       uuid.New(),
						TenantID: tenantID,
						Name:     "New Year",
						Date:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays?year=2026", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing year returns 400", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-integer year returns 400", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays?year=abc", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("optional category_id and month parsed", func(t *testing.T) {
		catID := uuid.New()
		monthVal := 12
		h := defaultHolidayHandler()
		h.listHolidaysUC = &mockListHolidaysUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error) {
				assert.Equal(t, 2026, query.Year)
				require.NotNil(t, query.CategoryID)
				assert.Equal(t, catID, *query.CategoryID)
				require.NotNil(t, query.Month)
				assert.Equal(t, monthVal, *query.Month)
				return []*entity.Holiday{}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays?year=2026&category_id="+catID.String()+"&month=12", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid category_id returns 400", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays?year=2026&category_id=not-uuid", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-integer month returns 400", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays?year=2026&month=december", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHolidayHandler_UpdateHoliday(t *testing.T) {
	holID := uuid.New()
	catID := uuid.New()
	dateStr := "2026-12-25"
	parsedDate, _ := time.Parse("2006-01-02", dateStr)

	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.updateHolidayUC = &mockUpdateHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, holID, input.ID)
				assert.Equal(t, parsedDate, input.Date)
				return &entity.Holiday{
					ID:       holID,
					TenantID: tenantID,
					Name:     input.Name,
					Date:     input.Date,
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.UpdateHolidayRequest{
			CategoryID: catID,
			Name:       "Updated Holiday",
			Date:       dateStr,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holidays/"+holID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("conflict 409", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.updateHolidayUC = &mockUpdateHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error) {
				return nil, domainerrors.ErrHolidayDuplicate
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.UpdateHolidayRequest{
			CategoryID: catID,
			Name:       "Duplicate",
			Date:       dateStr,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holidays/"+holID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.updateHolidayUC = &mockUpdateHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error) {
				return nil, domainerrors.ErrHolidayNotFound
			},
		}

		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.UpdateHolidayRequest{
			CategoryID: catID,
			Name:       "Missing",
			Date:       dateStr,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holidays/"+holID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("bad request 400 on invalid date format", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		body, _ := json.Marshal(holidaytypes.UpdateHolidayRequest{
			CategoryID: catID,
			Name:       "Updated Holiday",
			Date:       "not-a-date",
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holidays/"+holID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		var resp response.Envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotNil(t, resp.Error)
		assert.Equal(t, domainerrors.ErrHolidayDateInvalid.Error(), resp.Error.Message)
	})
}

func TestHolidayHandler_DeleteHoliday(t *testing.T) {
	holID := uuid.New()

	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.deleteHolidayUC = &mockDeleteHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteHolidayInput) error {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, holID, input.ID)
				return nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/holidays/"+holID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.deleteHolidayUC = &mockDeleteHolidayUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteHolidayInput) error {
				return domainerrors.ErrHolidayNotFound
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/holidays/"+holID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ============================================================================
// Holiday Calendar Settings Tests
// ============================================================================

func TestHolidayHandler_GetSettings(t *testing.T) {
	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.getSettingsUC = &mockGetSettingsUC{
			executeFn: func(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
				assert.Equal(t, testTenantID, tenantID)
				return &entity.HolidayCalendarSettings{
					TenantID:     tenantID,
					DefaultView:  "calendar",
					WeekStartDay: 1,
				}, nil
			},
		}

		r := setupHolidayTestRouter(h)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holiday-calendar/settings", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHolidayHandler_UpdateSettings(t *testing.T) {
	t.Run("success 200", func(t *testing.T) {
		h := defaultHolidayHandler()
		h.updateSettingsUC = &mockUpdateSettingsUC{
			executeFn: func(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateSettingsInput) (*entity.HolidayCalendarSettings, error) {
				assert.Equal(t, testTenantID, tenantID)
				assert.Equal(t, "list", input.DefaultView)
				assert.Equal(t, 0, input.WeekStartDay)
				return &entity.HolidayCalendarSettings{
					TenantID:     tenantID,
					DefaultView:  input.DefaultView,
					WeekStartDay: input.WeekStartDay,
				}, nil
			},
		}

		weekStartDay := 0
		r := setupHolidayTestRouter(h)
		body, _ := json.Marshal(holidaytypes.UpdateSettingsRequest{
			DefaultView:  "list",
			WeekStartDay: &weekStartDay,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holiday-calendar/settings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("bad request 400 on invalid default_view", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		weekStartDay := 1
		body, _ := json.Marshal(holidaytypes.UpdateSettingsRequest{
			DefaultView:  "invalid-view",
			WeekStartDay: &weekStartDay,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holiday-calendar/settings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("bad request 400 when missing week_start_day", func(t *testing.T) {
		h := defaultHolidayHandler()
		r := setupHolidayTestRouter(h)

		body := []byte(`{"default_view":"calendar"}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holiday-calendar/settings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
