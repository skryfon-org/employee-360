//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// leaveResp is a decoded envelope with the data kept raw.
type leaveResp struct {
	status int
	code   string // error code, empty on success
	raw    []byte
	data   json.RawMessage
	meta   *response.Meta
}

func (e *authFlowEnv) leave(t *testing.T, method, path string, body any, tok string) leaveResp {
	t.Helper()
	status, _, raw := e.doJSON(t, method, path, body, tok)
	var env struct {
		Data  json.RawMessage `json:"data"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
		Meta *response.Meta `json:"meta"`
	}
	_ = json.Unmarshal(raw, &env)
	r := leaveResp{status: status, raw: raw, data: env.Data, meta: env.Meta}
	if env.Error != nil {
		r.code = env.Error.Code
	}
	return r
}

func (r leaveResp) obj(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(r.data, &m), string(r.raw))
	return m
}

func (r leaveResp) list(t *testing.T) []map[string]any {
	t.Helper()
	var l []map[string]any
	require.NoError(t, json.Unmarshal(r.data, &l), string(r.raw))
	return l
}

func (r leaveResp) ids(t *testing.T, key string) []string {
	t.Helper()
	var out []string
	for _, m := range r.list(t) {
		out = append(out, m[key].(string))
	}
	return out
}

func TestLeaveFlow_EndToEnd(t *testing.T) {
	e := newAuthFlowEnv(t)
	other := newAuthFlowEnv(t)

	adminTok := e.login(t, authFlowPassword).AccessToken
	otherAdminTok := other.login(t, authFlowPassword).AccessToken

	mgr := seedEmployee(t, e, "mgr", true, true)
	emp1 := seedEmployee(t, e, "emp1", true, true)
	emp2 := seedEmployee(t, e, "emp2", true, true)
	foreign := seedEmployee(t, other, "foreign", true, true)

	// emp1 reports to mgr (employee_profiles.manager_id).
	require.NoError(t, e.db.Exec(
		"INSERT INTO employee_profiles (id, tenant_id, user_id, manager_id) VALUES (?, ?, ?, ?)",
		uuid.New(), e.tenantID, emp1.id, mgr.id).Error)

	// A second admin, used for the "admin applicant cannot decide own request" case.
	admin2 := seedEmployee(t, e, "admin2", true, true)
	var adminRole string
	require.NoError(t, e.db.Raw("SELECT id FROM roles WHERE tenant_id = ? AND name = ?", e.tenantID, entity.RoleAdmin).Scan(&adminRole).Error)
	require.NoError(t, e.db.Exec("INSERT INTO user_roles (id, tenant_id, user_id, role_id) VALUES (?, ?, ?, ?)",
		uuid.New(), e.tenantID, admin2.id, adminRole).Error)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// A single weekly off day that is neither today nor yesterday keeps the
	// today/yesterday window cases applicable on any weekday. Non-negative
	// offsets count working days (order-preserving, never an off day) so the
	// fixture ranges stay valid whatever weekday the suite runs on.
	offWeekday := (int(today.Weekday()) + 3) % 7
	require.NoError(t, e.db.Exec("INSERT INTO organization_settings (id, tenant_id, weekly_off_days) VALUES (?, ?, ?)",
		uuid.New(), e.tenantID, fmt.Sprintf("{%d}", offWeekday)).Error)
	d := func(offset int) string {
		if offset < 0 {
			return today.AddDate(0, 0, offset).Format("2006-01-02")
		}
		cur := today
		for n := -1; ; cur = cur.AddDate(0, 0, 1) {
			if int(cur.Weekday()) != offWeekday {
				n++
			}
			if n == offset {
				return cur.Format("2006-01-02")
			}
		}
	}

	apply := func(tok string, typeID, approver, start, end, part string) leaveResp {
		body := map[string]any{"leave_type_id": typeID, "start_date": start, "end_date": end, "approver_id": approver, "reason": "e2e"}
		if part != "" {
			body["day_part"] = part
		}
		return e.leave(t, http.MethodPost, "/api/v1/me/leave-requests", body, tok)
	}
	mustApply := func(tok, typeID, approver, start, end string) string {
		r := apply(tok, typeID, approver, start, end, "")
		require.Equal(t, http.StatusCreated, r.status, string(r.raw))
		return r.obj(t)["id"].(string)
	}

	// ---------------------------------------------------------------
	// Leave types
	// ---------------------------------------------------------------
	var casual, sick, old string
	t.Run("leave types", func(t *testing.T) {
		mk := func(name, code string) string {
			r := e.leave(t, http.MethodPost, "/api/v1/leave-types", map[string]any{"name": name, "code": code}, adminTok)
			require.Equal(t, http.StatusCreated, r.status, string(r.raw))
			return r.obj(t)["id"].(string)
		}
		casual, sick, old = mk("Casual", "casual"), mk("Sick", "SICK"), mk("Old", "OLD")
		r := e.leave(t, http.MethodGet, "/api/v1/leave-types", nil, adminTok)
		assert.Len(t, r.list(t), 3)

		// Duplicate code (case-insensitive) conflicts.
		r = e.leave(t, http.MethodPost, "/api/v1/leave-types", map[string]any{"name": "Dup", "code": "CASUAL"}, adminTok)
		assert.Equal(t, http.StatusConflict, r.status)
		// Employees cannot write.
		r = e.leave(t, http.MethodPost, "/api/v1/leave-types", map[string]any{"name": "X", "code": "X"}, emp1.token)
		assert.Equal(t, http.StatusForbidden, r.status)
		r = e.leave(t, http.MethodPatch, "/api/v1/leave-types/"+old, map[string]any{"is_active": false}, emp1.token)
		assert.Equal(t, http.StatusForbidden, r.status)

		// Deactivate (no DELETE route).
		r = e.leave(t, http.MethodPatch, "/api/v1/leave-types/"+old, map[string]any{"is_active": false}, adminTok)
		require.Equal(t, http.StatusOK, r.status, string(r.raw))
		assert.Equal(t, false, r.obj(t)["is_active"])
		r = e.leave(t, http.MethodPatch, "/api/v1/leave-types/"+casual, map[string]any{"description": "Short notice"}, adminTok)
		require.Equal(t, http.StatusOK, r.status)
		assert.Equal(t, "Short notice", r.obj(t)["description"])
		assert.Equal(t, http.StatusNotFound, e.leave(t, http.MethodDelete, "/api/v1/leave-types/"+old, nil, adminTok).status)

		// Employee sees active only, and include_inactive is admin-only.
		r = e.leave(t, http.MethodGet, "/api/v1/leave-types?include_inactive=true", nil, emp1.token)
		require.Equal(t, http.StatusOK, r.status)
		assert.ElementsMatch(t, []string{casual, sick}, r.ids(t, "id"))
		r = e.leave(t, http.MethodGet, "/api/v1/leave-types", nil, adminTok)
		assert.ElementsMatch(t, []string{casual, sick}, r.ids(t, "id"))
		r = e.leave(t, http.MethodGet, "/api/v1/leave-types?include_inactive=true", nil, adminTok)
		assert.ElementsMatch(t, []string{casual, sick, old}, r.ids(t, "id"))

		// Tenant isolation.
		r = e.leave(t, http.MethodGet, "/api/v1/leave-types", nil, otherAdminTok)
		assert.Empty(t, r.list(t))
		r = e.leave(t, http.MethodPatch, "/api/v1/leave-types/"+casual, map[string]any{"name": "hijack"}, otherAdminTok)
		assert.Equal(t, http.StatusNotFound, r.status)
		assert.Equal(t, http.StatusUnauthorized, e.leave(t, http.MethodGet, "/api/v1/leave-types", nil, "").status)
	})

	// ---------------------------------------------------------------
	// Approvers
	// ---------------------------------------------------------------
	t.Run("approvers", func(t *testing.T) {
		r := e.leave(t, http.MethodGet, "/api/v1/me/leave-approvers", nil, emp1.token)
		require.Equal(t, http.StatusOK, r.status, string(r.raw))
		list := r.list(t)
		require.Len(t, list, 3) // manager + two admins
		assert.Equal(t, mgr.id.String(), list[0]["user_id"])
		assert.Equal(t, "manager", list[0]["kind"])
		assert.ElementsMatch(t, []string{e.userID.String(), admin2.id.String()}, r.ids(t, "user_id")[1:])
		assert.NotContains(t, r.ids(t, "user_id"), emp1.id.String(), "self excluded")

		// No profile -> admins only.
		r = e.leave(t, http.MethodGet, "/api/v1/me/leave-approvers", nil, emp2.token)
		assert.ElementsMatch(t, []string{e.userID.String(), admin2.id.String()}, r.ids(t, "user_id"))

		// An admin never sees themselves.
		r = e.leave(t, http.MethodGet, "/api/v1/me/leave-approvers", nil, adminTok)
		assert.Equal(t, []string{admin2.id.String()}, r.ids(t, "user_id"))

		// Tenant isolation: the other tenant lists only its own admin.
		r = e.leave(t, http.MethodGet, "/api/v1/me/leave-approvers", nil, foreign.token)
		assert.Equal(t, []string{other.userID.String()}, r.ids(t, "user_id"))
	})

	// ---------------------------------------------------------------
	// Applying
	// ---------------------------------------------------------------
	var firstReq string
	t.Run("apply", func(t *testing.T) {
		r := apply(emp1.token, casual, mgr.id.String(), d(30), d(32), "")
		require.Equal(t, http.StatusCreated, r.status, string(r.raw))
		o := r.obj(t)
		firstReq = o["id"].(string)
		assert.Equal(t, "pending", o["status"])
		assert.Equal(t, "full", o["day_part"])
		assert.Equal(t, d(30), o["start_date"])
		assert.Equal(t, "CASUAL", o["leave_type_code"])
		assert.Equal(t, mgr.id.String(), o["approver_id"])

		// Date window: today and yesterday (UTC grace) accepted, today-2 and today-6 rejected, far future accepted.
		r = apply(emp1.token, sick, e.userID.String(), d(0), d(0), "")
		assert.Equal(t, http.StatusCreated, r.status, string(r.raw))
		r = apply(emp1.token, sick, e.userID.String(), d(-1), d(-1), "")
		assert.Equal(t, http.StatusCreated, r.status, string(r.raw))
		r = apply(emp1.token, sick, e.userID.String(), d(-2), d(-2), "")
		assert.Equal(t, http.StatusUnprocessableEntity, r.status)
		assert.Equal(t, "leave_start_too_old", r.code)
		r = apply(emp1.token, sick, e.userID.String(), d(-6), d(-6), "")
		assert.Equal(t, http.StatusUnprocessableEntity, r.status)
		r = apply(emp1.token, sick, e.userID.String(), d(2000), d(2000), "")
		assert.Equal(t, http.StatusCreated, r.status, string(r.raw))

		// Date order / day-part rules.
		assert.Equal(t, http.StatusBadRequest, apply(emp1.token, casual, mgr.id.String(), d(50), d(49), "").status)
		assert.Equal(t, http.StatusBadRequest, apply(emp1.token, casual, mgr.id.String(), d(50), d(51), "first_half").status, "half day over two days")
		assert.Equal(t, http.StatusBadRequest, apply(emp1.token, casual, mgr.id.String(), d(50), d(50), "quarter").status)
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodPost, "/api/v1/me/leave-requests", map[string]any{"start_date": "nope"}, emp1.token).status)

		// Approver validation.
		r = apply(emp1.token, casual, emp2.id.String(), d(50), d(50), "")
		assert.Equal(t, http.StatusUnprocessableEntity, r.status)
		assert.Equal(t, "invalid_approver", r.code, "plain colleague")
		r = apply(emp1.token, casual, emp1.id.String(), d(50), d(50), "")
		assert.Equal(t, "invalid_approver", r.code, "self")
		r = apply(emp1.token, casual, foreign.id.String(), d(50), d(50), "")
		assert.Equal(t, "invalid_approver", r.code, "other tenant's user")
		r = apply(adminTok, casual, e.userID.String(), d(50), d(50), "")
		assert.Equal(t, "invalid_approver", r.code, "admin picking themselves")

		// Leave type rules.
		r = apply(emp1.token, old, mgr.id.String(), d(50), d(50), "")
		assert.Equal(t, http.StatusUnprocessableEntity, r.status)
		assert.Equal(t, "leave_type_inactive", r.code)
		r = apply(emp1.token, uuid.NewString(), mgr.id.String(), d(50), d(50), "")
		assert.Equal(t, http.StatusNotFound, r.status)
		r = apply(foreign.token, casual, other.userID.String(), d(50), d(50), "")
		assert.Equal(t, http.StatusNotFound, r.status, "type of another tenant")

		// Listing own requests.
		r = e.leave(t, http.MethodGet, "/api/v1/me/leave-requests", nil, emp1.token)
		require.Equal(t, http.StatusOK, r.status)
		assert.Len(t, r.list(t), 4)
		require.NotNil(t, r.meta)
		assert.EqualValues(t, 4, r.meta.TotalItems)
		assert.Equal(t, 1, r.meta.Page)
		assert.Equal(t, 20, r.meta.PageSize)
		assert.Equal(t, 1, r.meta.TotalPages)
		pg := e.leave(t, http.MethodGet, "/api/v1/me/leave-requests?page=2&page_size=2", nil, emp1.token)
		require.Equal(t, http.StatusOK, pg.status, string(pg.raw))
		assert.Len(t, pg.list(t), 2, "last two items on page 2")
		assert.EqualValues(t, 4, pg.meta.TotalItems)
		assert.Equal(t, 2, pg.meta.TotalPages)
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, "/api/v1/me/leave-requests?page=x", nil, emp1.token).status)
		assert.Empty(t, e.leave(t, http.MethodGet, "/api/v1/me/leave-requests", nil, emp2.token).list(t))
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, "/api/v1/me/leave-requests?status=bogus", nil, emp1.token).status)
		// Date range filters (overlap semantics), on both caller-scoped lists.
		for _, path := range []string{"/api/v1/me/leave-requests", "/api/v1/leave-requests/team"} {
			tok := emp1.token
			if path != "/api/v1/me/leave-requests" {
				tok = mgr.token
			}
			far := e.leave(t, http.MethodGet, path+"?from=2000-01-01&to=2000-01-02", nil, tok)
			require.Equal(t, http.StatusOK, far.status, string(far.raw))
			assert.Empty(t, far.list(t), path)
			all := e.leave(t, http.MethodGet, path+"?from=2000-01-01", nil, tok)
			require.Equal(t, http.StatusOK, all.status)
			assert.NotEmpty(t, all.list(t), path)
			assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, path+"?from=2026-02-01&to=2026-01-01", nil, tok).status, path)
			assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, path+"?from=nope", nil, tok).status, path)
			// leave_type_id filter: unknown type is empty, known type only returns that type, malformed is 400.
			none := e.leave(t, http.MethodGet, path+"?leave_type_id=00000000-0000-4000-8000-000000000001", nil, tok)
			require.Equal(t, http.StatusOK, none.status, string(none.raw))
			assert.Empty(t, none.list(t), path)
			bySick := e.leave(t, http.MethodGet, path+"?leave_type_id="+sick, nil, tok)
			require.Equal(t, http.StatusOK, bySick.status, string(bySick.raw))
			for _, id := range bySick.ids(t, "leave_type_id") {
				assert.Equal(t, sick, id, path)
			}
			assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, path+"?leave_type_id=nope", nil, tok).status, path)
		}
	})

	t.Run("overlap", func(t *testing.T) {
		for _, rng := range [][2]string{{d(32), d(33)}, {d(29), d(30)}, {d(31), d(31)}} {
			r := apply(emp1.token, casual, mgr.id.String(), rng[0], rng[1], "")
			assert.Equal(t, http.StatusUnprocessableEntity, r.status, rng)
			assert.Equal(t, "leave_overlap", r.code, rng)
		}
		// Adjacent ranges and other users are fine.
		assert.Equal(t, http.StatusCreated, apply(emp1.token, casual, mgr.id.String(), d(33), d(33), "").status)
		assert.Equal(t, http.StatusCreated, apply(emp2.token, casual, e.userID.String(), d(30), d(32), "").status)

		// Opposite halves of one day coexist; the same half or a full day do not.
		day := d(60)
		assert.Equal(t, http.StatusCreated, apply(emp1.token, casual, mgr.id.String(), day, day, "first_half").status)
		assert.Equal(t, http.StatusCreated, apply(emp1.token, casual, mgr.id.String(), day, day, "second_half").status)
		r := apply(emp1.token, casual, mgr.id.String(), day, day, "first_half")
		assert.Equal(t, "leave_overlap", r.code)
		r = apply(emp1.token, casual, mgr.id.String(), day, day, "")
		assert.Equal(t, "leave_overlap", r.code)
	})

	// ---------------------------------------------------------------
	// Team inbox and admin listing
	// ---------------------------------------------------------------
	t.Run("team and admin lists", func(t *testing.T) {
		r := e.leave(t, http.MethodGet, "/api/v1/leave-requests/team", nil, mgr.token)
		require.Equal(t, http.StatusOK, r.status, string(r.raw))
		team := r.list(t)
		require.NotEmpty(t, team)
		for _, m := range team {
			assert.Equal(t, mgr.id.String(), m["approver_id"])
			assert.Equal(t, emp1.id.String(), m["user_id"])
		}
		assert.Contains(t, r.ids(t, "id"), firstReq)
		require.NotNil(t, r.meta)
		assert.EqualValues(t, len(team), r.meta.TotalItems)
		tp := e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?page=1&page_size=1", nil, mgr.token)
		require.Equal(t, http.StatusOK, tp.status, string(tp.raw))
		assert.Len(t, tp.list(t), 1)
		assert.Equal(t, r.meta.TotalItems, tp.meta.TotalItems, "total counts every item, not just the page")
		assert.Equal(t, int(r.meta.TotalItems), tp.meta.TotalPages)
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?page_size=x", nil, mgr.token).status)
		assert.Empty(t, e.leave(t, http.MethodGet, "/api/v1/leave-requests/team", nil, emp2.token).list(t))
		assert.Empty(t, e.leave(t, http.MethodGet, "/api/v1/leave-requests/team", nil, foreign.token).list(t))
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?status=x", nil, mgr.token).status)
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?exclude_status=x", nil, mgr.token).status)

		// exclude_status=pending: history only; totals exclude the pending rows.
		pend := e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?status=pending", nil, mgr.token)
		hist := e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?exclude_status=pending", nil, mgr.token)
		require.Equal(t, http.StatusOK, hist.status, string(hist.raw))
		require.NotNil(t, hist.meta)
		for _, m := range hist.list(t) {
			assert.NotEqual(t, "pending", m["status"])
		}
		assert.EqualValues(t, r.meta.TotalItems-pend.meta.TotalItems, hist.meta.TotalItems)
		assert.Empty(t, e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?status=pending&exclude_status=pending", nil, mgr.token).list(t))
		assert.Empty(t, e.leave(t, http.MethodGet, "/api/v1/leave-requests/team?exclude_status=pending", nil, foreign.token).list(t))

		// Admin listing: employees are forbidden.
		assert.Equal(t, http.StatusForbidden, e.leave(t, http.MethodGet, "/api/v1/leave-requests", nil, emp1.token).status)
		assert.Equal(t, http.StatusForbidden, e.leave(t, http.MethodGet, "/api/v1/leave-requests", nil, mgr.token).status)

		all := e.leave(t, http.MethodGet, "/api/v1/leave-requests", nil, adminTok)
		require.Equal(t, http.StatusOK, all.status, string(all.raw))
		require.NotNil(t, all.meta)
		total := all.meta.TotalItems
		assert.EqualValues(t, len(all.list(t)), total)

		page := e.leave(t, http.MethodGet, "/api/v1/leave-requests?page_size=2&page=1", nil, adminTok)
		assert.Len(t, page.list(t), 2)
		assert.Equal(t, total, page.meta.TotalItems)

		byEmp := e.leave(t, http.MethodGet, "/api/v1/leave-requests?employee_id="+emp2.id.String(), nil, adminTok)
		assert.Len(t, byEmp.list(t), 1)
		for _, id := range byEmp.ids(t, "user_id") {
			assert.Equal(t, emp2.id.String(), id)
		}
		byType := e.leave(t, http.MethodGet, "/api/v1/leave-requests?leave_type_id="+sick, nil, adminTok)
		assert.Len(t, byType.list(t), 3)
		pending := e.leave(t, http.MethodGet, "/api/v1/leave-requests?status=approved", nil, adminTok)
		assert.Empty(t, pending.list(t))

		q := url.Values{"from": {d(30)}, "to": {d(33)}, "employee_id": {emp1.id.String()}}
		byRange := e.leave(t, http.MethodGet, "/api/v1/leave-requests?"+q.Encode(), nil, adminTok)
		assert.Len(t, byRange.list(t), 2) // d30-32 and d33
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, "/api/v1/leave-requests?from=bad", nil, adminTok).status)
		assert.Equal(t, http.StatusBadRequest, e.leave(t, http.MethodGet, "/api/v1/leave-requests?status=bad", nil, adminTok).status)

		// Tenant isolation.
		fa := e.leave(t, http.MethodGet, "/api/v1/leave-requests", nil, otherAdminTok)
		assert.EqualValues(t, 0, fa.meta.TotalItems)
		assert.Empty(t, fa.list(t))
	})

	// ---------------------------------------------------------------
	// Decisions
	// ---------------------------------------------------------------
	decide := func(tok, id, action, comment string) leaveResp {
		var body any
		if comment != "" {
			body = map[string]any{"comment": comment}
		}
		return e.leave(t, http.MethodPost, "/api/v1/leave-requests/"+id+"/"+action, body, tok)
	}
	t.Run("approve and reject", func(t *testing.T) {
		id := mustApply(emp1.token, casual, mgr.id.String(), d(100), d(101))

		// The applicant never decides; a stranger sees nothing.
		assert.Equal(t, http.StatusForbidden, decide(emp1.token, id, "approve", "").status)
		assert.Equal(t, http.StatusForbidden, decide(emp1.token, id, "reject", "x").status)
		assert.Equal(t, http.StatusNotFound, decide(emp2.token, id, "approve", "").status)
		assert.Equal(t, http.StatusNotFound, decide(otherAdminTok, id, "approve", "").status, "cross-tenant admin")
		assert.Equal(t, http.StatusNotFound, decide(foreign.token, id, "reject", "").status)
		assert.Equal(t, http.StatusNotFound, decide(adminTok, uuid.NewString(), "approve", "").status)
		assert.Equal(t, http.StatusBadRequest, decide(adminTok, "not-a-uuid", "approve", "").status)
		still := e.leave(t, http.MethodGet, "/api/v1/me/leave-requests?status=pending", nil, emp1.token)
		assert.Contains(t, still.ids(t, "id"), id, "failed attempts left it pending")

		// Chosen approver approves with a comment.
		r := decide(mgr.token, id, "approve", "enjoy")
		require.Equal(t, http.StatusOK, r.status, string(r.raw))
		o := r.obj(t)
		assert.Equal(t, "approved", o["status"])
		assert.Equal(t, mgr.id.String(), o["decided_by"])
		assert.Equal(t, "enjoy", o["decision_comment"])
		assert.NotEmpty(t, o["decided_at"])

		// Only from pending.
		r = decide(mgr.token, id, "reject", "")
		assert.Equal(t, http.StatusConflict, r.status)
		assert.Equal(t, "leave_not_pending", r.code)
		assert.Equal(t, http.StatusConflict, decide(adminTok, id, "approve", "").status)

		// An admin who was not chosen may reject.
		id2 := mustApply(emp1.token, casual, mgr.id.String(), d(110), d(110))
		r = decide(adminTok, id2, "reject", "no capacity")
		require.Equal(t, http.StatusOK, r.status, string(r.raw))
		assert.Equal(t, "rejected", r.obj(t)["status"])
		assert.Equal(t, e.userID.String(), r.obj(t)["decided_by"])

		// The chosen approver being an admin works too.
		id3 := mustApply(emp1.token, casual, e.userID.String(), d(120), d(120))
		assert.Equal(t, http.StatusOK, decide(adminTok, id3, "approve", "").status)

		// An admin applicant cannot decide their own request, nor can the other admin... only themselves.
		own := mustApply(admin2.token, casual, e.userID.String(), d(130), d(130))
		assert.Equal(t, http.StatusForbidden, decide(admin2.token, own, "approve", "").status)
		assert.Equal(t, http.StatusOK, decide(adminTok, own, "approve", "").status)

		// Audit rows were written for the decisions.
		var n int64
		require.NoError(t, e.db.Raw("SELECT count(*) FROM audit_logs WHERE tenant_id = ? AND action IN ('leave_request.approve','leave_request.reject')", e.tenantID).Scan(&n).Error)
		assert.EqualValues(t, 4, n)
	})

	// ---------------------------------------------------------------
	// Cancel
	// ---------------------------------------------------------------
	t.Run("cancel", func(t *testing.T) {
		cancel := func(tok, id string) leaveResp {
			return e.leave(t, http.MethodPost, "/api/v1/me/leave-requests/"+id+"/cancel", nil, tok)
		}
		// Pending -> cancelled; then final.
		id := mustApply(emp2.token, casual, e.userID.String(), d(200), d(201))
		assert.Equal(t, http.StatusNotFound, cancel(emp1.token, id).status, "not the owner")
		assert.Equal(t, http.StatusNotFound, cancel(foreign.token, id).status, "other tenant")
		assert.Equal(t, http.StatusNotFound, cancel(adminTok, id).status, "even the approver")
		r := cancel(emp2.token, id)
		require.Equal(t, http.StatusOK, r.status, string(r.raw))
		assert.Equal(t, "cancelled", r.obj(t)["status"])
		r = cancel(emp2.token, id)
		assert.Equal(t, http.StatusConflict, r.status)
		assert.Equal(t, "leave_not_cancellable", r.code)
		// A cancelled request frees its dates.
		id = mustApply(emp2.token, casual, e.userID.String(), d(200), d(201))

		// Approved and not yet started -> cancellable.
		assert.Equal(t, http.StatusOK, decide(adminTok, id, "approve", "ok to go").status)
		r = cancel(emp2.token, id)
		require.Equal(t, http.StatusOK, r.status, string(r.raw))
		o := r.obj(t)
		assert.Equal(t, "cancelled", o["status"])
		assert.Equal(t, e.userID.String(), o["decided_by"], "cancel keeps the approver as decided_by")
		assert.Equal(t, "ok to go", o["decision_comment"])
		assert.NotEmpty(t, o["decided_at"])
		// A cancelled-while-pending request never had a decision.
		pend := mustApply(emp2.token, casual, e.userID.String(), d(205), d(205))
		o = cancel(emp2.token, pend).obj(t)
		assert.Nil(t, o["decided_by"])
		assert.Nil(t, o["decided_at"])

		// Approved but already started -> 409. Apply forbids past starts, so apply for
		// the future and then move the dates back in the DB to simulate time passing.
		started := mustApply(emp2.token, casual, e.userID.String(), d(300), d(301))
		require.NoError(t, e.db.Exec("UPDATE leave_requests SET start_date = ?, end_date = ? WHERE id = ?", d(-5), d(-4), started).Error)
		assert.Equal(t, http.StatusOK, decide(adminTok, started, "approve", "").status)
		r = cancel(emp2.token, started)
		assert.Equal(t, http.StatusConflict, r.status)
		assert.Equal(t, "leave_not_cancellable", r.code)

		// Starting today is still cancellable; rejected is final.
		todayReq := mustApply(emp2.token, casual, e.userID.String(), d(0), d(1))
		assert.Equal(t, http.StatusOK, decide(adminTok, todayReq, "approve", "").status)
		assert.Equal(t, http.StatusOK, cancel(emp2.token, todayReq).status)
		rej := mustApply(emp2.token, casual, e.userID.String(), d(210), d(210))
		assert.Equal(t, http.StatusOK, decide(adminTok, rej, "reject", "").status)
		assert.Equal(t, http.StatusConflict, cancel(emp2.token, rej).status)
		assert.Equal(t, http.StatusNotFound, cancel(emp2.token, uuid.NewString()).status)
	})

	t.Run("audit trail", func(t *testing.T) {
		count := func(q string, args ...any) int64 {
			var n int64
			require.NoError(t, e.db.Raw(q, args...).Scan(&n).Error)
			return n
		}
		var requests, cancelled int64
		requests = count("SELECT count(*) FROM leave_requests WHERE tenant_id = ?", e.tenantID)
		cancelled = count("SELECT count(*) FROM leave_requests WHERE tenant_id = ? AND status = 'cancelled'", e.tenantID)
		require.NotZero(t, requests)
		require.NotZero(t, cancelled)
		assert.Equal(t, requests, count("SELECT count(*) FROM audit_logs WHERE tenant_id = ? AND action = 'leave_request.apply'", e.tenantID), "one apply entry per request")
		assert.Equal(t, cancelled, count("SELECT count(*) FROM audit_logs WHERE tenant_id = ? AND action = 'leave_request.cancel'", e.tenantID), "one cancel entry per cancellation")
		assert.EqualValues(t, 2, count("SELECT count(*) FROM audit_logs WHERE tenant_id = ? AND action = 'leave_request.cancel' AND metadata->>'previous_status' = 'approved'", e.tenantID), "approved->cancelled records the previous status")
		assert.Positive(t, count("SELECT count(*) FROM audit_logs WHERE tenant_id = ? AND action = 'leave_request.cancel' AND metadata->>'previous_status' = 'pending'", e.tenantID))
		assert.Zero(t, count("SELECT count(*) FROM audit_logs WHERE tenant_id = ? AND action LIKE 'leave_request.%'", other.tenantID), "no cross-tenant audit rows")
	})

	// Everything created above is removed with the tenant (ON DELETE CASCADE),
	// including leave_types/leave_requests.
}
