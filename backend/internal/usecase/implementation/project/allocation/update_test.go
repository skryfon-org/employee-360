package allocation

import (
	"context"
	"testing"

	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAllocation(t *testing.T) {
	e := newAllocEnv(t)
	ctx := context.Background()
	r, err := e.create().Execute(ctx, e.tenant, e.actor, e.in(e.p1, 60, "2026-01-01", nil))
	require.NoError(t, err)
	_, err = e.create().Execute(ctx, e.tenant, e.actor, e.in(e.p2, 60, "2026-01-01", nil))
	require.NoError(t, err)
	up := NewUpdateAllocationUseCase(e.projects, e.allocs, e.rec, nil)
	id := r.Allocation.ID

	res, err := up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, AllocationPercent: opt(90)})
	require.NoError(t, err)
	assert.True(t, res.OverAllocated)
	assert.Equal(t, 150, res.UserTotalPercent)

	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, AllocationPercent: opt(101)})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, AllocationPercent: opt(0)})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, EndDate: optDate("2025-12-31")})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData, "end before start")
	_, err = up.Execute(ctx, uuid.New(), e.actor, prjtypes.UpdateAllocationInput{ID: id, AllocationPercent: opt(10)})
	assert.ErrorIs(t, err, domainerrors.ErrAllocationNotFound, "cross-tenant")

	// Ending the allocation removes it from the active total.
	res, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, EndDate: optDate("2026-06-01")})
	require.NoError(t, err)
	require.NotNil(t, res.Allocation.EndDate)

	// Updating cannot create an overlap with a sibling allocation.
	r2, err := e.create().Execute(ctx, e.tenant, e.actor, e.in(e.p1, 10, "2026-07-01", nil))
	require.NoError(t, err)
	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: r2.Allocation.ID, StartDate: optDate("2026-05-01")})
	assert.ErrorIs(t, err, domainerrors.ErrAllocationOverlap)
}

func TestUpdateAllocation_ProjectWindow(t *testing.T) {
	e := newAllocEnv(t)
	ctx := context.Background()
	p := e.p1
	p.StartDate, p.EndDate = d("2026-10-24"), dp("2026-10-29")
	require.NoError(t, e.projects.Update(ctx, e.tenant, e.actor, p))
	r, err := e.create().Execute(ctx, e.tenant, e.actor, e.in(p, 50, "2026-10-25", dp("2026-10-27")))
	require.NoError(t, err)
	up := NewUpdateAllocationUseCase(e.projects, e.allocs, e.rec, nil)
	id := r.Allocation.ID

	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, StartDate: optDate("2026-10-15")})
	assert.ErrorIs(t, err, ErrAllocationBeforeProjectStart)
	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, EndDate: optDate("2026-11-01")})
	assert.ErrorIs(t, err, ErrAllocationAfterProjectEnd)
	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, EndDate: emptypes.Optional[emptypes.Date]{Set: true, Null: true}})
	assert.NoError(t, err, "clearing end_date (open-ended) is allowed")
	_, err = up.Execute(ctx, e.tenant, e.actor, prjtypes.UpdateAllocationInput{ID: id, EndDate: optDate("2026-10-29")})
	assert.NoError(t, err)
}
