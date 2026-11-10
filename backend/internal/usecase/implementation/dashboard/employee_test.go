package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

// Fixed "now": Friday 2026-10-30. The 7-day window is Sun 10-24 .. Fri 10-30.
var fixedNow = time.Date(2026, 10, 30, 15, 0, 0, 0, time.UTC)

func d(y int, m time.Month, dd int) time.Time { return time.Date(y, m, dd, 0, 0, 0, 0, time.UTC) }

type empUsers struct {
	repository.UserRepository
	tenants []uuid.UUID
	roles   []entity.Role
}

func (f *empUsers) GetByIDWithRoles(_ context.Context, t, id uuid.UUID) (*entity.User, error) {
	f.tenants = append(f.tenants, t)
	return &entity.User{ID: id, TenantID: t, FirstName: "Asha", Roles: f.roles}, nil
}

type empAllocs struct {
	repository.ProjectAllocationRepository
	tenants []uuid.UUID
	items   []*entity.AllocationWithProject
}

func (f *empAllocs) ListByUserScope(_ context.Context, t, _ uuid.UUID, _ entity.AllocationScope, _ time.Time) ([]*entity.AllocationWithProject, error) {
	f.tenants = append(f.tenants, t)
	return f.items, nil
}

type empLogs struct {
	repository.StatusLogRepository
	tenants []uuid.UUID
	items   []*entity.StatusLogEntryWithProject
}

func (f *empLogs) ListByUserAndDateRange(_ context.Context, t, _ uuid.UUID, _, _ time.Time, _ *uuid.UUID) ([]*entity.StatusLogEntryWithProject, error) {
	f.tenants = append(f.tenants, t)
	return f.items, nil
}

type empLeave struct {
	repository.LeaveRequestRepository
	tenants      []uuid.UUID
	pending      int64
	approved     []*entity.LeaveRequestDetail
	recent       []*entity.LeaveRequestDetail
	approverPend int64
	approverAny  int64
	summaryCalls int
	approvedUser uuid.UUID
}

func (f *empLeave) ListByUser(_ context.Context, t, u uuid.UUID, fl repository.LeaveRequestFilter, _, _ int) ([]*entity.LeaveRequestDetail, int64, error) {
	f.tenants = append(f.tenants, t)
	switch fl.Status {
	case entity.LeaveStatusPending:
		return nil, f.pending, nil
	case entity.LeaveStatusApproved:
		f.approvedUser = u
		return f.approved, int64(len(f.approved)), nil
	}
	return f.recent, int64(len(f.recent)), nil
}

func (f *empLeave) ApproverSummary(_ context.Context, t, _ uuid.UUID) (int64, bool, error) {
	f.tenants = append(f.tenants, t)
	f.summaryCalls++
	return f.approverPend, f.approverAny > 0 || f.approverPend > 0, nil
}

type empHolidays struct {
	repository.HolidayRepository
	tenants []uuid.UUID
	items   []*entity.Holiday

	rangeCalls       int
	lastFrom, lastTo time.Time
}

func (f *empHolidays) ListByDateRange(_ context.Context, t uuid.UUID, from, to time.Time) ([]*entity.Holiday, error) {
	f.tenants = append(f.tenants, t)
	f.rangeCalls++
	f.lastFrom, f.lastTo = from, to
	var out []*entity.Holiday
	for _, h := range f.items {
		if !h.Date.Before(from) && h.Date.Before(to) {
			out = append(out, h)
		}
	}
	return out, nil
}

type empCats struct {
	repository.HolidayCategoryRepository
	tenants []uuid.UUID
	items   []*entity.HolidayCategory
}

func (f *empCats) List(_ context.Context, t uuid.UUID, _ *bool, _, _ int) ([]*entity.HolidayCategory, int64, error) {
	f.tenants = append(f.tenants, t)
	return f.items, int64(len(f.items)), nil
}

type empOrg struct {
	repository.OrganizationSettingsRepository
	tenants []uuid.UUID
	s       *entity.OrganizationSettings
}

func (f *empOrg) GetByTenantID(_ context.Context, t uuid.UUID) (*entity.OrganizationSettings, error) {
	f.tenants = append(f.tenants, t)
	if f.s == nil {
		return nil, domainerrors.ErrOrganizationSettingsNotFound
	}
	return f.s, nil
}

type empFx struct {
	users *empUsers
	alloc *empAllocs
	logs  *empLogs
	leave *empLeave
	hol   *empHolidays
	cats  *empCats
	org   *empOrg
}

func newFx() *empFx {
	return &empFx{&empUsers{}, &empAllocs{}, &empLogs{}, &empLeave{}, &empHolidays{}, &empCats{}, &empOrg{}}
}

