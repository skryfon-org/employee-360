package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	dashtypes "github.com/skryfon-org/employee-360/backend/internal/types/dashboard"
	dashusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/dashboard"
)

// DashboardHandler serves the read-only admin and super-admin dashboards. Role
// guards live in route registration; the tenant comes from the auth context.
type DashboardHandler struct {
	adminUC      dashusecase.AdminDashboardUseCase
	superAdminUC dashusecase.SuperAdminDashboardUseCase
	employeeUC   dashusecase.EmployeeDashboardUseCase
}

// NewDashboardHandler constructs a DashboardHandler.
func NewDashboardHandler(adminUC dashusecase.AdminDashboardUseCase, superAdminUC dashusecase.SuperAdminDashboardUseCase) *DashboardHandler {
	return &DashboardHandler{adminUC: adminUC, superAdminUC: superAdminUC}
}

// WithEmployee enables the employee dashboard endpoint and returns the handler.
func (h *DashboardHandler) WithEmployee(uc dashusecase.EmployeeDashboardUseCase) *DashboardHandler {
	h.employeeUC = uc
	return h
}

// HasEmployee reports whether the employee dashboard usecase is configured.
func (h *DashboardHandler) HasEmployee() bool { return h.employeeUC != nil }

func writeDashboardError(c *gin.Context, err error) {
	if errors.Is(err, domainerrors.ErrUnauthorized) {
		response.Unauthorized(c, "unauthorized")
		return
	}
	response.Internal(c, "an unexpected error occurred")
}

// Admin returns the caller's tenant dashboard (admin only).
//
// @Summary      Tenant admin dashboard
// @Description  Counts of users, invitations by status, departments and positions for the caller's tenant, plus the 5 most recent invitations.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=dashtypes.AdminDashboardResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/dashboard/admin [get]
func (h *DashboardHandler) Admin(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	res, err := h.adminUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		writeDashboardError(c, err)
		return
	}
	response.Success(c, dashtypes.ToAdminDashboardResponse(res))
}

// SuperAdmin returns the organisation overview (super_admin only).
//
// @Summary      Super admin dashboard
// @Description  Organisation overview of the caller's tenant: tenant info, user counts (total/active/inactive/pending-invited and by role), invitations by status, departments, positions and the 5 most recent invitations. super_admin only; no cross-tenant aggregates.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=dashtypes.SuperAdminDashboardResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/dashboard/super-admin [get]
func (h *DashboardHandler) SuperAdmin(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	res, err := h.superAdminUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		writeDashboardError(c, err)
		return
	}
	response.Success(c, dashtypes.ToSuperAdminDashboardResponse(res))
}

// Employee returns the caller's own home dashboard.
//
// @Summary      Employee dashboard
// @Description  Read-only summary for the authenticated user: greeting and today's day type, KPI tiles (pending leave, upcoming approved leave days, allocation percent, hours over the last 7 days, pending team approvals for approvers only), leave overview, next 5 holidays, active projects and the last-7-days (rolling, not calendar week) hours window with missing working days. Scoped to the caller's tenant and user.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=dashtypes.EmployeeDashboardResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/dashboard/employee [get]
func (h *DashboardHandler) Employee(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	res, err := h.employeeUC.Execute(c.Request.Context(), tenantID, userID)
	if err != nil {
		writeDashboardError(c, err)
		return
	}
	response.Success(c, res)
}
