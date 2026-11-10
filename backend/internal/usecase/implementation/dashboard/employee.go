package dashboard

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	dashtypes "github.com/skryfon-org/employee-360/backend/internal/types/dashboard"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/statuslog"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	dashusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/dashboard"
)

const (
	dateLayout = "2006-01-02"
	// listCap bounds the single-page repository reads used for aggregation.
	listCap = 200
)

// EmployeeDeps are the repositories the employee dashboard reads. All are
// queried with the explicit tenantID passed to Execute.
type EmployeeDeps struct {
	Users       repository.UserRepository
	Allocations repository.ProjectAllocationRepository
	StatusLogs  repository.StatusLogRepository
	Leave       repository.LeaveRequestRepository
	Holidays    repository.HolidayRepository
	Categories  repository.HolidayCategoryRepository
	OrgSettings repository.OrganizationSettingsRepository
}

// EmployeeDashboardUseCaseImpl implements dashusecase.EmployeeDashboardUseCase.
type EmployeeDashboardUseCaseImpl struct {
	d   EmployeeDeps
	now func() time.Time
}

var _ dashusecase.EmployeeDashboardUseCase = (*EmployeeDashboardUseCaseImpl)(nil)

// NewEmployeeDashboardUseCase constructs an EmployeeDashboardUseCaseImpl. A nil clock uses time.Now.
func NewEmployeeDashboardUseCase(d EmployeeDeps, clock func() time.Time) *EmployeeDashboardUseCaseImpl {
	if clock == nil {
		clock = time.Now
	}
	return &EmployeeDashboardUseCaseImpl{d: d, now: clock}
}

