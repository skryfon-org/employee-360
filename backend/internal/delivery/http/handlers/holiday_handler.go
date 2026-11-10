package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	holidayuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/holiday"
)

// HolidayHandler handles HTTP requests for holidays, categories, and calendar settings.
type HolidayHandler struct {
	createHolidayUC holidayuc.CreateHolidayUseCase
	getHolidayUC    holidayuc.GetHolidayUseCase
	listHolidaysUC  holidayuc.ListHolidaysByYearUseCase
	updateHolidayUC holidayuc.UpdateHolidayUseCase
	deleteHolidayUC holidayuc.DeleteHolidayUseCase

	createCategoryUC holidayuc.CreateCategoryUseCase
	getCategoryUC    holidayuc.GetCategoryUseCase
	listCategoriesUC holidayuc.ListCategoriesUseCase
	updateCategoryUC holidayuc.UpdateCategoryUseCase
	deleteCategoryUC holidayuc.DeleteCategoryUseCase

	getSettingsUC    holidayuc.GetSettingsUseCase
	updateSettingsUC holidayuc.UpdateSettingsUseCase
}

// NewHolidayHandler constructs a HolidayHandler.
func NewHolidayHandler(
	createHolidayUC holidayuc.CreateHolidayUseCase,
	getHolidayUC holidayuc.GetHolidayUseCase,
	listHolidaysUC holidayuc.ListHolidaysByYearUseCase,
	updateHolidayUC holidayuc.UpdateHolidayUseCase,
	deleteHolidayUC holidayuc.DeleteHolidayUseCase,
	createCategoryUC holidayuc.CreateCategoryUseCase,
	getCategoryUC holidayuc.GetCategoryUseCase,
	listCategoriesUC holidayuc.ListCategoriesUseCase,
	updateCategoryUC holidayuc.UpdateCategoryUseCase,
	deleteCategoryUC holidayuc.DeleteCategoryUseCase,
	getSettingsUC holidayuc.GetSettingsUseCase,
	updateSettingsUC holidayuc.UpdateSettingsUseCase,
) *HolidayHandler {
	return &HolidayHandler{
		createHolidayUC:  createHolidayUC,
		getHolidayUC:     getHolidayUC,
		listHolidaysUC:   listHolidaysUC,
		updateHolidayUC:  updateHolidayUC,
		deleteHolidayUC:  deleteHolidayUC,
		createCategoryUC: createCategoryUC,
		getCategoryUC:    getCategoryUC,
		listCategoriesUC: listCategoriesUC,
		updateCategoryUC: updateCategoryUC,
		deleteCategoryUC: deleteCategoryUC,
		getSettingsUC:    getSettingsUC,
		updateSettingsUC: updateSettingsUC,
	}
}

func writeHolidayError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrHolidayNotFound):
		response.NotFound(c, "holiday not found")
	case errors.Is(err, domainerrors.ErrHolidayCategoryNotFound):
		response.NotFound(c, "holiday category not found")
	case errors.Is(err, domainerrors.ErrHolidayCalendarSettingsNotFound):
		response.NotFound(c, "holiday calendar settings not found")
	case errors.Is(err, domainerrors.ErrHolidayDuplicate):
		response.Error(c, http.StatusConflict, "CONFLICT", "holiday already exists for this date and name")
	case errors.Is(err, domainerrors.ErrHolidayCategoryDuplicate):
		response.Error(c, http.StatusConflict, "CONFLICT", "holiday category already exists with this name")
	case errors.Is(err, domainerrors.ErrHolidayCategoryInUse):
		response.Error(c, http.StatusConflict, "CATEGORY_IN_USE", "holiday category is in use by holidays")
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, "insufficient permissions")
	case errors.Is(err, domainerrors.ErrHolidayNameRequired),
		errors.Is(err, domainerrors.ErrHolidayNameTooLong),
		errors.Is(err, domainerrors.ErrHolidayDateInvalid),
		errors.Is(err, domainerrors.ErrHolidayYearInvalid),
		errors.Is(err, domainerrors.ErrHolidayMonthInvalid),
		errors.Is(err, domainerrors.ErrHolidayCategoryInactive),
		errors.Is(err, domainerrors.ErrHolidayCategoryNameRequired),
		errors.Is(err, domainerrors.ErrHolidayCategoryNameTooLong),
		errors.Is(err, domainerrors.ErrHolidayCategoryColorInvalid),
		errors.Is(err, domainerrors.ErrHolidayCategoryDescTooLong),
		errors.Is(err, domainerrors.ErrHolidayCalendarSettingsInvalidDefaultView),
		errors.Is(err, domainerrors.ErrHolidayCalendarSettingsInvalidDefaultYear),
		errors.Is(err, domainerrors.ErrHolidayCalendarSettingsInvalidWeekStartDay):
		response.BadRequest(c, err.Error())
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// ============================================================================
// Holidays
// ============================================================================

// CreateHoliday creates a new holiday within the caller's tenant.
//
// @Summary      Create holiday
// @Tags         holidays
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      holidaytypes.CreateHolidayRequest  true  "Holiday details"
// @Success      201      {object}  response.Envelope{data=holidaytypes.HolidayResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/holidays [post]
func (h *HolidayHandler) CreateHoliday(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	var req holidaytypes.CreateHolidayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeHolidayError(c, domainerrors.ErrHolidayDateInvalid)
		return
	}

	holiday, err := h.createHolidayUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.CreateHolidayInput{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		Date:        date,
	})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Created(c, holidaytypes.ToHolidayResponse(holiday))
}

