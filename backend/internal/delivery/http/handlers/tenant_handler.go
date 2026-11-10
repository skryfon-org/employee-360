package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	tenanttypes "github.com/skryfon-org/employee-360/backend/internal/types/tenant"
	tenantusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/tenant"
)

// TenantUseCases bundles the current-tenant settings usecases the handler depends on.
type TenantUseCases struct {
	Get          tenantusecase.GetTenantUseCase
	Rename       tenantusecase.RenameTenantUseCase
	ListDomains  tenantusecase.ListTenantDomainsUseCase
	AddDomain    tenantusecase.AddTenantDomainUseCase
	UpdateDomain tenantusecase.UpdateTenantDomainUseCase
	DelDomain    tenantusecase.RemoveTenantDomainUseCase
}

// TenantHandler serves the super_admin current-tenant settings endpoints. The
// route group enforces Auth -> Tenant -> RequireRole(super_admin). The tenant
// and the actor come only from the auth context: no tenant id is accepted in
// the URL or body (the product is self-hosted; there is no cross-tenant
// management).
type TenantHandler struct {
	uc TenantUseCases
}

// NewTenantHandler constructs a TenantHandler.
func NewTenantHandler(uc TenantUseCases) *TenantHandler {
	return &TenantHandler{uc: uc}
}

func writeTenantError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrTenantNotFound):
		response.NotFound(c, "tenant not found")
	case errors.Is(err, domainerrors.ErrDomainNotFound):
		response.NotFound(c, "tenant domain not found")
	case errors.Is(err, domainerrors.ErrDomainAlreadyExists):
		response.Error(c, http.StatusConflict, "DOMAIN_ALREADY_EXISTS", "this domain is already registered")
	case errors.Is(err, domainerrors.ErrLastDomain):
		response.Error(c, http.StatusConflict, "LAST_DOMAIN", "a tenant must keep at least one domain; add another before removing this one")
	case errors.Is(err, domainerrors.ErrDomainInUse):
		response.Error(c, http.StatusConflict, "DOMAIN_IN_USE", "users in your organization still sign in with an @<domain> email; move or deactivate them before removing or changing this domain")
	case errors.Is(err, domainerrors.ErrInvalidDomain):
		response.Error(c, http.StatusBadRequest, "INVALID_DOMAIN", "domain must be a fully qualified domain with at least two labels, such as example.com (single-label hosts like localhost are not supported)")
	case errors.Is(err, domainerrors.ErrInvalidTenantName):
		response.Error(c, http.StatusBadRequest, "INVALID_NAME",
			fmt.Sprintf("name is required and must be at most %d characters", tenanttypes.MaxNameLen))
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

func pathUUID(c *gin.Context, param, label string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil || id == uuid.Nil {
		response.BadRequest(c, "invalid "+label)
		return uuid.Nil, false
	}
	return id, true
}

// Get returns the caller's tenant with its domains.
//
// @Summary      Get current tenant
// @Description  Returns the caller's own tenant and its live domains (super_admin only). The tenant is taken from the auth context.
// @Tags         tenant
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=tenanttypes.TenantDetailResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/tenant [get]
func (h *TenantHandler) Get(c *gin.Context) {
	tenantID, _, ok := tenantAndUser(c)
	if !ok {
		return
	}
	res, err := h.uc.Get.Execute(c.Request.Context(), tenantID)
	if err != nil {
		writeTenantError(c, err)
		return
	}
	response.Success(c, tenanttypes.ToTenantDetailResponse(res))
}

// Update renames the caller's tenant.
//
// @Summary      Rename current tenant
// @Tags         tenant
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      tenanttypes.UpdateTenantRequest  true  "New name"
// @Success      200      {object}  response.Envelope{data=tenanttypes.TenantResponse}
// @Failure      400      {object}  response.Envelope  "Invalid payload or INVALID_NAME"
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Router       /api/v1/tenant [patch]
func (h *TenantHandler) Update(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	var req tenanttypes.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	t, err := h.uc.Rename.Execute(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		writeTenantError(c, err)
		return
	}
	response.Success(c, tenanttypes.ToTenantResponse(t))
}