func day(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Execute aggregates the caller's own data within their tenant.
func (u *EmployeeDashboardUseCaseImpl) Execute(c context.Context, tenantID, userID uuid.UUID) (*dashtypes.EmployeeDashboardResponse, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	today := day(u.now())
	weekStart := today.AddDate(0, 0, -(dashtypes.EmployeeWeekDays - 1))

	user, err := u.d.Users.GetByIDWithRoles(c, tenantID, userID)
	if err != nil {
		return nil, err
	}

	settings, err := u.d.OrgSettings.GetByTenantID(c, tenantID)
	switch {
	case err == nil:
	case errors.Is(err, domainerrors.ErrOrganizationSettingsNotFound):
		settings = entity.DefaultOrganizationSettings(tenantID)
	default:
		return nil, err
	}
	welcome := ""
	if settings.WelcomeMessage != nil {
		welcome = *settings.WelcomeMessage
	}
	off := map[int]bool{}
	for _, d := range settings.WeeklyOffDays {
		off[d] = true
	}

	holidays, err := u.holidaysFrom(c, tenantID, weekStart, today)
	if err != nil {
		return nil, err
	}
	// Only public holidays (category is_public) are days off; optional holidays are working days.
	holidayOn := map[string]*entity.Holiday{}
	holidayDates := make([]time.Time, 0, len(holidayOn))
	for _, h := range ucshared.FilterPublicHolidays(holidays) {
		holidayOn[day(h.Date).Format(dateLayout)] = h
		holidayDates = append(holidayDates, h.Date)
	}

	stats := dashtypes.EmployeeStats{}
	resp := &dashtypes.EmployeeDashboardResponse{}

	// Leave.
	_, pendingTotal, err := u.d.Leave.ListByUser(c, tenantID, userID, repository.LeaveRequestFilter{Status: entity.LeaveStatusPending}, 1, 0)
	if err != nil {
		return nil, err
	}
	stats.PendingLeave = int(pendingTotal)
	approved, _, err := u.d.Leave.ListByUser(c, tenantID, userID, repository.LeaveRequestFilter{Status: entity.LeaveStatusApproved, From: &weekStart, OldestFirst: true}, listCap, 0)
	if err != nil {
		return nil, err
	}
	// leaveParts records, per date, the approved day parts (full, first_half, second_half).
	leaveParts := map[string]map[string]bool{}
	for _, r := range approved {
		part := r.DayPart
		if part == "" {
			part = entity.LeaveDayPartFull
		}
		for d := day(r.StartDate); !d.After(day(r.EndDate)); d = d.AddDate(0, 0, 1) {
			k := d.Format(dateLayout)
			if leaveParts[k] == nil {
				leaveParts[k] = map[string]bool{}
			}
			leaveParts[k][part] = true
		}
		if !day(r.EndDate).Before(today) {
			stats.UpcomingLeaveDays += remainingLeaveDays(r, today, settings.WeeklyOffDays, holidayDates)
			if resp.Leave.NextApproved == nil {
				it := leaveItem(r)
				resp.Leave.NextApproved = &it
			}
		}
	}
	recent, _, err := u.d.Leave.ListByUser(c, tenantID, userID, repository.LeaveRequestFilter{}, dashtypes.EmployeeRecentLeaveLimit, 0)
	if err != nil {
		return nil, err
	}
	resp.Leave.Recent = make([]dashtypes.EmployeeLeaveItem, 0, len(recent))
	for _, r := range recent {
		resp.Leave.Recent = append(resp.Leave.Recent, leaveItem(r))
	}

	// Pending team approvals: only for users who are (or have been) approvers, or admins.
	pa, err := u.pendingApprovals(c, tenantID, userID, user)
	if err != nil {
		return nil, err
	}
	stats.PendingApprovals = pa

	// Projects and allocation.
	allocs, err := u.d.Allocations.ListByUserScope(c, tenantID, userID, entity.AllocationScopeActive, today)
	if err != nil {
		return nil, err
	}
	resp.Projects = []dashtypes.EmployeeProject{}
	for _, a := range allocs {
		if !entity.IsLiveProjectStatus(a.ProjectStatus) || !a.ActiveOn(today) {
			continue
		}
		stats.AllocationPercent += a.AllocationPercent
		p := dashtypes.EmployeeProject{
			ProjectID: a.ProjectID.String(), ProjectName: a.ProjectName, ProjectCode: a.ProjectCode,
			RoleOnProject: a.RoleOnProject, AllocationPercent: a.AllocationPercent,
		}
		if a.EndDate != nil {
			s := day(*a.EndDate).Format(dateLayout)
			p.EndDate = &s
		}
		resp.Projects = append(resp.Projects, p)
	}

	// Hours for the 7-day window.
	logs, err := u.d.StatusLogs.ListByUserAndDateRange(c, tenantID, userID, weekStart, today, nil)
	if err != nil {
		return nil, err
	}
	hoursOn := map[string]float64{}
	for _, l := range logs {
		hoursOn[day(l.LogDate).Format(dateLayout)] += l.Hours
		stats.HoursThisWeek += l.Hours
	}
	resp.Week.Days = make([]dashtypes.EmployeeWeekDay, 0, dashtypes.EmployeeWeekDays)
	for d := weekStart; !d.After(today); d = d.AddDate(0, 0, 1) {
		key := d.Format(dateLayout)
		status := dayStatus(off, holidayOn, leaveParts, d, key)
		h := hoursOn[key]
		// Whether the day is a working day is independent of leave and of logging.
		working := !off[int(d.Weekday())] && holidayOn[key] == nil
		resp.Week.Days = append(resp.Week.Days, dashtypes.EmployeeWeekDay{Date: key, Hours: h, IsWorkingDay: working, Kind: kindOf(status)})
		// Only a working day with nothing logged and no full-day leave is missing; a half day still needs a log.
		if d.Before(today) && h == 0 && status == entity.DayStatusMissing {
			resp.Week.MissingDays++
		}
	}

	// Upcoming holidays with category display data.
	if resp.Holidays, err = u.upcomingHolidays(c, tenantID, holidays, today); err != nil {
		return nil, err
	}

	// Today's kind.
	todayKey := today.Format(dateLayout)
	t := dashtypes.EmployeeToday{Date: todayKey, Kind: kindOf(dayStatus(off, holidayOn, leaveParts, today, todayKey))}
	if h := holidayOn[todayKey]; h != nil {
		t.HolidayName = h.Name
	}
	resp.Greeting = dashtypes.EmployeeGreeting{FirstName: user.FirstName, WelcomeMessage: welcome, Today: t}
	resp.Stats = stats
	return resp, nil
}

// dayStatus classifies a calendar day with statuslog.ClassifyDay, the shared
// precedence rule, ignoring any logged hours (logged=false) so the result says what
// the calendar expects of the day: holiday, weekly_off, on_leave (full-day approved
// leave) or missing (a working day, including a half-day leave day).
func dayStatus(off map[int]bool, holidayOn map[string]*entity.Holiday, leaveParts map[string]map[string]bool, d time.Time, key string) string {
	full, partial := leaveOn(leaveParts[key])
	status, _ := statuslog.ClassifyDay(false, holidayOn[key] != nil, off[int(d.Weekday())], full || partial, partial)
	return status
}

// kindOf maps an expected-day status (see dayStatus) to the dashboard day kind.
func kindOf(status string) string {
	switch status {
	case entity.DayStatusHoliday:
		return dashtypes.DayKindHoliday
	case entity.DayStatusWeeklyOff:
		return dashtypes.DayKindWeeklyOff
	case entity.DayStatusOnLeave:
		return dashtypes.DayKindLeave
	}
	return dashtypes.DayKindWorking
}

// leaveOn reports whether approved parts cover the whole day (a full-day
// request, or both halves) or only part of it.
func leaveOn(parts map[string]bool) (full, half bool) {
	if parts[entity.LeaveDayPartFull] || (parts[entity.LeaveDayPartFirstHalf] && parts[entity.LeaveDayPartSecondHalf]) {
		return true, false
	}
	return false, len(parts) > 0
}

// holidaysFrom loads the tenant's holidays from the window start through the
// end of next year in one range query, in date order.
func (u *EmployeeDashboardUseCaseImpl) holidaysFrom(c context.Context, tenantID uuid.UUID, from, today time.Time) ([]*entity.Holiday, error) {
	to := time.Date(today.Year()+2, 1, 1, 0, 0, 0, 0, time.UTC) // exclusive
	return u.d.Holidays.ListByDateRange(c, tenantID, from, to)
}

func (u *EmployeeDashboardUseCaseImpl) upcomingHolidays(c context.Context, tenantID uuid.UUID, all []*entity.Holiday, today time.Time) ([]dashtypes.EmployeeHoliday, error) {
	out := []dashtypes.EmployeeHoliday{}
	var cats map[uuid.UUID]*entity.HolidayCategory
	for _, h := range all {
		d := day(h.Date)
		if d.Before(today) {
			continue
		}
		if len(out) == dashtypes.EmployeeHolidayLimit {
			break
		}
		if cats == nil {
			list, _, err := u.d.Categories.List(c, tenantID, nil, listCap, 0)
			if err != nil {
				return nil, err
			}
			cats = make(map[uuid.UUID]*entity.HolidayCategory, len(list))
			for _, cat := range list {
				cats[cat.ID] = cat
			}
		}
		item := dashtypes.EmployeeHoliday{
			ID: h.ID.String(), Name: h.Name, Date: d.Format(dateLayout),
			DaysUntil: int(d.Sub(today).Hours() / 24),
		}
		if cat := cats[h.CategoryID]; cat != nil {
			item.CategoryName, item.CategoryColor = cat.Name, cat.Color
		}
		out = append(out, item)
	}
	return out, nil
}

func (u *EmployeeDashboardUseCaseImpl) pendingApprovals(c context.Context, tenantID, userID uuid.UUID, user *entity.User) (*int, error) {
	isAdmin := false
	for _, r := range user.Roles {
		if r.Name == entity.RoleAdmin || r.Name == entity.RoleSuperAdmin {
			isAdmin = true
		}
	}
	pending, everAddressed, err := u.d.Leave.ApproverSummary(c, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && !everAddressed {
		return nil, nil
	}
	n := int(pending)
	return &n, nil
}

// remainingLeaveDays counts the request's days from max(start, today) through
// its end, excluding weekly off days and public holidays; a half day is 0.5 (0 once past).
func remainingLeaveDays(r *entity.LeaveRequestDetail, today time.Time, offDays []int, holidays []time.Time) float64 {
	start := day(r.StartDate)
	if start.Before(today) {
		start = today
	}
	end := day(r.EndDate)
	if end.Before(start) {
		return 0
	}
	return entity.CountLeaveDays(start, end, r.DayPart, offDays, holidays)
}

func leaveItem(r *entity.LeaveRequestDetail) dashtypes.EmployeeLeaveItem {
	return dashtypes.EmployeeLeaveItem{
		ID: r.ID.String(), LeaveTypeName: r.LeaveTypeName, LeaveTypeCode: r.LeaveTypeCode,
		StartDate: day(r.StartDate).Format(dateLayout), EndDate: day(r.EndDate).Format(dateLayout),
		DayPart: r.DayPart, Days: r.Days, Status: r.Status,
	}
}
