//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
)

// projectActor is a seeded user of an authFlowEnv tenant plus a valid bearer token.
type projectActor struct {
	id    uuid.UUID
	email string
	token string
}

// seedEmployee creates an employee-role user in the env's tenant. When login
// is true it also logs in and fills the bearer token.
func seedEmployee(t *testing.T, e *authFlowEnv, local string, active, login bool) projectActor {
	t.Helper()
	var existing string
	err := e.db.Raw("SELECT id FROM roles WHERE tenant_id = ? AND name = ?", e.tenantID, entity.RoleEmployee).Scan(&existing).Error
	require.NoError(t, err)
	roleID := uuid.New()
	if existing != "" {
		roleID = uuid.MustParse(existing)
	} else {
		require.NoError(t, e.db.Create(&entity.Role{ID: roleID, TenantID: e.tenantID, Name: entity.RoleEmployee}).Error)
	}
	hash, err := infraservice.NewHashService().HashPassword(authFlowPassword)
	require.NoError(t, err)
	a := projectActor{id: uuid.New(), email: local + "@" + e.domain}
	require.NoError(t, e.db.Create(&entity.User{
		ID: a.id, TenantID: e.tenantID, FirstName: local, LastName: "Emp", Email: a.email, PasswordHash: &hash, IsActive: active,
	}).Error)
	// GORM skips the zero-value bool on insert default; force the flag explicitly.
	require.NoError(t, e.db.Exec("UPDATE users SET is_active = ? WHERE id = ?", active, a.id).Error)
	require.NoError(t, e.db.Exec("INSERT INTO user_roles (id, tenant_id, user_id, role_id) VALUES (?, ?, ?, ?)",
		uuid.New(), e.tenantID, a.id, roleID).Error)
	if login {
		code, env, raw := e.post(t, "/login", map[string]string{"email": a.email, "password": authFlowPassword}, "")
		require.Equal(t, http.StatusOK, code, string(raw))
		var pair tokenPairBody
		require.NoError(t, json.Unmarshal(env.Data, &pair))
		a.token = pair.AccessToken
	}
	return a
}

// call performs a request and returns status plus the decoded "data" object.
func (e *authFlowEnv) call(t *testing.T, method, path string, body any, tok string) (int, map[string]any, []byte) {
	t.Helper()
	code, env, raw := e.doJSON(t, method, path, body, tok)
	var data map[string]any
	if len(env.Data) > 0 {
		_ = json.Unmarshal(env.Data, &data)
	}
	return code, data, raw
}

func (e *authFlowEnv) mustCreateProject(t *testing.T, tok, code string, extra map[string]any) string {
	t.Helper()
	body := map[string]any{"name": "Project " + code, "code": code, "start_date": "2026-01-01"}
	for k, v := range extra {
		body[k] = v
	}
	status, data, raw := e.call(t, http.MethodPost, "/api/v1/projects", body, tok)
	require.Equal(t, http.StatusCreated, status, string(raw))
	return data["id"].(string)
}

func allocBody(user uuid.UUID, pct any, start string, end any) map[string]any {
	b := map[string]any{"user_id": user, "allocation_percent": pct, "start_date": start}
	if end != nil {
		b["end_date"] = end
	}
	return b
}

