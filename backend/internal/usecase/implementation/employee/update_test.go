package employee

import (
	"testing"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate_UpsertsProfileAndValidatesReferences(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	dept := f.w.dept(f.tenant, true)
	pos := f.w.position(f.tenant, true)

	d, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"first_name":"Ann","employee_code":"EMP001","employment_type":"full_time","joining_date":"2024-01-15","department_id":"`+dept.String()+`","position_id":"`+pos.String()+`"}`))
	require.NoError(t, err)
	assert.Equal(t, "Ann", d.Summary.FirstName)
	require.NotNil(t, d.Profile, "profile is created on first update")
	assert.Equal(t, "EMP001", *d.Profile.EmployeeCode)
	assert.Equal(t, dept, *d.Summary.DepartmentID)
	assert.Contains(t, f.w.auditActions(), "employee.update")

	// Unchanged update writes nothing new.
	n := len(f.w.audit.Logs)
	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"first_name":"Ann"}`))
	require.NoError(t, err)
	assert.Len(t, f.w.audit.Logs, n, "no audit entry when nothing changed")
}

func TestUpdate_RejectsForeignOrInactiveReferences(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	other := f.w.tenant("Other")
	foreignDept := f.w.dept(other, true)
	foreignPos := f.w.position(other, true)
	foreignMgr := f.w.user(other, "m@other.test", true, entity.RoleEmployee)
	inactiveDept := f.w.dept(f.tenant, false)

	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"department_id":"`+foreignDept.String()+`"}`))
	assert.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"position_id":"`+foreignPos.String()+`"}`))
	assert.ErrorIs(t, err, domainerrors.ErrPositionNotFound)
	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"manager_id":"`+foreignMgr.ID.String()+`"}`))
	assert.ErrorIs(t, err, domainerrors.ErrManagerInvalid)
	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"department_id":"`+inactiveDept.String()+`"}`))
	assert.ErrorIs(t, err, domainerrors.ErrDepartmentInactive)
	_, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"manager_id":"`+f.employee.ID.String()+`"}`))
	assert.ErrorIs(t, err, domainerrors.ErrManagerInvalid, "cannot manage oneself")
}

func TestUpdate_Manager(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	inactive := f.w.user(f.tenant, "gone@acme.test", false, entity.RoleEmployee)

	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"manager_id":"`+inactive.ID.String()+`"}`))
	assert.ErrorIs(t, err, domainerrors.ErrManagerInvalid, "inactive manager")

	d, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"manager_id":"`+f.admin.ID.String()+`"}`))
	require.NoError(t, err)
	require.NotNil(t, d.Profile.ManagerID)
	assert.Equal(t, f.admin.ID, *d.Profile.ManagerID)

	d, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"manager_id":null}`))
	require.NoError(t, err)
	assert.Nil(t, d.Profile.ManagerID, "null clears the manager")
}

func TestUpdate_CrossTenantTargetIs404(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	other := f.w.tenant("Other")
	otherAdmin := f.w.user(other, "a@other.test", true, entity.RoleAdmin)

	_, err := uc.Execute(bg, f.w.actor(otherAdmin, entity.RoleAdmin), f.employee.ID, parse(t, `{"first_name":"Hacked"}`))
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	assert.Equal(t, "F", f.w.users[f.employee.ID].FirstName)
}

func TestUpdate_AdminCannotEditSuperAdmin(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	_, err := uc.Execute(bg, f.adminActor(), f.super.ID, parse(t, `{"first_name":"X"}`))
	assert.ErrorIs(t, err, domainerrors.ErrCannotActOnSuperAdmin)
	_, err = uc.Execute(bg, f.superActor(), f.super.ID, parse(t, `{"first_name":"X"}`))
	assert.NoError(t, err)
}

