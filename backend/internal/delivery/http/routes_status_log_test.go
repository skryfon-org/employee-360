package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
)

type dummyCreateStatusLog struct{}

func (dummyCreateStatusLog) Execute(_ context.Context, tenantID, actorID uuid.UUID, req statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
	return &entity.StatusLogEntryWithProject{
		StatusLogEntry: entity.StatusLogEntry{
			ID:       uuid.New(),
			TenantID: tenantID,
			UserID:   actorID,
			Hours:    req.Hours,
		},
	}, nil
}

type dummyUpdateStatusLog struct{}

func (dummyUpdateStatusLog) Execute(_ context.Context, tenantID, actorID, id uuid.UUID, req statuslog.UpdateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
	return &entity.StatusLogEntryWithProject{
		StatusLogEntry: entity.StatusLogEntry{
			ID:       id,
			TenantID: tenantID,
			UserID:   actorID,
		},
	}, nil
}

type dummyDeleteStatusLog struct{}

func (dummyDeleteStatusLog) Execute(_ context.Context, tenantID, actorID, id uuid.UUID) error {
	return nil
}

type dummyListMyStatusLogs struct{}

func (dummyListMyStatusLogs) Execute(_ context.Context, tenantID, userID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error) {
	return []*entity.DailyStatusLogGroup{}, nil
}

type dummyListMyDayStatuses struct{}

func (dummyListMyDayStatuses) Execute(_ context.Context, _, _ uuid.UUID, _, _ string) ([]*entity.MyDayStatus, error) {
	return []*entity.MyDayStatus{}, nil
}

type dummyGetDailyStatusLogs struct{}

func (dummyGetDailyStatusLogs) Execute(_ context.Context, tenantID uuid.UUID, dateStr, deptStr, projStr string) ([]*entity.DailyEmployeeSummary, error) {
	return []*entity.DailyEmployeeSummary{}, nil
}

type dummyGetMonthlyStatusLogs struct{}

func (dummyGetMonthlyStatusLogs) Execute(_ context.Context, _ uuid.UUID, _, _, _ string) (*entity.MonthlyStatusLogReport, error) {
	return &entity.MonthlyStatusLogReport{Month: "2026-10"}, nil
}

type dummyListEmployeeStatusLogs struct{}

func (dummyListEmployeeStatusLogs) Execute(_ context.Context, tenantID, employeeID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error) {
	return []*entity.DailyStatusLogGroup{}, nil
}

func statusLogRoutesEngine(t *testing.T) (*gin.Engine, domainservice.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtSvc := setupTestTokenService(t)
	cfg := &config.Config{CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}}
	ctr := testContainer(t, cfg, nil, nil)
	ctr.Auth = &container.AuthContainer{TokenService: jwtSvc, IdentityVerifier: verifierOf(jwtSvc)}
	ctr.Employee = &container.EmployeeContainer{
		Handler: handlers.NewEmployeeHandler(handlers.EmployeeUseCases{
			Me: okMe{}, MyProfile: okMyProfile{}, List: okList{}, ListContacts: okContacts{},
		}),
	}
	ctr.StatusLog = &container.StatusLogContainer{
		Handler: handlers.NewStatusLogHandler(handlers.StatusLogUseCases{
			Create:       dummyCreateStatusLog{},
			Update:       dummyUpdateStatusLog{},
			Delete:       dummyDeleteStatusLog{},
			ListMy:       dummyListMyStatusLogs{},
			MyDayStatus:  dummyListMyDayStatuses{},
			GetDaily:     dummyGetDailyStatusLogs{},
			GetMonthly:   dummyGetMonthlyStatusLogs{},
			ListEmployee: dummyListEmployeeStatusLogs{},
		}),
	}
	return SetupRouter(cfg, zerolog.Nop(), ctr), jwtSvc
}

func TestStatusLogRoutes_MeRoutes_AccessibleByEmployee(t *testing.T) {
	engine, jwtSvc := statusLogRoutesEngine(t)
	tenant := uuid.New()
	empToken := bearer(t, jwtSvc, tenant, "employee")

	// 1. Unauthenticated -> 401
	rec := do(engine, http.MethodGet, "/api/v1/me/status-logs", "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 2. Employee token -> 200
	rec = do(engine, http.MethodGet, "/api/v1/me/status-logs", empToken, "")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = do(engine, http.MethodGet, "/api/v1/me/status-logs/day-statuses", "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	rec = do(engine, http.MethodGet, "/api/v1/me/status-logs/day-statuses", empToken, "")
	assert.Equal(t, http.StatusOK, rec.Code)

	// 3. POST unauthenticated -> 401
	rec = do(engine, http.MethodPost, "/api/v1/me/status-logs", "", `{"log_date":"2026-10-27","hours":4.0}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 4. POST employee -> 201
	rec = do(engine, http.MethodPost, "/api/v1/me/status-logs", empToken, `{"log_date":"2026-10-27","hours":4.0}`)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestStatusLogRoutes_DailyAndEmployee_AdminOnly(t *testing.T) {
	engine, jwtSvc := statusLogRoutesEngine(t)
	tenant := uuid.New()
	empToken := bearer(t, jwtSvc, tenant, "employee")
	adminToken := bearer(t, jwtSvc, tenant, "admin")
	superToken := bearer(t, jwtSvc, tenant, "super_admin")
	empID := uuid.NewString()

	paths := []string{
		"/api/v1/status-logs/daily",
		"/api/v1/status-logs/monthly",
		"/api/v1/employees/" + empID + "/status-logs",
	}

	for _, p := range paths {
		// 1. Unauthenticated -> 401
		rec := do(engine, http.MethodGet, p, "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "path: %s", p)

		// 2. Employee -> 403 Forbidden
		rec = do(engine, http.MethodGet, p, empToken, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, "path: %s", p)

		// 3. Admin -> 200 OK
		rec = do(engine, http.MethodGet, p, adminToken, "")
		assert.Equal(t, http.StatusOK, rec.Code, "path: %s", p)

		// 4. Super Admin -> 200 OK
		rec = do(engine, http.MethodGet, p, superToken, "")
		assert.Equal(t, http.StatusOK, rec.Code, "path: %s", p)
	}
}
