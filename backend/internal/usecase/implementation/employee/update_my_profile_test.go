package employee

import (
	"encoding/json"
	"strings"
	"testing"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMyProfile_MissingFieldsAndValidationAndNoEmploymentFields(t *testing.T) {
	f := newFixture()
	upd := f.w.newUpdateMyProfile()

	var in emptypes.UpdateMyProfileInput
	require.NoError(t, json.Unmarshal([]byte(`{"phone":"+91 98765 43210"}`), &in))
	m, err := upd.Execute(bg, f.tenant, f.employee.ID, in)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"date_of_birth", "gender", "address_line1", "city", "country", "emergency_contact"}, m.MissingFields)

	require.NoError(t, json.Unmarshal([]byte(`{"gender":"bogus"}`), &in))
	_, err = upd.Execute(bg, f.tenant, f.employee.ID, in)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData)

	// Employment fields are not part of the self-service input type, so a JSON
	// body carrying them cannot set them.
	in = emptypes.UpdateMyProfileInput{}
	require.NoError(t, json.Unmarshal([]byte(`{"city":"Kochi","employee_code":"HACK","employment_status":"terminated"}`), &in))
	_, err = upd.Execute(bg, f.tenant, f.employee.ID, in)
	require.NoError(t, err)
	require.NotNil(t, f.w.profiles[f.employee.ID].EmployeeCode)
	assert.Equal(t, "EMP0001", *f.w.profiles[f.employee.ID].EmployeeCode, "generated, never the client-supplied value")
	assert.Equal(t, "active", f.w.profiles[f.employee.ID].EmploymentStatus)
}

func TestMyProfile_ReturnsDepartmentAndPositionNames_TenantScoped(t *testing.T) {
	f := newFixture()
	get := f.w.newGetMyProfile()

	m, err := get.Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.Nil(t, m.DepartmentID)
	assert.Empty(t, m.DepartmentName)

	d := f.w.dept(f.tenant, true)
	p := f.w.position(f.tenant, true)
	f.employee.DepartmentID, f.employee.PositionID = &d, &p
	m, err = get.Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.Equal(t, f.w.depts[d].Name, m.DepartmentName)
	assert.Equal(t, f.w.positions[p].Name, m.PositionName)
	assert.Equal(t, &d, m.DepartmentID)

	// Same via PATCH.
	var in emptypes.UpdateMyProfileInput
	require.NoError(t, json.Unmarshal([]byte(`{"city":"Kochi"}`), &in))
	m, err = f.w.newUpdateMyProfile().Execute(bg, f.tenant, f.employee.ID, in)
	require.NoError(t, err)
	assert.Equal(t, f.w.positions[p].Name, m.PositionName)

	// A department id belonging to another tenant is never resolved.
	other := f.w.tenant("other")
	od := f.w.dept(other, true)
	f.employee.DepartmentID = &od
	m, err = get.Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.Nil(t, m.DepartmentID)
	assert.Empty(t, m.DepartmentName)
}

func TestUpdateMyProfile_NamesUpdatedTrimmedAndValidated(t *testing.T) {
	f := newFixture()
	upd := f.w.newUpdateMyProfile()

	var in emptypes.UpdateMyProfileInput
	require.NoError(t, json.Unmarshal([]byte(`{"first_name":"  Asha  ","last_name":"Nair"}`), &in))
	m, err := upd.Execute(bg, f.tenant, f.employee.ID, in)
	require.NoError(t, err)
	assert.Equal(t, "Asha", m.FirstName)
	assert.Equal(t, "Nair", m.LastName)

	// Absent names are left unchanged.
	in = emptypes.UpdateMyProfileInput{}
	require.NoError(t, json.Unmarshal([]byte(`{"city":"Kochi"}`), &in))
	m, err = upd.Execute(bg, f.tenant, f.employee.ID, in)
	require.NoError(t, err)
	assert.Equal(t, "Asha", m.FirstName)

	for _, body := range []string{`{"first_name":"   "}`, `{"last_name":null}`, `{"first_name":"` + strings.Repeat("a", 101) + `"}`} {
		in = emptypes.UpdateMyProfileInput{}
		require.NoError(t, json.Unmarshal([]byte(body), &in))
		_, err = upd.Execute(bg, f.tenant, f.employee.ID, in)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData, body)
	}
	m, err = f.w.newGetMyProfile().Execute(bg, f.tenant, f.employee.ID)
	require.NoError(t, err)
	assert.Equal(t, "Asha", m.FirstName)
}
