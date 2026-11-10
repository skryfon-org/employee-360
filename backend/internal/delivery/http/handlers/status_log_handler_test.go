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
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
)

type createFn func(ctx context.Context, tenantID, actorID uuid.UUID, req statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error)

func (f createFn) Execute(ctx context.Context, tenantID, actorID uuid.UUID, req statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
	return f(ctx, tenantID, actorID, req)
}

type updateFn func(ctx context.Context, tenantID, actorID, id uuid.UUID, req statuslog.UpdateStatusLogRequest) (*entity.StatusLogEntryWithProject, error)

func (f updateFn) Execute(ctx context.Context, tenantID, actorID, id uuid.UUID, req statuslog.UpdateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
	return f(ctx, tenantID, actorID, id, req)
}

type deleteFn func(ctx context.Context, tenantID, actorID, id uuid.UUID) error

func (f deleteFn) Execute(ctx context.Context, tenantID, actorID, id uuid.UUID) error {
	return f(ctx, tenantID, actorID, id)
}

type listMyFn func(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error)

func (f listMyFn) Execute(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error) {
	return f(ctx, tenantID, userID, fromStr, toStr, projectStr)
}

type myDayFn func(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr string) ([]*entity.MyDayStatus, error)

func (f myDayFn) Execute(ctx context.Context, tenantID, userID uuid.UUID, fromStr, toStr string) ([]*entity.MyDayStatus, error) {
	return f(ctx, tenantID, userID, fromStr, toStr)
}

type getDailyFn func(ctx context.Context, tenantID uuid.UUID, dateStr, deptStr, projStr string) ([]*entity.DailyEmployeeSummary, error)

func (f getDailyFn) Execute(ctx context.Context, tenantID uuid.UUID, dateStr, deptStr, projStr string) ([]*entity.DailyEmployeeSummary, error) {
	return f(ctx, tenantID, dateStr, deptStr, projStr)
}

type getMonthlyFn func(ctx context.Context, tenantID uuid.UUID, monthStr, deptStr, projStr string) (*entity.MonthlyStatusLogReport, error)

func (f getMonthlyFn) Execute(ctx context.Context, tenantID uuid.UUID, monthStr, deptStr, projStr string) (*entity.MonthlyStatusLogReport, error) {
	return f(ctx, tenantID, monthStr, deptStr, projStr)
}

type listEmpFn func(ctx context.Context, tenantID, employeeID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error)

func (f listEmpFn) Execute(ctx context.Context, tenantID, employeeID uuid.UUID, fromStr, toStr, projectStr string) ([]*entity.DailyStatusLogGroup, error) {
	return f(ctx, tenantID, employeeID, fromStr, toStr, projectStr)
}

type statusLogMock struct {
	create       createFn
	update       updateFn
	delete       deleteFn
	listMy       listMyFn
	myDay        myDayFn
	getDaily     getDailyFn
	getMonthly   getMonthlyFn
	listEmployee listEmpFn
}

func (m *statusLogMock) useCases() StatusLogUseCases {
	return StatusLogUseCases{
		Create:       m.create,
		Update:       m.update,
		Delete:       m.delete,
		ListMy:       m.listMy,
		MyDayStatus:  m.myDay,
		GetDaily:     m.getDaily,
		GetMonthly:   m.getMonthly,
		ListEmployee: m.listEmployee,
	}
}

func newStatusLogRouter(m *statusLogMock, tenantID, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewStatusLogHandler(m.useCases())
	r := gin.New()
	g := r.Group("/api/v1", withIdentity(tenantID, userID))
	g.GET("/me/status-logs", h.ListMyStatusLogs)
	g.GET("/me/status-logs/day-statuses", h.ListMyDayStatuses)
	g.POST("/me/status-logs", h.CreateMyStatusLog)
	g.PATCH("/me/status-logs/:id", h.UpdateMyStatusLog)
	g.DELETE("/me/status-logs/:id", h.DeleteMyStatusLog)
	g.GET("/status-logs/daily", h.GetDailyStatusLogs)
	g.GET("/status-logs/monthly", h.GetMonthlyStatusLogs)
	g.GET("/employees/:id/status-logs", h.ListEmployeeStatusLogs)
	return r
}

func performStatusLogReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// Tests: window boundaries (today, today−6 accepted; today−7 and tomorrow rejected)
func TestStatusLogHandler_WindowBoundaries(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	entryID := uuid.New()

	mock := &statusLogMock{
		create: func(_ context.Context, tid, uid uuid.UUID, req statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
			switch req.LogDate {
			case "2026-10-27", "2026-10-21":
				d, _ := time.Parse("2006-01-02", req.LogDate)
				return &entity.StatusLogEntryWithProject{
					StatusLogEntry: entity.StatusLogEntry{
						ID:       entryID,
						TenantID: tid,
						UserID:   uid,
						LogDate:  d,
						Hours:    req.Hours,
					},
				}, nil
			case "2026-10-20":
				return nil, domainerrors.ErrLogWindowClosed
			case "2026-10-28":
				return nil, domainerrors.ErrFutureDate
			default:
				return nil, domainerrors.ErrInvalidStatusLogDate
			}
		},
		delete: func(_ context.Context, _, _, id uuid.UUID) error {
			if id == entryID {
				return domainerrors.ErrLogWindowClosed
			}
			return nil
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)

	cat := "meeting"

	// 1. Today accepted -> 201
	rec := performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            4.0,
	})
	assert.Equal(t, http.StatusCreated, rec.Code)

	// 2. Today - 6 accepted -> 201
	rec = performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-21",
		ActivityCategory: &cat,
		Hours:            4.0,
	})
	assert.Equal(t, http.StatusCreated, rec.Code)

	// 3. Today - 7 rejected -> 422 log_window_closed
	rec = performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-20",
		ActivityCategory: &cat,
		Hours:            4.0,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var env response.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "log_window_closed", env.Error.Code)

	// 4. Tomorrow rejected -> 422
	rec = performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-28",
		ActivityCategory: &cat,
		Hours:            4.0,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// 5. Delete outside window -> 422 log_window_closed
	rec = performStatusLogReq(r, http.MethodDelete, "/api/v1/me/status-logs/"+entryID.String(), nil)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var delEnv response.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &delEnv))
	require.NotNil(t, delEnv.Error)
	assert.Equal(t, "log_window_closed", delEnv.Error.Code)
}

// Tests: 24h cap -> 422
func TestStatusLogHandler_Daily24hCap(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	mock := &statusLogMock{
		create: func(_ context.Context, _, _ uuid.UUID, _ statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
			return nil, domainerrors.ErrDailyHoursExceeded
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)
	cat := "meeting"

	rec := performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            25.0,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var env response.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "daily_hours_exceeded", env.Error.Code)
}

