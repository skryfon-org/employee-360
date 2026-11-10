package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// Distinctive markers for the owner tenant's data so cross-tenant read tests
// can assert they never appear in another tenant's response.
const (
	ownerHolidayName  = "OwnerTenantSecretHoliday"
	ownerCategoryName = "OwnerTenantSecretCategory"
	ownerSettingsYear = 2099
)

type fakeHolidayUseCases struct {
	ownerTenant uuid.UUID
	holidayID   uuid.UUID
	categoryID  uuid.UUID

	createHolidayGotTenant  uuid.UUID
	listHolidayGotTenant    uuid.UUID
	createCategoryGotTenant uuid.UUID
	updateSettingsGotTenant uuid.UUID
}

func (f *fakeHolidayUseCases) CreateHoliday(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateHolidayInput) (*entity.Holiday, error) {
	f.createHolidayGotTenant = tenantID
	return &entity.Holiday{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		Date:        input.Date,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func (f *fakeHolidayUseCases) GetHoliday(ctx context.Context, tenantID, actorID uuid.UUID, query holidaytypes.GetHolidayQuery) (*entity.Holiday, error) {
	if query.ID == f.holidayID && tenantID != f.ownerTenant {
		return nil, domainerrors.ErrHolidayNotFound
	}
	return &entity.Holiday{
		ID:       query.ID,
		TenantID: tenantID,
		Name:     "Test Holiday",
		Date:     time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (f *fakeHolidayUseCases) ListHolidays(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error) {
	f.listHolidayGotTenant = tenantID
	if f.ownerTenant != uuid.Nil && tenantID != f.ownerTenant {
		return []*entity.Holiday{}, nil
	}
	return []*entity.Holiday{
		{
			ID:       uuid.New(),
			TenantID: tenantID,
			Name:     ownerHolidayName,
			Date:     time.Date(query.Year, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (f *fakeHolidayUseCases) UpdateHoliday(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateHolidayInput) (*entity.Holiday, error) {
	if input.ID == f.holidayID && tenantID != f.ownerTenant {
		return nil, domainerrors.ErrHolidayNotFound
	}
	return &entity.Holiday{
		ID:       input.ID,
		TenantID: tenantID,
		Name:     input.Name,
		Date:     input.Date,
	}, nil
}

func (f *fakeHolidayUseCases) DeleteHoliday(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteHolidayInput) error {
	if input.ID == f.holidayID && tenantID != f.ownerTenant {
		return domainerrors.ErrHolidayNotFound
	}
	return nil
}

func (f *fakeHolidayUseCases) CreateCategory(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error) {
	f.createCategoryGotTenant = tenantID
	return &entity.HolidayCategory{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      input.Name,
		Color:     input.Color,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeHolidayUseCases) GetCategory(ctx context.Context, tenantID uuid.UUID, query holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error) {
	if query.ID == f.categoryID && tenantID != f.ownerTenant {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}
	return &entity.HolidayCategory{
		ID:       query.ID,
		TenantID: tenantID,
		Name:     "Public",
		Color:    "#15803D",
		IsActive: true,
	}, nil
}

func (f *fakeHolidayUseCases) ListCategories(ctx context.Context, tenantID uuid.UUID, query holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error) {
	if f.ownerTenant != uuid.Nil && tenantID != f.ownerTenant {
		return &holidaytypes.ListCategoriesResult{Categories: []*entity.HolidayCategory{}, Page: 1, PageSize: 20}, nil
	}
	return &holidaytypes.ListCategoriesResult{
		Categories: []*entity.HolidayCategory{
			{
				ID:       uuid.New(),
				TenantID: tenantID,
				Name:     ownerCategoryName,
				Color:    "#15803D",
				IsActive: true,
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}, nil
}

func (f *fakeHolidayUseCases) UpdateCategory(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error) {
	if input.ID == f.categoryID && tenantID != f.ownerTenant {
		return nil, domainerrors.ErrHolidayCategoryNotFound
	}
	return &entity.HolidayCategory{
		ID:       input.ID,
		TenantID: tenantID,
		Name:     input.Name,
		Color:    input.Color,
	}, nil
}

func (f *fakeHolidayUseCases) DeleteCategory(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.DeleteCategoryInput) error {
	if input.ID == f.categoryID && tenantID != f.ownerTenant {
		return domainerrors.ErrHolidayCategoryNotFound
	}
	return nil
}

func (f *fakeHolidayUseCases) GetSettings(ctx context.Context, tenantID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
	if f.ownerTenant != uuid.Nil && tenantID != f.ownerTenant {
		// Foreign tenant gets defaults, never the owner's configured values.
		return &entity.HolidayCalendarSettings{TenantID: tenantID, DefaultView: "calendar", WeekStartDay: 1}, nil
	}
	year := ownerSettingsYear
	return &entity.HolidayCalendarSettings{
		TenantID:     tenantID,
		DefaultView:  "list",
		DefaultYear:  &year,
		WeekStartDay: 1,
	}, nil
}

func (f *fakeHolidayUseCases) UpdateSettings(ctx context.Context, tenantID, actorID uuid.UUID, input holidaytypes.UpdateSettingsInput) (*entity.HolidayCalendarSettings, error) {
	f.updateSettingsGotTenant = tenantID
	return &entity.HolidayCalendarSettings{
		TenantID:     tenantID,
		DefaultView:  input.DefaultView,
		DefaultYear:  input.DefaultYear,
		WeekStartDay: input.WeekStartDay,
	}, nil
}

// Adapt to interfaces
type fakeCreateHolidayAdapter struct{ f *fakeHolidayUseCases }

func (a fakeCreateHolidayAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, in holidaytypes.CreateHolidayInput) (*entity.Holiday, error) {
	return a.f.CreateHoliday(ctx, tID, aID, in)
}

type fakeGetHolidayAdapter struct{ f *fakeHolidayUseCases }

func (a fakeGetHolidayAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, q holidaytypes.GetHolidayQuery) (*entity.Holiday, error) {
	return a.f.GetHoliday(ctx, tID, aID, q)
}

type fakeListHolidaysAdapter struct{ f *fakeHolidayUseCases }

func (a fakeListHolidaysAdapter) Execute(ctx context.Context, tID uuid.UUID, q holidaytypes.ListHolidaysByYearQuery) ([]*entity.Holiday, error) {
	return a.f.ListHolidays(ctx, tID, q)
}

type fakeUpdateHolidayAdapter struct{ f *fakeHolidayUseCases }

func (a fakeUpdateHolidayAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, in holidaytypes.UpdateHolidayInput) (*entity.Holiday, error) {
	return a.f.UpdateHoliday(ctx, tID, aID, in)
}

type fakeDeleteHolidayAdapter struct{ f *fakeHolidayUseCases }

func (a fakeDeleteHolidayAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, in holidaytypes.DeleteHolidayInput) error {
	return a.f.DeleteHoliday(ctx, tID, aID, in)
}

type fakeCreateCategoryAdapter struct{ f *fakeHolidayUseCases }

func (a fakeCreateCategoryAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, in holidaytypes.CreateCategoryInput) (*entity.HolidayCategory, error) {
	return a.f.CreateCategory(ctx, tID, aID, in)
}

type fakeGetCategoryAdapter struct{ f *fakeHolidayUseCases }

func (a fakeGetCategoryAdapter) Execute(ctx context.Context, tID uuid.UUID, q holidaytypes.GetCategoryQuery) (*entity.HolidayCategory, error) {
	return a.f.GetCategory(ctx, tID, q)
}

type fakeListCategoriesAdapter struct{ f *fakeHolidayUseCases }

func (a fakeListCategoriesAdapter) Execute(ctx context.Context, tID uuid.UUID, q holidaytypes.ListCategoriesQuery) (*holidaytypes.ListCategoriesResult, error) {
	return a.f.ListCategories(ctx, tID, q)
}

type fakeUpdateCategoryAdapter struct{ f *fakeHolidayUseCases }

func (a fakeUpdateCategoryAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, in holidaytypes.UpdateCategoryInput) (*entity.HolidayCategory, error) {
	return a.f.UpdateCategory(ctx, tID, aID, in)
}

type fakeDeleteCategoryAdapter struct{ f *fakeHolidayUseCases }

func (a fakeDeleteCategoryAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, in holidaytypes.DeleteCategoryInput) error {
	return a.f.DeleteCategory(ctx, tID, aID, in)
}

type fakeGetSettingsAdapter struct{ f *fakeHolidayUseCases }

func (a fakeGetSettingsAdapter) Execute(ctx context.Context, tID uuid.UUID) (*entity.HolidayCalendarSettings, error) {
	return a.f.GetSettings(ctx, tID)
}

type fakeUpdateSettingsAdapter struct{ f *fakeHolidayUseCases }

func (a fakeUpdateSettingsAdapter) Execute(ctx context.Context, tID, aID uuid.UUID, in holidaytypes.UpdateSettingsInput) (*entity.HolidayCalendarSettings, error) {
	return a.f.UpdateSettings(ctx, tID, aID, in)
}

func holidayEngine(t *testing.T, f *fakeHolidayUseCases) (*gin.Engine, domainservice.TokenService) {
	return holidayEngineWith(t, f, nil)
}

// holidayEngineWith is holidayEngine plus an optional hook to wire extra containers.
func holidayEngineWith(t *testing.T, f *fakeHolidayUseCases, extra func(*container.AppContainer)) (*gin.Engine, domainservice.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtSvc := setupTestTokenService(t)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{
		TokenService:     jwtSvc,
		IdentityVerifier: verifierOf(jwtSvc),
	}

	h := handlers.NewHolidayHandler(
		fakeCreateHolidayAdapter{f: f},
		fakeGetHolidayAdapter{f: f},
		fakeListHolidaysAdapter{f: f},
		fakeUpdateHolidayAdapter{f: f},
		fakeDeleteHolidayAdapter{f: f},
		fakeCreateCategoryAdapter{f: f},
		fakeGetCategoryAdapter{f: f},
		fakeListCategoriesAdapter{f: f},
		fakeUpdateCategoryAdapter{f: f},
		fakeDeleteCategoryAdapter{f: f},
		fakeGetSettingsAdapter{f: f},
		fakeUpdateSettingsAdapter{f: f},
	)

	ctr.Holiday = &container.HolidayContainer{
		Handler: h,
	}
	// Holiday read routes sit behind the profile gate; default to a pass-through.
	ctr.Employee = &container.EmployeeContainer{ProfileGate: passGate{}}
	if extra != nil {
		extra(ctr)
	}

	return SetupRouter(cfg, zerolog.Nop(), ctr), jwtSvc
}

func TestHolidayRoutes_RoleBasedAccess(t *testing.T) {
	holID := uuid.New()
	catID := uuid.New()
	tenant := uuid.New()

	f := &fakeHolidayUseCases{
		ownerTenant: tenant,
		holidayID:   holID,
		categoryID:  catID,
	}
	engine, jwtSvc := holidayEngine(t, f)

	validHolidayBody := `{"category_id":"` + catID.String() + `","name":"Christmas","date":"2026-12-25"}`
	validCategoryBody := `{"name":"Public","color":"#15803D"}`
	validSettingsBody := `{"default_view":"calendar","week_start_day":1}`

	// Read routes: admin and employee both get 200; unauth gets 401
	readRoutes := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/holidays?year=2026"},
		{http.MethodGet, "/api/v1/holidays/" + holID.String()},
		{http.MethodGet, "/api/v1/holiday-categories"},
		{http.MethodGet, "/api/v1/holiday-categories/" + catID.String()},
		{http.MethodGet, "/api/v1/holiday-calendar/settings"},
	}

	for _, r := range readRoutes {
		t.Run("read "+r.method+" "+r.path, func(t *testing.T) {
			// No token -> 401
			rec := do(engine, r.method, r.path, "", "")
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "no token should return 401")

			// Bad token -> 401
			rec = do(engine, r.method, r.path, "Bearer garbage", "")
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "bad token should return 401")

			// Employee -> 200
			rec = do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, "employee"), "")
			assert.Equal(t, http.StatusOK, rec.Code, "employee should have read access (200)")

			// Admin -> 200
			rec = do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, "admin"), "")
			assert.Equal(t, http.StatusOK, rec.Code, "admin should have read access (200)")

			// Super Admin -> 200
			rec = do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, "super_admin"), "")
			assert.Equal(t, http.StatusOK, rec.Code, "super_admin should have read access (200)")
		})
	}

	// Write routes: employee gets 403; admin gets 2xx; super_admin gets 2xx; unauth gets 401
	writeRoutes := []struct {
		method, path, body string
		wantSuccessCode    int
	}{
		{http.MethodPost, "/api/v1/holidays", validHolidayBody, http.StatusCreated},
		{http.MethodPut, "/api/v1/holidays/" + holID.String(), validHolidayBody, http.StatusOK},
		{http.MethodDelete, "/api/v1/holidays/" + holID.String(), "", http.StatusOK},
		{http.MethodPost, "/api/v1/holiday-categories", validCategoryBody, http.StatusCreated},
		{http.MethodPut, "/api/v1/holiday-categories/" + catID.String(), validCategoryBody, http.StatusOK},
		{http.MethodDelete, "/api/v1/holiday-categories/" + catID.String(), "", http.StatusOK},
		{http.MethodPut, "/api/v1/holiday-calendar/settings", validSettingsBody, http.StatusOK},
	}

	for _, r := range writeRoutes {
		t.Run("write "+r.method+" "+r.path, func(t *testing.T) {
			// No token -> 401
			rec := do(engine, r.method, r.path, "", r.body)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "no token should return 401")

			// Bad token -> 401
			rec = do(engine, r.method, r.path, "Bearer invalid", r.body)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "bad token should return 401")

			// Employee -> 403 Forbidden on writes
			rec = do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, "employee"), r.body)
			assert.Equal(t, http.StatusForbidden, rec.Code, "employee must get 403 on write route")

			// Admin -> 2xx
			rec = do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, "admin"), r.body)
			assert.Equal(t, r.wantSuccessCode, rec.Code, "admin must get 2xx on write route: %s", rec.Body.String())

			// Super Admin -> 2xx
			rec = do(engine, r.method, r.path, bearer(t, jwtSvc, tenant, "super_admin"), r.body)
			assert.Equal(t, r.wantSuccessCode, rec.Code, "super_admin must get 2xx on write route: %s", rec.Body.String())
		})
	}
}

