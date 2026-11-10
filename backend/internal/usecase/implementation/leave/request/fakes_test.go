package request

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
)

type fakeTypes struct {
	rows map[uuid.UUID]*entity.LeaveType
}

func newFakeTypes() *fakeTypes { return &fakeTypes{rows: map[uuid.UUID]*entity.LeaveType{}} }

func (f *fakeTypes) add(tenant uuid.UUID, code string, active bool) *entity.LeaveType {
	lt := &entity.LeaveType{ID: uuid.New(), TenantID: tenant, Name: code, Code: code, IsActive: active}
	f.rows[lt.ID] = lt
	return lt
}
func (f *fakeTypes) Create(_ context.Context, tenantID, _ uuid.UUID, lt *entity.LeaveType) error {
	lt.TenantID = tenantID
	cp := *lt
	f.rows[lt.ID] = &cp
	return nil
}
func (f *fakeTypes) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.LeaveType, error) {
	lt, ok := f.rows[id]
	if !ok || lt.TenantID != tenantID {
		return nil, domainerrors.ErrLeaveTypeNotFound
	}
	cp := *lt
	return &cp, nil
}
func (f *fakeTypes) List(_ context.Context, tenantID uuid.UUID, activeOnly bool) ([]*entity.LeaveType, error) {
	var out []*entity.LeaveType
	for _, lt := range f.rows {
		if lt.TenantID == tenantID && (!activeOnly || lt.IsActive) {
			out = append(out, lt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func (f *fakeTypes) Update(_ context.Context, _, _ uuid.UUID, lt *entity.LeaveType) error {
	cp := *lt
	f.rows[lt.ID] = &cp
	return nil
}
func (f *fakeTypes) ExistsByCode(_ context.Context, tenantID uuid.UUID, code string, excludeID uuid.UUID) (bool, error) {
	for _, lt := range f.rows {
		if lt.TenantID == tenantID && lt.Code == code && lt.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

type fakeRequests struct {
	rows map[uuid.UUID]*entity.LeaveRequest
}

func newFakeRequests() *fakeRequests {
	return &fakeRequests{rows: map[uuid.UUID]*entity.LeaveRequest{}}
}

func (f *fakeRequests) LockUser(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeRequests) Create(_ context.Context, tenantID, _ uuid.UUID, r *entity.LeaveRequest) error {
	r.TenantID = tenantID
	cp := *r
	f.rows[r.ID] = &cp
	return nil
}
func (f *fakeRequests) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.LeaveRequest, error) {
	r, ok := f.rows[id]
	if !ok || r.TenantID != tenantID {
		return nil, domainerrors.ErrLeaveRequestNotFound
	}
	cp := *r
	return &cp, nil
}
func (f *fakeRequests) GetByIDForUpdate(c context.Context, t, id uuid.UUID) (*entity.LeaveRequest, error) {
	return f.GetByID(c, t, id)
}
func (f *fakeRequests) GetDetail(c context.Context, t, id uuid.UUID) (*entity.LeaveRequestDetail, error) {
	r, err := f.GetByID(c, t, id)
	if err != nil {
		return nil, err
	}
	d := &entity.LeaveRequestDetail{LeaveRequest: *r, UserName: "Ann Lee", UserEmail: "ann@example.com", LeaveTypeName: "CASUAL"}
	if r.DecidedBy != nil {
		d.DecidedByName = "Decider Name"
	}
	return d, nil
}
func (f *fakeRequests) UpdateStatus(_ context.Context, tenantID, _ uuid.UUID, r *entity.LeaveRequest) error {
	cur, ok := f.rows[r.ID]
	if !ok || cur.TenantID != tenantID {
		return domainerrors.ErrLeaveRequestNotFound
	}
	cur.Status = r.Status // status only: the decision trail is untouched
	return nil
}
func (f *fakeRequests) RecordDecision(_ context.Context, tenantID, _ uuid.UUID, r *entity.LeaveRequest) error {
	cur, ok := f.rows[r.ID]
	if !ok || cur.TenantID != tenantID {
		return domainerrors.ErrLeaveRequestNotFound
	}
	cp := *r
	f.rows[r.ID] = &cp
	return nil
}
func (f *fakeRequests) HasOverlap(_ context.Context, tenantID, userID uuid.UUID, start, end time.Time, dayPart string, excludeID uuid.UUID) (bool, error) {
	for _, r := range f.rows {
		if r.TenantID != tenantID || r.UserID != userID || r.ID == excludeID {
			continue
		}
		if r.Status != entity.LeaveStatusPending && r.Status != entity.LeaveStatusApproved {
			continue
		}
		if r.StartDate.After(end) || r.EndDate.Before(start) {
			continue
		}
		if start.Equal(end) && dayPart != entity.LeaveDayPartFull &&
			r.StartDate.Equal(r.EndDate) && r.DayPart != entity.LeaveDayPartFull && r.DayPart != dayPart {
			continue
		}
		return true, nil
	}
	return false, nil
}
func (f *fakeRequests) ListByUser(_ context.Context, tenantID, userID uuid.UUID, fl repository.LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error) {
	return f.page(tenantID, limit, offset, func(r *entity.LeaveRequest) bool {
		return r.UserID == userID && inFilter(r, fl)
	})
}
func (f *fakeRequests) ListByApprover(_ context.Context, tenantID, approverID uuid.UUID, fl repository.LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error) {
	return f.page(tenantID, limit, offset, func(r *entity.LeaveRequest) bool {
		return r.ApproverID == approverID && inFilter(r, fl)
	})
}
func (f *fakeRequests) ApproverSummary(_ context.Context, tenantID, approverID uuid.UUID) (int64, bool, error) {
	all := f.filter(tenantID, func(r *entity.LeaveRequest) bool { return r.ApproverID == approverID })
	var p int64
	for _, r := range all {
		if r.Status == entity.LeaveStatusPending {
			p++
		}
	}
	return p, len(all) > 0, nil
}
func (f *fakeRequests) List(_ context.Context, tenantID uuid.UUID, fl repository.LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error) {
	return f.page(tenantID, limit, offset, func(r *entity.LeaveRequest) bool {
		return inFilter(r, fl) && (fl.UserID == nil || r.UserID == *fl.UserID)
	})
}
func (f *fakeRequests) page(tenantID uuid.UUID, limit, offset int, keep func(*entity.LeaveRequest) bool) ([]*entity.LeaveRequestDetail, int64, error) {
	all := f.filter(tenantID, keep)
	sort.Slice(all, func(i, j int) bool { return all[i].StartDate.Before(all[j].StartDate) })
	total := int64(len(all))
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all, total, nil
}
func (f *fakeRequests) filter(tenantID uuid.UUID, keep func(*entity.LeaveRequest) bool) []*entity.LeaveRequestDetail {
	out := []*entity.LeaveRequestDetail{}
	for _, r := range f.rows {
		if r.TenantID == tenantID && keep(r) {
			out = append(out, &entity.LeaveRequestDetail{LeaveRequest: *r})
		}
	}
	return out
}

type fakeProfiles struct{ managers map[uuid.UUID]uuid.UUID }

func (f *fakeProfiles) GetByUserID(_ context.Context, tenantID, userID uuid.UUID) (*entity.EmployeeProfile, error) {
	m, ok := f.managers[userID]
	if !ok {
		return nil, domainerrors.ErrEmployeeProfileNotFound
	}
	return &entity.EmployeeProfile{TenantID: tenantID, UserID: userID, ManagerID: &m}, nil
}
func (f *fakeProfiles) Create(context.Context, uuid.UUID, *entity.EmployeeProfile) error { return nil }
func (f *fakeProfiles) Update(context.Context, uuid.UUID, uuid.UUID, *entity.EmployeeProfile) error {
	return nil
}
func (f *fakeProfiles) EmployeeCodeExists(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return false, nil
}

// fakeApprovers models users: active users and the admin subset.
type fakeApprovers struct {
	active map[uuid.UUID]bool
	admins []uuid.UUID
}

func (f *fakeApprovers) ListActiveAdmins(context.Context, uuid.UUID) ([]*entity.LeaveApprover, error) {
	var out []*entity.LeaveApprover
	for _, id := range f.admins {
		if f.active[id] {
			out = append(out, &entity.LeaveApprover{UserID: id, Name: "Admin Name", Email: "admin@example.com", Kind: entity.LeaveApproverAdmin})
		}
	}
	return out, nil
}
func (f *fakeApprovers) GetActiveUser(_ context.Context, _, id uuid.UUID) (*entity.LeaveApprover, bool, error) {
	if !f.active[id] {
		return nil, false, nil
	}
	return &entity.LeaveApprover{UserID: id, Name: "Mgr Name", Email: "mgr@example.com", Kind: entity.LeaveApproverManager}, true, nil
}

type recordedAudit struct {
	actions []string
	metas   []map[string]any
}

func (r *recordedAudit) Record(_ context.Context, _, _ uuid.UUID, action, _ string, _ uuid.UUID, meta map[string]any) error {
	r.actions = append(r.actions, action)
	r.metas = append(r.metas, meta)
	return nil
}

var (
	now    = time.Date(2026, 10, 29, 15, 30, 0, 0, time.UTC)
	clock  = func() time.Time { return now }
	bgctx  = context.Background()
	tenant = uuid.New()
)

type fakeOff struct{ days []int }

func (f *fakeOff) WeeklyOffDays(context.Context, uuid.UUID) ([]int, error) { return f.days, nil }

type fakeHolidays struct {
	repository.HolidayRepository
	items []*entity.Holiday
}

// add registers a public-category holiday (a company day off).
func (f *fakeHolidays) add(tenantID uuid.UUID, date string) {
	f.items = append(f.items, &entity.Holiday{ID: uuid.New(), TenantID: tenantID, Name: "H", Date: day(date), CategoryIsPublic: true})
}

// addOptional registers a holiday whose category is not public (counts as a working day).
func (f *fakeHolidays) addOptional(tenantID uuid.UUID, date string) {
	f.items = append(f.items, &entity.Holiday{ID: uuid.New(), TenantID: tenantID, Name: "O", Date: day(date)})
}

func (f *fakeHolidays) ListByDateRange(_ context.Context, tenantID uuid.UUID, from, to time.Time) ([]*entity.Holiday, error) {
	var out []*entity.Holiday
	for _, h := range f.items {
		if h.TenantID == tenantID && !h.Date.Before(from) && h.Date.Before(to) {
			out = append(out, h)
		}
	}
	return out, nil
}

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

type env struct {
	types     *fakeTypes
	requests  *fakeRequests
	profiles  *fakeProfiles
	approvers *fakeApprovers
	audit     *recordedAudit
	events    *recordingPublisher
	off       *fakeOff
	hol       *fakeHolidays

	typ                  *entity.LeaveType
	employee, mgr, admin uuid.UUID
	other                uuid.UUID
}

func newEnv() *env {
	e := &env{
		types: newFakeTypes(), requests: newFakeRequests(), audit: &recordedAudit{}, events: &recordingPublisher{}, off: &fakeOff{days: []int{0, 6}}, hol: &fakeHolidays{},
		employee: uuid.New(), mgr: uuid.New(), admin: uuid.New(), other: uuid.New(),
	}
	e.typ = e.types.add(tenant, "CASUAL", true)
	e.profiles = &fakeProfiles{managers: map[uuid.UUID]uuid.UUID{e.employee: e.mgr}}
	e.approvers = &fakeApprovers{
		active: map[uuid.UUID]bool{e.employee: true, e.mgr: true, e.admin: true, e.other: true},
		admins: []uuid.UUID{e.admin},
	}
	return e
}

func (e *env) applyUC() *applyLeave {
	return NewApplyLeaveUseCase(e.requests, e.types, e.profiles, e.approvers, e.off, e.hol, e.audit, nil, clock, e.events, testURLs).(*applyLeave)
}

func (e *env) in(start, end string) leavetypes.ApplyLeaveInput {
	return leavetypes.ApplyLeaveInput{LeaveTypeID: e.typ.ID, StartDate: day(start), EndDate: day(end), ApproverID: e.mgr}
}

func (e *env) applyOK(t *testing.T, in leavetypes.ApplyLeaveInput) *entity.LeaveRequestDetail {
	t.Helper()
	d, err := e.applyUC().Execute(bgctx, tenant, e.employee, in)
	require.NoError(t, err)
	return d
}

func (e *env) pending(t *testing.T) uuid.UUID {
	t.Helper()
	return e.applyOK(t, e.in("2026-11-10", "2026-11-11")).ID
}

// inFilter applies the status and overlapping date-range filters.
func inFilter(r *entity.LeaveRequest, fl repository.LeaveRequestFilter) bool {
	if fl.Status != "" && r.Status != fl.Status {
		return false
	}
	if fl.ExcludeStatus != "" && r.Status == fl.ExcludeStatus {
		return false
	}
	if fl.LeaveTypeID != nil && r.LeaveTypeID != *fl.LeaveTypeID {
		return false
	}
	if fl.From != nil && r.EndDate.Before(*fl.From) {
		return false
	}
	if fl.To != nil && r.StartDate.After(*fl.To) {
		return false
	}
	return true
}

var testURLs = leavecommon.AppURLs{Default: "https://app.test", Admin: "https://admin.test/", Employee: "https://emp.test"}

// recordingPublisher captures published events; err makes Publish fail.
type recordingPublisher struct {
	events []event.Event
	err    error
}

func (r *recordingPublisher) Publish(_ context.Context, evts ...event.Event) error {
	if r.err != nil {
		return r.err
	}
	r.events = append(r.events, evts...)
	return nil
}