func TestProjectFlow_ProjectsAndAllocations(t *testing.T) {
	e := newAuthFlowEnv(t)
	other := newAuthFlowEnv(t)
	admin := e.login(t, authFlowPassword).AccessToken
	otherAdmin := other.login(t, authFlowPassword).AccessToken
	emp := seedEmployee(t, e, "erin", true, true)
	inactive := seedEmployee(t, e, "ian", false, false)
	foreign := seedEmployee(t, other, "fred", true, false)

	// --- project CRUD ---
	t.Run("create, duplicate code, other tenant", func(t *testing.T) {
		status, data, raw := e.call(t, http.MethodPost, "/api/v1/projects", map[string]any{
			"name": "Apollo", "code": "apo", "start_date": "2026-01-01", "client_name": "ACME"}, admin)
		require.Equal(t, http.StatusCreated, status, string(raw))
		require.Equal(t, "APO", data["code"])
		require.Equal(t, "active", data["status"])

		status, _, _ = e.call(t, http.MethodPost, "/api/v1/projects", map[string]any{"name": "Dup", "code": "APO", "start_date": "2026-01-01"}, admin)
		require.Equal(t, http.StatusConflict, status)

		status, _, raw = other.call(t, http.MethodPost, "/api/v1/projects", map[string]any{"name": "Apollo", "code": "APO", "start_date": "2026-01-01"}, otherAdmin)
		require.Equal(t, http.StatusCreated, status, "same code is fine in another tenant: "+string(raw))

		status, _, _ = e.call(t, http.MethodPost, "/api/v1/projects", map[string]any{"name": "X", "code": "X", "start_date": "2026-02-01", "end_date": "2026-01-01"}, admin)
		require.Equal(t, http.StatusBadRequest, status, "end before start")
	})

	t.Run("generated code, collisions, suggestion, tenant isolation", func(t *testing.T) {
		mk := func(env *authFlowEnv, tok, name string) (int, map[string]any, []byte) {
			return env.call(t, http.MethodPost, "/api/v1/projects", map[string]any{"name": name, "start_date": "2026-01-01"}, tok)
		}
		status, data, raw := e.call(t, http.MethodGet, "/api/v1/projects/code-suggestion?name=Zeta+Mission", nil, admin)
		require.Equal(t, http.StatusOK, status, string(raw))
		require.Equal(t, "ZETA-MISSION", data["code"])

		status, data, raw = mk(e, admin, "Zeta Mission")
		require.Equal(t, http.StatusCreated, status, string(raw))
		require.Equal(t, "ZETA-MISSION", data["code"])
		status, data, _ = mk(e, admin, "zeta mission")
		require.Equal(t, http.StatusCreated, status)
		require.Equal(t, "ZETA-MISSION-2", data["code"])

		_, data, _ = e.call(t, http.MethodGet, "/api/v1/projects/code-suggestion?name=Zeta+Mission", nil, admin)
		require.Equal(t, "ZETA-MISSION-3", data["code"])
		_, data, _ = other.call(t, http.MethodGet, "/api/v1/projects/code-suggestion?name=Zeta+Mission", nil, otherAdmin)
		require.Equal(t, "ZETA-MISSION", data["code"], "other tenant's codes are not visible")
		status, data, _ = mk(other, otherAdmin, "Zeta Mission")
		require.Equal(t, http.StatusCreated, status)
		require.Equal(t, "ZETA-MISSION", data["code"])

		status, data, _ = mk(e, admin, "!!!")
		require.Equal(t, http.StatusCreated, status)
		require.Equal(t, "PROJECT", data["code"])
		status, _, _ = mk(e, admin, "  ")
		require.Equal(t, http.StatusBadRequest, status)
		status, _, _ = e.call(t, http.MethodGet, "/api/v1/projects/code-suggestion", nil, admin)
		require.Equal(t, http.StatusBadRequest, status)
		status, _, _ = e.call(t, http.MethodPost, "/api/v1/projects", map[string]any{"name": "Zeta Mission", "code": "ZETA-MISSION", "start_date": "2026-01-01"}, admin)
		require.Equal(t, http.StatusConflict, status, "explicit duplicate code still 409")
	})

	t.Run("list search status pagination", func(t *testing.T) {
		list := func(q string) (total int, n int) {
			_, env, raw := e.doJSON(t, http.MethodGet, "/api/v1/projects"+q, nil, admin)
			var full struct {
				Data []json.RawMessage `json:"data"`
				Meta struct {
					TotalItems int `json:"total_items"`
				} `json:"meta"`
			}
			require.NoError(t, json.Unmarshal(raw, &full), string(raw))
			_ = env
			return full.Meta.TotalItems, len(full.Data)
		}
		// Earlier subtests share this tenant, so count relative to what already exists.
		before, _ := list("")
		e.mustCreateProject(t, admin, "BRAVO", map[string]any{"status": "on_hold"})
		e.mustCreateProject(t, admin, "CHARLIE", nil)
		want := before + 2
		total, _ := list("")
		require.Equal(t, want, total)
		total, n := list("?search=acme")
		require.Equal(t, 1, total)
		require.Equal(t, 1, n)
		total, _ = list("?status=on_hold")
		require.Equal(t, 1, total)
		total, n = list("?page=2&page_size=2")
		require.Equal(t, want, total)
		require.Equal(t, 2, n)
		code, _, _ := e.call(t, http.MethodGet, "/api/v1/projects?status=bogus", nil, admin)
		require.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("get and patch incl. archive", func(t *testing.T) {
		id := e.mustCreateProject(t, admin, "DELTA", nil)
		status, data, _ := e.call(t, http.MethodGet, "/api/v1/projects/"+id, nil, admin)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "DELTA", data["code"])

		status, data, raw := e.call(t, http.MethodPatch, "/api/v1/projects/"+id, map[string]any{"name": "Delta 2", "status": "archived"}, admin)
		require.Equal(t, http.StatusOK, status, string(raw))
		require.Equal(t, "archived", data["status"])
		require.Equal(t, "Delta 2", data["name"])

		status, _, _ = e.call(t, http.MethodPatch, "/api/v1/projects/"+id, map[string]any{"code": "APO"}, admin)
		require.Equal(t, http.StatusConflict, status)
		status, _, _ = e.call(t, http.MethodPatch, "/api/v1/projects/"+id, map[string]any{"status": "bogus"}, admin)
		require.Equal(t, http.StatusBadRequest, status)

		// Archived project rejects new allocations (409) ...
		status, _, raw = e.call(t, http.MethodPost, "/api/v1/projects/"+id+"/allocations", allocBody(emp.id, 10, "2026-01-01", nil), admin)
		require.Equal(t, http.StatusConflict, status, string(raw))
		// ... but allows being reopened.
		status, _, _ = e.call(t, http.MethodPatch, "/api/v1/projects/"+id, map[string]any{"status": "active"}, admin)
		require.Equal(t, http.StatusOK, status)
	})

	// --- allocation validation ---
	pid := e.mustCreateProject(t, admin, "ECHO", nil)
	allocURL := "/api/v1/projects/" + pid + "/allocations"

	t.Run("allocation validation", func(t *testing.T) {
		for _, pct := range []int{0, 101} {
			status, _, _ := e.call(t, http.MethodPost, allocURL, allocBody(emp.id, pct, "2026-01-01", nil), admin)
			require.Equal(t, http.StatusBadRequest, status, fmt.Sprint("percent ", pct))
		}
		status, _, _ := e.call(t, http.MethodPost, allocURL, allocBody(emp.id, 50, "2026-03-01", "2026-02-01"), admin)
		require.Equal(t, http.StatusBadRequest, status, "end before start")
		status, _, _ = e.call(t, http.MethodPost, allocURL, allocBody(inactive.id, 50, "2026-01-01", nil), admin)
		require.Equal(t, http.StatusConflict, status, "inactive user")
		status, _, _ = e.call(t, http.MethodPost, allocURL, allocBody(foreign.id, 50, "2026-01-01", nil), admin)
		require.Equal(t, http.StatusNotFound, status, "user from another tenant")
		status, _, _ = e.call(t, http.MethodPost, "/api/v1/projects/"+uuid.NewString()+"/allocations", allocBody(emp.id, 50, "2026-01-01", nil), admin)
		require.Equal(t, http.StatusNotFound, status, "unknown project")
	})

	// --- overlap, over-allocation, end date, soft delete ---
	var firstAlloc string
	t.Run("create, overlap, over-allocation", func(t *testing.T) {
		status, data, raw := e.call(t, http.MethodPost, allocURL, allocBody(emp.id, 60, "2026-01-01", "2026-12-31"), admin)
		require.Equal(t, http.StatusCreated, status, string(raw))
		require.Equal(t, false, data["over_allocated"])
		firstAlloc = data["id"].(string)

		status, _, _ = e.call(t, http.MethodPost, allocURL, allocBody(emp.id, 10, "2026-12-31", nil), admin)
		require.Equal(t, http.StatusConflict, status, "overlap on same project")

		// A second live project pushes the total to 130%: succeeds, flagged.
		p2 := e.mustCreateProject(t, admin, "FOXTROT", nil)
		status, data, raw = e.call(t, http.MethodPost, "/api/v1/projects/"+p2+"/allocations", allocBody(emp.id, 70, "2026-01-01", nil), admin)
		require.Equal(t, http.StatusCreated, status, string(raw))
		require.Equal(t, true, data["over_allocated"])
		require.EqualValues(t, 130, data["user_total_percent"])
	})

	t.Run("employee allocations count only live projects", func(t *testing.T) {
		// Complete a third project that carries 50% of the user's time.
		p3 := e.mustCreateProject(t, admin, "GOLF", nil)
		status, _, raw := e.call(t, http.MethodPost, "/api/v1/projects/"+p3+"/allocations", allocBody(emp.id, 50, "2026-01-01", nil), admin)
		require.Equal(t, http.StatusCreated, status, string(raw))
		status, _, _ = e.call(t, http.MethodPatch, "/api/v1/projects/"+p3, map[string]any{"status": "completed"}, admin)
		require.Equal(t, http.StatusOK, status)

		status, data, raw := e.call(t, http.MethodGet, "/api/v1/employees/"+emp.id.String()+"/allocations", nil, admin)
		require.Equal(t, http.StatusOK, status, string(raw))
		items := data["allocations"].([]any)
		require.Len(t, items, 3, "admin sees every date-active allocation")
		statuses := map[string]bool{}
		for _, it := range items {
			m := it.(map[string]any)
			st := m["project_status"].(string)
			statuses[st] = true
			require.Equal(t, st != "completed", m["counts_toward_total"], "flag mirrors live status for "+st)
		}
		require.True(t, statuses["completed"] && statuses["active"])
		require.EqualValues(t, 130, data["total_percent"], "completed project excluded from the total")
		require.Equal(t, true, data["over_allocated"])

		// Employee sees only their own live allocations on /me/projects.
		status, data, raw = e.call(t, http.MethodGet, "/api/v1/me/projects", nil, emp.token)
		require.Equal(t, http.StatusOK, status, string(raw))
		mine := data["allocations"].([]any)
		require.Len(t, mine, 2)
		for _, it := range mine {
			require.Equal(t, emp.id.String(), it.(map[string]any)["user_id"])
			require.Equal(t, "active", it.(map[string]any)["project_status"])
			require.Equal(t, true, it.(map[string]any)["counts_toward_total"])
		}

		// scope param: active is the default; upcoming/past/all widen the read.
		pScope := e.mustCreateProject(t, admin, "SCOPE", map[string]any{"start_date": "2020-01-01"})
		future := time.Now().UTC().AddDate(0, 0, 30).Format("2006-01-02")
		for _, b := range []map[string]any{
			allocBody(emp.id, 10, "2020-02-01", "2020-03-01"),
			allocBody(emp.id, 20, future, nil),
		} {
			status, _, raw = e.call(t, http.MethodPost, "/api/v1/projects/"+pScope+"/allocations", b, admin)
			require.Equal(t, http.StatusCreated, status, string(raw))
		}
		scoped := func(q string) []any {
			st, d, rw := e.call(t, http.MethodGet, "/api/v1/me/projects"+q, nil, emp.token)
			require.Equal(t, http.StatusOK, st, string(rw))
			return d["allocations"].([]any)
		}
		require.Len(t, scoped(""), 2)
		require.Len(t, scoped("?scope=active"), 2)
		up := scoped("?scope=upcoming")
		require.Len(t, up, 1)
		require.Equal(t, "SCOPE", up[0].(map[string]any)["project_code"])
		require.Equal(t, future, up[0].(map[string]any)["start_date"].(string)[:10])
		pst := scoped("?scope=past")
		require.Len(t, pst, 1)
		require.Equal(t, "2020-02-01", pst[0].(map[string]any)["start_date"].(string)[:10])
		all := scoped("?scope=all")
		require.Len(t, all, 5, "2 live active + completed GOLF + upcoming + past")
		require.Equal(t, "2020-02-01", all[0].(map[string]any)["start_date"].(string)[:10], "ordered by start_date")
		status, _, _ = e.call(t, http.MethodGet, "/api/v1/me/projects?scope=bogus", nil, emp.token)
		require.Equal(t, http.StatusBadRequest, status)

		// Admin per-project state filter (?state=), default all. pScope has a
		// past (2020) and an upcoming (+30d) allocation for emp.
		byState := func(q string) []map[string]any {
			st, env, rw := e.doJSON(t, http.MethodGet, "/api/v1/projects/"+pScope+"/allocations"+q, nil, admin)
			require.Equal(t, http.StatusOK, st, string(rw))
			var list []map[string]any
			require.NoError(t, json.Unmarshal(env.Data, &list))
			return list
		}
		require.Len(t, byState(""), 2)
		require.NotEmpty(t, byState("")[0]["user_name"], "user_name is returned on the project allocations list")
		require.Len(t, byState("?state=all"), 2)
		require.Len(t, byState("?state=active"), 0)
		upl := byState("?state=upcoming")
		require.Len(t, upl, 1)
		require.Equal(t, future, upl[0]["start_date"].(string)[:10])
		pl := byState("?state=past")
		require.Len(t, pl, 1)
		require.Equal(t, "2020-02-01", pl[0]["start_date"].(string)[:10])
		status, _, _ = e.call(t, http.MethodGet, "/api/v1/projects/"+pScope+"/allocations?state=bogus", nil, admin)
		require.Equal(t, http.StatusBadRequest, status)

		// The admin has no allocations of their own.
		status, data, _ = e.call(t, http.MethodGet, "/api/v1/me/projects", nil, admin)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, data["allocations"])

		status, _, _ = e.call(t, http.MethodGet, "/api/v1/employees/"+uuid.NewString()+"/allocations", nil, admin)
		require.Equal(t, http.StatusNotFound, status)
		status, _, _ = e.call(t, http.MethodGet, "/api/v1/employees/"+foreign.id.String()+"/allocations", nil, admin)
		require.Equal(t, http.StatusNotFound, status, "other tenant's employee")
	})

	t.Run("allocation dates must fit the project window", func(t *testing.T) {
		pid := e.mustCreateProject(t, admin, "WIN", map[string]any{"start_date": "2026-10-24", "end_date": "2026-10-29"})
		path := "/api/v1/projects/" + pid + "/allocations"
		for name, b := range map[string]map[string]any{
			"before start": allocBody(emp.id, 50, "2026-10-15", "2026-10-26"),
			"after end":    allocBody(emp.id, 50, "2026-10-25", "2026-10-30"),
		} {
			status, _, raw := e.call(t, http.MethodPost, path, b, admin)
			require.Equal(t, http.StatusBadRequest, status, name+": "+string(raw))
		}
		status, data, raw := e.call(t, http.MethodPost, path, allocBody(emp.id, 50, "2026-10-25", "2026-10-27"), admin)
		require.Equal(t, http.StatusCreated, status, string(raw))
		id := data["id"].(string)
		status, _, raw = e.call(t, http.MethodPatch, "/api/v1/allocations/"+id, map[string]any{"end_date": "2026-11-01"}, admin)
		require.Equal(t, http.StatusBadRequest, status, string(raw))
		status, cleared, raw := e.call(t, http.MethodPatch, "/api/v1/allocations/"+id, map[string]any{"end_date": nil}, admin)
		require.Equal(t, http.StatusOK, status, string(raw))
		require.Nil(t, cleared["end_date"], "open-ended after clearing end_date")
	})

	t.Run("patch end date and soft delete", func(t *testing.T) {
		status, data, raw := e.call(t, http.MethodPatch, "/api/v1/allocations/"+firstAlloc, map[string]any{"end_date": "2026-06-30", "allocation_percent": 40}, admin)
		require.Equal(t, http.StatusOK, status, string(raw))
		require.Equal(t, "2026-06-30", data["end_date"])
		require.EqualValues(t, 40, data["allocation_percent"])

		status, _, _ = e.call(t, http.MethodPatch, "/api/v1/allocations/"+firstAlloc, map[string]any{"allocation_percent": 101}, admin)
		require.Equal(t, http.StatusBadRequest, status)
		status, _, _ = e.call(t, http.MethodPatch, "/api/v1/allocations/"+firstAlloc, map[string]any{"end_date": "2025-01-01"}, admin)
		require.Equal(t, http.StatusBadRequest, status)

		status, _, raw = e.call(t, http.MethodDelete, "/api/v1/allocations/"+firstAlloc, nil, admin)
		require.Equal(t, http.StatusOK, status, string(raw))
		status, _, _ = e.call(t, http.MethodDelete, "/api/v1/allocations/"+firstAlloc, nil, admin)
		require.Equal(t, http.StatusNotFound, status)

		var deleted int64
		require.NoError(t, e.db.Raw("SELECT count(*) FROM project_allocations WHERE id = ? AND deleted_at IS NOT NULL", firstAlloc).Scan(&deleted).Error)
		require.EqualValues(t, 1, deleted, "row is soft-deleted, not removed")

		// The project allocation list returns a bare array in data.
		status, env, raw := e.doJSON(t, http.MethodGet, allocURL, nil, admin)
		require.Equal(t, http.StatusOK, status, string(raw))
		var listed []map[string]any
		require.NoError(t, json.Unmarshal(env.Data, &listed), string(raw))
		for _, it := range listed {
			require.NotEqual(t, firstAlloc, it["id"], "soft-deleted allocation must not be listed")
		}
	})

	t.Run("employee role is forbidden on management endpoints", func(t *testing.T) {
		for _, c := range []struct{ method, path string }{
			{http.MethodPost, "/api/v1/projects"},
			{http.MethodGet, "/api/v1/projects"},
			{http.MethodGet, "/api/v1/projects/" + pid},
			{http.MethodPatch, "/api/v1/projects/" + pid},
			{http.MethodGet, allocURL},
			{http.MethodPost, allocURL},
			{http.MethodPatch, "/api/v1/allocations/" + uuid.NewString()},
			{http.MethodDelete, "/api/v1/allocations/" + uuid.NewString()},
			{http.MethodGet, "/api/v1/employees/" + emp.id.String() + "/allocations"},
		} {
			status, _, _ := e.call(t, c.method, c.path, map[string]any{}, emp.token)
			require.Equal(t, http.StatusForbidden, status, c.method+" "+c.path)
			status, _, _ = e.call(t, c.method, c.path, map[string]any{}, "")
			require.Equal(t, http.StatusUnauthorized, status, c.method+" "+c.path+" unauthenticated")
		}
	})

	t.Run("cross-tenant access is 404", func(t *testing.T) {
		// Another tenant's admin cannot see or touch this tenant's project/allocations.
		status, _, _ := other.call(t, http.MethodGet, "/api/v1/projects/"+pid, nil, otherAdmin)
		require.Equal(t, http.StatusNotFound, status)
		status, _, _ = other.call(t, http.MethodPatch, "/api/v1/projects/"+pid, map[string]any{"name": "pwn"}, otherAdmin)
		require.Equal(t, http.StatusNotFound, status)
		status, _, _ = other.call(t, http.MethodGet, allocURL, nil, otherAdmin)
		require.Equal(t, http.StatusNotFound, status)
		status, _, _ = other.call(t, http.MethodPost, allocURL, allocBody(foreign.id, 10, "2026-01-01", nil), otherAdmin)
		require.Equal(t, http.StatusNotFound, status)

		var allocID string
		require.NoError(t, e.db.Raw("SELECT id FROM project_allocations WHERE tenant_id = ? AND deleted_at IS NULL LIMIT 1", e.tenantID).Scan(&allocID).Error)
		require.NotEmpty(t, allocID)
		status, _, _ = other.call(t, http.MethodPatch, "/api/v1/allocations/"+allocID, map[string]any{"allocation_percent": 5}, otherAdmin)
		require.Equal(t, http.StatusNotFound, status)
		status, _, _ = other.call(t, http.MethodDelete, "/api/v1/allocations/"+allocID, nil, otherAdmin)
		require.Equal(t, http.StatusNotFound, status)

		var still int64
		require.NoError(t, e.db.Raw("SELECT count(*) FROM project_allocations WHERE id = ? AND deleted_at IS NULL", allocID).Scan(&still).Error)
		require.EqualValues(t, 1, still, "foreign delete must not have touched the row")
	})
}