func (f *empFx) uc() *EmployeeDashboardUseCaseImpl {
	return NewEmployeeDashboardUseCase(EmployeeDeps{
		Users: f.users, Allocations: f.alloc, StatusLogs: f.logs, Leave: f.leave,
		Holidays: f.hol, Categories: f.cats, OrgSettings: f.org,
	}, func() time.Time { return fixedNow })
}

func logOn(day time.Time, h float64) *entity.StatusLogEntryWithProject {
	return &entity.StatusLogEntryWithProject{StatusLogEntry: entity.StatusLogEntry{LogDate: day, Hours: h}}
}

func TestEmployeeDashboard_RejectsNilIDs(t *testing.T) {
	_, err := newFx().uc().Execute(context.Background(), uuid.Nil, uuid.New())
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	_, err = newFx().uc().Execute(context.Background(), uuid.New(), uuid.Nil)
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

func TestEmployeeDashboard_ScopesEveryRepoToTenant(t *testing.T) {
	f := newFx()
	f.hol.items = []*entity.Holiday{{ID: uuid.New(), Date: d(2026, 12, 25), Name: "Christmas", CategoryIsPublic: true}}
	tid := uuid.New()
	_, err := f.uc().Execute(context.Background(), tid, uuid.New())
	require.NoError(t, err)
	for name, got := range map[string][]uuid.UUID{
		"users": f.users.tenants, "allocs": f.alloc.tenants, "logs": f.logs.tenants,
		"leave": f.leave.tenants, "holidays": f.hol.tenants, "cats": f.cats.tenants, "org": f.org.tenants,
	} {
		require.NotEmpty(t, got, name)
		for _, g := range got {
			assert.Equal(t, tid, g, name)
		}
	}
}

func TestEmployeeDashboard_WeeklyOffAndHolidayAndMissingDays(t *testing.T) {
	f := newFx()
	// Off days default Sun(0)/Sat(6). Window: Sun 24, Mon 25, Tue 26, Wed 27, Thu 28, Fri 29?? -> 24..30.
	// 10-24 is Saturday, 10-25 Sunday, 26 Mon, 27 Tue, 28 Wed, 29 Thu, 30 Fri.
	f.org.s = entity.DefaultOrganizationSettings(uuid.New())
	f.hol.items = []*entity.Holiday{{ID: uuid.New(), Date: d(2026, 10, 27), Name: "Local holiday", CategoryIsPublic: true}}
	f.logs.items = []*entity.StatusLogEntryWithProject{logOn(d(2026, 10, 26), 8), logOn(d(2026, 10, 29), 7.5)}
	// Approved leave covers Wed 28 -> not "missing".
	f.leave.approved = []*entity.LeaveRequestDetail{{LeaveRequest: entity.LeaveRequest{
		StartDate: d(2026, 10, 28), EndDate: d(2026, 10, 28), DayPart: entity.LeaveDayPartFull, Status: entity.LeaveStatusApproved}}}

	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)

	require.Len(t, res.Week.Days, 7)
	working := map[string]bool{}
	for _, dd := range res.Week.Days {
		working[dd.Date] = dd.IsWorkingDay
	}
	assert.False(t, working["2026-10-24"], "saturday off")
	assert.False(t, working["2026-10-25"], "sunday off")
	assert.False(t, working["2026-10-27"], "holiday")
	assert.True(t, working["2026-10-26"])
	// Missing: none of Mon(logged), Tue(holiday), Wed(leave), Thu(logged); today excluded.
	assert.Equal(t, 0, res.Week.MissingDays)
	assert.InDelta(t, 15.5, res.Stats.HoursThisWeek, 0.001)
	assert.Equal(t, "working_day", res.Greeting.Today.Kind)
}

func TestEmployeeDashboard_MissingDaysCount(t *testing.T) {
	f := newFx()
	f.logs.items = []*entity.StatusLogEntryWithProject{logOn(d(2026, 10, 26), 8)}
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	// Working past days: Mon26(logged) Tue27 Wed28 Thu29 -> 3 missing.
	assert.Equal(t, 3, res.Week.MissingDays)
}

func TestEmployeeDashboard_TodayKind(t *testing.T) {
	f := newFx()
	f.hol.items = []*entity.Holiday{{ID: uuid.New(), Date: d(2026, 10, 30), Name: "Founders Day", CategoryIsPublic: true}}
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "holiday", res.Greeting.Today.Kind)
	assert.Equal(t, "Founders Day", res.Greeting.Today.HolidayName)

	f = newFx()
	f.org.s = &entity.OrganizationSettings{WeeklyOffDays: []int{5}} // Friday
	res, err = f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "weekly_off", res.Greeting.Today.Kind)
}

