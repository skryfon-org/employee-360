package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
	orguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/organization"
)

// WeeklyOffDaysHandler serves the employee-readable weekly off days. The route
// group enforces Auth -> Tenant -> RequireRole(admin, super_admin, employee); the
// tenant comes only from the auth context.
type WeeklyOffDaysHandler struct {
	get orguc.GetWeeklyOffDaysUseCase
}

// NewWeeklyOffDaysHandler constructs a WeeklyOffDaysHandler.
func NewWeeklyOffDaysHandler(get orguc.GetWeeklyOffDaysUseCase) *WeeklyOffDaysHandler {
	return &WeeklyOffDaysHandler{get: get}
}

// GetWeeklyOffDays returns the caller's tenant weekly off days (0 = Sunday ..
// 6 = Saturday; default [0,6]).
//
// @Summary      Get weekly off days
// @Tags         organization-settings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=orgtypes.WeeklyOffDaysResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/organization/weekly-off-days [get]
func (h *WeeklyOffDaysHandler) GetWeeklyOffDays(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	days, err := h.get.Execute(c.Request.Context(), tenantID)
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	if days == nil {
		days = []int{}
	}
	response.Success(c, orgtypes.WeeklyOffDaysResponse{WeeklyOffDays: days})
}
