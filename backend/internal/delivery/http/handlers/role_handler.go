package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	roletypes "github.com/skryfon-org/employee-360/backend/internal/types/role"
	roleusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/role"
)

// RoleHandler handles HTTP requests for role lookups. Tenant identity is read
// once via the ctx package (populated by the Auth/Tenant middleware) and passed
// to the usecase explicitly.
type RoleHandler struct {
	listUC roleusecase.ListAssignableRolesUseCase
}

// NewRoleHandler constructs a RoleHandler.
func NewRoleHandler(listUC roleusecase.ListAssignableRolesUseCase) *RoleHandler {
	return &RoleHandler{listUC: listUC}
}

// List returns the roles the caller may assign (admin only).
//
// @Summary      List assignable roles
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=[]roletypes.RoleResponse}
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Router       /api/v1/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	roles, err := h.listUC.Execute(c.Request.Context(), actor)
	if err != nil {
		switch {
		case errors.Is(err, domainerrors.ErrUnauthorized):
			response.Unauthorized(c, "unauthorized")
		case errors.Is(err, domainerrors.ErrForbidden):
			response.Forbidden(c, "insufficient permissions")
		default:
			response.Internal(c, "an unexpected error occurred")
		}
		return
	}
	out := make([]roletypes.RoleResponse, 0, len(roles))
	for _, r := range roles {
		out = append(out, roletypes.ToRoleResponse(r))
	}
	response.Success(c, out)
}