func TestHolidayRoutes_CrossTenant404(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	holID := uuid.New()
	catID := uuid.New()

	f := &fakeHolidayUseCases{
		ownerTenant: tenantA,
		holidayID:   holID,
		categoryID:  catID,
	}
	engine, jwtSvc := holidayEngine(t, f)

	routes := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/v1/holidays/" + holID.String(), ""},
		{http.MethodPut, "/api/v1/holidays/" + holID.String(), `{"category_id":"` + catID.String() + `","name":"Cross","date":"2026-12-25"}`},
		{http.MethodDelete, "/api/v1/holidays/" + holID.String(), ""},
		{http.MethodGet, "/api/v1/holiday-categories/" + catID.String(), ""},
		{http.MethodPut, "/api/v1/holiday-categories/" + catID.String(), `{"name":"Cross","color":"#15803D"}`},
		{http.MethodDelete, "/api/v1/holiday-categories/" + catID.String(), ""},
	}

	for _, r := range routes {
		t.Run("tenantB accessing tenantA resource "+r.method+" "+r.path, func(t *testing.T) {
			rec := do(engine, r.method, r.path, bearer(t, jwtSvc, tenantB, "admin"), r.body)
			assert.Equal(t, http.StatusNotFound, rec.Code, "cross-tenant admin should get 404: %s", rec.Body.String())

			// Super Admin in tenantB also receives 404 (strictly tenant-scoped)
			rec = do(engine, r.method, r.path, bearer(t, jwtSvc, tenantB, "super_admin"), r.body)
			assert.Equal(t, http.StatusNotFound, rec.Code, "cross-tenant super_admin should get 404: %s", rec.Body.String())
		})
	}
}

