package statuslog

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

// MonthLayout is the wire format for the month query parameter.
const MonthLayout = "2006-01"

type getMonthlyStatusLogsUseCase struct {
	repo  repository.StatusLogRepository
	off   repository.WeeklyOffDaysReader
	leave repository.LeaveRequestRepository
	hols  repository.HolidayRepository
	clock Clock
}

// NewGetMonthlyStatusLogsUseCase constructs GetMonthlyStatusLogsUseCase.
func NewGetMonthlyStatusLogsUseCase(repo repository.StatusLogRepository, off repository.WeeklyOffDaysReader, leave repository.LeaveRequestRepository, hols repository.HolidayRepository, clock Clock) statusloguc.GetMonthlyStatusLogsUseCase {
	if clock == nil {
		clock = RealClock
	}
	return &getMonthlyStatusLogsUseCase{repo: repo, off: off, leave: leave, hols: hols, clock: clock}
}

// WorkingDaysInMonth counts days in the month starting at first whose weekday
// (0 = Sunday .. 6 = Saturday) is not in offDays.
func WorkingDaysInMonth(first time.Time, offDays []int) int {
	off := make(map[time.Weekday]bool, len(offDays))
	for _, d := range offDays {
		off[time.Weekday(d)] = true
	}
	n := 0
	for d := first; d.Month() == first.Month(); d = d.AddDate(0, 0, 1) {
		if !off[d.Weekday()] {
			n++
		}
	}
	return n
}

func (uc *getMonthlyStatusLogsUseCase) Execute(ctx context.Context, tenantID uuid.UUID, monthStr, deptStr, projStr string) (*entity.MonthlyStatusLogReport, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}

	now := uc.clock().UTC()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if strings.TrimSpace(monthStr) != "" {
		t, err := time.ParseInLocation(MonthLayout, strings.TrimSpace(monthStr), time.UTC)
		if err != nil {
			return nil, domainerrors.ErrInvalidStatusLogDate
		}
		first = t
	}
	last := first.AddDate(0, 1, -1)

	var deptID *uuid.UUID
	if strings.TrimSpace(deptStr) != "" {
		d, err := uuid.Parse(strings.TrimSpace(deptStr))
		if err != nil {
			return nil, domainerrors.ErrDepartmentNotFound
		}
		deptID = &d
	}
	projID, err := ParseProjectFilter(projStr)
	if err != nil {
		return nil, err
	}

	offDays, err := ucshared.ResolveOffDays(ctx, uc.off, tenantID)
	if err != nil {
		return nil, err
	}

	rows, err := uc.repo.MonthlySummary(ctx, tenantID, first, last, offDays, deptID, projID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*entity.MonthlyEmployeeSummary{}
	}

	holidays, err := ucshared.PublicHolidayDates(ctx, uc.hols, tenantID, first, last)
	if err != nil {
		return nil, err
	}
	leaveByUser, err := uc.leaveByUser(ctx, tenantID, first, last, offDays, holidays, rows)
	if err != nil {
		return nil, err
	}
	workingDays := entity.CountLeaveDays(first, last, entity.LeaveDayPartFull, offDays, holidays)

	report := &entity.MonthlyStatusLogReport{
		Month:          first.Format(MonthLayout),
		WorkingDays:    int(workingDays),
		TotalEmployees: len(rows),
		Employees:      rows,
	}
	for _, r := range rows {
		leave := leaveByUser[r.UserID]
		for _, f := range leave {
			r.LeaveDays += f
		}
		r.ExpectedDays = math.Max(0, workingDays-r.LeaveDays)
		r.MissingDays = missingDays(first, last, offDays, holidays, r.LoggedDates, leave)
		if r.DaysLogged > 0 {
			report.LoggedEmployees++
		}
		if r.MissingDays > 0 {
			report.NotLoggedEmployees++
		}
		report.TotalHours += r.TotalHours
	}
	report.TotalHours = RoundHours(report.TotalHours)
	return report, nil
}

// missingDays counts the expected working days with nothing logged. Only days that
// are neither weekly off days nor public holidays count, a logged day contributes
// nothing wherever it falls (so work on an off day, holiday or leave day never hides
// a different missing day), and approved leave on an unlogged day reduces it by the
// leave fraction (0.5 for a half day).
func missingDays(first, last time.Time, offDays []int, holidays []time.Time, logged []time.Time, leave map[time.Time]float64) float64 {
	isLogged := make(map[time.Time]bool, len(logged))
	for _, d := range logged {
		isLogged[DateOnly(d)] = true
	}
	hol := make(map[time.Time]bool, len(holidays))
	for _, h := range holidays {
		hol[DateOnly(h)] = true
	}
	missing := 0.0
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if IsWeeklyOffDay(offDays, d) || hol[d] || isLogged[d] {
			continue
		}
		missing += 1 - leave[d]
	}
	return missing
}

// leaveByUser returns, per employee, the fraction of each working day of the month
// covered by approved leave (1 full day, 0.5 half day; weekly off days and public
// holidays excluded, capped at 1). One tenant-wide leave query serves all employees.
func (uc *getMonthlyStatusLogsUseCase) leaveByUser(ctx context.Context, tenantID uuid.UUID, first, last time.Time, offDays []int, holidays []time.Time, rows []*entity.MonthlyEmployeeSummary) (map[uuid.UUID]map[time.Time]float64, error) {
	out := map[uuid.UUID]map[time.Time]float64{}
	if len(rows) == 0 {
		return out, nil
	}
	// The repository scopes these to the tenant, approved only, overlapping [first, last].
	reqs, _, err := uc.leave.List(ctx, tenantID, repository.LeaveRequestFilter{Status: entity.LeaveStatusApproved, From: &first, To: &last}, ucshared.NoLimit, 0)
	if err != nil {
		return nil, err
	}
	for _, r := range reqs {
		s, e := DateOnly(r.StartDate), DateOnly(r.EndDate)
		if s.Before(first) {
			s = first
		}
		if e.After(last) {
			e = last
		}
		for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
			f := entity.CountLeaveDays(d, d, r.DayPart, offDays, holidays)
			if f == 0 {
				continue
			}
			days := out[r.UserID]
			if days == nil {
				days = map[time.Time]float64{}
				out[r.UserID] = days
			}
			days[d] = math.Min(1, days[d]+f)
		}
	}
	return out, nil
}
