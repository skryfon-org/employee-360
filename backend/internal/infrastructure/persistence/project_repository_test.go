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

func newProject(code string) *entity.Project {
	return &entity.Project{Name: "Proj " + code, Code: code, Status: entity.ProjectStatusActive, StartDate: day(2026, 1, 1)}
}

func TestProjectRepository_TenantScopingUniquenessAndFilters(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormProjectRepository(e.db)
	actor := uuid.New()

	pA := newProject("ALPHA")
	pA.ClientName = str("Acme")
	require.NoError(t, repo.Create(c, e.tA.ID, actor, pA))
	require.Equal(t, e.tA.ID, pA.TenantID)

	// Code uniqueness is per tenant.
	assert.ErrorIs(t, repo.Create(c, e.tA.ID, actor, newProject("ALPHA")), domainerrors.ErrProjectCodeTaken)
	pB := newProject("ALPHA")
	require.NoError(t, repo.Create(c, e.tB.ID, actor, pB))

	// Cross-tenant reads are not found.
	_, err := repo.GetByID(c, e.tB.ID, pA.ID)
	assert.ErrorIs(t, err, domainerrors.ErrProjectNotFound)
	got, err := repo.GetByID(c, e.tA.ID, pA.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme", *got.ClientName)
	taken, _ := repo.ExistsByCode(c, e.tA.ID, "ALPHA")
	assert.True(t, taken)
	taken, _ = repo.ExistsByCode(c, e.tA.ID, "NOPE")
	assert.False(t, taken)

	// Cross-tenant update is a not-found, and does not touch the row.
	hijack := *pA
	hijack.Name = "hijacked"
	assert.ErrorIs(t, repo.Update(c, e.tB.ID, actor, &hijack), domainerrors.ErrProjectNotFound)
	got, _ = repo.GetByID(c, e.tA.ID, pA.ID)
	assert.NotEqual(t, "hijacked", got.Name)

	// Filters, search (with LIKE metacharacters) and pagination.
	p2 := newProject("BETA")
	p2.Status = entity.ProjectStatusOnHold
	require.NoError(t, repo.Create(c, e.tA.ID, actor, p2))
	list, total, err := repo.List(c, e.tA.ID, repository.ProjectFilter{}, 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, list, 2)
	_, total, _ = repo.List(c, e.tA.ID, repository.ProjectFilter{Status: "on_hold"}, 10, 0)
	assert.EqualValues(t, 1, total)
	_, total, _ = repo.List(c, e.tA.ID, repository.ProjectFilter{Search: "acme"}, 10, 0)
	assert.EqualValues(t, 1, total, "search matches client name case-insensitively")
	_, total, _ = repo.List(c, e.tA.ID, repository.ProjectFilter{Search: "%"}, 10, 0)
	assert.EqualValues(t, 0, total, "wildcards are escaped")
	list, total, _ = repo.List(c, e.tA.ID, repository.ProjectFilter{}, 1, 1)
	assert.EqualValues(t, 2, total)
	assert.Len(t, list, 1)

	// Status change + code clash on update.
	got.Status = entity.ProjectStatusArchived
	require.NoError(t, repo.Update(c, e.tA.ID, actor, got))
	got.Code = "BETA"
	assert.ErrorIs(t, repo.Update(c, e.tA.ID, actor, got), domainerrors.ErrProjectCodeTaken)

	// DB-level date-order CHECK.
	bad := newProject("BADDATES")
	end := day(2025, 12, 31)
	bad.EndDate = &end
	assert.Error(t, repo.Create(c, e.tA.ID, actor, bad))

	// DB-level status CHECK.
	bad = newProject("BADSTATUS")
	bad.Status = "bogus"
	assert.Error(t, repo.Create(c, e.tA.ID, actor, bad))
}

func TestProjectAllocationRepository_TenantScopingOverlapAndBounds(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	projects := NewGormProjectRepository(e.db)
	repo := NewGormProjectAllocationRepository(e.db)
	actor := uuid.New()

	pA := newProject("ALLOC")
	require.NoError(t, projects.Create(c, e.tA.ID, actor, pA))
	pA2 := newProject("ALLOC2")
	require.NoError(t, projects.Create(c, e.tA.ID, actor, pA2))

	end := day(2026, 3, 31)
	a1 := &entity.ProjectAllocation{ProjectID: pA.ID, UserID: e.uA.ID, AllocationPercent: 60, StartDate: day(2026, 1, 1), EndDate: &end}
	require.NoError(t, repo.Create(c, e.tA.ID, actor, a1))
	a2 := &entity.ProjectAllocation{ProjectID: pA2.ID, UserID: e.uA.ID, AllocationPercent: 70, StartDate: day(2026, 2, 1)}
	require.NoError(t, repo.Create(c, e.tA.ID, actor, a2))

	// Cross-tenant isolation.
	_, err := repo.GetByID(c, e.tB.ID, a1.ID)
	assert.ErrorIs(t, err, domainerrors.ErrAllocationNotFound)
	other := *a1
	other.AllocationPercent = 5
	assert.ErrorIs(t, repo.Update(c, e.tB.ID, actor, &other), domainerrors.ErrAllocationNotFound)
	assert.ErrorIs(t, repo.SoftDelete(c, e.tB.ID, a1.ID, actor), domainerrors.ErrAllocationNotFound)
	lst, err := repo.ListByProject(c, e.tB.ID, pA.ID, entity.AllocationScopeAll, day(2026, 2, 15))
	require.NoError(t, err)
	assert.Empty(t, lst)
	for _, sc := range []entity.AllocationScope{entity.AllocationScopeActive, entity.AllocationScopeUpcoming, entity.AllocationScopePast} {
		lst, err = repo.ListByProject(c, e.tB.ID, pA.ID, sc, day(2026, 2, 15))
		require.NoError(t, err)
		assert.Empty(t, lst, "tenant B sees nothing in scope %s", sc)
	}
	// State filter in tenant A (project ALLOC has only a1: Jan 1 - Mar 31).
	for sc, want := range map[entity.AllocationScope]int{
		entity.AllocationScopeAll: 1, entity.AllocationScopeActive: 1, entity.AllocationScopeUpcoming: 0, entity.AllocationScopePast: 0,
	} {
		lst, err = repo.ListByProject(c, e.tA.ID, pA.ID, sc, day(2026, 2, 15))
		require.NoError(t, err)
		assert.Len(t, lst, want, "scope %s on 2026-02-15", sc)
	}
	lst, _ = repo.ListByProject(c, e.tA.ID, pA.ID, entity.AllocationScopePast, day(2026, 4, 1))
	assert.Len(t, lst, 1)
	// The user's name is joined in, and still resolves once the user is soft-deleted.
	assert.Equal(t, "Test User", lst[0].UserName())
	require.NoError(t, e.db.Exec("UPDATE users SET deleted_at = now() WHERE id = ?", e.uA.ID).Error)
	lst, err = repo.ListByProject(c, e.tA.ID, pA.ID, entity.AllocationScopeAll, day(2026, 2, 15))
	require.NoError(t, err)
	require.Len(t, lst, 1)
	assert.Equal(t, "Test User", lst[0].UserName())
	require.NoError(t, e.db.Exec("UPDATE users SET deleted_at = NULL WHERE id = ?", e.uA.ID).Error)
	lst, _ = repo.ListByProject(c, e.tA.ID, pA.ID, entity.AllocationScopeUpcoming, day(2025, 12, 31))
	assert.Len(t, lst, 1)
	lst, _ = repo.ListByProject(c, e.tA.ID, pA.ID, "bogus", day(2026, 2, 15))
	assert.Empty(t, lst)
	act, err := repo.ListByUserScope(c, e.tB.ID, e.uA.ID, entity.AllocationScopeActive, day(2026, 2, 15))
	require.NoError(t, err)
	assert.Empty(t, act)

	// Active view: both active mid-Feb (130% total, no cap), only a2 after a1 ends.
	act, err = repo.ListByUserScope(c, e.tA.ID, e.uA.ID, entity.AllocationScopeActive, day(2026, 2, 15))
	require.NoError(t, err)
	require.Len(t, act, 2)
	assert.Equal(t, "Proj ALLOC", act[0].ProjectName)
	assert.Equal(t, "ALLOC", act[0].ProjectCode)
	assert.Equal(t, entity.ProjectStatusActive, act[0].ProjectStatus, "joined read exposes project status")
	assert.Equal(t, 130, act[0].AllocationPercent+act[1].AllocationPercent)
	act, _ = repo.ListByUserScope(c, e.tA.ID, e.uA.ID, entity.AllocationScopeActive, day(2026, 4, 1))
	require.Len(t, act, 1)
	assert.Equal(t, a2.ID, act[0].ID)
	act, _ = repo.ListByUserScope(c, e.tA.ID, e.uA.ID, entity.AllocationScopeActive, day(2025, 12, 31))
	assert.Empty(t, act, "not started yet")

	// Scoped reads: a1 = Jan1-Mar31 (pA), a2 = Feb1-open (pA2).
	sc := func(tid uuid.UUID, scope entity.AllocationScope, asOf time.Time) []*entity.AllocationWithProject {
		l, err := repo.ListByUserScope(c, tid, e.uA.ID, scope, asOf)
		require.NoError(t, err)
		return l
	}
	assert.Empty(t, sc(e.tB.ID, entity.AllocationScopeAll, day(2026, 2, 15)), "scoped reads are tenant scoped")
	assert.Len(t, sc(e.tA.ID, entity.AllocationScopeActive, day(2026, 3, 31)), 2, "end date inclusive")
	assert.Len(t, sc(e.tA.ID, entity.AllocationScopeActive, day(2026, 4, 1)), 1, "open-ended stays active")
	up := sc(e.tA.ID, entity.AllocationScopeUpcoming, day(2026, 1, 15))
	require.Len(t, up, 1)
	assert.Equal(t, a2.ID, up[0].ID)
	assert.Empty(t, sc(e.tA.ID, entity.AllocationScopeUpcoming, day(2026, 2, 1)), "start == today is active, not upcoming")
	past := sc(e.tA.ID, entity.AllocationScopePast, day(2026, 4, 1))
	require.Len(t, past, 1)
	assert.Equal(t, a1.ID, past[0].ID)
	assert.Equal(t, "ALLOC", past[0].ProjectCode)
	assert.Empty(t, sc(e.tA.ID, entity.AllocationScopePast, day(2026, 3, 31)), "end == today is active, not past")
	all := sc(e.tA.ID, entity.AllocationScopeAll, day(2026, 4, 1))
	require.Len(t, all, 2)
	assert.Equal(t, a1.ID, all[0].ID, "ordered by start_date")
	assert.Equal(t, a2.ID, all[1].ID)

	// Per-scope counts (one query): a1 Jan1-Mar31 on pA, a2 Feb1-open on pA2.
	cnt := func(tid uuid.UUID, asOf time.Time) entity.AllocationScopeCounts {
		n, err := repo.CountByUserScopes(c, tid, e.uA.ID, asOf)
		require.NoError(t, err)
		return n
	}
	assert.Equal(t, entity.AllocationScopeCounts{Active: 2}, cnt(e.tA.ID, day(2026, 2, 15)))
	assert.Equal(t, entity.AllocationScopeCounts{Active: 1, Upcoming: 1}, cnt(e.tA.ID, day(2026, 1, 15)))
	assert.Equal(t, entity.AllocationScopeCounts{Active: 1, Past: 1}, cnt(e.tA.ID, day(2026, 4, 1)))
	assert.Equal(t, entity.AllocationScopeCounts{}, cnt(e.tB.ID, day(2026, 2, 15)), "tenant scoped")
	require.NoError(t, e.db.Exec("UPDATE projects SET status = 'archived' WHERE id = ?", pA2.ID).Error)
	assert.Equal(t, entity.AllocationScopeCounts{Active: 1}, cnt(e.tA.ID, day(2026, 2, 15)), "archived project not in active")
	assert.Equal(t, entity.AllocationScopeCounts{Active: 0, Past: 1}, cnt(e.tA.ID, day(2026, 4, 1)))

	// Picker aggregate on the employee directory (archived pA2 excluded).
	dir := NewGormEmployeeDirectoryRepository(e.db)
	alloc := func(tid uuid.UUID, asOf time.Time) map[uuid.UUID]*int {
		rows, _, err := dir.List(c, tid, entity.EmployeeFilter{IncludeAllocation: true, AllocationAsOf: asOf}, 50, 0)
		require.NoError(t, err)
		m := map[uuid.UUID]*int{}
		for _, r := range rows {
			m[r.UserID] = r.AllocatedPercent
		}
		return m
	}
	m := alloc(e.tA.ID, day(2026, 2, 15))
	require.NotNil(t, m[e.uA.ID])
	assert.Equal(t, 60, *m[e.uA.ID])
	assert.Equal(t, 0, *alloc(e.tA.ID, day(2026, 4, 1))[e.uA.ID])
	rows, _, err := dir.List(c, e.tA.ID, entity.EmployeeFilter{}, 50, 0)
	require.NoError(t, err)
	for _, r := range rows {
		assert.Nil(t, r.AllocatedPercent, "omitted unless requested")
	}
	_, ok := alloc(e.tB.ID, day(2026, 2, 15))[e.uA.ID]
	assert.False(t, ok, "other tenant never sees the user")
	require.NoError(t, e.db.Exec("UPDATE projects SET status = 'active' WHERE id = ?", pA2.ID).Error)

	// Overlap detection (same project + user only).
	ov, err := repo.HasOverlap(c, e.tA.ID, pA.ID, e.uA.ID, day(2026, 3, 31), nil, uuid.Nil)
	require.NoError(t, err)
	assert.True(t, ov)
	ov, _ = repo.HasOverlap(c, e.tA.ID, pA.ID, e.uA.ID, day(2026, 4, 1), nil, uuid.Nil)
	assert.False(t, ov)
	ov, _ = repo.HasOverlap(c, e.tA.ID, pA.ID, e.uA.ID, day(2025, 1, 1), ptr(day(2025, 12, 31)), uuid.Nil)
	assert.False(t, ov)
	ov, _ = repo.HasOverlap(c, e.tA.ID, pA.ID, e.uA.ID, day(2026, 3, 1), nil, a1.ID)
	assert.False(t, ov, "own row excluded")
	ov, _ = repo.HasOverlap(c, e.tB.ID, pA.ID, e.uA.ID, day(2026, 3, 1), nil, uuid.Nil)
	assert.False(t, ov, "other tenant sees nothing")

	// Update + soft delete.
	a1.AllocationPercent = 40
	require.NoError(t, repo.Update(c, e.tA.ID, actor, a1))
	got, _ := repo.GetByID(c, e.tA.ID, a1.ID)
	assert.Equal(t, 40, got.AllocationPercent)
	require.NoError(t, repo.SoftDelete(c, e.tA.ID, a1.ID, actor))
	_, err = repo.GetByID(c, e.tA.ID, a1.ID)
	assert.ErrorIs(t, err, domainerrors.ErrAllocationNotFound)
	ov, _ = repo.HasOverlap(c, e.tA.ID, pA.ID, e.uA.ID, day(2026, 3, 1), nil, uuid.Nil)
	assert.False(t, ov, "soft-deleted rows do not block")

	// DB-level CHECK constraints: percent bounds and date order.
	for _, pct := range []int{0, 101, -1} {
		bad := &entity.ProjectAllocation{ProjectID: pA2.ID, UserID: e.uA.ID, AllocationPercent: pct, StartDate: day(2027, 1, 1)}
		assert.Error(t, repo.Create(c, e.tA.ID, actor, bad), "percent %d", pct)
	}
	for _, pct := range []int{1, 100} {
		ok := &entity.ProjectAllocation{ProjectID: pA2.ID, UserID: e.uA.ID, AllocationPercent: pct, StartDate: day(2027, time.Month(pct%12+1), 1)}
		assert.NoError(t, repo.Create(c, e.tA.ID, actor, ok), "percent %d", pct)
	}
	bad := &entity.ProjectAllocation{ProjectID: pA2.ID, UserID: e.uA.ID, AllocationPercent: 10, StartDate: day(2028, 1, 1), EndDate: ptr(day(2027, 12, 31))}
	assert.Error(t, repo.Create(c, e.tA.ID, actor, bad))
}

func ptr[T any](v T) *T { return &v }

// The DB itself (not just the repository) must reject an allocation whose
// project or user belongs to a different tenant than the allocation row.
func TestProjectAllocations_DBRejectsCrossTenantReferences(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	projects := NewGormProjectRepository(e.db)
	actor := uuid.New()

	pA := newProject("XT-A")
	require.NoError(t, projects.Create(c, e.tA.ID, actor, pA))
	pB := newProject("XT-B")
	require.NoError(t, projects.Create(c, e.tB.ID, actor, pB))

	insert := func(tenantID, projectID, userID uuid.UUID) error {
		return e.db.WithContext(c).Exec(
			`INSERT INTO project_allocations (tenant_id, project_id, user_id, allocation_percent, start_date)
			 VALUES (?, ?, ?, 50, '2026-01-01')`, tenantID, projectID, userID).Error
	}

	assert.Error(t, insert(e.tA.ID, pB.ID, e.uA.ID), "tenant A allocation referencing tenant B project")
	assert.Error(t, insert(e.tA.ID, pA.ID, e.uB.ID), "tenant A allocation referencing tenant B user")
	assert.Error(t, insert(e.tB.ID, pA.ID, e.uB.ID), "tenant B allocation referencing tenant A project")
	assert.Error(t, insert(e.tA.ID, pB.ID, e.uB.ID), "both references in the other tenant")
	assert.NoError(t, insert(e.tA.ID, pA.ID, e.uA.ID), "same-tenant references are accepted")
}
