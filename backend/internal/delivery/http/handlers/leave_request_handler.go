package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
	leaverequestuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/request"
)

// LeaveRequestUseCases groups the usecases behind the leave request endpoints.
type LeaveRequestUseCases struct {
	ListApprovers leaverequestuc.ListApproversUseCase
	Apply         leaverequestuc.ApplyLeaveUseCase
	ListMy        leaverequestuc.ListMyLeaveUseCase
	Cancel        leaverequestuc.CancelLeaveUseCase
	ListTeam      leaverequestuc.ListTeamLeaveUseCase
	ListAll       leaverequestuc.ListAllLeaveUseCase
	Approve       leaverequestuc.DecideLeaveUseCase
	Reject        leaverequestuc.DecideLeaveUseCase
}

// LeaveRequestHandler serves leave requests: /me/leave-*, /leave-requests.
type LeaveRequestHandler struct{ uc LeaveRequestUseCases }

// NewLeaveRequestHandler constructs a LeaveRequestHandler.
func NewLeaveRequestHandler(uc LeaveRequestUseCases) *LeaveRequestHandler {
	return &LeaveRequestHandler{uc: uc}
}

// callerIsAdmin reports whether the authenticated caller holds a tenant admin
// role (roles come from the auth context, loaded from the database).
func callerIsAdmin(c *gin.Context) bool {
	roles, _ := ctx.RolesFromContext(c.Request.Context())
	for _, r := range roles {
		if r == entity.RoleAdmin || r == entity.RoleSuperAdmin {
			return true
		}
	}
	return false
}

// ListMyLeaveApprovers lists who the caller can address a leave request to.
//
// @Summary      List my leave approvers
// @Description  The caller's active manager plus active tenant admins, excluding the caller.
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=[]leavetypes.LeaveApproverResponse}
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/me/leave-approvers [get]
func (h *LeaveRequestHandler) ListMyLeaveApprovers(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	out, err := h.uc.ListApprovers.Execute(c.Request.Context(), tenantID, userID)
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Success(c, leavetypes.ToLeaveApproverResponses(out))
}

// ApplyLeave creates a pending leave request for the caller.
//
// @Summary      Apply for leave
// @Tags         me
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  leavetypes.ApplyLeaveRequest  true  "Leave request"
// @Success      201  {object}  response.Envelope{data=leavetypes.LeaveRequestResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /api/v1/me/leave-requests [post]
func (h *LeaveRequestHandler) ApplyLeave(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	var req leavetypes.ApplyLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	d, err := h.uc.Apply.Execute(c.Request.Context(), tenantID, userID, leavetypes.ApplyLeaveInput{
		LeaveTypeID: req.LeaveTypeID, StartDate: req.StartDate.Time, EndDate: req.EndDate.Time,
		DayPart: req.DayPart, Reason: req.Reason, ApproverID: req.ApproverID,
	})
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Created(c, leavetypes.ToLeaveRequestResponse(d))
}

// ListMyLeaveRequests lists the caller's own leave requests.
//
// @Summary      List my leave requests
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        page_size  query  int     false  "Page size (default 20, max 100)"
// @Param        status  query  string  false  "Filter by status"  Enums(pending,approved,rejected,cancelled)
// @Param        from    query  string  false  "Requests ending on/after this date (YYYY-MM-DD)"
// @Param        to      query  string  false  "Requests starting on/before this date (YYYY-MM-DD)"
// @Param        leave_type_id  query  string  false  "Filter by leave type ID (UUID)"
// @Success      200  {object}  response.Envelope{data=[]leavetypes.LeaveRequestResponse,meta=response.Meta}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/me/leave-requests [get]
func (h *LeaveRequestHandler) ListMyLeaveRequests(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}
	out, err := h.uc.ListMy.Execute(c.Request.Context(), tenantID, userID, leavetypes.ListMyLeaveQuery{
		Page: page, PageSize: pageSize, Status: c.Query("status"),
		From: c.Query("from"), To: c.Query("to"), LeaveTypeID: c.Query("leave_type_id"),
	})
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Paginated(c, leavetypes.ToLeaveRequestResponses(out.Items), pageMeta(out.Page, out.PageSize, out.Total))
}

// CancelMyLeaveRequest cancels the caller's pending, or approved-but-not-started, request.
//
// @Summary      Cancel my leave request
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Leave request UUID"
// @Success      200  {object}  response.Envelope{data=leavetypes.LeaveRequestResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/me/leave-requests/{id}/cancel [post]
func (h *LeaveRequestHandler) CancelMyLeaveRequest(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "leave")
	if !ok {
		return
	}
	d, err := h.uc.Cancel.Execute(c.Request.Context(), tenantID, userID, id)
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Success(c, leavetypes.ToLeaveRequestResponse(d))
}

