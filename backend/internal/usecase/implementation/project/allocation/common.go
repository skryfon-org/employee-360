// Package allocation implements the project-allocation usecases. All
// repository calls take the explicit tenantID resolved by the handler from the
// auth context; nothing here reads identity from context.Context.
package allocation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	projectcommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/project"
)

// now is the clock; replaced in tests.
var now = time.Now

// today returns the current calendar date at UTC midnight.
func today() time.Time {
	n := now().UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// Validation sentinels (each matches domainerrors.ErrInvalidProjectData).
var (
	ErrUserIDRequired      = projectcommon.Required("user_id")
	ErrPercentOutOfRange   = projectcommon.Invalid("allocation_percent: must be between 1 and 100")
	ErrRoleTooLong         = projectcommon.TooLong("role_on_project", 100)
	ErrInvalidScope        = projectcommon.Invalid("scope: must be one of active, upcoming, past, all") // also used for the project-allocations state filter
	ErrAllocationDateOrder = projectcommon.Invalid("end_date: must not be before start_date")
)

// Project-window validation sentinels (each matches domainerrors.ErrInvalidProjectData).
var (
	ErrAllocationBeforeProjectStart = projectcommon.Invalid("start_date: must not be before the project start date")
	ErrAllocationAfterProjectEnd    = projectcommon.Invalid("end_date: must not be after the project end date")
	ErrAllocationStartAfterProject  = projectcommon.Invalid("start_date: must not be after the project end date")
)

// validateWithinProject checks the allocation's dates against the project's
// window. end is optional (nil = open-ended); when set on a project with an end
// date it must not fall after it.
func validateWithinProject(p *entity.Project, start time.Time, end *time.Time) error {
	if start.Before(p.StartDate) {
		return ErrAllocationBeforeProjectStart
	}
	if p.EndDate == nil {
		return nil
	}
	if start.After(*p.EndDate) {
		return ErrAllocationStartAfterProject
	}
	if end != nil && end.After(*p.EndDate) {
		return ErrAllocationAfterProjectEnd
	}
	return nil
}

// liveOnly keeps allocations on live projects (active or on_hold).
func liveOnly(items []*entity.AllocationWithProject) []*entity.AllocationWithProject {
	out := make([]*entity.AllocationWithProject, 0, len(items))
	for _, a := range items {
		if entity.IsLiveProjectStatus(a.ProjectStatus) {
			out = append(out, a)
		}
	}
	return out
}

// sumPercent totals only allocations on live projects, so completed and
// archived projects never contribute to over-allocation.
func sumPercent(items []*entity.AllocationWithProject) int {
	total := 0
	for _, a := range liveOnly(items) {
		total += a.AllocationPercent
	}
	return total
}

// liveTotalOn sums the percent of allocations that cover ref and sit on live
// projects, so upcoming/past items are listed but never counted.
func liveTotalOn(items []*entity.AllocationWithProject, ref time.Time) int {
	total := 0
	for _, a := range items {
		if entity.IsLiveProjectStatus(a.ProjectStatus) && a.ActiveOn(ref) {
			total += a.AllocationPercent
		}
	}
	return total
}

// totalFor computes the user's total allocation on the reference date for a
// just-written allocation: today when the allocation covers today, otherwise
// its start date (so a future allocation is judged against what overlaps it).
// The result is advisory only.
func totalFor(c context.Context, repo repository.ProjectAllocationRepository, tenantID uuid.UUID, a *entity.ProjectAllocation) (int, error) {
	ref := today()
	if !a.ActiveOn(ref) {
		ref = a.StartDate
	}
	items, err := repo.ListByUserScope(c, tenantID, a.UserID, entity.AllocationScopeActive, ref)
	if err != nil {
		return 0, err
	}
	return sumPercent(items), nil
}
