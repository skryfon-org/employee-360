package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
	statusloguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/statuslog"
)

// StatusLogUseCases groups the status log use cases.
type StatusLogUseCases struct {
	Create       statusloguc.CreateStatusLogUseCase
	Update       statusloguc.UpdateStatusLogUseCase
	Delete       statusloguc.DeleteStatusLogUseCase
	ListMy       statusloguc.ListMyStatusLogsUseCase
	MyDayStatus  statusloguc.ListMyDayStatusesUseCase
	GetDaily     statusloguc.GetDailyStatusLogsUseCase
	GetMonthly   statusloguc.GetMonthlyStatusLogsUseCase
	ListEmployee statusloguc.ListEmployeeStatusLogsUseCase
}

// StatusLogHandler serves status log endpoints for employees and admins.
type StatusLogHandler struct {
	uc StatusLogUseCases
}

// NewStatusLogHandler constructs a StatusLogHandler.
func NewStatusLogHandler(uc StatusLogUseCases) *StatusLogHandler {
	return &StatusLogHandler{uc: uc}
}

func writeStatusLogError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrStatusLogNotFound),
		errors.Is(err, domainerrors.ErrUserNotFound),
		errors.Is(err, domainerrors.ErrProjectNotFound),
		errors.Is(err, domainerrors.ErrDepartmentNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrLogWindowClosed):
		response.Error(c, http.StatusUnprocessableEntity, "log_window_closed", err.Error())
	case errors.Is(err, domainerrors.ErrFutureDate):
		response.Error(c, http.StatusUnprocessableEntity, "future_date_not_allowed", err.Error())
	case errors.Is(err, domainerrors.ErrDailyHoursExceeded):
		response.Error(c, http.StatusUnprocessableEntity, "daily_hours_exceeded", err.Error())
	case errors.Is(err, domainerrors.ErrNoActiveAllocation):
		response.Error(c, http.StatusUnprocessableEntity, "no_active_allocation", err.Error())
	case errors.Is(err, domainerrors.ErrProjectOrCategoryRequired),
		errors.Is(err, domainerrors.ErrInvalidActivityCategory),
		errors.Is(err, domainerrors.ErrInvalidStatusLogHours),
		errors.Is(err, domainerrors.ErrInvalidStatusLogDate),
		errors.Is(err, domainerrors.ErrStatusLogDateRangeExceeded),
		errors.Is(err, domainerrors.ErrInvalidProjectData):
		response.BadRequest(c, err.Error())
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// ListMyStatusLogs lists the authenticated employee's status logs grouped by date.
//
// @Summary      List my status logs
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Param        from     query     string  false  "Start date (YYYY-MM-DD)"
// @Param        to       query     string  false  "End date (YYYY-MM-DD)"
// @Param        project  query     string  false  "Filter by project UUID"
// @Success      200      {object}  response.Envelope{data=[]statuslog.DailyStatusLogGroupResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Router       /api/v1/me/status-logs [get]
func (h *StatusLogHandler) ListMyStatusLogs(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")
	projectStr := c.Query("project")
	if projectStr == "" {
		projectStr = c.Query("project_id")
	}

	groups, err := h.uc.ListMy.Execute(c.Request.Context(), tenantID, userID, fromStr, toStr, projectStr)
	if err != nil {
		writeStatusLogError(c, err)
		return
	}

	response.Success(c, statuslog.ToDailyStatusLogGroupResponses(groups))
}

// ListMyDayStatuses returns the authenticated employee's per-day status over a range.
//
// @Summary      List my day statuses
// @Description  One item per calendar day in [from, to], oldest first, classified as logged, holiday (public categories only), weekly_off, on_leave (approved leave only) or missing, with the same precedence as the admin daily summary. Days after today are omitted. Defaults and the 366-day cap match GET /me/status-logs.
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Param        from  query     string  false  "Start date (YYYY-MM-DD), default today-6"
// @Param        to    query     string  false  "End date (YYYY-MM-DD), default today"
// @Success      200   {object}  response.Envelope{data=[]statuslog.MyDayStatusResponse}
// @Failure      400   {object}  response.Envelope
// @Failure      401   {object}  response.Envelope
// @Router       /api/v1/me/status-logs/day-statuses [get]
func (h *StatusLogHandler) ListMyDayStatuses(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	days, err := h.uc.MyDayStatus.Execute(c.Request.Context(), tenantID, userID, c.Query("from"), c.Query("to"))
	if err != nil {
		writeStatusLogError(c, err)
		return
	}
	response.Success(c, statuslog.ToMyDayStatusResponses(days))
}

// CreateMyStatusLog creates a status log entry for the authenticated employee.
//
// @Summary      Create status log entry
// @Tags         me
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      statuslog.CreateStatusLogRequest  true  "Status log entry"
// @Success      201      {object}  response.Envelope{data=statuslog.StatusLogEntryResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Router       /api/v1/me/status-logs [post]
func (h *StatusLogHandler) CreateMyStatusLog(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	var req statuslog.CreateStatusLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	entry, err := h.uc.Create.Execute(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		writeStatusLogError(c, err)
		return
	}

	response.Created(c, statuslog.ToStatusLogEntryResponse(entry))
}

