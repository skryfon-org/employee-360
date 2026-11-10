//go:build integration

package persistence

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

func newLeaveReq(tenant, user, typ, approver uuid.UUID, start, end time.Time, part string) *entity.LeaveRequest {
	return &entity.LeaveRequest{
		TenantID: tenant, UserID: user, LeaveTypeID: typ, ApproverID: approver,
		StartDate: start, EndDate: end, DayPart: part, Status: entity.LeaveStatusPending,
	}
}

func TestLeaveRequestRepository_ScopingConstraintsAndOverlap(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	types := NewGormLeaveTypeRepository(e.db)
	repo := NewGormLeaveRequestRepository(e.db)
	actor := uuid.New()

	lt := &entity.LeaveType{Name: "Casual", Code: "CASUAL", IsActive: true}
	require.NoError(t, types.Create(c, e.tA.ID, actor, lt))
	ltB := &entity.LeaveType{Name: "Casual", Code: "CASUAL", IsActive: true}
	require.NoError(t, types.Create(c, e.tB.ID, actor, ltB))
	mgr := createTestUser(t, e.db, e.tA.ID, "mgr-"+uuid.NewString()+"@a.test")

	r := newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 11, 10), day(2026, 11, 12), entity.LeaveDayPartFull)
	require.NoError(t, repo.Create(c, e.tA.ID, e.uA.ID, r))

	// Schema CHECKs.
	assert.Error(t, repo.Create(c, e.tA.ID, e.uA.ID, newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 11, 12), day(2026, 11, 10), "full")), "end before start")
	assert.Error(t, repo.Create(c, e.tA.ID, e.uA.ID, newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 12, 1), day(2026, 12, 2), "first_half")), "half over two days")
	assert.Error(t, repo.Create(c, e.tA.ID, e.uA.ID, newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 12, 1), day(2026, 12, 1), "quarter")), "bad day part")
	bad := newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 12, 1), day(2026, 12, 1), "full")
	bad.Status = "weird"
	assert.Error(t, repo.Create(c, e.tA.ID, e.uA.ID, bad), "bad status")
	// Tenant-scoped FKs: a tenant-B leave type / approver cannot be used by tenant A.
	assert.Error(t, repo.Create(c, e.tA.ID, e.uA.ID, newLeaveReq(e.tA.ID, e.uA.ID, ltB.ID, mgr.ID, day(2026, 12, 1), day(2026, 12, 1), "full")), "cross-tenant type")
	assert.Error(t, repo.Create(c, e.tA.ID, e.uA.ID, newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, e.uB.ID, day(2026, 12, 1), day(2026, 12, 1), "full")), "cross-tenant approver")

	// Cross-tenant reads / writes behave as not found.
	_, err := repo.GetByID(c, e.tB.ID, r.ID)
	assert.ErrorIs(t, err, domainerrors.ErrLeaveRequestNotFound)
	_, err = repo.GetDetail(c, e.tB.ID, r.ID)
	assert.ErrorIs(t, err, domainerrors.ErrLeaveRequestNotFound)
	hijack := *r
	hijack.Status = entity.LeaveStatusApproved
	assert.ErrorIs(t, repo.UpdateStatus(c, e.tB.ID, e.uB.ID, &hijack), domainerrors.ErrLeaveRequestNotFound)
	cur, _ := repo.GetByID(c, e.tA.ID, r.ID)
	assert.Equal(t, entity.LeaveStatusPending, cur.Status)

	d, err := repo.GetDetail(c, e.tA.ID, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "CASUAL", d.LeaveTypeCode)
	assert.Equal(t, "Test User", d.UserName)
	assert.Equal(t, "Test User", d.ApproverName)

	// Lists are tenant scoped.
	mine, n, err := repo.ListByUser(c, e.tA.ID, e.uA.ID, repository.LeaveRequestFilter{}, 10, 0)
	require.NoError(t, err)
	assert.Len(t, mine, 1)
	assert.EqualValues(t, 1, n)
	none, n, _ := repo.ListByUser(c, e.tB.ID, e.uA.ID, repository.LeaveRequestFilter{}, 10, 0)
	assert.Empty(t, none)
	assert.EqualValues(t, 0, n)
	team, n, _ := repo.ListByApprover(c, e.tA.ID, mgr.ID, repository.LeaveRequestFilter{Status: "pending"}, 10, 0)
	assert.Len(t, team, 1)
	assert.EqualValues(t, 1, n)
	none, n, _ = repo.ListByApprover(c, e.tB.ID, mgr.ID, repository.LeaveRequestFilter{}, 10, 0)
	assert.Empty(t, none)
	assert.EqualValues(t, 0, n)
	// exclude_status: pending row is dropped from items and total, still tenant/approver scoped.
	notPending, n, err := repo.ListByApprover(c, e.tA.ID, mgr.ID, repository.LeaveRequestFilter{ExcludeStatus: entity.LeaveStatusPending}, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, notPending)
	assert.EqualValues(t, 0, n)
	_, n, _ = repo.ListByApprover(c, e.tA.ID, mgr.ID, repository.LeaveRequestFilter{ExcludeStatus: entity.LeaveStatusApproved}, 10, 0)
	assert.EqualValues(t, 1, n)
	_, n, _ = repo.ListByApprover(c, e.tA.ID, mgr.ID, repository.LeaveRequestFilter{Status: "pending", ExcludeStatus: "pending"}, 10, 0)
	assert.EqualValues(t, 0, n)
	_, n, _ = repo.ListByApprover(c, e.tB.ID, mgr.ID, repository.LeaveRequestFilter{ExcludeStatus: entity.LeaveStatusApproved}, 10, 0)
	assert.EqualValues(t, 0, n, "other tenant")
	// Paging windows the items but the total still counts everything.
	second := newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 12, 1), day(2026, 12, 1), entity.LeaveDayPartFull)
	require.NoError(t, repo.Create(c, e.tA.ID, e.uA.ID, second))
	win, n, _ := repo.ListByUser(c, e.tA.ID, e.uA.ID, repository.LeaveRequestFilter{}, 1, 1)
	assert.Len(t, win, 1)
	assert.EqualValues(t, 2, n)
	win, n, _ = repo.ListByApprover(c, e.tA.ID, mgr.ID, repository.LeaveRequestFilter{}, 1, 0)
	assert.Len(t, win, 1)
	assert.EqualValues(t, 2, n)
	pend, ever, err := repo.ApproverSummary(c, e.tA.ID, mgr.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, pend)
	assert.True(t, ever)
	pend, ever, err = repo.ApproverSummary(c, e.tB.ID, mgr.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, pend, "tenant B never counts tenant A requests for the same approver id")
	assert.False(t, ever)
	pend, ever, _ = repo.ApproverSummary(c, e.tA.ID, uuid.New())
	assert.EqualValues(t, 0, pend)
	assert.False(t, ever)
	require.NoError(t, e.db.Exec("UPDATE leave_requests SET deleted_at = NOW() WHERE id = ?", second.ID).Error)
	pend, ever, err = repo.ApproverSummary(c, e.tA.ID, mgr.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, pend, "soft-deleted excluded")
	assert.True(t, ever)
	pg, total, err := repo.List(c, e.tA.ID, repository.LeaveRequestFilter{}, 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Len(t, pg, 1)
	_, total, _ = repo.List(c, e.tB.ID, repository.LeaveRequestFilter{}, 10, 0)
	assert.EqualValues(t, 0, total)

	// Date-range and filter semantics.
	from, to := day(2026, 11, 12), day(2026, 11, 20)
	hit, _, _ := repo.List(c, e.tA.ID, repository.LeaveRequestFilter{From: &from, To: &to}, 10, 0)
	assert.Len(t, hit, 1, "range touching the last day intersects")
	from = day(2026, 11, 13)
	miss, _, _ := repo.List(c, e.tA.ID, repository.LeaveRequestFilter{From: &from}, 10, 0)
	assert.Empty(t, miss)

	// Overlap detection.
	ov := func(s, en time.Time, part string) bool {
		got, err := repo.HasOverlap(c, e.tA.ID, e.uA.ID, s, en, part, uuid.Nil)
		require.NoError(t, err)
		return got
	}
	assert.True(t, ov(day(2026, 11, 12), day(2026, 11, 15), "full"))
	assert.True(t, ov(day(2026, 11, 8), day(2026, 11, 10), "full"))
	assert.False(t, ov(day(2026, 11, 13), day(2026, 11, 13), "full"))
	got, _ := repo.HasOverlap(c, e.tB.ID, e.uA.ID, day(2026, 11, 10), day(2026, 11, 10), "full", uuid.Nil)
	assert.False(t, got, "other tenant")
	got, _ = repo.HasOverlap(c, e.tA.ID, e.uA.ID, day(2026, 11, 10), day(2026, 11, 12), "full", r.ID)
	assert.False(t, got, "excludeID")

	half := newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 11, 20), day(2026, 11, 20), "first_half")
	require.NoError(t, repo.Create(c, e.tA.ID, e.uA.ID, half))
	assert.False(t, ov(day(2026, 11, 20), day(2026, 11, 20), "second_half"))
	assert.True(t, ov(day(2026, 11, 20), day(2026, 11, 20), "first_half"))
	assert.True(t, ov(day(2026, 11, 20), day(2026, 11, 20), "full"))

	// Rejected/cancelled requests free the dates.
	half.Status = entity.LeaveStatusCancelled
	require.NoError(t, repo.UpdateStatus(c, e.tA.ID, e.uA.ID, half))
	assert.False(t, ov(day(2026, 11, 20), day(2026, 11, 20), "first_half"))
}

