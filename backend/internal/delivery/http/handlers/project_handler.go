package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	prjuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/project"
)

// ProjectUseCases groups the usecases behind the project endpoints.
type ProjectUseCases struct {
	Create      prjuc.CreateProjectUseCase
	Get         prjuc.GetProjectUseCase
	List        prjuc.ListProjectsUseCase
	Update      prjuc.UpdateProjectUseCase
	SuggestCode prjuc.SuggestProjectCodeUseCase
}

// ProjectHandler serves the project endpoints (/projects).
type ProjectHandler struct{ uc ProjectUseCases }

// NewProjectHandler constructs a ProjectHandler.
func NewProjectHandler(uc ProjectUseCases) *ProjectHandler { return &ProjectHandler{uc: uc} }

// writeProjectError maps project and allocation domain errors to HTTP responses;
// it is shared by ProjectHandler and ProjectAllocationHandler.
func writeProjectError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrProjectNotFound),
		errors.Is(err, domainerrors.ErrAllocationNotFound),
		errors.Is(err, domainerrors.ErrUserNotFound):
		response.NotFound(c, "not found")
	case errors.Is(err, domainerrors.ErrInvalidProjectData):
		response.BadRequest(c, err.Error())
	case errors.Is(err, domainerrors.ErrProjectCodeTaken),
		errors.Is(err, domainerrors.ErrAllocationOverlap),
		errors.Is(err, domainerrors.ErrProjectArchived),
		errors.Is(err, domainerrors.ErrAllocationUserInactive):
		response.Error(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, "insufficient permissions")
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// CreateProject creates a project.
//
// @Summary      Create project
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  prjtypes.CreateProjectRequest  true  "Project details"
// @Success      201  {object}  response.Envelope{data=prjtypes.ProjectResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	var req prjtypes.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	in := prjtypes.CreateProjectInput{
		Name: req.Name, Code: req.Code, Description: req.Description, ClientName: req.ClientName,
		Status: req.Status, StartDate: req.StartDate.Time,
	}
	if req.EndDate != nil {
		in.EndDate = &req.EndDate.Time
	}
	p, err := h.uc.Create.Execute(c.Request.Context(), tenantID, actorID, in)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Created(c, prjtypes.ToProjectResponse(p))
}

// ListProjects lists projects with search, status filter and pagination.
//
// @Summary      List projects
// @Tags         projects
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        page_size  query  int     false  "Page size (default 20, max 100)"
// @Param        search     query  string  false  "Match on name, code or client name"
// @Param        status     query  string  false  "Filter by status"  Enums(active,on_hold,completed,archived)
// @Success      200  {object}  response.Envelope{data=[]prjtypes.ProjectResponse,meta=response.Meta}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/projects [get]
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}
	out, err := h.uc.List.Execute(c.Request.Context(), tenantID, prjtypes.ListProjectsQuery{
		Page: page, PageSize: pageSize, Search: c.Query("search"), Status: c.Query("status"),
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	items := make([]prjtypes.ProjectResponse, 0, len(out.Projects))
	for _, p := range out.Projects {
		items = append(items, prjtypes.ToProjectResponse(p))
	}
	response.Paginated(c, items, pageMeta(out.Page, out.PageSize, out.Total))
}

// GetProject returns one project.
//
// @Summary      Get project
// @Tags         projects
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Project UUID"
// @Success      200  {object}  response.Envelope{data=prjtypes.ProjectResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/projects/{id} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "project id")
	if !ok {
		return
	}
	p, err := h.uc.Get.Execute(c.Request.Context(), tenantID, id)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Success(c, prjtypes.ToProjectResponse(p))
}

// UpdateProject patches a project, including its status (projects are archived, never deleted).
//
// @Summary      Update project
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Project UUID"
// @Param        request  body  prjtypes.UpdateProjectRequest  true  "Fields to change"
// @Success      200  {object}  response.Envelope{data=prjtypes.ProjectResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/projects/{id} [patch]
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "project id")
	if !ok {
		return
	}
	var req prjtypes.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	p, err := h.uc.Update.Execute(c.Request.Context(), tenantID, actorID, prjtypes.UpdateProjectInput{
		ID: id, Name: req.Name, Code: req.Code, Description: req.Description, ClientName: req.ClientName,
		Status: req.Status, StartDate: req.StartDate, EndDate: req.EndDate,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Success(c, prjtypes.ToProjectResponse(p))
}

// SuggestProjectCode previews the code that would be generated for a name.
//
// @Summary      Suggest project code
// @Description  Returns the tenant-unique code that creating a project with this name and no explicit code would get. Advisory only.
// @Tags         projects
// @Produce      json
// @Security     BearerAuth
// @Param        name  query  string  true  "Project name"
// @Success      200  {object}  response.Envelope{data=prjtypes.ProjectCodeSuggestionResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/projects/code-suggestion [get]
func (h *ProjectHandler) SuggestProjectCode(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	code, err := h.uc.SuggestCode.Execute(c.Request.Context(), tenantID, c.Query("name"))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Success(c, prjtypes.ProjectCodeSuggestionResponse{Code: code})
}