// ListTeamLeaveRequests lists requests addressed to the caller as approver.
//
// @Summary      List leave requests awaiting my decision (team)
// @Tags         leave
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        page_size  query  int     false  "Page size (default 20, max 100)"
// @Param        status  query  string  false  "Filter by status"  Enums(pending,approved,rejected,cancelled)
// @Param        exclude_status  query  string  false  "Omit requests with this status (e.g. pending for history); equal to status yields an empty page"  Enums(pending,approved,rejected,cancelled)
// @Param        from    query  string  false  "Requests ending on/after this date (YYYY-MM-DD)"
// @Param        to      query  string  false  "Requests starting on/before this date (YYYY-MM-DD)"
// @Param        leave_type_id  query  string  false  "Filter by leave type ID (UUID)"
// @Success      200  {object}  response.Envelope{data=[]leavetypes.LeaveRequestResponse,meta=response.Meta}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/leave-requests/team [get]
func (h *LeaveRequestHandler) ListTeamLeaveRequests(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}
	out, err := h.uc.ListTeam.Execute(c.Request.Context(), tenantID, userID, leavetypes.ListMyLeaveQuery{
		Page: page, PageSize: pageSize, Status: c.Query("status"), ExcludeStatus: c.Query("exclude_status"),
		From: c.Query("from"), To: c.Query("to"), LeaveTypeID: c.Query("leave_type_id"),
	})
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Paginated(c, leavetypes.ToLeaveRequestResponses(out.Items), pageMeta(out.Page, out.PageSize, out.Total))
}

// ListAllLeaveRequests lists every leave request of the tenant (admin).
//
// @Summary      List all leave requests
// @Tags         leave
// @Produce      json
// @Security     BearerAuth
// @Param        page         query  int     false  "Page number (default 1)"
// @Param        page_size    query  int     false  "Page size (default 20, max 100)"
// @Param        status       query  string  false  "Filter by status"  Enums(pending,approved,rejected,cancelled)
// @Param        employee_id  query  string  false  "Filter by applicant user UUID"
// @Param        leave_type_id query  string  false  "Filter by leave type UUID"
// @Param        from         query  string  false  "Requests ending on/after this date (YYYY-MM-DD)"
// @Param        to           query  string  false  "Requests starting on/before this date (YYYY-MM-DD)"
// @Success      200  {object}  response.Envelope{data=[]leavetypes.LeaveRequestResponse,meta=response.Meta}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/leave-requests [get]
func (h *LeaveRequestHandler) ListAllLeaveRequests(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}
	out, err := h.uc.ListAll.Execute(c.Request.Context(), tenantID, leavetypes.ListLeaveRequestsQuery{
		Page: page, PageSize: pageSize, Status: c.Query("status"), EmployeeID: c.Query("employee_id"),
		LeaveTypeID: c.Query("leave_type_id"), From: c.Query("from"), To: c.Query("to"),
	})
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Paginated(c, leavetypes.ToLeaveRequestResponses(out.Items), pageMeta(out.Page, out.PageSize, out.Total))
}

func (h *LeaveRequestHandler) decide(c *gin.Context, uc leaverequestuc.DecideLeaveUseCase) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "leave")
	if !ok {
		return
	}
	var req leavetypes.DecisionRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request payload")
			return
		}
	}
	d, err := uc.Execute(c.Request.Context(), tenantID, actorID, callerIsAdmin(c), id, req.Comment)
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Success(c, leavetypes.ToLeaveRequestResponse(d))
}

// ApproveLeaveRequest approves a pending request (chosen approver or tenant admin; never the applicant).
//
// @Summary      Approve leave request
// @Tags         leave
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                        true   "Leave request UUID"
// @Param        request  body  leavetypes.DecisionRequest    false  "Optional comment"
// @Success      200  {object}  response.Envelope{data=leavetypes.LeaveRequestResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/leave-requests/{id}/approve [post]
func (h *LeaveRequestHandler) ApproveLeaveRequest(c *gin.Context) { h.decide(c, h.uc.Approve) }

// RejectLeaveRequest rejects a pending request (chosen approver or tenant admin; never the applicant).
//
// @Summary      Reject leave request
// @Tags         leave
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                        true   "Leave request UUID"
// @Param        request  body  leavetypes.DecisionRequest    false  "Optional comment"
// @Success      200  {object}  response.Envelope{data=leavetypes.LeaveRequestResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/leave-requests/{id}/reject [post]
func (h *LeaveRequestHandler) RejectLeaveRequest(c *gin.Context) { h.decide(c, h.uc.Reject) }
