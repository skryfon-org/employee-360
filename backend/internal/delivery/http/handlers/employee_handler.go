package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

// EmployeeUseCases groups the usecases behind the admin employee endpoints and /me.
type EmployeeUseCases struct {
	List          empuc.ListEmployeesUseCase
	Get           empuc.GetEmployeeUseCase
	Update        empuc.UpdateEmployeeUseCase
	Assignments   empuc.ListAssignmentsUseCase
	ListContacts  empuc.ListEmergencyContactsUseCase
	CreateContact empuc.CreateEmergencyContactUseCase
	UpdateContact empuc.UpdateEmergencyContactUseCase
	DeleteContact empuc.DeleteEmergencyContactUseCase
	Deactivate    empuc.DeactivateEmployeeUseCase
	Activate      empuc.ActivateEmployeeUseCase
	ChangeRole    empuc.ChangeEmployeeRoleUseCase
	Me            empuc.GetMeUseCase
	MyProfile     empuc.GetMyProfileUseCase
	UpdateMyProf  empuc.UpdateMyProfileUseCase
}

// EmployeeHandler serves admin employee management and the employee's own /me endpoints.
type EmployeeHandler struct{ uc EmployeeUseCases }

// NewEmployeeHandler constructs an EmployeeHandler.
func NewEmployeeHandler(uc EmployeeUseCases) *EmployeeHandler { return &EmployeeHandler{uc: uc} }

func writeEmployeeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrUserNotFound),
		errors.Is(err, domainerrors.ErrEmergencyContactNotFound),
		errors.Is(err, domainerrors.ErrEmployeeProfileNotFound):
		response.NotFound(c, "not found")
	case errors.Is(err, domainerrors.ErrDepartmentNotFound):
		response.BadRequest(c, "department not found")
	case errors.Is(err, domainerrors.ErrPositionNotFound):
		response.BadRequest(c, "position not found")
	case errors.Is(err, domainerrors.ErrDepartmentInactive):
		response.Error(c, http.StatusConflict, "CONFLICT", "department is inactive")
	case errors.Is(err, domainerrors.ErrPositionInactive):
		response.Error(c, http.StatusConflict, "CONFLICT", "position is inactive")
	case errors.Is(err, domainerrors.ErrManagerInvalid),
		errors.Is(err, domainerrors.ErrAssignmentDateOrder),
		errors.Is(err, domainerrors.ErrInvalidEmployeeData):
		response.BadRequest(c, err.Error())
	case errors.Is(err, domainerrors.ErrEmployeeCodeTaken),
		errors.Is(err, domainerrors.ErrEmployeeProfileExists),
		errors.Is(err, domainerrors.ErrLastActiveAdmin),
		errors.Is(err, domainerrors.ErrUserNotOnboarded):
		response.Error(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, domainerrors.ErrCannotActOnSelf),
		errors.Is(err, domainerrors.ErrCannotActOnSuperAdmin),
		errors.Is(err, domainerrors.ErrRoleChangeNotAllowed),
		errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized),
		errors.Is(err, domainerrors.ErrTenantNotFound):
		response.Unauthorized(c, "unauthorized")
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// actor builds the caller from the authenticated request context.
func actorFrom(c *gin.Context) (emptypes.Actor, bool) {
	tid, uid, ok := tenantAndUser(c)
	if !ok {
		return emptypes.Actor{}, false
	}
	roles, _ := ctx.RolesFromContext(c.Request.Context())
	return emptypes.Actor{TenantID: tid, UserID: uid, Roles: roles}, true
}

// List returns a paginated, filterable employee directory.
//
// @Summary      List employees
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        include            query  string false "Set to allocation to add allocated_percent (today's allocation on live projects) to each item" Enums(allocation)
// @Param        search             query  string false "Search name, email or employee code"
// @Param        status             query  string false "active or inactive" Enums(active, inactive)
// @Param        department_id      query  string false "Department UUID"
// @Param        position_id        query  string false "Position UUID"
// @Param        employment_status  query  string false "Employment status" Enums(active, probation, on_leave, notice_period, resigned, terminated)
// @Param        page               query  int    false "Page number (default 1)"
// @Param        page_size          query  int    false "Page size (default 20, max 100)"
// @Success      200  {object}  response.Envelope{data=[]emptypes.EmployeeSummaryResponse,meta=response.Meta}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/employees [get]
func (h *EmployeeHandler) List(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}
	var req emptypes.ListEmployeesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid query parameters")
		return
	}
	f, err := req.ToFilter()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.uc.List.Execute(c.Request.Context(), tenantID, emptypes.ListEmployeesQuery{Page: page, PageSize: pageSize, Filter: f})
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	items := make([]emptypes.EmployeeSummaryResponse, 0, len(out.Employees))
	for _, e := range out.Employees {
		items = append(items, emptypes.ToSummaryResponse(e))
	}
	response.Paginated(c, items, pageMeta(out.Page, out.PageSize, out.Total))
}

