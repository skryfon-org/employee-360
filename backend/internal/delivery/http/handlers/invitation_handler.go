package handlers

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
	invusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/invitation"
)

// InvitationHandler handles HTTP requests for onboarding invitations.
// Tenant and actor identity come exclusively from the request context populated
// by the Auth/Tenant middleware; request bodies never carry a tenant ID.
type InvitationHandler struct {
	inviteUC   invusecase.InviteUserUseCase
	acceptUC   invusecase.AcceptInvitationUseCase
	resendUC   invusecase.ResendInvitationUseCase
	revokeUC   invusecase.RevokeInvitationUseCase
	listUC     invusecase.ListInvitationsUseCase
	validateUC invusecase.ValidateInvitationUseCase
}

// NewInvitationHandler constructs an InvitationHandler.
func NewInvitationHandler(
	inviteUC invusecase.InviteUserUseCase,
	acceptUC invusecase.AcceptInvitationUseCase,
	resendUC invusecase.ResendInvitationUseCase,
	revokeUC invusecase.RevokeInvitationUseCase,
	listUC invusecase.ListInvitationsUseCase,
	validateUC invusecase.ValidateInvitationUseCase,
) *InvitationHandler {
	return &InvitationHandler{
		inviteUC:   inviteUC,
		acceptUC:   acceptUC,
		resendUC:   resendUC,
		revokeUC:   revokeUC,
		listUC:     listUC,
		validateUC: validateUC,
	}
}

// writeInvitationError maps domain errors to HTTP responses.
func writeInvitationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, "insufficient permissions")
	case errors.Is(err, domainerrors.ErrInvitationNotFound):
		response.NotFound(c, "invitation not found")
	case errors.Is(err, domainerrors.ErrInvitationNotPending):
		response.Error(c, 409, "CONFLICT", "invitation is no longer pending")
	case errors.Is(err, domainerrors.ErrEmailAlreadyExists):
		response.Error(c, 409, "CONFLICT", "a user with that email already exists")
	case errors.Is(err, domainerrors.ErrInvitationExpired):
		response.Error(c, http.StatusGone, "INVITATION_EXPIRED", "this invitation has expired; ask your administrator to resend it")
	case errors.Is(err, domainerrors.ErrInvitationRevoked):
		response.Error(c, http.StatusForbidden, "INVITATION_REVOKED", "this invitation has been revoked; contact your administrator")
	case errors.Is(err, domainerrors.ErrInvitationAccepted):
		response.Error(c, http.StatusConflict, "INVITATION_ACCEPTED", "this invitation has already been accepted; sign in instead")
	case errors.Is(err, domainerrors.ErrInvalidToken):
		response.Error(c, http.StatusBadRequest, "INVALID_TOKEN", "invalid invitation token")
	case errors.Is(err, domainerrors.ErrEmailDomainNotAllowed):
		response.Error(c, http.StatusBadRequest, "EMAIL_DOMAIN_NOT_ALLOWED", "the email domain is not registered for your organization; invite an address on one of its domains")
	case errors.Is(err, domainerrors.ErrDepartmentInactive):
		response.Error(c, http.StatusBadRequest, "DEPARTMENT_INACTIVE", "the selected department is inactive; choose an active department")
	case errors.Is(err, domainerrors.ErrPositionInactive):
		response.Error(c, http.StatusBadRequest, "POSITION_INACTIVE", "the selected position is inactive; choose an active position")
	case errors.Is(err, domainerrors.ErrRoleNotFound),
		errors.Is(err, domainerrors.ErrDepartmentNotFound),
		errors.Is(err, domainerrors.ErrPositionNotFound),
		errors.Is(err, domainerrors.ErrInvalidRole),
		errors.Is(err, domainerrors.ErrInvalidEmail),
		errors.Is(err, domainerrors.ErrInvalidInvitationFilter),
		errors.Is(err, domainerrors.ErrInvalidPassword):
		response.BadRequest(c, err.Error())
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

// Invite creates a pending user and invitation (admin only).
//
// @Summary      Invite user
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      invtypes.InviteUserRequest  true  "Invitation details"
// @Success      201      {object}  response.Envelope{data=invtypes.InvitationResponse}
// @Failure      400      {object}  response.Envelope  "Invalid payload, email domain not registered for the tenant (EMAIL_DOMAIN_NOT_ALLOWED), inactive department (DEPARTMENT_INACTIVE), or inactive position (POSITION_INACTIVE)"
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Router       /api/v1/users/invitations [post]
func (h *InvitationHandler) Invite(c *gin.Context) {
	var req invtypes.InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	inv, err := h.inviteUC.Execute(c.Request.Context(), actor, req)
	if err != nil {
		writeInvitationError(c, err)
		return
	}
	response.Created(c, invtypes.ToInvitationResponse(inv))
}