func TestLeaveRequestRepository_UpdateStatusKeepsDecisionTrail(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	types := NewGormLeaveTypeRepository(e.db)
	repo := NewGormLeaveRequestRepository(e.db)
	lt := &entity.LeaveType{Name: "Casual", Code: "CASUAL", IsActive: true}
	require.NoError(t, types.Create(c, e.tA.ID, uuid.New(), lt))
	mgr := createTestUser(t, e.db, e.tA.ID, "mgr-"+uuid.NewString()+"@a.test")

	r := newLeaveReq(e.tA.ID, e.uA.ID, lt.ID, mgr.ID, day(2026, 11, 10), day(2026, 11, 10), entity.LeaveDayPartFull)
	require.NoError(t, repo.Create(c, e.tA.ID, e.uA.ID, r))

	decidedAt := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	r.Status, r.DecidedBy, r.DecidedAt, r.DecisionComment = entity.LeaveStatusApproved, &mgr.ID, &decidedAt, "enjoy"
	require.NoError(t, repo.RecordDecision(c, e.tA.ID, mgr.ID, r))

	// Cancel writes the status only, even if the in-memory struct is mangled.
	other := uuid.New()
	now := time.Now().UTC()
	r.Status, r.DecidedBy, r.DecidedAt, r.DecisionComment = entity.LeaveStatusCancelled, &other, &now, "overwritten"
	require.NoError(t, repo.UpdateStatus(c, e.tA.ID, e.uA.ID, r))

	got, err := repo.GetByID(c, e.tA.ID, r.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.LeaveStatusCancelled, got.Status)
	require.NotNil(t, got.DecidedBy)
	assert.Equal(t, mgr.ID, *got.DecidedBy)
	assert.True(t, decidedAt.Equal(*got.DecidedAt))
	assert.Equal(t, "enjoy", got.DecisionComment)
	assert.Equal(t, e.uA.ID, *got.UpdatedBy)
	d, err := repo.GetDetail(c, e.tA.ID, r.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, d.DecidedByName)
	// Another tenant must not resolve the decider name.
	_, err = repo.GetDetail(c, e.tB.ID, r.ID)
	assert.Error(t, err)
}