func TestEmployeeDashboard_GreetingWelcomeAndHolidays(t *testing.T) {
	f := newFx()
	msg := "Welcome aboard"
	f.org.s = &entity.OrganizationSettings{WelcomeMessage: &msg, WeeklyOffDays: []int{0, 6}}
	cat := uuid.New()
	f.cats.items = []*entity.HolidayCategory{{ID: cat, Name: "National", Color: "#112233"}}
	for i := 0; i < 7; i++ {
		f.hol.items = append(f.hol.items, &entity.Holiday{ID: uuid.New(), CategoryID: cat, Name: "H", Date: d(2026, 11, 2+i)})
	}
	f.hol.items = append(f.hol.items, &entity.Holiday{ID: uuid.New(), Name: "Past", Date: d(2026, 1, 1)})
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "Asha", res.Greeting.FirstName)
	assert.Equal(t, msg, res.Greeting.WelcomeMessage)
	require.Len(t, res.Holidays, 5)
	assert.Equal(t, "National", res.Holidays[0].CategoryName)
	assert.Equal(t, 3, res.Holidays[0].DaysUntil)
}

func TestEmployeeDashboard_LeaveAndProjectsStats(t *testing.T) {
	f := newFx()
	f.leave.pending = 2
	f.leave.approved = []*entity.LeaveRequestDetail{
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 10, 26), EndDate: d(2026, 10, 26), DayPart: "full"}},
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 11, 10), EndDate: d(2026, 11, 12), DayPart: "full"}},
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 11, 20), EndDate: d(2026, 11, 20), DayPart: "first_half"}},
	}
	end := d(2026, 12, 31)
	f.alloc.items = []*entity.AllocationWithProject{
		{ProjectAllocation: entity.ProjectAllocation{AllocationPercent: 60, StartDate: d(2026, 1, 1), EndDate: &end}, ProjectStatus: entity.ProjectStatusActive},
		{ProjectAllocation: entity.ProjectAllocation{AllocationPercent: 30, StartDate: d(2026, 1, 1)}, ProjectStatus: entity.ProjectStatusOnHold},
		{ProjectAllocation: entity.ProjectAllocation{AllocationPercent: 50, StartDate: d(2026, 1, 1)}, ProjectStatus: entity.ProjectStatusCompleted},
		{ProjectAllocation: entity.ProjectAllocation{AllocationPercent: 40, StartDate: d(2027, 1, 1)}, ProjectStatus: entity.ProjectStatusActive},
	}
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 2, res.Stats.PendingLeave)
	assert.InDelta(t, 3.5, res.Stats.UpcomingLeaveDays, 0.001)
	require.NotNil(t, res.Leave.NextApproved)
	assert.Equal(t, "2026-11-10", res.Leave.NextApproved.StartDate)
	assert.Equal(t, 90, res.Stats.AllocationPercent)
	assert.Len(t, res.Projects, 2)
	require.NotNil(t, res.Projects[0].EndDate)
	assert.Equal(t, "2026-12-31", *res.Projects[0].EndDate)
}

func TestEmployeeDashboard_UpcomingLeaveExcludesHolidays(t *testing.T) {
	f := newFx()
	f.leave.approved = []*entity.LeaveRequestDetail{
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 11, 10), EndDate: d(2026, 11, 12), DayPart: "full"}},
	}
	f.hol.items = []*entity.Holiday{{ID: uuid.New(), Date: d(2026, 11, 11), Name: "Mid", CategoryIsPublic: true}}
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.InDelta(t, 2.0, res.Stats.UpcomingLeaveDays, 0.001)
}

func TestEmployeeDashboard_PendingApprovalsOnlyForApprovers(t *testing.T) {
	f := newFx()
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, res.Stats.PendingApprovals, "plain employee")

	f = newFx()
	f.leave.approverPend, f.leave.approverAny = 4, 9
	res, err = f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res.Stats.PendingApprovals)
	assert.Equal(t, 4, *res.Stats.PendingApprovals)

	assert.Equal(t, 1, f.leave.summaryCalls, "single summary query")

	f = newFx()
	f.leave.approverAny = 3 // manager with an empty queue still sees the tile
	res, err = f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res.Stats.PendingApprovals)
	assert.Equal(t, 0, *res.Stats.PendingApprovals)

	f = newFx()
	f.users.roles = []entity.Role{{Name: entity.RoleAdmin}}
	res, err = f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res.Stats.PendingApprovals)
	assert.Equal(t, 0, *res.Stats.PendingApprovals)
}

func TestEmployeeDashboard_UpcomingLeaveDaysCountsFromToday(t *testing.T) {
	f := newFx()
	f.leave.approved = []*entity.LeaveRequestDetail{
		// Ended before today (Fri 10-30): not counted.
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 10, 26), EndDate: d(2026, 10, 28), DayPart: "full"}},
		// Spans today: 10-28..11-02 -> only 10-30..11-02 remain: Fri + Mon = 2 working days (Sat/Sun off).
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 10, 28), EndDate: d(2026, 11, 2), DayPart: "full"}},
		// Half day in the future stays 0.5.
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 11, 5), EndDate: d(2026, 11, 5), DayPart: "second_half"}},
	}
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.InDelta(t, 2.5, res.Stats.UpcomingLeaveDays, 0.001)
}

