package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
)

// tenantOnly reads the authenticated tenant ID from the request context
// (populated by the Auth/Tenant middleware via the ctx package). On a missing
// or invalid value it writes a 401 and returns ok=false; the caller must stop.
func tenantOnly(c *gin.Context) (tenantID uuid.UUID, ok bool) {
	tid, ok := parseIdentity(c.Request.Context(), ctx.TenantIDFromContext)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return uuid.Nil, false
	}
	return tid, true
}

// tenantAndUser reads the authenticated tenant and user IDs from the request
// context. On a missing or invalid value it writes a 401 and returns ok=false.
func tenantAndUser(c *gin.Context) (tenantID, userID uuid.UUID, ok bool) {
	rc := c.Request.Context()
	tid, tok := parseIdentity(rc, ctx.TenantIDFromContext)
	uid, uok := parseIdentity(rc, ctx.UserIDFromContext)
	if !tok || !uok {
		response.Unauthorized(c, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	return tid, uid, true
}

// parseIdentity reads a UUID string via read and rejects missing, malformed or nil values.
func parseIdentity(rc context.Context, read func(context.Context) (string, bool)) (uuid.UUID, bool) {
	raw, ok := read(rc)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}
