package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
	deptimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/department"
	deptuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/department"
)

// DepartmentHandler handles HTTP requests for tenant departments.
type DepartmentHandler struct {
	createUC deptuc.CreateDepartmentUseCase
	getUC    deptuc.GetDepartmentUseCase
	listUC   deptuc.ListDepartmentsUseCase
	updateUC deptuc.UpdateDepartmentUseCase
	deleteUC deptuc.DeleteDepartmentUseCase
}

// NewDepartmentHandler constructs a DepartmentHandler.
func NewDepartmentHandler(
	createUC deptuc.CreateDepartmentUseCase,
	getUC deptuc.GetDepartmentUseCase,
	listUC deptuc.ListDepartmentsUseCase,
	updateUC deptuc.UpdateDepartmentUseCase,
	deleteUC deptuc.DeleteDepartmentUseCase,
) *DepartmentHandler {
	return &DepartmentHandler{
		createUC: createUC,
		getUC:    getUC,
		listUC:   listUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
	}
}

func writeDepartmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrDepartmentNotFound):
		response.NotFound(c, "department not found")
	case errors.Is(err, domainerrors.ErrDepartmentNameTaken):
		response.Error(c, http.StatusConflict, "CONFLICT", "a department with this name already exists")
	case errors.Is(err, domainerrors.ErrDepartmentInUse):
		response.Error(c, http.StatusConflict, "CONFLICT", "department cannot be deleted because it is assigned to users or invitations")
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, "insufficient permissions")
	case errors.Is(err, deptimpl.ErrDepartmentNameRequired),
		errors.Is(err, deptimpl.ErrDepartmentNameTooLong),
		errors.Is(err, deptimpl.ErrDescriptionTooLong):
		response.BadRequest(c, err.Error())
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// Create creates a new department within the caller's tenant.
//
// @Summary      Create department
// @Tags         departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      depttypes.CreateDepartmentRequest  true  "Department details"
// @Success      201      {object}  response.Envelope{data=depttypes.DepartmentResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/departments [post]
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req depttypes.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	dept, err := h.createUC.Execute(c.Request.Context(), tenantID, actorID, depttypes.CreateDepartmentInput{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
	})
	if err != nil {
		writeDepartmentError(c, err)
		return
	}

	response.Created(c, depttypes.ToDepartmentResponse(dept))
}

// GetByID looks up a single department by ID within the caller's tenant.
//
// @Summary      Get department by ID
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Department UUID"
// @Success      200  {object}  response.Envelope{data=depttypes.DepartmentResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/departments/{id} [get]
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	dept, err := h.getUC.Execute(c.Request.Context(), tenantID, depttypes.GetDepartmentQuery{ID: id})
	if err != nil {
		writeDepartmentError(c, err)
		return
	}

	response.Success(c, depttypes.ToDepartmentResponse(dept))
}

// List returns a paginated list of departments for the caller's tenant.
//
// @Summary      List departments
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int  false  "Page number (default 1)"
// @Param        page_size  query     int  false  "Page size (default 20, max 100)"
// @Param        is_active  query     bool false  "Filter by active flag (true/false); omit for all"
// @Success      200        {object}  response.Envelope{data=[]depttypes.DepartmentResponse,meta=response.Meta}
// @Failure      400        {object}  response.Envelope
// @Failure      401        {object}  response.Envelope
// @Failure      403        {object}  response.Envelope
// @Router       /api/v1/departments [get]
func (h *DepartmentHandler) List(c *gin.Context) {
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

	out, err := h.listUC.Execute(c.Request.Context(), tenantID, depttypes.ListDepartmentsQuery{
		Page:     page,
		PageSize: pageSize,
		IsActive: isActive,
	})
	if err != nil {
		writeDepartmentError(c, err)
		return
	}

	items := make([]depttypes.DepartmentResponse, 0, len(out.Departments))
	for _, d := range out.Departments {
		items = append(items, depttypes.ToDepartmentResponse(d))
	}

	response.Paginated(c, items, pageMeta(out.Page, out.PageSize, out.Total))
}

// Update modifies an existing department within the caller's tenant.
//
// @Summary      Update department
// @Tags         departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                             true  "Department UUID"
// @Param        request  body      depttypes.UpdateDepartmentRequest  true  "Updated department details"
// @Success      200      {object}  response.Envelope{data=depttypes.DepartmentResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/departments/{id} [put]
func (h *DepartmentHandler) Update(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	var req depttypes.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	dept, err := h.updateUC.Execute(c.Request.Context(), tenantID, actorID, depttypes.UpdateDepartmentInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
	})
	if err != nil {
		writeDepartmentError(c, err)
		return
	}

	response.Success(c, depttypes.ToDepartmentResponse(dept))
}

// Delete removes a department within the caller's tenant.
//
// @Summary      Delete department
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Department UUID"
// @Success      200  {object}  response.Envelope
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/departments/{id} [delete]
func (h *DepartmentHandler) Delete(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	if err := h.deleteUC.Execute(c.Request.Context(), tenantID, actorID, depttypes.DeleteDepartmentInput{ID: id}); err != nil {
		writeDepartmentError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "department deleted successfully"})
}
