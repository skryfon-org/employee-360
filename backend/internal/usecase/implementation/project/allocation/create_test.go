package allocation

import (
	"context"
	"testing"
	"time"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAllocation_OverAllocationIsWarnOnly(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()

	r1, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 60, "2026-01-01", nil))
	require.NoError(t, err)
	assert.False(t, r1.OverAllocated)
	assert.Equal(t, 60, r1.UserTotalPercent)

	r2, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p2, 70, "2026-02-01", nil))
	require.NoError(t, err, "total above 100 must not be rejected")
	assert.True(t, r2.OverAllocated)
	assert.Equal(t, 130, r2.UserTotalPercent)

	list, err := NewListUserAllocationsUseCase(e.users, e.allocs).Execute(ctx, e.tenant, e.user, "")
	require.NoError(t, err)
	assert.Equal(t, 130, list.TotalPercent)
	assert.True(t, list.OverAllocated)
	assert.Len(t, list.Allocations, 2)
}

func TestCreateAllocation_FutureAllocationJudgedAtItsStart(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()
	_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 80, "2026-01-01", nil))
	require.NoError(t, err)
	r, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p2, 50, "2026-09-01", nil))
	require.NoError(t, err)
	assert.True(t, r.OverAllocated)
	assert.Equal(t, 130, r.UserTotalPercent)
}

func TestCreateAllocation_Validation(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()

	for _, pct := range []int{-5, 0, 101, 1000} {
		_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, pct, "2026-01-01", nil))
		assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData, "percent %d", pct)
	}
	for i, pct := range []int{1, 100} {
		in := e.in(e.p1, pct, []string{"2026-01-01", "2026-02-01"}[i], dp([]string{"2026-01-02", "2026-02-02"}[i]))
		_, err := uc.Execute(ctx, e.tenant, e.actor, in)
		assert.NoError(t, err, "percent %d", pct)
	}
	_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 50, "2026-03-01", dp("2026-02-28")))
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData, "end before start")
	_, err = uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 50, "2026-03-01", dp("2026-03-01")))
	assert.NoError(t, err, "end == start is a one-day allocation")

	in := e.in(e.p1, 50, "2026-01-01", nil)
	in.UserID = uuid.Nil
	_, err = uc.Execute(ctx, e.tenant, e.actor, in)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
}

func TestCreateAllocation_TenantAndUserChecks(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()

	_, err := uc.Execute(ctx, uuid.New(), e.actor, e.in(e.p1, 50, "2026-01-01", nil))
	assert.ErrorIs(t, err, domainerrors.ErrProjectNotFound, "project of another tenant")

	in := e.in(e.p1, 50, "2026-01-01", nil)
	in.UserID = uuid.New()
	_, err = uc.Execute(ctx, e.tenant, e.actor, in)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound, "user outside tenant")

	in = e.in(e.p1, 50, "2026-01-01", nil)
	in.UserID = e.inactive
	_, err = uc.Execute(ctx, e.tenant, e.actor, in)
	assert.ErrorIs(t, err, domainerrors.ErrAllocationUserInactive)
	assert.Empty(t, e.allocs.rows)
}

func TestCreateAllocation_Overlap(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()
	_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 50, "2026-01-01", dp("2026-03-31")))
	require.NoError(t, err)

	_, err = uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 50, "2026-03-31", nil))
	assert.ErrorIs(t, err, domainerrors.ErrAllocationOverlap)
	_, err = uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 50, "2026-02-01", nil))
	assert.ErrorIs(t, err, domainerrors.ErrAllocationOverlap, "open-ended range overlapping")
	_, err = uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 50, "2026-04-01", nil))
	assert.NoError(t, err, "adjacent range is fine")
	_, err = uc.Execute(ctx, e.tenant, e.actor, e.in(e.p2, 50, "2026-01-01", nil))
	assert.NoError(t, err, "same dates on another project is fine")
}

func TestCreateAllocation_RejectsArchivedProjectOnly(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()

	e.setStatus(e.p1, entity.ProjectStatusArchived)
	_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 50, "2026-01-01", nil))
	assert.ErrorIs(t, err, domainerrors.ErrProjectArchived)
	assert.Empty(t, e.allocs.rows)

	for i, st := range []string{entity.ProjectStatusActive, entity.ProjectStatusOnHold, entity.ProjectStatusCompleted} {
		e.setStatus(e.p1, st)
		start := []string{"2026-01-01", "2026-02-01", "2026-03-01"}[i]
		_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 10, start, dp(start)))
		assert.NoError(t, err, st)
	}
}

func TestCreateAllocation_TotalCountsOnlyLiveProjects(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()

	// 90% on a project that then completes must not count toward the next allocation.
	_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p1, 90, "2026-01-01", nil))
	require.NoError(t, err)
	e.setStatus(e.p1, entity.ProjectStatusCompleted)
	r, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p2, 50, "2026-01-01", nil))
	require.NoError(t, err)
	assert.Equal(t, 50, r.UserTotalPercent)
	assert.False(t, r.OverAllocated)

	// on_hold still counts.
	e.setStatus(e.p1, entity.ProjectStatusOnHold)
	up := NewUpdateAllocationUseCase(e.projects, e.allocs, e.rec, nil)
	r, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: r.Allocation.ID, AllocationPercent: opt(60)})
	require.NoError(t, err)
	assert.Equal(t, 150, r.UserTotalPercent)
	assert.True(t, r.OverAllocated)
}

func TestCreateAllocation_ProjectWindow(t *testing.T) {
	e := newAllocEnv(t)
	uc := e.create()
	ctx := context.Background()
	fixed := e.p1
	fixed.StartDate, fixed.EndDate = d("2026-10-24"), dp("2026-10-29")
	require.NoError(t, e.projects.Update(ctx, e.tenant, e.actor, fixed))

	cases := []struct {
		name  string
		start string
		end   *time.Time
		ok    bool
	}{
		{"before project start", "2026-10-15", dp("2026-10-26"), false},
		{"ends after project end", "2026-10-25", dp("2026-10-30"), false},
		{"starts after project end", "2026-10-30", dp("2026-10-30"), false},
		{"exactly the window", "2026-10-24", dp("2026-10-29"), true},
	}
	for _, c := range cases {
		_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(fixed, 10, c.start, c.end))
		if c.ok {
			assert.NoError(t, err, c.name)
		} else {
			assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData, c.name)
		}
	}

	// Open-ended allocation is fine when the project has no end date.
	_, err := uc.Execute(ctx, e.tenant, e.actor, e.in(e.p2, 10, "2026-03-01", nil))
	assert.NoError(t, err)
	_, err = uc.Execute(ctx, e.tenant, e.actor, e.in(e.p2, 10, "2025-12-31", nil))
	assert.ErrorIs(t, err, ErrAllocationBeforeProjectStart)

	// Open-ended allocation is also allowed on a project that has an end date.
	p3 := e.p1
	p3.EndDate = dp("2027-12-31")
	require.NoError(t, e.projects.Update(ctx, e.tenant, e.actor, p3))
	_, err = uc.Execute(ctx, e.tenant, e.actor, e.in(p3, 10, "2027-01-01", nil))
	assert.NoError(t, err)
}