// Get returns one employee with profile and emergency contacts.
//
// @Summary      Get employee
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string true "User UUID"
// @Success      200  {object}  response.Envelope{data=emptypes.EmployeeDetailResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/employees/{id} [get]
func (h *EmployeeHandler) Get(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	d, err := h.uc.Get.Execute(c.Request.Context(), tenantID, id)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToDetailResponse(d))
}

// Update updates user fields and upserts the employee profile. Department or
// position changes close/open the assignment history in one transaction.
//
// @Summary      Update employee
// @Description  Absent fields are unchanged; null or "" clears optional fields. PUT and PATCH behave identically: both are partial updates (PUT is not a full replace).
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string true "User UUID"
// @Param        request  body  emptypes.UpdateEmployeeInput true "Fields to update"
// @Success      200  {object}  response.Envelope{data=emptypes.EmployeeDetailResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/employees/{id} [put]
// @Router       /api/v1/employees/{id} [patch]
func (h *EmployeeHandler) Update(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	var in emptypes.UpdateEmployeeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	d, err := h.uc.Update.Execute(c.Request.Context(), actor, id, in)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToDetailResponse(d))
}

// ListAssignments returns the department/position history, newest first.
//
// @Summary      List employee assignment history
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string true "User UUID"
// @Success      200  {object}  response.Envelope{data=[]emptypes.AssignmentResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/assignments [get]
func (h *EmployeeHandler) ListAssignments(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	as, err := h.uc.Assignments.Execute(c.Request.Context(), tenantID, id)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToAssignmentResponses(as))
}

// Deactivate deactivates an employee and revokes their refresh tokens.
//
// @Summary      Deactivate employee
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string true "User UUID"
// @Success      200  {object}  response.Envelope{data=emptypes.EmployeeDetailResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/deactivate [post]
func (h *EmployeeHandler) Deactivate(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	d, err := h.uc.Deactivate.Execute(c.Request.Context(), actor, id)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToDetailResponse(d))
}

// Activate reactivates a deactivated employee.
//
// @Summary      Activate employee
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string true "User UUID"
// @Success      200  {object}  response.Envelope{data=emptypes.EmployeeDetailResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/activate [post]
func (h *EmployeeHandler) Activate(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	d, err := h.uc.Activate.Execute(c.Request.Context(), actor, id)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToDetailResponse(d))
}

// ChangeRole switches an employee between the employee and admin roles
// (super_admin only; never self; super_admin role is untouchable).
//
// @Summary      Change employee role
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string true "User UUID"
// @Param        request  body  emptypes.ChangeRoleInput true "New role"
// @Success      200  {object}  response.Envelope{data=emptypes.EmployeeDetailResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/role [put]
func (h *EmployeeHandler) ChangeRole(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	var in emptypes.ChangeRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "role must be employee or admin")
		return
	}
	d, err := h.uc.ChangeRole.Execute(c.Request.Context(), actor, id, in.Role)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToDetailResponse(d))
}

// ListContacts lists an employee's emergency contacts.
//
// @Summary      List employee emergency contacts
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string true "User UUID"
// @Success      200  {object}  response.Envelope{data=[]emptypes.EmergencyContactResponse}
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/emergency-contacts [get]
func (h *EmployeeHandler) ListContacts(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	h.listContacts(c, tenantID, id)
}

// CreateContact adds an emergency contact for an employee.
//
// @Summary      Add employee emergency contact
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string true "User UUID"
// @Param        request  body  emptypes.EmergencyContactInput true "Contact"
// @Success      201  {object}  response.Envelope{data=emptypes.EmergencyContactResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/emergency-contacts [post]
func (h *EmployeeHandler) CreateContact(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	h.createContact(c, actor, id)
}

// UpdateContact replaces an emergency contact of an employee.
//
// @Summary      Update employee emergency contact
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string true "User UUID"
// @Param        contactId  path  string true "Contact UUID"
// @Param        request    body  emptypes.EmergencyContactInput true "Contact"
// @Success      200  {object}  response.Envelope{data=emptypes.EmergencyContactResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/emergency-contacts/{contactId} [put]
func (h *EmployeeHandler) UpdateContact(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	h.updateContact(c, actor, id)
}

// DeleteContact removes an emergency contact of an employee.
//
// @Summary      Delete employee emergency contact
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string true "User UUID"
// @Param        contactId  path  string true "Contact UUID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/employees/{id}/emergency-contacts/{contactId} [delete]
func (h *EmployeeHandler) DeleteContact(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id", "employee id")
	if !ok {
		return
	}
	h.deleteContact(c, actor, id)
}