// UpdateMyStatusLog updates an existing status log entry owned by the employee.
//
// @Summary      Update status log entry
// @Tags         me
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                            true  "Status log UUID"
// @Param        request  body      statuslog.UpdateStatusLogRequest  true  "Status log updates"
// @Success      200      {object}  response.Envelope{data=statuslog.StatusLogEntryResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Router       /api/v1/me/status-logs/{id} [patch]
func (h *StatusLogHandler) UpdateMyStatusLog(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, ok := pathUUID(c, "id", "status log id")
	if !ok {
		return
	}

	var req statuslog.UpdateStatusLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	entry, err := h.uc.Update.Execute(c.Request.Context(), tenantID, userID, id, req)
	if err != nil {
		writeStatusLogError(c, err)
		return
	}

	response.Success(c, statuslog.ToStatusLogEntryResponse(entry))
}

// DeleteMyStatusLog soft-deletes a status log entry owned by the employee.
//
// @Summary      Delete status log entry
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Status log UUID"
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Router       /api/v1/me/status-logs/{id} [delete]
func (h *StatusLogHandler) DeleteMyStatusLog(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	id, ok := pathUUID(c, "id", "status log id")
	if !ok {
		return
	}

	if err := h.uc.Delete.Execute(c.Request.Context(), tenantID, userID, id); err != nil {
		writeStatusLogError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "status log entry deleted successfully"})
}

// GetDailyStatusLogs returns admin daily summary of status logging across active employees.
//
// @Summary      Get daily status logs summary
// @Tags         status-logs
// @Produce      json
// @Security     BearerAuth
// @Param        date        query     string  false  "Calendar date (YYYY-MM-DD)"
// @Param        department  query     string  false  "Filter by department UUID"
// @Param        project     query     string  false  "Filter by project UUID"
// @Success      200         {object}  response.Envelope{data=[]statuslog.DailyEmployeeSummaryResponse}
// @Failure      400         {object}  response.Envelope
// @Failure      401         {object}  response.Envelope
// @Failure      403         {object}  response.Envelope
// @Router       /api/v1/status-logs/daily [get]
func (h *StatusLogHandler) GetDailyStatusLogs(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	dateStr := c.Query("date")
	deptStr := c.Query("department")
	if deptStr == "" {
		deptStr = c.Query("department_id")
	}
	projStr := c.Query("project")
	if projStr == "" {
		projStr = c.Query("project_id")
	}

	summaries, err := h.uc.GetDaily.Execute(c.Request.Context(), tenantID, dateStr, deptStr, projStr)
	if err != nil {
		writeStatusLogError(c, err)
		return
	}

	response.Success(c, statuslog.ToDailyEmployeeSummaryResponses(summaries))
}

// GetMonthlyStatusLogs returns the admin org-wide monthly status log report.
//
// @Summary      Get monthly status logs report
// @Tags         status-logs
// @Produce      json
// @Security     BearerAuth
// @Param        month       query     string  false  "Calendar month (YYYY-MM); defaults to the current month"
// @Param        department  query     string  false  "Filter by department UUID"
// @Param        project     query     string  false  "Filter by project UUID"
// @Success      200         {object}  response.Envelope{data=statuslog.MonthlyStatusLogReportResponse}
// @Failure      400         {object}  response.Envelope
// @Failure      401         {object}  response.Envelope
// @Failure      403         {object}  response.Envelope
// @Router       /api/v1/status-logs/monthly [get]
func (h *StatusLogHandler) GetMonthlyStatusLogs(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	report, err := h.uc.GetMonthly.Execute(c.Request.Context(), tenantID, c.Query("month"), c.Query("department"), c.Query("project"))
	if err != nil {
		writeStatusLogError(c, err)
		return
	}

	response.Success(c, statuslog.ToMonthlyStatusLogReportResponse(report))
}

// ListEmployeeStatusLogs returns a specific employee's status logs grouped by date.
//
// @Summary      List employee status logs
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string  true   "Employee UUID"
// @Param        from     query     string  false  "Start date (YYYY-MM-DD)"
// @Param        to       query     string  false  "End date (YYYY-MM-DD)"
// @Param        project  query     string  false  "Filter by project UUID"
// @Success      200      {object}  response.Envelope{data=[]statuslog.DailyStatusLogGroupResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Router       /api/v1/employees/{id}/status-logs [get]
func (h *StatusLogHandler) ListEmployeeStatusLogs(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")
	projectStr := c.Query("project")
	if projectStr == "" {
		projectStr = c.Query("project_id")
	}

	groups, err := h.uc.ListEmployee.Execute(c.Request.Context(), tenantID, id, fromStr, toStr, projectStr)
	if err != nil {
		writeStatusLogError(c, err)
		return
	}

	response.Success(c, statuslog.ToDailyStatusLogGroupResponses(groups))
}