func TestEmployeeDashboard_HolidaysSingleRangeQuery(t *testing.T) {
	f := newFx()
	_, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 1, f.hol.rangeCalls)
	assert.Equal(t, d(2026, 10, 24), f.hol.lastFrom, "week start")
	assert.Equal(t, d(2028, 1, 1), f.hol.lastTo, "exclusive end: start of the year after next")
}

func TestEmployeeDashboard_OptionalHolidayIsWorkingDay(t *testing.T) {
	f := newFx()
	f.leave.approved = []*entity.LeaveRequestDetail{
		{LeaveRequest: entity.LeaveRequest{ID: uuid.New(), StartDate: d(2026, 11, 10), EndDate: d(2026, 11, 12), DayPart: "full"}},
	}
	f.hol.items = []*entity.Holiday{
		{ID: uuid.New(), Date: d(2026, 11, 11), Name: "Optional"},                       // not public: counts
		{ID: uuid.New(), Date: d(2026, 11, 12), Name: "Public", CategoryIsPublic: true}, // excluded
		{ID: uuid.New(), Date: d(2026, 10, 27), Name: "Optional week day"},
	}
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.InDelta(t, 2.0, res.Stats.UpcomingLeaveDays, 0.001)
	for _, dd := range res.Week.Days {
		if dd.Date == "2026-10-27" {
			assert.True(t, dd.IsWorkingDay, "optional holiday is a working day")
		}
	}
	assert.Len(t, res.Holidays, 2+0, "optional holidays still listed as upcoming")
}

func TestEmployeeDashboard_LeaveDayKinds(t *testing.T) {
	f := newFx()
	// Today Fri 10-30. Tue 27 full leave, Wed 28 first half, Thu 29 both halves.
	f.leave.approved = []*entity.LeaveRequestDetail{
		{LeaveRequest: entity.LeaveRequest{StartDate: d(2026, 10, 27), EndDate: d(2026, 10, 27), DayPart: "full"}},
		{LeaveRequest: entity.LeaveRequest{StartDate: d(2026, 10, 28), EndDate: d(2026, 10, 28), DayPart: "first_half"}},
		{LeaveRequest: entity.LeaveRequest{StartDate: d(2026, 10, 29), EndDate: d(2026, 10, 29), DayPart: "first_half"}},
		{LeaveRequest: entity.LeaveRequest{StartDate: d(2026, 10, 29), EndDate: d(2026, 10, 29), DayPart: "second_half"}},
		// Spans the weekend: Sat/Sun keep weekly_off kind.
		{LeaveRequest: entity.LeaveRequest{StartDate: d(2026, 10, 23), EndDate: d(2026, 10, 25), DayPart: "full"}},
	}
	tid, uid := uuid.New(), uuid.New()
	res, err := f.uc().Execute(context.Background(), tid, uid)
	require.NoError(t, err)
	kinds := map[string]string{}
	for _, dd := range res.Week.Days {
		kinds[dd.Date] = dd.Kind
	}
	assert.Equal(t, "weekly_off", kinds["2026-10-24"])
	assert.Equal(t, "weekly_off", kinds["2026-10-25"])
	assert.Equal(t, "working_day", kinds["2026-10-26"])
	assert.Equal(t, "leave", kinds["2026-10-27"])
	assert.Equal(t, "working_day", kinds["2026-10-28"], "half day stays a working day")
	assert.Equal(t, "leave", kinds["2026-10-29"], "both halves = full day")
	// Mon 26 and half-day Wed 28 are unlogged past working days (full leave days are excused).
	assert.Equal(t, 2, res.Week.MissingDays)
	// Only the caller's own approved leave is queried.
	assert.Equal(t, uid, f.leave.approvedUser)
	for _, got := range f.leave.tenants {
		assert.Equal(t, tid, got)
	}
}

func TestEmployeeDashboard_TodayOnLeave(t *testing.T) {
	f := newFx()
	f.leave.approved = []*entity.LeaveRequestDetail{
		{LeaveRequest: entity.LeaveRequest{StartDate: d(2026, 10, 30), EndDate: d(2026, 10, 30), DayPart: "full"}},
	}
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "leave", res.Greeting.Today.Kind)
}

// Pending/rejected/cancelled leave is never fetched for expectations: the
// usecase only reads the approved list, so pending leave leaves days missing.
func TestEmployeeDashboard_PendingLeaveDoesNotExcuse(t *testing.T) {
	f := newFx()
	f.leave.pending = 1
	res, err := f.uc().Execute(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 4, res.Week.MissingDays)
}
