package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

func TestApply_StartDateWindow(t *testing.T) {
	e := newEnv()
	uc := e.applyUC()
	cases := []struct {
		name  string
		start string
		err   error
	}{
		{"today-1 accepted (client-local grace)", "2026-10-28", nil},
		{"today-2 rejected", "2026-10-27", domainerrors.ErrLeaveStartTooOld},
		{"today-6 rejected", "2026-10-23", domainerrors.ErrLeaveStartTooOld},
		{"today accepted", "2026-10-29", nil},
		{"far future accepted", "2030-01-01", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e.requests.rows = map[uuid.UUID]*entity.LeaveRequest{}
			_, err := uc.Execute(bgctx, tenant, e.employee, e.in(tc.start, tc.start))
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestApply_DateOrderAndDayPart(t *testing.T) {
	e := newEnv()
	uc := e.applyUC()

	_, err := uc.Execute(bgctx, tenant, e.employee, e.in("2026-11-05", "2026-11-04"))
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)

	in := e.in("2026-11-05", "2026-11-06")
	in.DayPart = entity.LeaveDayPartFirstHalf
	_, err = uc.Execute(bgctx, tenant, e.employee, in)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData, "half day over multiple days")

	in = e.in("2026-11-05", "2026-11-05")
	in.DayPart = "quarter"
	_, err = uc.Execute(bgctx, tenant, e.employee, in)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)

	in = e.in("2026-11-05", "2026-11-05")
	in.DayPart = entity.LeaveDayPartSecondHalf
	d, err := uc.Execute(bgctx, tenant, e.employee, in)
	require.NoError(t, err)
	assert.Equal(t, entity.LeaveDayPartSecondHalf, d.DayPart)
	assert.Equal(t, entity.LeaveStatusPending, d.Status)
	assert.Contains(t, e.audit.actions, "leave_request.apply")

	in = e.in("2026-12-01", "2026-12-03")
	d, err = uc.Execute(bgctx, tenant, e.employee, in)
	require.NoError(t, err)
	assert.Equal(t, entity.LeaveDayPartFull, d.DayPart, "day part defaults to full")
}

func TestApply_Overlap(t *testing.T) {
	e := newEnv()
	d := e.applyOK(t, e.in("2026-11-10", "2026-11-12"))

	for _, r := range [][2]string{{"2026-11-12", "2026-11-14"}, {"2026-11-09", "2026-11-10"}, {"2026-11-11", "2026-11-11"}} {
		_, err := e.applyUC().Execute(bgctx, tenant, e.employee, e.in(r[0], r[1]))
		assert.ErrorIs(t, err, domainerrors.ErrLeaveOverlap, r)
	}
	// Adjacent ranges do not overlap.
	e.applyOK(t, e.in("2026-11-13", "2026-11-13"))

	// Cancelled requests free the dates.
	e.requests.rows[d.ID].Status = entity.LeaveStatusCancelled
	e.applyOK(t, e.in("2026-11-10", "2026-11-10"))

	// Another user's overlapping dates are fine.
	_, err := e.applyUC().Execute(bgctx, tenant, e.other, leavetypes.ApplyLeaveInput{
		LeaveTypeID: e.typ.ID, StartDate: day("2026-11-10"), EndDate: day("2026-11-10"), ApproverID: e.admin,
	})
	require.NoError(t, err)
}

func TestApply_OverlapHalfDays(t *testing.T) {
	e := newEnv()
	first := e.in("2026-11-20", "2026-11-20")
	first.DayPart = entity.LeaveDayPartFirstHalf
	e.applyOK(t, first)

	second := first
	second.DayPart = entity.LeaveDayPartSecondHalf
	e.applyOK(t, second) // opposite half is allowed

	_, err := e.applyUC().Execute(bgctx, tenant, e.employee, first)
	assert.ErrorIs(t, err, domainerrors.ErrLeaveOverlap, "same half twice")

	full := e.in("2026-11-20", "2026-11-20")
	_, err = e.applyUC().Execute(bgctx, tenant, e.employee, full)
	assert.ErrorIs(t, err, domainerrors.ErrLeaveOverlap, "full day over a half")
}

func TestApply_ApproverValidation(t *testing.T) {
	e := newEnv()
	uc := e.applyUC()

	in := e.in("2026-11-05", "2026-11-05")
	in.ApproverID = e.mgr
	e.applyOK(t, in) // manager

	in = e.in("2026-11-06", "2026-11-06")
	in.ApproverID = e.admin
	e.applyOK(t, in) // tenant admin

	in = e.in("2026-11-07", "2026-11-07")
	in.ApproverID = e.employee
	_, err := uc.Execute(bgctx, tenant, e.employee, in)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveApprover, "self")

	in.ApproverID = e.other // plain colleague: neither manager nor admin
	_, err = uc.Execute(bgctx, tenant, e.employee, in)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveApprover)

	e.approvers.active[e.mgr] = false // inactive manager is not offered
	in.ApproverID = e.mgr
	_, err = uc.Execute(bgctx, tenant, e.employee, in)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveApprover)

	// An admin applicant cannot pick themselves.
	_, err = uc.Execute(bgctx, tenant, e.admin, leavetypes.ApplyLeaveInput{
		LeaveTypeID: e.typ.ID, StartDate: day("2026-11-05"), EndDate: day("2026-11-05"), ApproverID: e.admin,
	})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveApprover)
}

