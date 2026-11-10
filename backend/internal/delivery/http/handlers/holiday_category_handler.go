package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
)

// ============================================================================
// Holiday Categories
// ============================================================================

// CreateCategory creates a new holiday category within the caller's tenant.
//
// @Summary      Create holiday category
// @Tags         holiday-categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      holidaytypes.CreateCategoryRequest  true  "Category details"
// @Success      201      {object}  response.Envelope{data=holidaytypes.CategoryResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/holiday-categories [post]
func (h *HolidayHandler) CreateCategory(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	var req holidaytypes.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	category, err := h.createCategoryUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.CreateCategoryInput{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		IsActive:    req.IsActive,
	})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Created(c, holidaytypes.ToCategoryResponse(category))
}

// GetCategoryByID retrieves a single holiday category by ID within the caller's tenant.
//
// @Summary      Get holiday category by ID
// @Tags         holiday-categories
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Category UUID"
// @Success      200  {object}  response.Envelope{data=holidaytypes.CategoryResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/holiday-categories/{id} [get]
func (h *HolidayHandler) GetCategoryByID(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid holiday category id")
		return
	}

	category, err := h.getCategoryUC.Execute(c.Request.Context(), tenantID, holidaytypes.GetCategoryQuery{ID: id})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.ToCategoryResponse(category))
}

// ListCategories retrieves holiday categories within the caller's tenant.
//
// @Summary      List holiday categories
// @Tags         holiday-categories
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int   false  "Page number (default 1)"
// @Param        page_size  query     int   false  "Page size (default 20, max 100)"
// @Param        is_active  query     bool  false  "Filter by active status (true/false)"
// @Success      200        {object}  response.Envelope{data=[]holidaytypes.CategoryResponse,meta=response.Meta}
// @Failure      400        {object}  response.Envelope
// @Failure      401        {object}  response.Envelope
// @Failure      403        {object}  response.Envelope
// @Router       /api/v1/holiday-categories [get]
func (h *HolidayHandler) ListCategories(c *gin.Context) {
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

	out, err := h.listCategoriesUC.Execute(c.Request.Context(), tenantID, holidaytypes.ListCategoriesQuery{
		Page:     page,
		PageSize: pageSize,
		IsActive: isActive,
	})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	items := make([]holidaytypes.CategoryResponse, 0, len(out.Categories))
	for _, cat := range out.Categories {
		items = append(items, holidaytypes.ToCategoryResponse(cat))
	}

	response.Paginated(c, items, pageMeta(out.Page, out.PageSize, out.Total))
}

// UpdateCategory updates an existing holiday category within the caller's tenant.
//
// @Summary      Update holiday category
// @Tags         holiday-categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                              true  "Category UUID"
// @Param        request  body      holidaytypes.UpdateCategoryRequest  true  "Updated category details"
// @Success      200      {object}  response.Envelope{data=holidaytypes.CategoryResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/holiday-categories/{id} [put]
func (h *HolidayHandler) UpdateCategory(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid holiday category id")
		return
	}

	var req holidaytypes.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	category, err := h.updateCategoryUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.UpdateCategoryInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		IsActive:    req.IsActive,
	})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.ToCategoryResponse(category))
}

// DeleteCategory removes a holiday category within the caller's tenant.
//
// @Summary      Delete holiday category
// @Tags         holiday-categories
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Category UUID"
// @Success      200  {object}  response.Envelope{data=holidaytypes.MessageResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/holiday-categories/{id} [delete]
func (h *HolidayHandler) DeleteCategory(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid holiday category id")
		return
	}

	if err := h.deleteCategoryUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.DeleteCategoryInput{ID: id}); err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.MessageResponse{Message: "holiday category deleted successfully"})
}
