package entity

import (
	"time"

	"github.com/google/uuid"
)

// Project statuses. Projects are never hard-deleted; they move to archived.
const (
	ProjectStatusActive    = "active"
	ProjectStatusOnHold    = "on_hold"
	ProjectStatusCompleted = "completed"
	ProjectStatusArchived  = "archived"
)

// ProjectStatuses lists the allowed project statuses.
var ProjectStatuses = []string{ProjectStatusActive, ProjectStatusOnHold, ProjectStatusCompleted, ProjectStatusArchived}

// Project is a tenant-scoped project employees can be allocated to.
// Date fields are calendar dates (UTC midnight).
type Project struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Description string     `json:"description"`
	ClientName  *string    `json:"client_name,omitempty"`
	Status      string     `json:"status"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// ProjectAllocation allocates a user to a project at a percentage for a date
// range. EndDate nil means open-ended.
type ProjectAllocation struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	ProjectID         uuid.UUID  `json:"project_id"`
	UserID            uuid.UUID  `json:"user_id"`
	AllocationPercent int        `json:"allocation_percent"`
	RoleOnProject     *string    `json:"role_on_project,omitempty"`
	StartDate         time.Time  `json:"start_date"`
	EndDate           *time.Time `json:"end_date,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"-"`
	UpdatedBy *uuid.UUID `json:"-"`
	DeletedBy *uuid.UUID `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// ActiveOn reports whether the allocation covers the given calendar date.
func (a *ProjectAllocation) ActiveOn(d time.Time) bool {
	if a.StartDate.After(d) {
		return false
	}
	return a.EndDate == nil || !a.EndDate.Before(d)
}

// AllocationScope selects which of a user's allocations a read returns,
// relative to a reference date.
type AllocationScope string

// Allocation scopes for the /me/projects read.
const (
	AllocationScopeActive   AllocationScope = "active"   // ActiveOn(today)
	AllocationScopeUpcoming AllocationScope = "upcoming" // start_date > today
	AllocationScopePast     AllocationScope = "past"     // end_date < today
	AllocationScopeAll      AllocationScope = "all"      // every non-deleted allocation
)

// ParseAllocationScope maps a raw value to a scope; empty means active.
func ParseAllocationScope(s string) (AllocationScope, bool) {
	switch AllocationScope(s) {
	case "", AllocationScopeActive:
		return AllocationScopeActive, true
	case AllocationScopeUpcoming, AllocationScopePast, AllocationScopeAll:
		return AllocationScope(s), true
	}
	return "", false
}

// AllocationWithProject is an allocation joined with its project's identity,
// used by the per-employee and /me read models.
type AllocationWithProject struct {
	ProjectAllocation
	ProjectName   string `json:"project_name"`
	ProjectCode   string `json:"project_code"`
	ProjectStatus string `json:"project_status"`
}

// AllocationWithUser is an allocation joined with the allocated user's name,
// used by the per-project read model. The user may be soft-deleted.
type AllocationWithUser struct {
	ProjectAllocation
	UserFirstName string `json:"user_first_name"`
	UserLastName  string `json:"user_last_name"`
}

// UserName returns the allocated user's full name (first and last, space
// separated, same convention as User.FullName).
func (a AllocationWithUser) UserName() string {
	u := User{FirstName: a.UserFirstName, LastName: a.UserLastName}
	return u.FullName()
}

// AllocationScopeCounts holds per-scope allocation counts for one user as of a
// date, using the same definitions as the corresponding list scopes.
type AllocationScopeCounts struct {
	Active   int
	Upcoming int
	Past     int
}

// CountsTowardTotal reports whether this allocation contributes to the user's
// total allocation percent (its project is live).
func (a AllocationWithProject) CountsTowardTotal() bool {
	return IsLiveProjectStatus(a.ProjectStatus)
}

// IsLiveProjectStatus reports whether allocations on a project with this status
// count toward a user's workload (active or on_hold). Completed and archived
// projects no longer do.
func IsLiveProjectStatus(status string) bool {
	return status == ProjectStatusActive || status == ProjectStatusOnHold
}

// MaxTotalAllocationPercent is the point above which a user is flagged as
// over-allocated. It is advisory only and never rejects a write.
const MaxTotalAllocationPercent = 100