// GetHolidayByID retrieves a single holiday by ID within the caller's tenant.
//
// @Summary      Get holiday by ID
// @Tags         holidays
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Holiday UUID"
// @Success      200  {object}  response.Envelope{data=holidaytypes.HolidayResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/holidays/{id} [get]
func (h *HolidayHandler) GetHolidayByID(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid holiday id")
		return
	}

	holiday, err := h.getHolidayUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.GetHolidayQuery{ID: id})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.ToHolidayResponse(holiday))
}

// ListHolidays retrieves holidays for a given year within the caller's tenant.
//
// @Summary      List holidays by year
// @Tags         holidays
// @Produce      json
// @Security     BearerAuth
// @Param        year         query     int     true   "Year (required, between 2000 and 2100)"
// @Param        category_id  query     string  false  "Category UUID filter"
// @Param        month        query     int     false  "Month filter (1-12)"
// @Success      200          {object}  response.Envelope{data=[]holidaytypes.HolidayResponse}
// @Failure      400          {object}  response.Envelope
// @Failure      401          {object}  response.Envelope
// @Failure      403          {object}  response.Envelope
// @Router       /api/v1/holidays [get]
func (h *HolidayHandler) ListHolidays(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	rawYear := c.Query("year")
	if rawYear == "" {
		response.BadRequest(c, "year query parameter is required")
		return
	}

	year, err := strconv.Atoi(rawYear)
	if err != nil {
		response.BadRequest(c, "year must be an integer")
		return
	}

	var categoryID *uuid.UUID
	if rawCatID := c.Query("category_id"); rawCatID != "" {
		catID, err := uuid.Parse(rawCatID)
		if err != nil {
			response.BadRequest(c, "category_id must be a valid UUID")
			return
		}
		categoryID = &catID
	}

	var month *int
	if rawMonth := c.Query("month"); rawMonth != "" {
		m, err := strconv.Atoi(rawMonth)
		if err != nil {
			response.BadRequest(c, "month must be an integer")
			return
		}
		month = &m
	}

	holidays, err := h.listHolidaysUC.Execute(c.Request.Context(), tenantID, holidaytypes.ListHolidaysByYearQuery{
		Year:       year,
		CategoryID: categoryID,
		Month:      month,
	})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	items := make([]holidaytypes.HolidayResponse, 0, len(holidays))
	for _, hol := range holidays {
		items = append(items, holidaytypes.ToHolidayResponse(hol))
	}

	response.Success(c, items)
}

// UpdateHoliday updates an existing holiday within the caller's tenant.
//
// @Summary      Update holiday
// @Tags         holidays
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                             true  "Holiday UUID"
// @Param        request  body      holidaytypes.UpdateHolidayRequest  true  "Updated holiday details"
// @Success      200      {object}  response.Envelope{data=holidaytypes.HolidayResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/holidays/{id} [put]
func (h *HolidayHandler) UpdateHoliday(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid holiday id")
		return
	}

	var req holidaytypes.UpdateHolidayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeHolidayError(c, domainerrors.ErrHolidayDateInvalid)
		return
	}

	holiday, err := h.updateHolidayUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.UpdateHolidayInput{
		ID:          id,
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		Date:        date,
	})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.ToHolidayResponse(holiday))
}

// DeleteHoliday removes a holiday within the caller's tenant.
//
// @Summary      Delete holiday
// @Tags         holidays
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Holiday UUID"
// @Success      200  {object}  response.Envelope{data=holidaytypes.MessageResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/holidays/{id} [delete]
func (h *HolidayHandler) DeleteHoliday(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid holiday id")
		return
	}

	if err := h.deleteHolidayUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.DeleteHolidayInput{ID: id}); err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.MessageResponse{Message: "holiday deleted successfully"})
}

// ============================================================================
// Holiday Calendar Settings
// ============================================================================

// GetSettings retrieves holiday calendar settings for the caller's tenant.
//
// @Summary      Get holiday calendar settings
// @Tags         holiday-calendar-settings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=holidaytypes.SettingsResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/holiday-calendar/settings [get]
func (h *HolidayHandler) GetSettings(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	settings, err := h.getSettingsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.ToSettingsResponse(settings))
}

// UpdateSettings updates holiday calendar settings for the caller's tenant.
//
// @Summary      Update holiday calendar settings
// @Tags         holiday-calendar-settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      holidaytypes.UpdateSettingsRequest  true  "Updated calendar settings"
// @Success      200      {object}  response.Envelope{data=holidaytypes.SettingsResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Router       /api/v1/holiday-calendar/settings [put]
func (h *HolidayHandler) UpdateSettings(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	var req holidaytypes.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	settings, err := h.updateSettingsUC.Execute(c.Request.Context(), tenantID, actorID, holidaytypes.UpdateSettingsInput{
		DefaultView:  req.DefaultView,
		DefaultYear:  req.DefaultYear,
		WeekStartDay: *req.WeekStartDay,
	})
	if err != nil {
		writeHolidayError(c, err)
		return
	}

	response.Success(c, holidaytypes.ToSettingsResponse(settings))
}
