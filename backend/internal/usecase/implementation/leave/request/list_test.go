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

func TestLists(t *testing.T) {
	e := newEnv()
	id := e.pending(t)

	q := func(status string) leavetypes.ListMyLeaveQuery { return leavetypes.ListMyLeaveQuery{Status: status} }
	mine, err := NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.employee, q(""))
	require.NoError(t, err)
	assert.Len(t, mine.Items, 1)
	assert.EqualValues(t, 1, mine.Total)
	assert.Equal(t, 1, mine.Page)
	assert.Equal(t, 20, mine.PageSize, "default page size")
	none, _ := NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.other, q(""))
	assert.Empty(t, none.Items)
	_, err = NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.employee, q("bogus"))
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)

	team, _ := NewListTeamLeaveUseCase(e.requests).Execute(bgctx, tenant, e.mgr, q("pending"))
	require.Len(t, team.Items, 1)
	assert.Equal(t, id, team.Items[0].ID)
	team, _ = NewListTeamLeaveUseCase(e.requests).Execute(bgctx, tenant, e.other, q(""))
	assert.Empty(t, team.Items)
	team, _ = NewListTeamLeaveUseCase(e.requests).Execute(bgctx, uuid.New(), e.mgr, q(""))
	assert.Empty(t, team.Items, "other tenant")

	// Pagination: total counts everything, items are windowed, size is clamped.
	e.applyOK(t, e.in("2026-12-01", "2026-12-01"))
	e.applyOK(t, e.in("2026-12-02", "2026-12-02"))
	p1, _ := NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.employee, leavetypes.ListMyLeaveQuery{Page: 1, PageSize: 2})
	assert.Len(t, p1.Items, 2)
	assert.EqualValues(t, 3, p1.Total)
	p2, _ := NewListTeamLeaveUseCase(e.requests).Execute(bgctx, tenant, e.mgr, leavetypes.ListMyLeaveQuery{Page: 2, PageSize: 2})
	assert.Len(t, p2.Items, 1)
	assert.EqualValues(t, 3, p2.Total)
	assert.Equal(t, 2, p2.Page)
	big, _ := NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.employee, leavetypes.ListMyLeaveQuery{PageSize: 500})
	assert.Equal(t, 100, big.PageSize)

	// Date range: overlap semantics, either bound optional, inverted rejected.
	rng := func(from, to string) int {
		p, err := NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.employee, leavetypes.ListMyLeaveQuery{From: from, To: to})
		require.NoError(t, err)
		tp, err := NewListTeamLeaveUseCase(e.requests).Execute(bgctx, tenant, e.mgr, leavetypes.ListMyLeaveQuery{From: from, To: to})
		require.NoError(t, err)
		assert.Equal(t, len(p.Items), len(tp.Items))
		return len(p.Items)
	}
	assert.Equal(t, 3, rng("", ""))
	assert.Equal(t, 1, rng("2026-12-02", ""), "from only: ends on/after")
	assert.Equal(t, 2, rng("", "2026-12-01"), "to only: starts on/before")
	assert.Equal(t, 1, rng("2026-12-01", "2026-12-01"))
	assert.Equal(t, 0, rng("2000-01-01", "2000-01-02"))
	for _, q := range []leavetypes.ListMyLeaveQuery{{From: "2026-02-01", To: "2026-01-01"}, {From: "bad"}, {To: "2026/01/01"}} {
		_, err := NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.employee, q)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData, q)
		_, err = NewListTeamLeaveUseCase(e.requests).Execute(bgctx, tenant, e.mgr, q)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData, q)
	}

	// Leave type filter: matching, non-matching, malformed (both caller-scoped lists).
	byType := func(id string) (int, int, error) {
		p, err := NewListMyLeaveUseCase(e.requests).Execute(bgctx, tenant, e.employee, leavetypes.ListMyLeaveQuery{LeaveTypeID: id})
		if err != nil {
			return 0, 0, err
		}
		tp, err := NewListTeamLeaveUseCase(e.requests).Execute(bgctx, tenant, e.mgr, leavetypes.ListMyLeaveQuery{LeaveTypeID: id})
		if err != nil {
			return 0, 0, err
		}
		return len(p.Items), len(tp.Items), nil
	}
	m, tm, err := byType(e.typ.ID.String())
	require.NoError(t, err)
	assert.Equal(t, 3, m)
	assert.Equal(t, 3, tm)
	m, tm, err = byType(uuid.NewString())
	require.NoError(t, err)
	assert.Zero(t, m)
	assert.Zero(t, tm)
	_, _, err = byType("not-a-uuid")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)
	_, err = NewListTeamLeaveUseCase(e.requests).Execute(bgctx, tenant, e.mgr, leavetypes.ListMyLeaveQuery{LeaveTypeID: "bad"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)

	all := NewListAllLeaveUseCase(e.requests)
	page, err := all.Execute(bgctx, tenant, leavetypes.ListLeaveRequestsQuery{PageSize: 500})
	require.NoError(t, err)
	assert.Equal(t, 100, page.PageSize)
	assert.EqualValues(t, 3, page.Total)
	for _, q := range []leavetypes.ListLeaveRequestsQuery{
		{EmployeeID: "nope"}, {From: "2026/01/01"}, {From: "2026-02-01", To: "2026-01-01"}, {Status: "x"},
	} {
		_, err = all.Execute(bgctx, tenant, q)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData, q)
	}
}

func TestListTeam_ExcludeStatus(t *testing.T) {
	e := newEnv()
	e.pending(t)
	approved := e.applyOK(t, e.in("2026-12-01", "2026-12-01")).ID
	e.requests.rows[approved].Status = entity.LeaveStatusApproved
	e.applyOK(t, e.in("2026-12-02", "2026-12-02"))

	uc := NewListTeamLeaveUseCase(e.requests)
	run := func(q leavetypes.ListMyLeaveQuery) *leavetypes.LeaveRequestPage {
		p, err := uc.Execute(bgctx, tenant, e.mgr, q)
		require.NoError(t, err)
		return p
	}
	p := run(leavetypes.ListMyLeaveQuery{ExcludeStatus: "pending"})
	require.Len(t, p.Items, 1)
	assert.Equal(t, approved, p.Items[0].ID)
	assert.EqualValues(t, 1, p.Total, "total excludes pending rows")

	assert.EqualValues(t, 2, run(leavetypes.ListMyLeaveQuery{ExcludeStatus: "approved"}).Total)
	assert.EqualValues(t, 3, run(leavetypes.ListMyLeaveQuery{ExcludeStatus: "rejected"}).Total)

	same := run(leavetypes.ListMyLeaveQuery{Status: "pending", ExcludeStatus: "pending"})
	assert.Empty(t, same.Items, "status == exclude_status yields an empty page")
	assert.EqualValues(t, 0, same.Total)

	_, err := uc.Execute(bgctx, tenant, e.mgr, leavetypes.ListMyLeaveQuery{ExcludeStatus: "bogus"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)
	other, _ := uc.Execute(bgctx, tenant, e.other, leavetypes.ListMyLeaveQuery{ExcludeStatus: "pending"})
	assert.Empty(t, other.Items, "scoped to approver")
}