func TestHolidayRoutes_CrossTenantReadsExposeNoForeignData(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	f := &fakeHolidayUseCases{ownerTenant: tenantA}
	engine, jwtSvc := holidayEngine(t, f)

	paths := []string{
		"/api/v1/holidays?year=2026",
		"/api/v1/holiday-categories",
		"/api/v1/holiday-calendar/settings",
	}
	foreignMarkers := []string{ownerHolidayName, ownerCategoryName, "2099", tenantA.String()}

	for _, p := range paths {
		for _, role := range []string{"employee", "admin"} {
			t.Run(role+" tenantB "+p, func(t *testing.T) {
				// Sanity: the owner tenant does see its own data on this route.
				own := do(engine, http.MethodGet, p, bearer(t, jwtSvc, tenantA, role), "")
				require.Equal(t, http.StatusOK, own.Code)

				rec := do(engine, http.MethodGet, p, bearer(t, jwtSvc, tenantB, role), "")
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				for _, m := range foreignMarkers {
					assert.NotContains(t, rec.Body.String(), m, "tenant B response must not contain tenant A data")
				}
			})
		}
	}
}

func TestHolidayRoutes_TenantFromTokenNotClient(t *testing.T) {
	tenantA := uuid.New()
	spoofedTenant := uuid.New()
	catID := uuid.New()

	f := &fakeHolidayUseCases{
		ownerTenant: tenantA,
	}
	engine, jwtSvc := holidayEngine(t, f)

	t.Run("POST /holidays ignores forged tenant_id in body and header", func(t *testing.T) {
		body := `{"category_id":"` + catID.String() + `","name":"Christmas","date":"2026-12-25","tenant_id":"` + spoofedTenant.String() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holidays", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer(t, jwtSvc, tenantA, "admin"))
		req.Header.Set("X-Tenant-ID", spoofedTenant.String())
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, tenantA, f.createHolidayGotTenant, "usecase must receive tenant from token, not client")
	})

	t.Run("GET /holidays ignores forged tenant_id in query and header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/holidays?year=2026&tenant_id="+spoofedTenant.String(), nil)
		req.Header.Set("Authorization", bearer(t, jwtSvc, tenantA, "employee"))
		req.Header.Set("X-Tenant-ID", spoofedTenant.String())
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, tenantA, f.listHolidayGotTenant, "usecase must receive tenant from token, not query")
	})

	t.Run("POST /holiday-categories ignores forged tenant_id in body", func(t *testing.T) {
		body := `{"name":"Public","color":"#15803D","tenant_id":"` + spoofedTenant.String() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/holiday-categories", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer(t, jwtSvc, tenantA, "admin"))
		req.Header.Set("X-Tenant-ID", spoofedTenant.String())
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, tenantA, f.createCategoryGotTenant, "usecase must receive tenant from token")
	})

	t.Run("PUT /holiday-calendar/settings ignores forged tenant_id in body", func(t *testing.T) {
		body := `{"default_view":"list","week_start_day":0,"tenant_id":"` + spoofedTenant.String() + `"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/holiday-calendar/settings", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer(t, jwtSvc, tenantA, "admin"))
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, tenantA, f.updateSettingsGotTenant, "usecase must receive tenant from token")
	})
}

func TestHolidayRoutes_AllRegistered(t *testing.T) {
	f := &fakeHolidayUseCases{}
	engine, _ := holidayEngine(t, f)

	registered := map[string]bool{}
	for _, r := range engine.Routes() {
		registered[r.Method+" "+r.Path] = true
	}

	expectedRoutes := []string{
		"GET /api/v1/holidays",
		"GET /api/v1/holidays/:id",
		"POST /api/v1/holidays",
		"PUT /api/v1/holidays/:id",
		"DELETE /api/v1/holidays/:id",
		"GET /api/v1/holiday-categories",
		"GET /api/v1/holiday-categories/:id",
		"POST /api/v1/holiday-categories",
		"PUT /api/v1/holiday-categories/:id",
		"DELETE /api/v1/holiday-categories/:id",
		"GET /api/v1/holiday-calendar/settings",
		"PUT /api/v1/holiday-calendar/settings",
	}

	for _, route := range expectedRoutes {
		assert.True(t, registered[route], "expected route %s to be registered", route)
	}
}

func TestHolidayRoutes_NilHandlerPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Holiday = &container.HolidayContainer{
		Handler: nil,
	}

	assert.PanicsWithValue(t, "routes: holiday handler must not be nil when holiday container is configured", func() {
		SetupRouter(cfg, zerolog.Nop(), ctr)
	})
}

func TestHolidayRoutes_NilEmployeeContainerPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSvc := setupTestTokenService(t)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{
		TokenService:     jwtSvc,
		IdentityVerifier: verifierOf(jwtSvc),
	}
	ctr.Holiday = &container.HolidayContainer{Handler: &handlers.HolidayHandler{}}
	ctr.Employee = nil

	assert.PanicsWithValue(t, "routes: Employee container (profile gate) is required for holiday routes", func() {
		SetupRouter(cfg, zerolog.Nop(), ctr)
	})
}

// passGate is a ProfileGateUseCase that reports every profile as complete.
type passGate struct{}

func (passGate) Execute(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil }
