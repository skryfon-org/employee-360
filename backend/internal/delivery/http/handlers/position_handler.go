package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	postypes "github.com/skryfon-org/employee-360/backend/internal/types/position"
	posimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/position"
	posuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/position"
)

// Request and response DTO aliases
type CreatePositionRequest = postypes.CreatePositionRequest
type UpdatePositionRequest = postypes.UpdatePositionRequest
type PositionResponse = postypes.PositionResponse

// PositionHandler handles HTTP requests for tenant positions.
type PositionHandler struct {
	createUC posuc.CreatePositionUseCase
	getUC    posuc.GetPositionUseCase
	listUC   posuc.ListPositionsUseCase
	updateUC posuc.UpdatePositionUseCase
	deleteUC posuc.DeletePositionUseCase
}

// NewPositionHandler constructs a PositionHandler.
func NewPositionHandler(
	createUC posuc.CreatePositionUseCase,
	getUC posuc.GetPositionUseCase,
	listUC posuc.ListPositionsUseCase,
	updateUC posuc.UpdatePositionUseCase,
	deleteUC posuc.DeletePositionUseCase,
) *PositionHandler {
	return &PositionHandler{
		createUC: createUC,
		getUC:    getUC,
		listUC:   listUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
	}
}

func writePositionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrPositionNotFound):
		response.NotFound(c, "position not found")
	case errors.Is(err, domainerrors.ErrPositionNameTaken):
		response.Error(c, http.StatusConflict, "CONFLICT", "a position with this name already exists")
	case errors.Is(err, domainerrors.ErrPositionInUse):
		response.Error(c, http.StatusConflict, "CONFLICT", "position cannot be deleted because it is assigned to users or invitations")
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, "insufficient permissions")
	case errors.Is(err, posimpl.ErrPositionNameRequired),
		errors.Is(err, posimpl.ErrPositionNameTooLong),
		errors.Is(err, posimpl.ErrDescriptionTooLong):
		response.BadRequest(c, err.Error())
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// Create creates a new position within the caller's tenant.
//
// @Summary      Create position
// @Tags         positions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      postypes.CreatePositionRequest  true  "Position details"
// @Success      201      {object}  response.Envelope{data=postypes.PositionResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/positions [post]
func (h *PositionHandler) Create(c *gin.Context) {
	var req postypes.CreatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	pos, err := h.createUC.Execute(c.Request.Context(), tenantID, actorID, postypes.CreatePositionInput{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
	})
	if err != nil {
		writePositionError(c, err)
		return
	}

	response.Created(c, postypes.ToPositionResponse(pos))
}

// GetByID looks up a single position by ID within the caller's tenant.
//
// @Summary      Get position by ID
// @Tags         positions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Position UUID"
// @Success      200  {object}  response.Envelope{data=postypes.PositionResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/positions/{id} [get]
func (h *PositionHandler) GetByID(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid position id")
		return
	}

	pos, err := h.getUC.Execute(c.Request.Context(), tenantID, postypes.GetPositionQuery{ID: id})
	if err != nil {
		writePositionError(c, err)
		return
	}

	response.Success(c, postypes.ToPositionResponse(pos))
}

// List returns a paginated list of positions for the caller's tenant.
//
// @Summary      List positions
// @Tags         positions
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int   false  "Page number (default 1)"
// @Param        page_size  query     int   false  "Page size (default 20, max 100)"
// @Param        is_active  query     bool  false  "Filter by active flag (true/false); omit for all"
// @Success      200        {object}  response.Envelope{data=[]postypes.PositionResponse,meta=response.Meta}
// @Failure      400        {object}  response.Envelope
// @Failure      401        {object}  response.Envelope
// @Failure      403        {object}  response.Envelope
// @Failure      404        {object}  response.Envelope
// @Failure      409        {object}  response.Envelope
// @Router       /api/v1/positions [get]
func (h *PositionHandler) List(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}

	var isActive *bool
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil || (raw != "true" && raw != "false") {
			response.BadRequest(c, "is_active must be true or false")
			return
		}
		isActive = &v
	}

	out, err := h.listUC.Execute(c.Request.Context(), tenantID, postypes.ListPositionsQuery{
		Page:     page,
		PageSize: pageSize,
		IsActive: isActive,
	})
	if err != nil {
		writePositionError(c, err)
		return
	}

	items := make([]postypes.PositionResponse, 0, len(out.Positions))
	for _, p := range out.Positions {
		items = append(items, postypes.ToPositionResponse(p))
	}

	response.Paginated(c, items, pageMeta(out.Page, out.PageSize, out.Total))
}

// Update modifies an existing position within the caller's tenant.
//
// @Summary      Update position
// @Tags         positions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                          true  "Position UUID"
// @Param        request  body      postypes.UpdatePositionRequest  true  "Updated position details"
// @Success      200      {object}  response.Envelope{data=postypes.PositionResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/positions/{id} [put]
func (h *PositionHandler) Update(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid position id")
		return
	}

	var req postypes.UpdatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	pos, err := h.updateUC.Execute(c.Request.Context(), tenantID, actorID, postypes.UpdatePositionInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
	})
	if err != nil {
		writePositionError(c, err)
		return
	}

	response.Success(c, postypes.ToPositionResponse(pos))
}

// Delete removes a position within the caller's tenant.
//
// @Summary      Delete position
// @Tags         positions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Position UUID"
// @Success      200  {object}  response.Envelope
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/positions/{id} [delete]
func (h *PositionHandler) Delete(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid position id")
		return
	}

	if err := h.deleteUC.Execute(c.Request.Context(), tenantID, actorID, postypes.DeletePositionInput{ID: id}); err != nil {
		writePositionError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "position deleted successfully"})
}
