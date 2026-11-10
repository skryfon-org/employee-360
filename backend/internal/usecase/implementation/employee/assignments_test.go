package employee

import (
	"testing"
	"time"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignments_CloseOpenOnChange(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	list := f.w.newListAssignments()
	d1, d2 := f.w.dept(f.tenant, true), f.w.dept(f.tenant, true)
	p1 := f.w.position(f.tenant, true)

	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"joining_date":"2024-01-01","department_id":"`+d1.String()+`","position_id":"`+p1.String()+`"}`))
	require.NoError(t, err)
	hist, _ := list.Execute(bg, f.tenant, f.employee.ID)
	require.Len(t, hist, 1)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), hist[0].EffectiveFrom, "first stint starts at joining date")
	assert.Nil(t, hist[0].EffectiveTo)

	// Department change -> close + open.
	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"department_id":"`+d2.String()+`","effective_date":"2024-06-01","change_reason":"transfer"}`))
	require.NoError(t, err)
	hist, _ = list.Execute(bg, f.tenant, f.employee.ID)
	require.Len(t, hist, 2)
	assert.Equal(t, d2, *hist[0].DepartmentID, "newest first")
	assert.Nil(t, hist[0].EffectiveTo)
	assert.Equal(t, "transfer", *hist[0].ChangeReason)
	require.NotNil(t, hist[1].EffectiveTo)
	assert.Equal(t, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), *hist[1].EffectiveTo)
	assert.Equal(t, 1, f.w.openCount(f.tenant, f.employee.ID), "exactly one open row")
	assert.Equal(t, d2, *f.w.users[f.employee.ID].DepartmentID, "users row mirrors current")
	assert.Contains(t, f.w.auditActions(), "employee.assignment_change")

	// No change -> no new row.
	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"department_id":"`+d2.String()+`","work_location":"X"}`))
	require.NoError(t, err)
	hist, _ = list.Execute(bg, f.tenant, f.employee.ID)
	assert.Len(t, hist, 2)
}

func TestAssignments_DateOrderAndTenantIsolation(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	list := f.w.newListAssignments()
	d1, d2 := f.w.dept(f.tenant, true), f.w.dept(f.tenant, true)
	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"department_id":"`+d1.String()+`","effective_date":"2024-06-01"}`))
	require.NoError(t, err)

	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"department_id":"`+d2.String()+`","effective_date":"2024-01-01"}`))
	assert.ErrorIs(t, err, domainerrors.ErrAssignmentDateOrder)
	assert.Equal(t, 1, f.w.openCount(f.tenant, f.employee.ID))

	other := f.w.tenant("Other")
	_, err = list.Execute(bg, other, f.employee.ID)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound, "cross-tenant history is 404")
}

// ---- deactivate / activate ----
