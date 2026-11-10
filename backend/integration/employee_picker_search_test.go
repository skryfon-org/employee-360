//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// GET /api/v1/employees powers the "Add allocation" picker: one search box over
// name, employee code and email, active-only, tenant-scoped and paginated.
func TestEmployeePickerSearch(t *testing.T) {
	e := newAuthFlowEnv(t)
	other := newAuthFlowEnv(t)
	admin := e.login(t, authFlowPassword).AccessToken

	erin := seedEmployee(t, e, "erin", true, false)
	_ = seedEmployee(t, e, "ian", false, false)
	_ = seedEmployee(t, other, "erin", true, false) // same name, other tenant
	require.NoError(t, e.db.Exec("UPDATE users SET last_name = 'O_Brien%' WHERE id = ?", erin.id).Error)
	require.NoError(t, e.db.Exec(
		"INSERT INTO employee_profiles (id, tenant_id, user_id, employee_code, employment_status) VALUES (gen_random_uuid(), ?, ?, 'EMP-0042', 'active')",
		e.tenantID, erin.id).Error)

	type page struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			TotalItems int `json:"total_items"`
		} `json:"meta"`
	}
	get := func(q string) page {
		code, _, raw := e.call(t, http.MethodGet, "/api/v1/employees"+q, nil, admin)
		require.Equal(t, http.StatusOK, code, string(raw))
		var p page
		require.NoError(t, json.Unmarshal(raw, &p), string(raw))
		return p
	}

	for name, q := range map[string]string{
		"first last": "?search=ERIN%20o_brien", "code": "?search=emp-004",
		"email": "?search=erin@" + e.domain, "last name": "?search=o_brien%25",
	} {
		p := get(q)
		require.Equal(t, 1, p.Meta.TotalItems, name)
		row := p.Data[0]
		require.Equal(t, erin.id.String(), row["id"], name)
		require.Equal(t, "erin", row["first_name"])
		require.Equal(t, "O_Brien%", row["last_name"])
		require.Equal(t, "EMP-0042", row["employee_code"])
		require.Equal(t, erin.email, row["email"])
	}

	require.Equal(t, 1, get("?search=ian&status=inactive").Meta.TotalItems)
	require.Equal(t, 0, get("?search=ian&status=active").Meta.TotalItems, "active-only")
	require.Equal(t, 1, get("?search=%25").Meta.TotalItems, "bare % matches only the literal in Erin's surname")
	require.Equal(t, 0, get("?search=n%25o").Meta.TotalItems, "% is not a wildcard")
	require.Equal(t, 0, get("?search=er_n").Meta.TotalItems, "bare _ is literal")

	_, isSet := get("?search=erin").Data[0]["allocated_percent"]
	require.False(t, isSet, "allocated_percent omitted unless include=allocation")
	withAlloc := get("?search=erin&include=allocation").Data[0]
	require.EqualValues(t, 0, withAlloc["allocated_percent"], "no allocations -> 0")
	code, _, _ := e.call(t, http.MethodGet, "/api/v1/employees?include=bogus", nil, admin)
	require.Equal(t, http.StatusBadRequest, code)

	p := get("?status=active&page_size=1")
	require.Len(t, p.Data, 1)
	require.GreaterOrEqual(t, p.Meta.TotalItems, 2, "admin + erin; other tenant excluded")
	for _, row := range get("?search=erin").Data {
		require.Contains(t, row["email"], e.domain, "tenant isolation")
	}
}