func TestApply_LeaveTypeRules(t *testing.T) {
	e := newEnv()
	uc := e.applyUC()

	inactive := e.types.add(tenant, "OLD", false)
	in := e.in("2026-11-05", "2026-11-05")
	in.LeaveTypeID = inactive.ID
	_, err := uc.Execute(bgctx, tenant, e.employee, in)
	assert.ErrorIs(t, err, domainerrors.ErrLeaveTypeInactive)

	foreign := e.types.add(uuid.New(), "X", true) // another tenant's type
	in.LeaveTypeID = foreign.ID
	_, err = uc.Execute(bgctx, tenant, e.employee, in)
	assert.ErrorIs(t, err, domainerrors.ErrLeaveTypeNotFound)
}

func TestApply_WorkingDays(t *testing.T) {
	cases := []struct {
		name       string
		off        []int
		start, end string
		part       string
		wantDays   float64
		wantErr    bool
	}{
		{"Thu..Mon excludes Sat/Sun", []int{0, 6}, "2026-11-26", "2026-11-30", "", 3, false},
		{"starts on off day is allowed", []int{0, 6}, "2026-11-28", "2026-11-30", "", 1, false},
		{"single off day rejected", []int{0, 6}, "2026-11-28", "2026-11-28", "", 0, true},
		{"range entirely off days rejected", []int{0, 6}, "2026-11-28", "2026-11-29", "", 0, true},
		{"half day on off day rejected", []int{0, 6}, "2026-11-29", "2026-11-29", entity.LeaveDayPartFirstHalf, 0, true},
		{"half day on working day", []int{0, 6}, "2026-11-30", "2026-11-30", entity.LeaveDayPartSecondHalf, 0.5, false},
		{"custom Fri/Sat off", []int{5, 6}, "2026-11-26", "2026-11-30", "", 3, false},
		{"custom: Sunday is working", []int{5, 6}, "2026-11-29", "2026-11-29", "", 1, false},
		{"custom: Friday rejected", []int{5}, "2026-11-27", "2026-11-27", "", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			e.off.days = tc.off
			in := e.in(tc.start, tc.end)
			in.DayPart = tc.part
			d, err := e.applyUC().Execute(bgctx, tenant, e.employee, in)
			if tc.wantErr {
				assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)
				assert.Contains(t, err.Error(), "weekly off days or public holidays")
				assert.Empty(t, e.requests.rows)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantDays, d.Days)
		})
	}
}

func TestApply_Holidays(t *testing.T) {
	cases := []struct {
		name       string
		holidays   []string
		optional   []string
		otherTen   []string
		start, end string
		part       string
		wantDays   float64
		wantErr    bool
	}{
		{"holiday mid-range excluded", []string{"2026-11-04"}, nil, nil, "2026-11-03", "2026-11-05", "", 2, false},
		{"only holiday plus weekend rejected", []string{"2026-11-27"}, nil, nil, "2026-11-27", "2026-11-29", "", 0, true},
		{"half day on holiday rejected", []string{"2026-11-03"}, nil, nil, "2026-11-03", "2026-11-03", entity.LeaveDayPartFirstHalf, 0, true},
		{"other tenant holiday ignored", nil, nil, []string{"2026-11-04"}, "2026-11-03", "2026-11-05", "", 3, false},
		{"optional holiday mid-range counted as working day", nil, []string{"2026-11-04"}, nil, "2026-11-03", "2026-11-05", "", 3, false},
		{"range of only an optional holiday allowed", nil, []string{"2026-11-04"}, nil, "2026-11-04", "2026-11-04", "", 1, false},
		{"half day on optional holiday allowed", nil, []string{"2026-11-04"}, nil, "2026-11-04", "2026-11-04", entity.LeaveDayPartFirstHalf, 0.5, false},
		{"optional holiday plus weekend: only optional counts", nil, []string{"2026-11-27"}, nil, "2026-11-27", "2026-11-29", "", 1, false},
		{"public holiday plus weekend rejected despite optional elsewhere", []string{"2026-11-27"}, []string{"2026-11-30"}, nil, "2026-11-27", "2026-11-29", "", 0, true},
		{"holiday on range edge excluded", []string{"2026-11-05"}, nil, nil, "2026-11-03", "2026-11-05", "", 2, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			for _, h := range tc.holidays {
				e.hol.add(tenant, h)
			}
			for _, h := range tc.optional {
				e.hol.addOptional(tenant, h)
			}
			for _, h := range tc.otherTen {
				e.hol.add(uuid.New(), h)
			}
			in := e.in(tc.start, tc.end)
			in.DayPart = tc.part
			d, err := e.applyUC().Execute(bgctx, tenant, e.employee, in)
			if tc.wantErr {
				assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)
				assert.Contains(t, err.Error(), "weekly off days or public holidays")
				assert.Empty(t, e.requests.rows)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantDays, d.Days)
		})
	}
}
