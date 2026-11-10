//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
)

func (e *authFlowEnv) callStatusLog(t *testing.T, method, path string, body any, tok string) (int, response.Envelope, []byte) {
	t.Helper()
	code, _, raw := e.doJSON(t, method, path, body, tok)
	var env response.Envelope
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &env)
	}
	return code, env, raw
}

func TestStatusLogFlow_EndToEnd(t *testing.T) {
	e := newAuthFlowEnv(t)
	other := newAuthFlowEnv(t)

	adminTok := e.login(t, authFlowPassword).AccessToken
	otherAdminTok := other.login(t, authFlowPassword).AccessToken

	emp1 := seedEmployee(t, e, "emp1", true, true)
	emp2 := seedEmployee(t, e, "emp2", true, true)
	foreignEmp := seedEmployee(t, other, "foreign", true, true)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayStr := today.Format("2006-01-02")
	tMinus6Str := today.AddDate(0, 0, -6).Format("2006-01-02")
	tMinus7Str := today.AddDate(0, 0, -7).Format("2006-01-02")
	tomorrowStr := today.AddDate(0, 0, 1).Format("2006-01-02")

	// Seed Project Alpha in Tenant A and allocate emp1 to it
	projectAID := e.mustCreateProject(t, adminTok, "ALPHASL", nil)
	projectAStart := today.AddDate(0, 0, -30).Format("2006-01-02")
	st, _, raw := e.call(t, http.MethodPost, "/api/v1/projects/"+projectAID+"/allocations",
		allocBody(emp1.id, 50, projectAStart, nil), adminTok)
	require.Equal(t, http.StatusCreated, st, string(raw))

	// Seed Project Beta in Tenant A (no allocation for emp1)
	projectBID := e.mustCreateProject(t, adminTok, "BETASL", nil)

	// -------------------------------------------------------------
	// 1. Window Boundaries: today, today-6 accepted; today-7 & tomorrow rejected
	// -------------------------------------------------------------
	t.Run("window boundaries", func(t *testing.T) {
		cat := "meeting"

		// Today -> 201 Created
		code, env, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          todayStr,
			ActivityCategory: &cat,
			Hours:            2.0,
			Notes:            "today log",
		}, emp1.token)
		require.Equal(t, http.StatusCreated, code, string(raw))
		require.True(t, env.Success)

		// Today - 6 -> 201 Created
		code, env, raw = e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          tMinus6Str,
			ActivityCategory: &cat,
			Hours:            1.5,
			Notes:            "t-6 log",
		}, emp1.token)
		require.Equal(t, http.StatusCreated, code, string(raw))
		require.True(t, env.Success)

		// Today - 7 -> 422 log_window_closed
		code, env, raw = e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          tMinus7Str,
			ActivityCategory: &cat,
			Hours:            2.0,
			Notes:            "too old",
		}, emp1.token)
		require.Equal(t, http.StatusUnprocessableEntity, code, string(raw))
		require.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, "log_window_closed", env.Error.Code)

		// Tomorrow -> 422 future_date_not_allowed
		code, env, raw = e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          tomorrowStr,
			ActivityCategory: &cat,
			Hours:            2.0,
			Notes:            "tomorrow",
		}, emp1.token)
		require.Equal(t, http.StatusUnprocessableEntity, code, string(raw))
		require.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, "future_date_not_allowed", env.Error.Code)
	})

	// -------------------------------------------------------------
	// 2. 24h Cap on total daily hours
	// -------------------------------------------------------------
	t.Run("daily 24h cap", func(t *testing.T) {
		cat := "internal"
		// On tMinus6Str, emp1 already logged 1.5h.
		// Log 22.5h -> total 24.0h -> 201 Created
		code, _, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          tMinus6Str,
			ActivityCategory: &cat,
			Hours:            22.5,
			Notes:            "marathon",
		}, emp1.token)
		require.Equal(t, http.StatusCreated, code, string(raw))

		// Log 0.5h more on tMinus6Str -> total 24.5h -> 422 daily_hours_exceeded
		code, env, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          tMinus6Str,
			ActivityCategory: &cat,
			Hours:            0.5,
			Notes:            "overflow",
		}, emp1.token)
		require.Equal(t, http.StatusUnprocessableEntity, code, string(raw))
		require.NotNil(t, env.Error)
		assert.Equal(t, "daily_hours_exceeded", env.Error.Code)
	})

	// -------------------------------------------------------------
	// 3. One-of Project or Category
	// -------------------------------------------------------------
	t.Run("one of project or category", func(t *testing.T) {
		cat := "meeting"
		pID := uuid.MustParse(projectAID)

		// Both -> 400
		code, _, _ := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          todayStr,
			ProjectID:        &pID,
			ActivityCategory: &cat,
			Hours:            1.0,
		}, emp1.token)
		assert.Equal(t, http.StatusBadRequest, code)

		// Neither -> 400
		code, _, _ = e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate: todayStr,
			Hours:   1.0,
		}, emp1.token)
		assert.Equal(t, http.StatusBadRequest, code)

		// Invalid category enum -> 400
		badCat := "invalid_cat"
		code, _, _ = e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          todayStr,
			ActivityCategory: &badCat,
			Hours:            1.0,
		}, emp1.token)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	// -------------------------------------------------------------
	// 4. Project entries require active allocation on log_date
	// -------------------------------------------------------------
	t.Run("project entries require active allocation", func(t *testing.T) {
		pAID := uuid.MustParse(projectAID)
		pBID := uuid.MustParse(projectBID)

		// emp1 is allocated to Project Alpha -> 201 Created
		code, _, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:   todayStr,
			ProjectID: &pAID,
			Hours:     3.0,
			Notes:     "alpha work",
		}, emp1.token)
		require.Equal(t, http.StatusCreated, code, string(raw))

		// emp1 is NOT allocated to Project Beta -> 422 no_active_allocation
		code, env, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:   todayStr,
			ProjectID: &pBID,
			Hours:     2.0,
			Notes:     "unallocated project",
		}, emp1.token)
		require.Equal(t, http.StatusUnprocessableEntity, code, string(raw))
		require.NotNil(t, env.Error)
		assert.Equal(t, "no_active_allocation", env.Error.Code)
	})

	// -------------------------------------------------------------
	// 5. Cannot edit another user's entry
	// -------------------------------------------------------------
	t.Run("cannot edit another user entry", func(t *testing.T) {
		cat := "training"
		code, _, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          todayStr,
			ActivityCategory: &cat,
			Hours:            1.0,
			Notes:            "emp1 private entry",
		}, emp1.token)
		require.Equal(t, http.StatusCreated, code, string(raw))

		var created struct {
			ID string `json:"id"`
		}
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(raw, &env)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		// emp2 tries to PATCH emp1's entry -> 404 Not Found (prevents id leakage)
		newNotes := "hacked"
		code, _, _ = e.callStatusLog(t, http.MethodPatch, "/api/v1/me/status-logs/"+created.ID, statuslog.UpdateStatusLogRequest{
			Notes: &newNotes,
		}, emp2.token)
		assert.Equal(t, http.StatusNotFound, code)

		// emp2 tries to DELETE emp1's entry -> 404 Not Found (prevents id leakage)
		code, _, _ = e.callStatusLog(t, http.MethodDelete, "/api/v1/me/status-logs/"+created.ID, nil, emp2.token)
		assert.Equal(t, http.StatusNotFound, code)

		// emp1 CAN update own entry -> 200 OK
		code, _, raw = e.callStatusLog(t, http.MethodPatch, "/api/v1/me/status-logs/"+created.ID, statuslog.UpdateStatusLogRequest{
			Notes: &newNotes,
		}, emp1.token)
		assert.Equal(t, http.StatusOK, code, string(raw))

		// emp1 CAN delete own entry -> 200 OK
		code, _, raw = e.callStatusLog(t, http.MethodDelete, "/api/v1/me/status-logs/"+created.ID, nil, emp1.token)
		assert.Equal(t, http.StatusOK, code, string(raw))
	})

	// -------------------------------------------------------------
	// 6. Cross-tenant 404
	// -------------------------------------------------------------
	t.Run("cross tenant 404", func(t *testing.T) {
		cat := "support"
		code, _, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          todayStr,
			ActivityCategory: &cat,
			Hours:            2.0,
			Notes:            "emp1 support",
		}, emp1.token)
		require.Equal(t, http.StatusCreated, code, string(raw))

		var created struct {
			ID string `json:"id"`
		}
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(raw, &env)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		// foreign employee in other tenant tries to PATCH emp1's entry -> 404 Not Found
		newNotes := "foreign edit"
		code, _, _ = other.callStatusLog(t, http.MethodPatch, "/api/v1/me/status-logs/"+created.ID, statuslog.UpdateStatusLogRequest{
			Notes: &newNotes,
		}, foreignEmp.token)
		assert.Equal(t, http.StatusNotFound, code)

		// foreign admin tries to GET emp1's status logs -> 404 Not Found
		code, _, _ = other.callStatusLog(t, http.MethodGet, "/api/v1/employees/"+emp1.id.String()+"/status-logs", nil, otherAdminTok)
		assert.Equal(t, http.StatusNotFound, code)
	})

	// -------------------------------------------------------------
	// 7. GET /me/status-logs: grouped by date with per-day totals
	// -------------------------------------------------------------
	t.Run("get me status logs grouped by date", func(t *testing.T) {
		code, _, raw := e.callStatusLog(t, http.MethodGet, fmt.Sprintf("/api/v1/me/status-logs?from=%s&to=%s", tMinus6Str, todayStr), nil, emp1.token)
		require.Equal(t, http.StatusOK, code, string(raw))

		var res struct {
			Success bool                                    `json:"success"`
			Data    []statuslog.DailyStatusLogGroupResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &res))
		require.True(t, res.Success)
		assert.NotEmpty(t, res.Data)

		for _, group := range res.Data {
			assert.NotEmpty(t, group.Date)
			assert.Greater(t, group.TotalHours, 0.0)
			assert.NotEmpty(t, group.Entries)
		}
	})

	// -------------------------------------------------------------
	// 8. Admin GET /status-logs/daily lists non-loggers
	// -------------------------------------------------------------
	t.Run("daily view lists non-loggers", func(t *testing.T) {
		// Non-admin employee cannot access admin daily route -> 403
		code, _, _ := e.callStatusLog(t, http.MethodGet, "/api/v1/status-logs/daily?date="+todayStr, nil, emp1.token)
		assert.Equal(t, http.StatusForbidden, code)

		// Admin calls daily route -> 200 OK
		code, _, raw := e.callStatusLog(t, http.MethodGet, "/api/v1/status-logs/daily?date="+todayStr, nil, adminTok)
		require.Equal(t, http.StatusOK, code, string(raw))

		var res struct {
			Success bool                                     `json:"success"`
			Data    []statuslog.DailyEmployeeSummaryResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &res))
		require.True(t, res.Success)

		var emp1Sum, emp2Sum *statuslog.DailyEmployeeSummaryResponse
		for i := range res.Data {
			if res.Data[i].UserID == emp1.id {
				emp1Sum = &res.Data[i]
			} else if res.Data[i].UserID == emp2.id {
				emp2Sum = &res.Data[i]
			}
		}

		// emp1 logged time today -> logged: true, total_hours > 0
		require.NotNil(t, emp1Sum)
		assert.True(t, emp1Sum.Logged)
		assert.Greater(t, emp1Sum.TotalHours, 0.0)

		// emp2 did NOT log time today -> logged: false, total_hours: 0
		require.NotNil(t, emp2Sum)
		assert.False(t, emp2Sum.Logged)
		assert.Equal(t, 0.0, emp2Sum.TotalHours)
		assert.Empty(t, emp2Sum.HoursByProject)
		assert.Empty(t, emp2Sum.HoursByCategory)
	})

	// -------------------------------------------------------------
	// 9. Admin GET /employees/:id/status-logs
	// -------------------------------------------------------------
	t.Run("admin gets employee status logs", func(t *testing.T) {
		// Non-admin employee cannot access -> 403
		code, _, _ := e.callStatusLog(t, http.MethodGet, "/api/v1/employees/"+emp1.id.String()+"/status-logs", nil, emp2.token)
		assert.Equal(t, http.StatusForbidden, code)

		// Admin can access -> 200 OK
		code, _, raw := e.callStatusLog(t, http.MethodGet, "/api/v1/employees/"+emp1.id.String()+"/status-logs", nil, adminTok)
		require.Equal(t, http.StatusOK, code, string(raw))

		var res struct {
			Success bool                                    `json:"success"`
			Data    []statuslog.DailyStatusLogGroupResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &res))
		require.True(t, res.Success)
		assert.NotEmpty(t, res.Data)
	})

	// -------------------------------------------------------------
	// 10. Date range capped at 366 days
	// -------------------------------------------------------------
	t.Run("date range capped at 366 days", func(t *testing.T) {
		code, _, _ := e.callStatusLog(t, http.MethodGet, "/api/v1/me/status-logs?from=0001-01-01&to=9999-12-31", nil, emp1.token)
		assert.Equal(t, http.StatusBadRequest, code)

		code, _, _ = e.callStatusLog(t, http.MethodGet, "/api/v1/employees/"+emp1.id.String()+"/status-logs?from=0001-01-01&to=9999-12-31", nil, adminTok)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	// -------------------------------------------------------------
	// 11. Patch nil UUID project_id alone is rejected
	// -------------------------------------------------------------
	t.Run("patch nil uuid project alone is rejected", func(t *testing.T) {
		cat := "internal"
		code, _, raw := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
			LogDate:          todayStr,
			ActivityCategory: &cat,
			Hours:            1.0,
			Notes:            "test patch nil",
		}, emp1.token)
		require.Equal(t, http.StatusCreated, code, string(raw))

		var created struct {
			ID string `json:"id"`
		}
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(raw, &env)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		nilUUID := uuid.Nil
		code, _, _ = e.callStatusLog(t, http.MethodPatch, "/api/v1/me/status-logs/"+created.ID, statuslog.UpdateStatusLogRequest{
			ProjectID: &nilUUID,
		}, emp1.token)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	// -------------------------------------------------------------
	// 12. Concurrency test: 24h cap under concurrent POSTs
	// -------------------------------------------------------------
	t.Run("concurrent posts enforce 24h cap", func(t *testing.T) {
		testDateStr := today.AddDate(0, 0, -2).Format("2006-01-02")
		cat := "support"

		concurrency := 2
		startSig := make(chan struct{})
		results := make(chan int, concurrency)

		var wg sync.WaitGroup
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startSig
				code, _, _ := e.callStatusLog(t, http.MethodPost, "/api/v1/me/status-logs", statuslog.CreateStatusLogRequest{
					LogDate:          testDateStr,
					ActivityCategory: &cat,
					Hours:            14.0,
					Notes:            "concurrent 14h",
				}, emp2.token)
				results <- code
			}()
		}

		close(startSig)
		wg.Wait()
		close(results)

		var successCount, rejectedCount int
		for code := range results {
			if code == http.StatusCreated {
				successCount++
			} else if code == http.StatusUnprocessableEntity {
				rejectedCount++
			}
		}

		assert.Equal(t, 1, successCount, "exactly one concurrent 14h insert should succeed")
		assert.Equal(t, 1, rejectedCount, "the second concurrent 14h insert should be rejected")

		// Verify total hours on testDateStr for emp2 does not exceed 24h
		code, _, raw := e.callStatusLog(t, http.MethodGet, fmt.Sprintf("/api/v1/me/status-logs?from=%s&to=%s", testDateStr, testDateStr), nil, emp2.token)
		require.Equal(t, http.StatusOK, code, string(raw))

		var res struct {
			Data []statuslog.DailyStatusLogGroupResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &res))
		require.Len(t, res.Data, 1)
		assert.Equal(t, 14.0, res.Data[0].TotalHours)
	})
}
