// Package project holds API DTOs and usecase inputs/results for projects and
// project allocations. Tenant identity is never part of a payload.
package project

import (
	"time"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
)

// CreateProjectRequest is the body of POST /projects.
type CreateProjectRequest struct {
	Name        string         `json:"name" binding:"required"`
	Code        string         `json:"code"` // optional; generated from name when blank
	Description string         `json:"description"`
	ClientName  *string        `json:"client_name"`
	Status      string         `json:"status" enums:"active,on_hold,completed,archived"` // defaults to active
	StartDate   emptypes.Date  `json:"start_date" swaggertype:"string" format:"date"`
	EndDate     *emptypes.Date `json:"end_date" swaggertype:"string" format:"date"`
}

// UpdateProjectRequest is the body of PATCH /projects/:id. Absent fields are
// left unchanged; explicit null clears client_name / end_date.
type UpdateProjectRequest struct {
	Name        emptypes.Optional[string]        `json:"name" swaggertype:"string"`
	Code        emptypes.Optional[string]        `json:"code" swaggertype:"string"`
	Description emptypes.Optional[string]        `json:"description" swaggertype:"string"`
	ClientName  emptypes.Optional[string]        `json:"client_name" swaggertype:"string"`
	Status      emptypes.Optional[string]        `json:"status" swaggertype:"string" enums:"active,on_hold,completed,archived"`
	StartDate   emptypes.Optional[emptypes.Date] `json:"start_date" swaggertype:"string" format:"date"`
	EndDate     emptypes.Optional[emptypes.Date] `json:"end_date" swaggertype:"string" format:"date"`
}

// ProjectCodeSuggestionResponse is the data of GET /projects/code-suggestion.
type ProjectCodeSuggestionResponse struct {
	Code string `json:"code"`
}

// CreateProjectInput is the validated-at-usecase input for creating a project.
type CreateProjectInput struct {
	Name        string
	Code        string
	Description string
	ClientName  *string
	Status      string
	StartDate   time.Time
	EndDate     *time.Time
}

// UpdateProjectInput patches a project.
type UpdateProjectInput struct {
	ID          uuid.UUID
	Name        emptypes.Optional[string]
	Code        emptypes.Optional[string]
	Description emptypes.Optional[string]
	ClientName  emptypes.Optional[string]
	Status      emptypes.Optional[string]
	StartDate   emptypes.Optional[emptypes.Date]
	EndDate     emptypes.Optional[emptypes.Date]
}

// ListProjectsQuery holds paging and filters for listing projects.
type ListProjectsQuery struct {
	Page     int
	PageSize int
	Search   string
	Status   string
}

// ListProjectsResult is a page of projects.
type ListProjectsResult struct {
	Projects []*entity.Project
	Total    int64
	Page     int
	PageSize int
}

// ProjectResponse is the project API model.
type ProjectResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	ClientName  *string   `json:"client_name,omitempty"`
	Status      string    `json:"status" enums:"active,on_hold,completed,archived"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToProjectResponse maps a project entity to its API model.
func ToProjectResponse(p *entity.Project) ProjectResponse {
	return ProjectResponse{
		ID: p.ID, Name: p.Name, Code: p.Code, Description: p.Description, ClientName: p.ClientName,
		Status: p.Status, StartDate: p.StartDate.UTC().Format(emptypes.DateLayout), EndDate: emptypes.FormatDate(p.EndDate),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// CreateAllocationRequest is the body of POST /projects/:id/allocations.
type CreateAllocationRequest struct {
	UserID            uuid.UUID      `json:"user_id" binding:"required"`
	AllocationPercent int            `json:"allocation_percent" minimum:"1" maximum:"100"`
	RoleOnProject     *string        `json:"role_on_project"`
	StartDate         emptypes.Date  `json:"start_date" swaggertype:"string" format:"date"`
	EndDate           *emptypes.Date `json:"end_date" swaggertype:"string" format:"date"`
}

// UpdateAllocationRequest is the body of PATCH /allocations/:id. Absent fields
// are unchanged; explicit null clears role_on_project / end_date.
type UpdateAllocationRequest struct {
	AllocationPercent emptypes.Optional[int]           `json:"allocation_percent" swaggertype:"integer" minimum:"1" maximum:"100"`
	RoleOnProject     emptypes.Optional[string]        `json:"role_on_project" swaggertype:"string"`
	StartDate         emptypes.Optional[emptypes.Date] `json:"start_date" swaggertype:"string" format:"date"`
	EndDate           emptypes.Optional[emptypes.Date] `json:"end_date" swaggertype:"string" format:"date"`
}

// CreateAllocationInput is the usecase input for creating an allocation.
type CreateAllocationInput struct {
	ProjectID         uuid.UUID
	UserID            uuid.UUID
	AllocationPercent int
	RoleOnProject     *string
	StartDate         time.Time
	EndDate           *time.Time
}

// UpdateAllocationInput patches an allocation.
type UpdateAllocationInput struct {
	ID                uuid.UUID
	AllocationPercent emptypes.Optional[int]
	RoleOnProject     emptypes.Optional[string]
	StartDate         emptypes.Optional[emptypes.Date]
	EndDate           emptypes.Optional[emptypes.Date]
}

// AllocationResult is the outcome of an allocation write: the allocation plus
// the user's total allocation on the reference date. OverAllocated is advisory
// (total > 100); it never causes a rejection.
type AllocationResult struct {
	Allocation       *entity.ProjectAllocation
	UserTotalPercent int
	OverAllocated    bool
}

// UserAllocationsResult is a user's active allocations with their total.
type UserAllocationsResult struct {
	Allocations   []*entity.AllocationWithProject
	TotalPercent  int
	OverAllocated bool
	Counts        *entity.AllocationScopeCounts // set by GET /me/projects only
}

// AllocationResponse is the allocation API model (without project info).
type AllocationResponse struct {
	ID                uuid.UUID `json:"id"`
	ProjectID         uuid.UUID `json:"project_id"`
	UserID            uuid.UUID `json:"user_id"`
	AllocationPercent int       `json:"allocation_percent"`
	RoleOnProject     *string   `json:"role_on_project,omitempty"`
	StartDate         string    `json:"start_date"`
	EndDate           *string   `json:"end_date,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ProjectAllocationResponse is an allocation on GET /projects/:id/allocations,