func (h *EmployeeHandler) listContacts(c *gin.Context, tenantID, userID uuid.UUID) {
	cs, err := h.uc.ListContacts.Execute(c.Request.Context(), tenantID, userID)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToContactResponses(cs))
}

func (h *EmployeeHandler) createContact(c *gin.Context, actor emptypes.Actor, userID uuid.UUID) {
	var in emptypes.EmergencyContactInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	ec, err := h.uc.CreateContact.Execute(c.Request.Context(), actor, userID, in)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Created(c, emptypes.ToContactResponse(ec))
}

func (h *EmployeeHandler) updateContact(c *gin.Context, actor emptypes.Actor, userID uuid.UUID) {
	cid, ok := pathUUID(c, "contactId", "contact id")
	if !ok {
		return
	}
	var in emptypes.EmergencyContactInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	ec, err := h.uc.UpdateContact.Execute(c.Request.Context(), actor, userID, cid, in)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToContactResponse(ec))
}

func (h *EmployeeHandler) deleteContact(c *gin.Context, actor emptypes.Actor, userID uuid.UUID) {
	cid, ok := pathUUID(c, "contactId", "contact id")
	if !ok {
		return
	}
	if err := h.uc.DeleteContact.Execute(c.Request.Context(), actor, userID, cid); err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "emergency contact deleted successfully"})
}

// ---- /me (the caller's own data; user id from auth context only) ----

// Me returns the caller's identity, organization and onboarding state.
//
// @Summary      Current user
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=emptypes.MeResponse}
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/me [get]
func (h *EmployeeHandler) Me(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	m, err := h.uc.Me.Execute(c.Request.Context(), tenantID, userID)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToMeResponse(m))
}

// GetMyProfile returns the caller's profile, contacts and completion state.
//
// @Summary      Get my profile
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=emptypes.MyProfileResponse}
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/me/profile [get]
func (h *EmployeeHandler) GetMyProfile(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	m, err := h.uc.MyProfile.Execute(c.Request.Context(), tenantID, userID)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToMyProfileResponse(m))
}

// UpdateMyProfile updates the caller's self-service profile fields.
//
// @Summary      Update my profile
// @Description  Personal and address fields only; employment fields are read-only for the employee.
// @Tags         me
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  emptypes.UpdateMyProfileInput true "Profile fields"
// @Success      200  {object}  response.Envelope{data=emptypes.MyProfileResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Router       /api/v1/me/profile [patch]
func (h *EmployeeHandler) UpdateMyProfile(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	var in emptypes.UpdateMyProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	m, err := h.uc.UpdateMyProf.Execute(c.Request.Context(), tenantID, userID, in)
	if err != nil {
		writeEmployeeError(c, err)
		return
	}
	response.Success(c, emptypes.ToMyProfileResponse(m))
}

// ListMyContacts lists the caller's emergency contacts.
//
// @Summary      List my emergency contacts
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=[]emptypes.EmergencyContactResponse}
// @Router       /api/v1/me/emergency-contacts [get]
func (h *EmployeeHandler) ListMyContacts(c *gin.Context) {
	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	h.listContacts(c, tenantID, userID)
}

// CreateMyContact adds an emergency contact for the caller.
//
// @Summary      Add my emergency contact
// @Tags         me
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  emptypes.EmergencyContactInput true "Contact"
// @Success      201  {object}  response.Envelope{data=emptypes.EmergencyContactResponse}
// @Failure      400  {object}  response.Envelope
// @Router       /api/v1/me/emergency-contacts [post]
func (h *EmployeeHandler) CreateMyContact(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	h.createContact(c, actor, actor.UserID)
}

// UpdateMyContact replaces one of the caller's emergency contacts.
//
// @Summary      Update my emergency contact
// @Tags         me
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        contactId  path  string true "Contact UUID"
// @Param        request    body  emptypes.EmergencyContactInput true "Contact"
// @Success      200  {object}  response.Envelope{data=emptypes.EmergencyContactResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/me/emergency-contacts/{contactId} [put]
func (h *EmployeeHandler) UpdateMyContact(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	h.updateContact(c, actor, actor.UserID)
}

// DeleteMyContact removes one of the caller's emergency contacts.
//
// @Summary      Delete my emergency contact
// @Tags         me
// @Produce      json
// @Security     BearerAuth
// @Param        contactId  path  string true "Contact UUID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/me/emergency-contacts/{contactId} [delete]
func (h *EmployeeHandler) DeleteMyContact(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	h.deleteContact(c, actor, actor.UserID)
}
