package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
	orguc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/organization"
)

// OrganizationHandler serves the organization settings endpoints. The route group
// enforces Auth -> Tenant -> RequireRole(admin, super_admin); the tenant and actor
// come only from the auth context, never from the URL or body.
type OrganizationHandler struct {
	getSettings    orguc.GetSettingsUseCase
	updateSettings orguc.UpdateSettingsUseCase
}

// NewOrganizationHandler constructs an OrganizationHandler.
func NewOrganizationHandler(get orguc.GetSettingsUseCase, update orguc.UpdateSettingsUseCase) *OrganizationHandler {
	return &OrganizationHandler{getSettings: get, updateSettings: update}
}

func writeOrganizationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrInvalidOrganizationSettings):
		response.BadRequest(c, err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// GetSettings returns the organization settings of the caller's tenant (defaults
// when none were saved yet).
//
// @Summary      Get organization settings
// @Tags         organization-settings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=orgtypes.SettingsResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/organization/settings [get]
func (h *OrganizationHandler) GetSettings(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	s, err := h.getSettings.Execute(c.Request.Context(), tenantID)
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	response.Success(c, orgtypes.ToSettingsResponse(s))
}

// UpdateSettings saves the employee code format of the caller's tenant. The
// counter is server-managed and cannot be set. A prefix change affects newly
// generated codes only.
//
// @Summary      Update organization settings
// @Tags         organization-settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      orgtypes.UpdateSettingsRequest  true  "Employee code format and weekly off days"
// @Success      200      {object}  response.Envelope{data=orgtypes.SettingsResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Router       /api/v1/organization/settings [put]
func (h *OrganizationHandler) UpdateSettings(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	var req orgtypes.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	s, err := h.updateSettings.Execute(c.Request.Context(), tenantID, actorID, orgtypes.UpdateSettingsInput{
		EmployeeCodePrefix:      req.EmployeeCodePrefix,
		EmployeeCodePadding:     *req.EmployeeCodePadding,
		EmployeeCodeStartNumber: *req.EmployeeCodeStartNumber,
		WelcomeMessage:          req.WelcomeMessage,
		WeeklyOffDays:           req.WeeklyOffDays,
	})
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	response.Success(c, orgtypes.ToSettingsResponse(s))
}