// Resend reissues the token of a pending invitation (admin only).
//
// @Summary      Resend invitation
// @Tags         invitations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Invitation ID"
// @Success      200  {object}  response.Envelope{data=invtypes.InvitationResponse}
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/users/invitations/{id}/resend [post]
func (h *InvitationHandler) Resend(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid invitation id")
		return
	}
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	inv, err := h.resendUC.Execute(c.Request.Context(), actor, id)
	if err != nil {
		writeInvitationError(c, err)
		return
	}
	response.Success(c, invtypes.ToInvitationResponse(inv))
}

// Revoke cancels a pending invitation (admin only).
//
// @Summary      Revoke invitation
// @Tags         invitations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Invitation ID"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Router       /api/v1/users/invitations/{id} [delete]
func (h *InvitationHandler) Revoke(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid invitation id")
		return
	}
	tenantID, actorID, ok := tenantAndUser(c)
	if !ok {
		return
	}
	if err := h.revokeUC.Execute(c.Request.Context(), tenantID, actorID, id); err != nil {
		writeInvitationError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "invitation revoked"})
}

// List returns the caller's tenant invitations (admin only), filtered,
// searched and paginated.
//
// @Summary      List invitations
// @Tags         invitations
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int     false  "Page number (default 1)"
// @Param        page_size  query  int     false  "Page size (default 20, max 100)"
// @Param        status     query  string  false  "Filter by status"  Enums(pending, accepted, expired, revoked)
// @Param        search     query  string  false  "Case-insensitive substring match on invitee email (max 100 chars)"
// @Success      200  {object}  response.Envelope{data=[]invtypes.InvitationListItemResponse,meta=response.Meta}
// @Failure      400  {object}  response.Envelope
// @Router       /api/v1/users/invitations [get]
func (h *InvitationHandler) List(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}
	q := invtypes.ListInvitationsQuery{Page: page, PageSize: pageSize}
	q.Status = entity.InvitationStatus(strings.ToLower(strings.TrimSpace(c.Query("status"))))
	q.Search = strings.TrimSpace(c.Query("search"))
	if utf8.RuneCountInString(q.Search) > invtypes.MaxSearchLen {
		response.BadRequest(c, "search is too long")
		return
	}
	res, err := h.listUC.Execute(c.Request.Context(), tenantID, q)
	if err != nil {
		writeInvitationError(c, err)
		return
	}
	out := make([]invtypes.InvitationListItemResponse, 0, len(res.Items))
	for _, i := range res.Items {
		out = append(out, invtypes.ToInvitationListItemResponse(i))
	}
	response.Paginated(c, out, pageMeta(res.Page, res.PageSize, res.Total))
}

// Accept consumes an invitation token and sets the invitee's password (unauthenticated).
//
// @Summary      Accept invitation
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Param        request  body      invtypes.AcceptInvitationRequest  true  "Token and new password"
// @Success      200      {object}  response.Envelope
// @Failure      400      {object}  response.Envelope  "INVALID_TOKEN or validation error"
// @Failure      403      {object}  response.Envelope  "INVITATION_REVOKED"
// @Failure      409      {object}  response.Envelope  "INVITATION_ACCEPTED"
// @Failure      410      {object}  response.Envelope  "INVITATION_EXPIRED"
// @Router       /api/v1/invitations/accept [post]
func (h *InvitationHandler) Accept(c *gin.Context) {
	var req invtypes.AcceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	if err := h.acceptUC.Execute(c.Request.Context(), req); err != nil {
		writeInvitationError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "invitation accepted"})
}

// Validate checks an invitation token and returns the invitee email and role (unauthenticated).
//
// @Summary      Validate invitation token
// @Tags         invitations
// @Produce      json
// @Param        token  query     string  true  "Invitation token"
// @Success      200    {object}  response.Envelope{data=invtypes.ValidateInvitationResponse}
// @Failure      400    {object}  response.Envelope  "INVALID_TOKEN"
// @Failure      403    {object}  response.Envelope  "INVITATION_REVOKED"
// @Failure      409    {object}  response.Envelope  "INVITATION_ACCEPTED"
// @Failure      410    {object}  response.Envelope  "INVITATION_EXPIRED"
// @Router       /api/v1/invitations/validate [get]
func (h *InvitationHandler) Validate(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_TOKEN", "invalid invitation token")
		return
	}
	res, err := h.validateUC.Execute(c.Request.Context(), token)
	if err != nil {
		writeInvitationError(c, err)
		return
	}
	response.Success(c, res)
}