func TestUpdate_Validation(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	cases := map[string]string{
		"bad gender":      `{"gender":"robot"}`,
		"bad type":        `{"employment_type":"slave"}`,
		"bad status":      `{"employment_status":"vacation"}`,
		"null status":     `{"employment_status":null}`,
		"exit<joining":    `{"joining_date":"2024-05-01","exit_date":"2024-01-01"}`,
		"bad code":        `{"employee_code":"a b"}`,
		"blank first":     `{"first_name":"  "}`,
		"bad phone":       `{"phone":"abc"}`,
		"bad email":       `{"personal_email":"nope"}`,
		"future dob":      `{"date_of_birth":"2999-01-01"}`,
		"confirm<joining": `{"joining_date":"2024-05-01","confirmation_date":"2024-01-01"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, body))
			assert.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData)
		})
	}
}

func TestUpdate_EmployeeCodeUniquePerTenant(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	e2 := f.w.user(f.tenant, "e2@acme.test", true, entity.RoleEmployee)
	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"employee_code":"EMP1"}`))
	require.NoError(t, err)
	_, err = uc.Execute(bg, f.adminActor(), e2.ID, parse(t, `{"employee_code":"EMP1"}`))
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeCodeTaken)

	// Same code in another tenant is fine.
	other := f.w.tenant("Other")
	oa := f.w.user(other, "oa@other.test", true, entity.RoleAdmin)
	oe := f.w.user(other, "oe@other.test", true, entity.RoleEmployee)
	_, err = uc.Execute(bg, f.w.actor(oa, entity.RoleAdmin), oe.ID, parse(t, `{"employee_code":"EMP1"}`))
	assert.NoError(t, err)
}

func TestUpdate_NullClearsOptionalField(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"work_location":"Kochi"}`))
	require.NoError(t, err)
	d, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"work_location":null}`))
	require.NoError(t, err)
	assert.Nil(t, d.Profile.WorkLocation)
}

func TestAdminEditCanCompleteProfile(t *testing.T) {
	f := newFixture()
	_, err := f.w.newCreateEmergencyContact().Execute(bg, f.adminActor(), f.employee.ID, contact("Spouse", true))
	require.NoError(t, err)
	_, err = f.w.newUpdateEmployee().Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"phone":"+91 98765 43210","date_of_birth":"1990-04-02","gender":"male","address_line1":"x","city":"c","country":"IN"}`))
	require.NoError(t, err)
	ok, _ := f.w.newProfileGate().Execute(bg, f.tenant, f.employee.ID)
	assert.True(t, ok)
}

func TestUpdate_ManualEmployeeCodeMustMatchTenantPrefix(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	e2 := f.w.user(f.tenant, "e3@acme.test", true, entity.RoleEmployee)

	// Default prefix EMP applies when the tenant has no settings row.
	for _, code := range []string{"XYZ0001", "emp0001", "EMP", "EMP-0001", "EMP00A1"} {
		_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"employee_code":"`+code+`"}`))
		require.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData, code)
		assert.Contains(t, err.Error(), `"EMP"`, "error names the required prefix")
	}
	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"employee_code":"EMP0042"}`))
	require.NoError(t, err)

	// A tenant with a custom prefix: only that prefix is accepted.
	custom := f.w.tenant("Custom")
	f.w.settings[custom] = &entity.OrganizationSettings{TenantID: custom, EmployeeCodePrefix: "SKRYFON", EmployeeCodePadding: 4, EmployeeCodeStartNumber: 40001, EmployeeCodeNextNumber: 40001}
	ca := f.w.user(custom, "ca@custom.test", true, entity.RoleAdmin)
	ce := f.w.user(custom, "ce@custom.test", true, entity.RoleEmployee)
	_, err = uc.Execute(bg, f.w.actor(ca, entity.RoleAdmin), ce.ID, parse(t, `{"employee_code":"EMP0042"}`))
	assert.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData)
	_, err = uc.Execute(bg, f.w.actor(ca, entity.RoleAdmin), ce.ID, parse(t, `{"employee_code":"SKRYFON40001"}`))
	assert.NoError(t, err)

	// Duplicate manual code gives a clear conflict, not a format error.
	_, err = uc.Execute(bg, f.adminActor(), e2.ID, parse(t, `{"employee_code":"EMP0042"}`))
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeCodeTaken)
}

func TestUpdate_PrefixChangeDoesNotRewriteExistingCodes(t *testing.T) {
	f := newFixture()
	uc := f.w.newUpdateEmployee()
	_, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"employee_code":"EMP0001"}`))
	require.NoError(t, err)

	f.w.settings[f.tenant] = &entity.OrganizationSettings{TenantID: f.tenant, EmployeeCodePrefix: "ACME", EmployeeCodePadding: 4, EmployeeCodeStartNumber: 1, EmployeeCodeNextNumber: 1}

	// Unrelated edits keep the legacy code and do not re-validate it.
	d, err := uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"work_location":"Kochi"}`))
	require.NoError(t, err)
	assert.Equal(t, "EMP0001", *d.Profile.EmployeeCode)
	// Re-sending the unchanged legacy code is also fine.
	d, err = uc.Execute(bg, f.adminActor(), f.employee.ID, parse(t, `{"employee_code":"EMP0001"}`))
	require.NoError(t, err)
	assert.Equal(t, "EMP0001", *d.Profile.EmployeeCode)
}