// carrying the allocated user's full name so clients need no per-user lookup.
type ProjectAllocationResponse struct {
	AllocationResponse
	// UserName is the allocated user's full name (first and last), also
	// resolved for soft-deleted users.
	UserName string `json:"user_name"`
}

// ToProjectAllocationResponse maps an allocation-with-user to its API model.
func ToProjectAllocationResponse(a *entity.AllocationWithUser) ProjectAllocationResponse {
	return ProjectAllocationResponse{AllocationResponse: ToAllocationResponse(&a.ProjectAllocation), UserName: a.UserName()}
}

// ToAllocationResponse maps an allocation entity to its API model.
func ToAllocationResponse(a *entity.ProjectAllocation) AllocationResponse {
	return AllocationResponse{
		ID: a.ID, ProjectID: a.ProjectID, UserID: a.UserID, AllocationPercent: a.AllocationPercent,
		RoleOnProject: a.RoleOnProject, StartDate: a.StartDate.UTC().Format(emptypes.DateLayout),
		EndDate: emptypes.FormatDate(a.EndDate), CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

// AllocationWriteResponse is returned by allocation create/update.
type AllocationWriteResponse struct {
	AllocationResponse
	UserTotalPercent int  `json:"user_total_percent"`
	OverAllocated    bool `json:"over_allocated"`
}

// ToAllocationWriteResponse maps a write result to its API model.
func ToAllocationWriteResponse(r *AllocationResult) AllocationWriteResponse {
	return AllocationWriteResponse{
		AllocationResponse: ToAllocationResponse(r.Allocation),
		UserTotalPercent:   r.UserTotalPercent,
		OverAllocated:      r.OverAllocated,
	}
}

// UserAllocationResponse is an allocation with its project identity.
type UserAllocationResponse struct {
	AllocationResponse
	ProjectName   string `json:"project_name"`
	ProjectCode   string `json:"project_code"`
	ProjectStatus string `json:"project_status" enums:"active,on_hold,completed,archived"`
	// CountsTowardTotal is true when this allocation is included in the
	// response's total_percent (its project is active or on_hold).
	CountsTowardTotal bool `json:"counts_toward_total"`
}

// UserAllocationsResponse is a user's active allocations with their total.
type UserAllocationsResponse struct {
	Allocations   []UserAllocationResponse `json:"allocations"`
	TotalPercent  int                      `json:"total_percent"`
	OverAllocated bool                     `json:"over_allocated"`
	// Counts is present on GET /me/projects only: the caller's allocation
	// counts per scope as of today, independent of the requested scope.
	Counts *ScopeCountsResponse `json:"counts,omitempty"`
}

// ScopeCountsResponse holds per-scope allocation counts.
type ScopeCountsResponse struct {
	Active   int `json:"active"`
	Upcoming int `json:"upcoming"`
	Past     int `json:"past"`
}

// ToUserAllocationsResponse maps a result to its API model.
func ToUserAllocationsResponse(r *UserAllocationsResult) UserAllocationsResponse {
	items := make([]UserAllocationResponse, 0, len(r.Allocations))
	for _, a := range r.Allocations {
		items = append(items, UserAllocationResponse{
			AllocationResponse: ToAllocationResponse(&a.ProjectAllocation),
			ProjectName:        a.ProjectName, ProjectCode: a.ProjectCode, ProjectStatus: a.ProjectStatus,
			CountsTowardTotal: a.CountsTowardTotal(),
		})
	}
	out := UserAllocationsResponse{Allocations: items, TotalPercent: r.TotalPercent, OverAllocated: r.OverAllocated}
	if r.Counts != nil {
		out.Counts = &ScopeCountsResponse{Active: r.Counts.Active, Upcoming: r.Counts.Upcoming, Past: r.Counts.Past}
	}
	return out
}
