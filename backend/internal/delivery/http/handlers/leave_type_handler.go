package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
	leavetypeuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/leave/leavetype"
)

// LeaveTypeUseCases groups the usecases behind the leave type endpoints.
type LeaveTypeUseCases struct {
	List   leavetypeuc.ListLeaveTypesUseCase
	Create leavetypeuc.CreateLeaveTypeUseCase
	Update leavetypeuc.UpdateLeaveTypeUseCase
}

// LeaveTypeHandler serves the leave type endpoints (/leave-types).
type LeaveTypeHandler struct{ uc LeaveTypeUseCases }

// NewLeaveTypeHandler constructs a LeaveTypeHandler.
func NewLeaveTypeHandler(uc LeaveTypeUseCases) *LeaveTypeHandler { return &LeaveTypeHandler{uc: uc} }

// writeLeaveError maps leave domain errors to HTTP responses; it is shared by
// LeaveTypeHandler and LeaveRequestHandler.
func writeLeaveError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrLeaveTypeNotFound),
		errors.Is(err, domainerrors.ErrLeaveRequestNotFound):
		response.NotFound(c, "not found")
	case errors.Is(err, domainerrors.ErrInvalidLeaveData):
		response.BadRequest(c, err.Error())
	case errors.Is(err, domainerrors.ErrLeaveOverlap):
		response.Error(c, http.StatusUnprocessableEntity, "leave_overlap", err.Error())
	case errors.Is(err, domainerrors.ErrLeaveStartTooOld):
		response.Error(c, http.StatusUnprocessableEntity, "leave_start_too_old", err.Error())
	case errors.Is(err, domainerrors.ErrInvalidLeaveApprover):
		response.Error(c, http.StatusUnprocessableEntity, "invalid_approver", err.Error())
	case errors.Is(err, domainerrors.ErrLeaveTypeInactive):
		response.Error(c, http.StatusUnprocessableEntity, "leave_type_inactive", err.Error())
	case errors.Is(err, domainerrors.ErrLeaveTypeCodeTaken):
		response.Error(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, domainerrors.ErrLeaveNotPending):
		response.Error(c, http.StatusConflict, "leave_not_pending", err.Error())
	case errors.Is(err, domainerrors.ErrLeaveNotCancellable):
		response.Error(c, http.StatusConflict, "leave_not_cancellable", err.Error())
	case errors.Is(err, domainerrors.ErrLeaveSelfDecision):
		response.Forbidden(c, err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, "insufficient permissions")
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// ListLeaveTypes lists active leave types (admins may include inactive ones).
//
// @Summary      List leave types
// @Tags         leave
// @Produce      json
// @Security     BearerAuth
// @Param        include_inactive  query  bool  false  "Include inactive types (admin only; ignored otherwise)"
// @Success      200  {object}  response.Envelope{data=[]leavetypes.LeaveTypeResponse}
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/leave-types [get]
func (h *LeaveTypeHandler) ListLeaveTypes(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	includeInactive := callerIsAdmin(c) && c.Query("include_inactive") == "true"
	out, err := h.uc.List.Execute(c.Request.Context(), tenantID, includeInactive)
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Success(c, leavetypes.ToLeaveTypeResponses(out))
}

// CreateLeaveType creates a leave type.
//
// @Summary      Create leave type
// @Tags         leave
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  leavetypes.CreateLeaveTypeRequest  true  "Leave type"
// @Success      201  {object}  response.Envelope{data=leavetypes.LeaveTypeResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope  "error.code CONFLICT: leave type code already exists"
// @Router       /api/v1/leave-types [post]
func (h *LeaveTypeHandler) CreateLeaveType(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	var req leavetypes.CreateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	lt, err := h.uc.Create.Execute(c.Request.Context(), tenantID, actorID, leavetypes.CreateLeaveTypeInput{
		Name: req.Name, Code: req.Code, Description: req.Description, IsActive: req.IsActive,
	})
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Created(c, leavetypes.ToLeaveTypeResponse(lt))
}

// UpdateLeaveType patches a leave type; set is_active=false to deactivate.
//
// @Summary      Update or deactivate leave type
// @Tags         leave
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                             true  "Leave type UUID"
// @Param        request  body  leavetypes.UpdateLeaveTypeRequest  true  "Fields to change"
// @Success      200  {object}  response.Envelope{data=leavetypes.LeaveTypeResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope  "error.code CONFLICT: leave type code already exists"
// @Router       /api/v1/leave-types/{id} [patch]
func (h *LeaveTypeHandler) UpdateLeaveType(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "leave")
	if !ok {
		return
	}
	var req leavetypes.UpdateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	lt, err := h.uc.Update.Execute(c.Request.Context(), tenantID, actorID, id, leavetypes.UpdateLeaveTypeInput{
		Name: req.Name, Code: req.Code, Description: req.Description, IsActive: req.IsActive,
	})
	if err != nil {
		writeLeaveError(c, err)
		return
	}
	response.Success(c, leavetypes.ToLeaveTypeResponse(lt))
}
