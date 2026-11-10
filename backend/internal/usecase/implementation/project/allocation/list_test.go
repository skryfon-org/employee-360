package allocation

import (
	"context"
	"testing"
	"time"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjectAllocationsAndUserView_TenantScoped(t *testing.T) {
	e := newAllocEnv(t)
	ctx := context.Background()
	_, err := e.create().Execute(ctx, e.tenant, e.actor, e.in(e.p1, 40, "2026-01-01", nil))
	require.NoError(t, err)

	lp := NewListProjectAllocationsUseCase(e.projects, e.allocs)
	as, err := lp.Execute(ctx, e.tenant, e.p1.ID, "")
	require.NoError(t, err)
	assert.Len(t, as, 1)
	assert.Equal(t, "Ada Lovelace", as[0].UserName())
	_, err = lp.Execute(ctx, uuid.New(), e.p1.ID, "")
	assert.ErrorIs(t, err, domainerrors.ErrProjectNotFound)

	lu := NewListUserAllocationsUseCase(e.users, e.allocs)
	_, err = lu.Execute(ctx, uuid.New(), e.user, "")
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	res, err := lu.Execute(ctx, e.tenant, e.inactive, "")
	require.NoError(t, err, "inactive users can still be inspected")
	assert.Empty(t, res.Allocations)
	assert.False(t, res.OverAllocated)
}

func TestUserViews_LiveProjectFiltering(t *testing.T) {
	e := newAllocEnv(t)
	ctx := context.Background()
	p3 := &entity.Project{ID: uuid.New(), TenantID: e.tenant, Name: "P3", Code: "P3", Status: entity.ProjectStatusActive, StartDate: d("2026-01-01")}
	require.NoError(t, e.projects.Create(ctx, e.tenant, e.actor, p3))
	for _, p := range []*entity.Project{e.p1, e.p2, p3} {
		_, err := e.create().Execute(ctx, e.tenant, e.actor, e.in(p, 60, "2026-01-01", nil))
		require.NoError(t, err)
	}
	e.setStatus(e.p1, entity.ProjectStatusArchived)
	e.setStatus(e.p2, entity.ProjectStatusOnHold)

	// Admin view lists everything (with project status) but counts only live projects.
	admin, err := NewListUserAllocationsUseCase(e.users, e.allocs).Execute(ctx, e.tenant, e.user, "")
	require.NoError(t, err)
	assert.Len(t, admin.Allocations, 3)
	statuses := map[string]bool{}
	for _, a := range admin.Allocations {
		statuses[a.ProjectStatus] = true
	}
	assert.Equal(t, map[string]bool{"archived": true, "on_hold": true, "active": true}, statuses)
	assert.Equal(t, 120, admin.TotalPercent)
	assert.True(t, admin.OverAllocated)
	sum := 0
	for _, a := range admin.Allocations {
		assert.Equal(t, entity.IsLiveProjectStatus(a.ProjectStatus), a.CountsTowardTotal())
		if a.CountsTowardTotal() {
			sum += a.AllocationPercent
		}
	}
	assert.Equal(t, admin.TotalPercent, sum, "flagged items sum to total_percent")

	// /me/projects drops completed and archived projects.
	me, err := NewListMyProjectsUseCase(e.users, e.allocs).Execute(ctx, e.tenant, e.user, entity.AllocationScopeActive)
	require.NoError(t, err)
	require.Len(t, me.Allocations, 2)
	for _, a := range me.Allocations {
		assert.True(t, entity.IsLiveProjectStatus(a.ProjectStatus))
	}
	assert.Equal(t, 120, me.TotalPercent)

	e.setStatus(e.p2, entity.ProjectStatusCompleted)
	me, err = NewListMyProjectsUseCase(e.users, e.allocs).Execute(ctx, e.tenant, e.user, "")
	require.NoError(t, err)
	assert.Len(t, me.Allocations, 1)
	assert.False(t, me.OverAllocated)

	_, err = NewListMyProjectsUseCase(e.users, e.allocs).Execute(ctx, uuid.New(), e.user, "")
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
}

func TestListMyProjects_Scopes(t *testing.T) {
	e := newAllocEnv(t) // today = 2026-06-15
	ctx := context.Background()
	add := func(p *entity.Project, pct int, start string, end *time.Time) *entity.ProjectAllocation {
		a := &entity.ProjectAllocation{ID: uuid.New(), TenantID: e.tenant, ProjectID: p.ID, UserID: e.user, AllocationPercent: pct, StartDate: d(start), EndDate: end}
		require.NoError(t, e.allocs.Create(ctx, e.tenant, e.actor, a))
		return a
	}
	endsToday := add(e.p1, 10, "2026-01-01", dp("2026-06-15")) // end inclusive: active
	openEnded := add(e.p2, 20, "2026-06-15", nil)              // starts today: active, not upcoming
	upcoming := add(e.p1, 30, "2026-06-16", nil)
	past := add(e.p2, 40, "2026-01-01", dp("2026-06-14"))
	deleted := add(e.p1, 50, "2026-07-01", nil)
	require.NoError(t, e.allocs.SoftDelete(ctx, e.tenant, deleted.ID, e.actor))
	e.setStatus(e.p2, entity.ProjectStatusArchived)

	run := func(scope entity.AllocationScope) *prjtypes.UserAllocationsResult {
		res, err := NewListMyProjectsUseCase(e.users, e.allocs).Execute(ctx, e.tenant, e.user, scope)
		require.NoError(t, err)
		return res
	}
	ids := func(r *prjtypes.UserAllocationsResult) map[uuid.UUID]bool {
		m := map[uuid.UUID]bool{}
		for _, a := range r.Allocations {
			m[a.ID] = true
		}
		return m
	}

	act := ids(run(""))
	assert.Equal(t, map[uuid.UUID]bool{endsToday.ID: true}, act, "default = active on live projects; archived p2 dropped")
	assert.Equal(t, act, ids(run(entity.AllocationScopeActive)))
	assert.Equal(t, map[uuid.UUID]bool{upcoming.ID: true}, ids(run(entity.AllocationScopeUpcoming)))
	assert.Equal(t, map[uuid.UUID]bool{past.ID: true}, ids(run(entity.AllocationScopePast)), "past includes archived projects")
	all := run(entity.AllocationScopeAll)
	assert.Equal(t, map[uuid.UUID]bool{endsToday.ID: true, openEnded.ID: true, upcoming.ID: true, past.ID: true}, ids(all), "soft-deleted excluded")
	assert.Equal(t, 10, all.TotalPercent, "total = today's load on live projects only")
	assert.Equal(t, 0, run(entity.AllocationScopeUpcoming).TotalPercent)
	want := entity.AllocationScopeCounts{Active: 1, Upcoming: 1, Past: 1}
	for _, sc := range []entity.AllocationScope{"", entity.AllocationScopeActive, entity.AllocationScopeUpcoming, entity.AllocationScopePast, entity.AllocationScopeAll} {
		require.NotNil(t, run(sc).Counts)
		assert.Equal(t, want, *run(sc).Counts, "counts independent of scope %q (archived p2 not in active)", sc)
	}

	_, err := NewListMyProjectsUseCase(e.users, e.allocs).Execute(ctx, e.tenant, e.user, "bogus")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
}

func TestListUserAllocations_Scopes(t *testing.T) {
	e := newAllocEnv(t) // today = 2026-06-15
	ctx := context.Background()
	add := func(p *entity.Project, pct int, start string, end *time.Time) *entity.ProjectAllocation {
		a := &entity.ProjectAllocation{ID: uuid.New(), TenantID: e.tenant, ProjectID: p.ID, UserID: e.user, AllocationPercent: pct, StartDate: d(start), EndDate: end}
		require.NoError(t, e.allocs.Create(ctx, e.tenant, e.actor, a))
		return a
	}
	activeLive := add(e.p1, 10, "2026-01-01", nil)
	activeArchived := add(e.p2, 20, "2026-01-01", nil)
	upcoming := add(e.p1, 30, "2026-06-16", nil)
	past := add(e.p2, 40, "2026-01-01", dp("2026-06-14"))
	e.setStatus(e.p2, entity.ProjectStatusArchived)

	uc := NewListUserAllocationsUseCase(e.users, e.allocs)
	run := func(scope entity.AllocationScope) (map[uuid.UUID]bool, *prjtypes.UserAllocationsResult) {
		res, err := uc.Execute(ctx, e.tenant, e.user, scope)
		require.NoError(t, err)
		m := map[uuid.UUID]bool{}
		for _, a := range res.Allocations {
			m[a.ID] = true
		}
		return m, res
	}

	// Default is unchanged: every date-active allocation, any project status.
	def, res := run("")
	assert.Equal(t, map[uuid.UUID]bool{activeLive.ID: true, activeArchived.ID: true}, def)
	assert.Equal(t, 10, res.TotalPercent)
	assert.Nil(t, res.Counts)
	act, _ := run(entity.AllocationScopeActive)
	assert.Equal(t, def, act)

	up, res := run(entity.AllocationScopeUpcoming)
	assert.Equal(t, map[uuid.UUID]bool{upcoming.ID: true}, up)
	assert.Equal(t, 0, res.TotalPercent, "upcoming never counts")
	assert.False(t, res.OverAllocated)

	pa, res := run(entity.AllocationScopePast)
	assert.Equal(t, map[uuid.UUID]bool{past.ID: true}, pa, "past includes archived projects")
	assert.Equal(t, 0, res.TotalPercent)

	all, res := run(entity.AllocationScopeAll)
	assert.Len(t, all, 4)
	assert.Equal(t, 10, res.TotalPercent, "only today's live-project load counts")

	_, err := uc.Execute(ctx, e.tenant, e.user, "bogus")
	assert.ErrorIs(t, err, ErrInvalidScope)
	_, err = uc.Execute(ctx, uuid.New(), e.user, entity.AllocationScopeAll)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
}

func TestListProjectAllocations_State(t *testing.T) {
	e := newAllocEnv(t) // today = 2026-06-15
	ctx := context.Background()
	add := func(pct int, start string, end *time.Time) *entity.ProjectAllocation {
		a := &entity.ProjectAllocation{ID: uuid.New(), TenantID: e.tenant, ProjectID: e.p1.ID, UserID: e.user, AllocationPercent: pct, StartDate: d(start), EndDate: end}
		require.NoError(t, e.allocs.Create(ctx, e.tenant, e.actor, a))
		return a
	}
	endsToday := add(10, "2026-01-01", dp("2026-06-15"))
	startsToday := add(20, "2026-06-15", nil)
	upcoming := add(30, "2026-06-16", nil)
	past := add(40, "2026-01-01", dp("2026-06-14"))
	deleted := add(50, "2026-07-01", nil)
	require.NoError(t, e.allocs.SoftDelete(ctx, e.tenant, deleted.ID, e.actor))

	lp := NewListProjectAllocationsUseCase(e.projects, e.allocs)
	ids := func(scope entity.AllocationScope) map[uuid.UUID]bool {
		as, err := lp.Execute(ctx, e.tenant, e.p1.ID, scope)
		require.NoError(t, err)
		m := map[uuid.UUID]bool{}
		for _, a := range as {
			m[a.ID] = true
		}
		return m
	}
	assert.Equal(t, map[uuid.UUID]bool{endsToday.ID: true, startsToday.ID: true}, ids(entity.AllocationScopeActive))
	assert.Equal(t, map[uuid.UUID]bool{upcoming.ID: true}, ids(entity.AllocationScopeUpcoming))
	assert.Equal(t, map[uuid.UUID]bool{past.ID: true}, ids(entity.AllocationScopePast))
	all := map[uuid.UUID]bool{endsToday.ID: true, startsToday.ID: true, upcoming.ID: true, past.ID: true}
	assert.Equal(t, all, ids(entity.AllocationScopeAll))
	assert.Equal(t, all, ids(""), "empty defaults to all")

	_, err := lp.Execute(ctx, e.tenant, e.p1.ID, "bogus")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
	assert.ErrorIs(t, err, ErrInvalidScope)
	_, err = lp.Execute(ctx, uuid.New(), e.p1.ID, entity.AllocationScopeActive)
	assert.ErrorIs(t, err, domainerrors.ErrProjectNotFound, "other tenant")
}
