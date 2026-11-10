package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	allocuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/allocation"
)

// ProjectAllocationUseCases groups the usecases behind the allocation endpoints.
type ProjectAllocationUseCases struct {
	ListByProject allocuc.ListProjectAllocationsUseCase
	Create        allocuc.CreateAllocationUseCase
	Update        allocuc.UpdateAllocationUseCase
	Delete        allocuc.DeleteAllocationUseCase
	ByUser        allocuc.ListUserAllocationsUseCase
	MyProjects    allocuc.ListMyProjectsUseCase
}

// ProjectAllocationHandler serves project allocations, an employee's allocations
// and /me/projects.
type ProjectAllocationHandler struct{ uc ProjectAllocationUseCases }

// NewProjectAllocationHandler constructs a ProjectAllocationHandler.
func NewProjectAllocationHandler(uc ProjectAllocationUseCases) *ProjectAllocationHandler {
	return &ProjectAllocationHandler{uc: uc}
}

// ListProjectAllocations lists a project's allocations. The optional state
// query param filters server-side relative to today (UTC): active (start_date
// <= today and end_date null or >= today), upcoming (start_date > today), past
// (end_date < today) or all (default).
//
// @Summary      List project allocations
// @Tags         projects
// @Produce      json
// @Security     BearerAuth
// @Param        id     path   string  true   "Project UUID"
// @Param        state  query  string  false  "Allocation state filter"  Enums(all, active, upcoming, past)  default(all)
// @Success      200  {object}  response.Envelope{data=[]prjtypes.ProjectAllocationResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/projects/{id}/allocations [get]
func (h *ProjectAllocationHandler) ListProjectAllocations(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "project id")
	if !ok {
		return
	}
	scope := entity.AllocationScopeAll
	if raw := c.Query("state"); raw != "" {
		var valid bool
		if scope, valid = entity.ParseAllocationScope(raw); !valid {
			response.BadRequest(c, "state: must be one of all, active, upcoming, past")
			return
		}
	}
	as, err := h.uc.ListByProject.Execute(c.Request.Context(), tenantID, id, scope)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	items := make([]prjtypes.ProjectAllocationResponse, 0, len(as))
	for _, a := range as {
		items = append(items, prjtypes.ToProjectAllocationResponse(a))
	}
	response.Success(c, items)
}

// CreateAllocation allocates a user to a project. Over-allocation (total above
// 100%) is allowed and reported via over_allocated. The total counts only
// allocations on live projects (active or on_hold). Allocating to an archived
// project is rejected with 409; completed, on_hold and active projects are allowed.
// The allocation must fit the project's date window: start_date on or after the
// project start; end_date is optional (omit or null for an open-ended allocation)
// and, when given on a project with an end date, must be on or before it.
//
// @Summary      Create project allocation
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Project UUID"
// @Param        request  body  prjtypes.CreateAllocationRequest  true  "Allocation details"
// @Success      201  {object}  response.Envelope{data=prjtypes.AllocationWriteResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/projects/{id}/allocations [post]
func (h *ProjectAllocationHandler) CreateAllocation(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "project id")
	if !ok {
		return
	}
	var req prjtypes.CreateAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	in := prjtypes.CreateAllocationInput{
		ProjectID: id, UserID: req.UserID, AllocationPercent: req.AllocationPercent,
		RoleOnProject: req.RoleOnProject, StartDate: req.StartDate.Time,
	}
	if req.EndDate != nil {
		in.EndDate = &req.EndDate.Time
	}
	res, err := h.uc.Create.Execute(c.Request.Context(), tenantID, actorID, in)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Created(c, prjtypes.ToAllocationWriteResponse(res))
}

// UpdateAllocation patches an allocation (percent, role, dates; set end_date to end it).
// When dates change they must still fit the project's window (400 otherwise).
//
// @Summary      Update allocation
// @Tags         allocations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Allocation UUID"
// @Param        request  body  prjtypes.UpdateAllocationRequest  true  "Fields to change"
// @Success      200  {object}  response.Envelope{data=prjtypes.AllocationWriteResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/allocations/{id} [patch]
func (h *ProjectAllocationHandler) UpdateAllocation(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "allocation id")
	if !ok {
		return
	}
	var req prjtypes.UpdateAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	res, err := h.uc.Update.Execute(c.Request.Context(), tenantID, actorID, prjtypes.UpdateAllocationInput{
		ID: id, AllocationPercent: req.AllocationPercent, RoleOnProject: req.RoleOnProject,
		StartDate: req.StartDate, EndDate: req.EndDate,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Success(c, prjtypes.ToAllocationWriteResponse(res))
}

// DeleteAllocation soft-deletes an allocation.
//
// @Summary      Delete allocation
// @Tags         allocations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Allocation UUID"
// @Success      200  {object}  response.Envelope
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/allocations/{id} [delete]
func (h *ProjectAllocationHandler) DeleteAllocation(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "allocation id")
	if !ok {
		return
	}
	if err := h.uc.Delete.Execute(c.Request.Context(), tenantID, actorID, id); err != nil {
		writeProjectError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "allocation deleted successfully"})
}

// ListEmployeeAllocations returns an employee's allocations, each with its
// project_status. The optional scope query param selects active (default,
// date-active today), upcoming (start_date > today), past (end_date < today) or
// all. Every scope lists allocations on any project status. total_percent and
// over_allocated (warn-only) count only allocations active today on live
// projects (status active or on_hold), so upcoming/past items and completed or
// archived projects are listed but never counted; each item's
// counts_toward_total says whether its project is live.
//
// @Summary      List employee allocations
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id     path   string  true   "User UUID"
// @Param        scope  query  string  false  "Which allocations to return"  Enums(active, upcoming, past, all)  default(active)
// @Success      200  {object}  response.Envelope{data=prjtypes.UserAllocationsResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/allocations [get]
func (h *ProjectAllocationHandler) ListEmployeeAllocations(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	scope, valid := entity.ParseAllocationScope(c.Query("scope"))
	if !valid {
		response.BadRequest(c, "scope: must be one of active, upcoming, past, all")
		return
	}
	res, err := h.uc.ByUser.Execute(c.Request.Context(), tenantID, id, scope)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Success(c, prjtypes.ToUserAllocationsResponse(res))
}

// ListMyProjects returns the caller's date-active allocations on live projects
// (status active or on_hold); completed and archived projects are excluded.
// The optional scope query param selects active (default), upcoming
// (start_date > today), past (end_date < today) or all (every non-deleted
// allocation, any project status), ordered by start_date for non-active scopes.
//
// @Summary      List my projects
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Param        scope  query  string  false  "Which allocations to return"  Enums(active, upcoming, past, all)  default(active)
// @Success      200  {object}  response.Envelope{data=prjtypes.UserAllocationsResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/me/projects [get]
func (h *ProjectAllocationHandler) ListMyProjects(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	scope, valid := entity.ParseAllocationScope(c.Query("scope"))
	if !valid {
		response.BadRequest(c, "scope: must be one of active, upcoming, past, all")
		return
	}
	res, err := h.uc.MyProjects.Execute(c.Request.Context(), tenantID, userID, scope)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Success(c, prjtypes.ToUserAllocationsResponse(res))
}
