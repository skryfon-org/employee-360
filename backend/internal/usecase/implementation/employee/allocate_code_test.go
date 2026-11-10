package employee

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func str(s string) *string { return &s }

func TestAllocateEmployeeCode_DefaultsThenSequential(t *testing.T) {
	w := newWorld()
	a, b := w.tenant("a"), w.tenant("b")
	for _, want := range []string{"EMP0001", "EMP0002"} {
		got, err := allocateEmployeeCode(bg, (*fSettings)(w), (*fProfiles)(w), a)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	got, err := allocateEmployeeCode(bg, (*fSettings)(w), (*fProfiles)(w), b)
	require.NoError(t, err)
	assert.Equal(t, "EMP0001", got, "tenant B counter is independent")
	assert.EqualValues(t, 3, w.settings[a].EmployeeCodeNextNumber)
	assert.EqualValues(t, 2, w.settings[b].EmployeeCodeNextNumber)
}

func TestAllocateEmployeeCode_ConfiguredFormatAndOverflow(t *testing.T) {
	w := newWorld()
	a := w.tenant("a")
	s := entity.DefaultOrganizationSettings(a)
	s.EmployeeCodePrefix, s.EmployeeCodePadding, s.EmployeeCodeNextNumber = "ACME", 4, 9999
	w.settings[a] = s
	for _, want := range []string{"ACME9999", "ACME10000"} {
		got, err := allocateEmployeeCode(bg, (*fSettings)(w), (*fProfiles)(w), a)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestAllocateEmployeeCode_SkipsManuallyTakenCodesInSameTenantOnly(t *testing.T) {
	w := newWorld()
	a, b := w.tenant("a"), w.tenant("b")
	ua, ub := w.user(a, "a@x.test", true, "employee"), w.user(b, "b@x.test", true, "employee")
	w.profiles[ua.ID] = &entity.EmployeeProfile{TenantID: a, UserID: ua.ID, EmployeeCode: str("EMP0001")}
	w.profiles[ub.ID] = &entity.EmployeeProfile{TenantID: b, UserID: ub.ID, EmployeeCode: str("EMP0001")}

	got, err := allocateEmployeeCode(bg, (*fSettings)(w), (*fProfiles)(w), a)
	require.NoError(t, err)
	assert.Equal(t, "EMP0002", got)
	assert.EqualValues(t, 3, w.settings[a].EmployeeCodeNextNumber, "the skipped number is consumed")
}

type errSettings struct {
	fSettings
	err error
}

func (e *errSettings) NextEmployeeCodeNumber(context.Context, uuid.UUID) (entity.EmployeeCodeCounter, error) {
	return entity.EmployeeCodeCounter{}, e.err
}

func TestAllocateEmployeeCode_ExhaustionAndErrors(t *testing.T) {
	w := newWorld()
	a := w.tenant("a")
	// Occupy every candidate the allocator will try.
	for i := int64(1); i <= maxCodeAllocationAttempts; i++ {
		id := uuid.New()
		w.profiles[id] = &entity.EmployeeProfile{TenantID: a, UserID: id, EmployeeCode: str(entity.FormatEmployeeCode("EMP", 4, i))}
	}
	_, err := allocateEmployeeCode(bg, (*fSettings)(w), (*fProfiles)(w), a)
	assert.ErrorIs(t, err, ErrEmployeeCodeExhausted)

	boom := errors.New("db down")
	_, err = allocateEmployeeCode(bg, &errSettings{err: boom}, (*fProfiles)(w), a)
	assert.ErrorIs(t, err, boom)
}

func TestSaveProfile_AssignsCodeOnlyWhenNewAndUnset(t *testing.T) {
	w := newWorld()
	a := w.tenant("a")
	u1, u2 := uuid.New(), uuid.New()

	p1 := &entity.EmployeeProfile{TenantID: a, UserID: u1}
	require.NoError(t, saveProfile(bg, (*fProfiles)(w), (*fSettings)(w), a, u1, p1, true))
	assert.Equal(t, "EMP0001", *p1.EmployeeCode)

	manual := &entity.EmployeeProfile{TenantID: a, UserID: u2, EmployeeCode: str("CUSTOM1")}
	require.NoError(t, saveProfile(bg, (*fProfiles)(w), (*fSettings)(w), a, u2, manual, true))
	assert.Equal(t, "CUSTOM1", *manual.EmployeeCode)
	assert.EqualValues(t, 2, w.settings[a].EmployeeCodeNextNumber, "manual code does not consume the counter")

	require.NoError(t, saveProfile(bg, (*fProfiles)(w), (*fSettings)(w), a, u1, p1, false))
	assert.EqualValues(t, 2, w.settings[a].EmployeeCodeNextNumber, "updates never allocate")
}

func TestCreateEmergencyContact_NewProfileGetsGeneratedCode(t *testing.T) {
	f := newFixture()
	_, err := f.w.newCreateEmergencyContact().Execute(bg, f.adminActor(), f.employee.ID, contact("A", false))
	require.NoError(t, err)
	p := f.w.profiles[f.employee.ID]
	require.NotNil(t, p)
	require.NotNil(t, p.EmployeeCode)
	assert.Equal(t, "EMP0001", *p.EmployeeCode)
}