// ListDomains returns the caller's tenant domains.
//
// @Summary      List current tenant domains
// @Tags         tenant
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=[]tenanttypes.TenantDomainResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/tenant/domains [get]
func (h *TenantHandler) ListDomains(c *gin.Context) {
	tenantID, _, ok := tenantAndUser(c)
	if !ok {
		return
	}
	ds, err := h.uc.ListDomains.Execute(c.Request.Context(), tenantID)
	if err != nil {
		writeTenantError(c, err)
		return
	}
	response.Success(c, tenanttypes.ToTenantDomainResponses(ds))
}

// AddDomain registers a domain for the caller's tenant.
//
// @Summary      Add current tenant domain
// @Tags         tenant
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      tenanttypes.AddDomainRequest  true  "Domain"
// @Success      201      {object}  response.Envelope{data=tenanttypes.TenantDomainResponse}
// @Failure      400      {object}  response.Envelope  "Invalid payload or INVALID_DOMAIN"
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope  "DOMAIN_ALREADY_EXISTS"
// @Router       /api/v1/tenant/domains [post]
func (h *TenantHandler) AddDomain(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	var req tenanttypes.AddDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	d, err := h.uc.AddDomain.Execute(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		writeTenantError(c, err)
		return
	}
	response.Created(c, tenanttypes.ToTenantDomainResponse(d))
}

// UpdateDomain changes the value of one of the caller's tenant domains.
//
// @Summary      Update current tenant domain
// @Description  Same normalisation and global uniqueness rules as add. Setting the current value is a no-op. A domain that is not the caller's tenant's (or is removed) is 404.
// @Tags         tenant
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        domainId  path      string                           true  "Domain ID"
// @Param        request   body      tenanttypes.UpdateDomainRequest  true  "New domain value"
// @Success      200       {object}  response.Envelope{data=tenanttypes.TenantDomainResponse}
// @Failure      400       {object}  response.Envelope  "Invalid payload, invalid id or INVALID_DOMAIN"
// @Failure      401       {object}  response.Envelope
// @Failure      403       {object}  response.Envelope
// @Failure      404       {object}  response.Envelope
// @Failure      409       {object}  response.Envelope  "DOMAIN_ALREADY_EXISTS or DOMAIN_IN_USE"
// @Router       /api/v1/tenant/domains/{domainId} [patch]
func (h *TenantHandler) UpdateDomain(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	domainID, ok := pathUUID(c, "domainId", "domain id")
	if !ok {
		return
	}
	var req tenanttypes.UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	d, err := h.uc.UpdateDomain.Execute(c.Request.Context(), actorID, tenantID, domainID, req)
	if err != nil {
		writeTenantError(c, err)
		return
	}
	response.Success(c, tenanttypes.ToTenantDomainResponse(d))
}

// RemoveDomain soft-deletes one of the caller's tenant domains.
//
// @Summary      Remove current tenant domain
// @Description  The tenant's last remaining domain cannot be removed (409 LAST_DOMAIN).
// @Tags         tenant
// @Produce      json
// @Security     BearerAuth
// @Param        domainId  path      string  true  "Domain ID"
// @Success      200       {object}  response.Envelope
// @Failure      400       {object}  response.Envelope
// @Failure      401       {object}  response.Envelope
// @Failure      403       {object}  response.Envelope
// @Failure      404       {object}  response.Envelope
// @Failure      409       {object}  response.Envelope  "LAST_DOMAIN or DOMAIN_IN_USE"
// @Router       /api/v1/tenant/domains/{domainId} [delete]
func (h *TenantHandler) RemoveDomain(c *gin.Context) {
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	domainID, ok := pathUUID(c, "domainId", "domain id")
	if !ok {
		return
	}
	if err := h.uc.DelDomain.Execute(c.Request.Context(), actorID, tenantID, domainID); err != nil {
		writeTenantError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "domain removed"})
}