// Tests: one-of project/category -> 400
func TestStatusLogHandler_OneOfProjectOrCategory(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	mock := &statusLogMock{
		create: func(_ context.Context, _, _ uuid.UUID, req statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
			if req.ProjectID != nil && req.ActivityCategory != nil {
				return nil, domainerrors.ErrProjectOrCategoryRequired
			}
			if req.ProjectID == nil && req.ActivityCategory == nil {
				return nil, domainerrors.ErrProjectOrCategoryRequired
			}
			return nil, nil
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)

	cat := "meeting"
	pid := uuid.New()

	// Both
	rec := performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ProjectID:        &pid,
		ActivityCategory: &cat,
		Hours:            4.0,
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Neither
	rec = performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate: "2026-10-27",
		Hours:   4.0,
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Tests: active allocation required for project entries -> 422
func TestStatusLogHandler_ActiveAllocationRequired(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	mock := &statusLogMock{
		create: func(_ context.Context, _, _ uuid.UUID, _ statuslog.CreateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
			return nil, domainerrors.ErrNoActiveAllocation
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)
	pid := uuid.New()

	rec := performStatusLogReq(r, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
		LogDate:   "2026-10-27",
		ProjectID: &pid,
		Hours:     4.0,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var env response.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "no_active_allocation", env.Error.Code)
}

// Tests: cannot edit another user's entry -> 403
func TestStatusLogHandler_CannotEditAnotherUserEntry(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	targetID := uuid.New()

	mock := &statusLogMock{
		update: func(_ context.Context, _, _, id uuid.UUID, _ statuslog.UpdateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
			if id == targetID {
				return nil, domainerrors.ErrForbidden
			}
			return nil, nil
		},
		delete: func(_ context.Context, _, _, id uuid.UUID) error {
			if id == targetID {
				return domainerrors.ErrForbidden
			}
			return nil
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)

	notes := "edit"
	rec := performStatusLogReq(r, http.MethodPatch, "/api/v1/me/status-logs/"+targetID.String(), statuslog.UpdateStatusLogRequest{
		Notes: &notes,
	})
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = performStatusLogReq(r, http.MethodDelete, "/api/v1/me/status-logs/"+targetID.String(), nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Tests: cross-tenant 404
func TestStatusLogHandler_CrossTenant404(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	otherTenantID := uuid.New()

	mock := &statusLogMock{
		update: func(_ context.Context, _, _, id uuid.UUID, _ statuslog.UpdateStatusLogRequest) (*entity.StatusLogEntryWithProject, error) {
			return nil, domainerrors.ErrStatusLogNotFound
		},
		delete: func(_ context.Context, _, _, id uuid.UUID) error {
			return domainerrors.ErrStatusLogNotFound
		},
		listEmployee: func(_ context.Context, _, _ uuid.UUID, _, _, _ string) ([]*entity.DailyStatusLogGroup, error) {
			return nil, domainerrors.ErrUserNotFound
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)

	rec := performStatusLogReq(r, http.MethodPatch, "/api/v1/me/status-logs/"+otherTenantID.String(), statuslog.UpdateStatusLogRequest{})
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = performStatusLogReq(r, http.MethodDelete, "/api/v1/me/status-logs/"+otherTenantID.String(), nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = performStatusLogReq(r, http.MethodGet, "/api/v1/employees/"+otherTenantID.String()+"/status-logs", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// Tests: daily view lists non-loggers
func TestStatusLogHandler_DailyViewListsNonLoggers(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	u1, u2 := uuid.New(), uuid.New()

	mock := &statusLogMock{
		getDaily: func(_ context.Context, _ uuid.UUID, _, _, _ string) ([]*entity.DailyEmployeeSummary, error) {
			return []*entity.DailyEmployeeSummary{
				{
					UserID:     u1,
					FirstName:  "Alice",
					LastName:   "Smith",
					Logged:     true,
					TotalHours: 8.0,
				},
				{
					UserID:     u2,
					FirstName:  "Bob",
					LastName:   "Jones",
					Logged:     false,
					TotalHours: 0.0,
				},
			}, nil
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)

	rec := performStatusLogReq(r, http.MethodGet, "/api/v1/status-logs/daily?date=2026-10-27", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	var env struct {
		Success bool                                     `json:"success"`
		Data    []statuslog.DailyEmployeeSummaryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Len(t, env.Data, 2)

	assert.Equal(t, u1, env.Data[0].UserID)
	assert.True(t, env.Data[0].Logged)
	assert.Equal(t, 8.0, env.Data[0].TotalHours)

	assert.Equal(t, u2, env.Data[1].UserID)
	assert.False(t, env.Data[1].Logged)
	assert.Equal(t, 0.0, env.Data[1].TotalHours)
}

func TestStatusLogHandler_Monthly(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	u1 := uuid.New()
	var gotTenant uuid.UUID
	var gotMonth, gotDept, gotProj string

	mock := &statusLogMock{
		getMonthly: func(_ context.Context, tid uuid.UUID, month, dept, proj string) (*entity.MonthlyStatusLogReport, error) {
			if month == "bad" {
				return nil, domainerrors.ErrInvalidStatusLogDate
			}
			gotTenant, gotMonth, gotDept, gotProj = tid, month, dept, proj
			return &entity.MonthlyStatusLogReport{
				Month: "2026-10", WorkingDays: 22, TotalEmployees: 1, LoggedEmployees: 1, TotalHours: 12.5,
				Employees: []*entity.MonthlyEmployeeSummary{{
					UserID: u1, FirstName: "Alice", DaysLogged: 3, TotalHours: 12.5,
					HoursByCategory: []entity.CategoryHoursBreakdown{{Category: "meeting", Hours: 12.5}},
				}},
			}, nil
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)

	rec := performStatusLogReq(r, http.MethodGet, "/api/v1/status-logs/monthly?month=2026-10&department=d1&project=p1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, tenantID, gotTenant)
	assert.Equal(t, "2026-10", gotMonth)
	assert.Equal(t, "d1", gotDept)
	assert.Equal(t, "p1", gotProj)

	var env struct {
		Data statuslog.MonthlyStatusLogReportResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, 22, env.Data.WorkingDays)
	require.Len(t, env.Data.Employees, 1)
	assert.Equal(t, 3, env.Data.Employees[0].DaysLogged)
	assert.NotNil(t, env.Data.Employees[0].HoursByProject)
	assert.Len(t, env.Data.Employees[0].HoursByCategory, 1)

	rec = performStatusLogReq(r, http.MethodGet, "/api/v1/status-logs/monthly?month=bad", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStatusLogHandler_ListMyDayStatuses(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	mock := &statusLogMock{
		myDay: func(_ context.Context, tid, uid uuid.UUID, from, to string) ([]*entity.MyDayStatus, error) {
			assert.Equal(t, tenantID, tid)
			assert.Equal(t, userID, uid)
			if from == "bad" {
				return nil, domainerrors.ErrInvalidStatusLogDate
			}
			assert.Equal(t, "2026-10-26", from)
			assert.Equal(t, "2026-10-28", to)
			d := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
			return []*entity.MyDayStatus{
				{Date: d, DayStatus: entity.DayStatusHoliday, HolidayName: "Diwali"},
				{Date: d.AddDate(0, 0, 1), DayStatus: entity.DayStatusOnLeave, HalfDay: true, LeaveTypeName: "Casual"},
				{Date: d.AddDate(0, 0, 2), DayStatus: entity.DayStatusLogged, TotalHours: 7.5},
			}, nil
		},
	}
	r := newStatusLogRouter(mock, tenantID, userID)

	rec := performStatusLogReq(r, http.MethodGet, "/api/v1/me/status-logs/day-statuses?from=2026-10-26&to=2026-10-28", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var env struct {
		Data []map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Len(t, env.Data, 3)
	assert.Equal(t, "2026-10-26", env.Data[0]["date"])
	assert.Equal(t, "holiday", env.Data[0]["day_status"])
	assert.Equal(t, "Diwali", env.Data[0]["holiday_name"])
	assert.NotContains(t, env.Data[0], "leave_type_name")
	assert.Equal(t, "on_leave", env.Data[1]["day_status"])
	assert.Equal(t, true, env.Data[1]["half_day"])
	assert.Equal(t, "Casual", env.Data[1]["leave_type_name"])
	assert.Equal(t, 7.5, env.Data[2]["total_hours"])

	rec = performStatusLogReq(r, http.MethodGet, "/api/v1/me/status-logs/day-statuses?from=bad", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
