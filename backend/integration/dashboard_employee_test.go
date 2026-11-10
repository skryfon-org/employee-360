//go:build integration

package integration

import (
	"encoding/json"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashtypes "github.com/skryfon-org/employee-360/backend/internal/types/dashboard"
)

// seedDashboardData inserts leave, an allocation, a status log and a holiday for
// one employee of env e, tagged with label so cross-tenant leaks are detectable.
func seedDashboardData(t *testing.T, e *authFlowEnv, emp projectActor, label string, today time.Time,
	pending int, approvedStart, approvedEnd int, pct int, hours float64, holidayOffset int) {
	t.Helper()
	day := func(off int) string { return today.AddDate(0, 0, off).Format("2006-01-02") }

	// Complete profile so the RequireCompleteProfile gate passes.
	require.NoError(t, e.db.Exec(
		"INSERT INTO employee_profiles (id, tenant_id, user_id, profile_completed_at) VALUES (?, ?, ?, NOW())",
		uuid.New(), e.tenantID, emp.id).Error)

	ltID := uuid.New()
	require.NoError(t, e.db.Exec("INSERT INTO leave_types (id, tenant_id, name, code) VALUES (?, ?, ?, ?)",
		ltID, e.tenantID, label+" leave", label+"LT").Error)
	insertLeave := func(status string, start, end int) {
		require.NoError(t, e.db.Exec(`INSERT INTO leave_requests
			(id, tenant_id, user_id, leave_type_id, start_date, end_date, approver_id, status, days)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.New(), e.tenantID, emp.id, ltID, day(start), day(end), e.userID, status,
			entity.CountLeaveDays(today.AddDate(0, 0, start), today.AddDate(0, 0, end), "full", entity.DefaultWeeklyOffDays(), nil)).Error)
	}
	for i := 0; i < pending; i++ {
		insertLeave("pending", 20+i*2, 20+i*2)
	}
	insertLeave("approved", approvedStart, approvedEnd)

	projID := uuid.New()
	require.NoError(t, e.db.Exec("INSERT INTO projects (id, tenant_id, name, code, start_date) VALUES (?, ?, ?, ?, ?)",
		projID, e.tenantID, label+" Project", label+"PRJ", day(-60)).Error)
	require.NoError(t, e.db.Exec(
		"INSERT INTO project_allocations (id, tenant_id, project_id, user_id, allocation_percent, start_date) VALUES (?, ?, ?, ?, ?, ?)",
		uuid.New(), e.tenantID, projID, emp.id, pct, day(-30)).Error)
	require.NoError(t, e.db.Exec(
		"INSERT INTO status_log_entries (id, tenant_id, user_id, log_date, project_id, hours) VALUES (?, ?, ?, ?, ?, ?)",
		uuid.New(), e.tenantID, emp.id, day(-1), projID, hours).Error)

	catID := uuid.New()
	require.NoError(t, e.db.Exec("INSERT INTO holiday_categories (id, tenant_id, name, color) VALUES (?, ?, ?, '#112233')",
		catID, e.tenantID, label+" category").Error)
	require.NoError(t, e.db.Exec("INSERT INTO holidays (id, tenant_id, category_id, name, date) VALUES (?, ?, ?, ?, ?)",
		uuid.New(), e.tenantID, catID, label+" Holiday", day(holidayOffset)).Error)
}

func (e *authFlowEnv) employeeDashboard(t *testing.T, tok string) (int, dashtypes.EmployeeDashboardResponse, []byte) {
	t.Helper()
	status, env, raw := e.doJSON(t, http.MethodGet, "/api/v1/dashboard/employee", nil, tok)
	var out dashtypes.EmployeeDashboardResponse
	if status == http.StatusOK {
		require.NoError(t, json.Unmarshal(env.Data, &out), string(raw))
	}
	return status, out, raw
}

func TestEmployeeDashboard_TenantIsolation(t *testing.T) {
	a := newAuthFlowEnv(t)
	b := newAuthFlowEnv(t)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	empA := seedEmployee(t, a, "dasha", true, true)
	empB := seedEmployee(t, b, "dashb", true, true)

	// A: 1 pending; approved leave started yesterday and ends in 2 days (3 days remain incl. today); 40%; 3h; holiday +3.
	seedDashboardData(t, a, empA, "TA", today, 1, -1, 2, 40, 3, 3)
	// B: 2 pending; approved leave +5..+6; 70%; 8h; holiday +4 (earlier than nothing in A's list).
	seedDashboardData(t, b, empB, "TB", today, 2, 5, 6, 70, 8, 4)

	t.Run("tenant A sees only its own data", func(t *testing.T) {
		status, res, raw := a.employeeDashboard(t, empA.token)
		require.Equal(t, http.StatusOK, status, string(raw))

		assert.Equal(t, 1, res.Stats.PendingLeave)
		assert.InDelta(t, entity.CountLeaveDays(today, today.AddDate(0, 0, 2), "full", entity.DefaultWeeklyOffDays(), nil), res.Stats.UpcomingLeaveDays, 0.001, "only today..end working days count, not the elapsed day")
		assert.Equal(t, 40, res.Stats.AllocationPercent)
		assert.InDelta(t, 3.0, res.Stats.HoursThisWeek, 0.001)

		require.NotNil(t, res.Leave.NextApproved)
		require.Len(t, res.Projects, 1)
		assert.Equal(t, "TA Project", res.Projects[0].ProjectName)
		require.Len(t, res.Holidays, 1)
		assert.Equal(t, "TA Holiday", res.Holidays[0].Name)
		assert.Equal(t, "TA category", res.Holidays[0].CategoryName)
		for _, l := range res.Leave.Recent {
			assert.Equal(t, "TA leave", l.LeaveTypeName)
		}
		assert.NotContains(t, string(raw), "TB", "tenant B data leaked into tenant A response")
	})

	t.Run("tenant B sees only its own data", func(t *testing.T) {
		status, res, raw := b.employeeDashboard(t, empB.token)
		require.Equal(t, http.StatusOK, status, string(raw))
		assert.Equal(t, 2, res.Stats.PendingLeave)
		assert.InDelta(t, entity.CountLeaveDays(today.AddDate(0, 0, 5), today.AddDate(0, 0, 6), "full", entity.DefaultWeeklyOffDays(), nil), res.Stats.UpcomingLeaveDays, 0.001)
		assert.Equal(t, 70, res.Stats.AllocationPercent)
		assert.InDelta(t, 8.0, res.Stats.HoursThisWeek, 0.001)
		require.Len(t, res.Holidays, 1)
		assert.Equal(t, "TB Holiday", res.Holidays[0].Name)
		assert.NotContains(t, string(raw), "TA", "tenant A data leaked into tenant B response")
	})

	t.Run("unauthenticated is 401", func(t *testing.T) {
		status, _, _ := a.employeeDashboard(t, "")
		assert.Equal(t, http.StatusUnauthorized, status)
	})
}
