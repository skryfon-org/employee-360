package statuslog

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// IsWeeklyOffDay reports whether date's weekday is one of offDays (0 = Sunday .. 6 = Saturday).
func IsWeeklyOffDay(offDays []int, date time.Time) bool {
	for _, d := range offDays {
		if time.Weekday(d) == date.Weekday() {
			return true
		}
	}
	return false
}

// ClassifyDay is the single day-status precedence rule shared by the admin daily
// summary and the employee day-statuses view: logged, holiday, weekly_off, on_leave
// (approved full-day leave), missing. Leave on a holiday or weekly off day is not
// on_leave, matching entity.CountLeaveDays. An approved half day with nothing logged
// is still missing (the other half is a working half) but reports halfDay=true; a
// half day that was logged is logged. halfDay is only ever true for on_leave and
// for that missing case.
func ClassifyDay(logged, isHoliday, isOff, onLeave, leaveIsHalf bool) (status string, halfDay bool) {
	switch {
	case logged:
		return entity.DayStatusLogged, false
	case isHoliday:
		return entity.DayStatusHoliday, false
	case isOff:
		return entity.DayStatusWeeklyOff, false
	case onLeave && leaveIsHalf:
		return entity.DayStatusMissing, true
	case onLeave:
		return entity.DayStatusOnLeave, false
	default:
		return entity.DayStatusMissing, false
	}
}

// IsHalfDayLeave reports whether a leave request's day part makes date a half day
// (a half day on a weekly off day counts 0, so it is not a half day).
func IsHalfDayLeave(date time.Time, dayPart string, offDays []int) bool {
	return entity.CountLeaveDays(date, date, dayPart, offDays, nil) == 0.5
}

// dayLeave is the approved leave covering one employee on one date.
type dayLeave struct {
	// half is true when every covering request is a half day.
	half bool
	// typeName is the leave type of the first covering request.
	typeName string
}

// leaveCoverage maps each employee to the approved leave covering date. reqs must
// already be the approved requests overlapping date (the repository filters tenant,
// status and range); the per-request range check only matters for ranges wider
// than one day.
func leaveCoverage(date time.Time, offDays []int, reqs []*entity.LeaveRequestDetail) map[uuid.UUID]dayLeave {
	out := map[uuid.UUID]dayLeave{}
	for _, r := range reqs {
		if date.Before(DateOnly(r.StartDate)) || date.After(DateOnly(r.EndDate)) {
			continue
		}
		half := IsHalfDayLeave(date, r.DayPart, offDays)
		if prev, ok := out[r.UserID]; ok {
			prev.half = prev.half && half
			out[r.UserID] = prev
			continue
		}
		out[r.UserID] = dayLeave{half: half, typeName: r.LeaveTypeName}
	}
	return out
}

// inclusiveDayCount is the number of calendar days in [from, to]. It works on the
// calendar dates, so it is exact regardless of the times' locations or DST.
func inclusiveDayCount(from, to time.Time) int {
	f := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	t := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(t.Sub(f).Hours()/24) + 1
}
