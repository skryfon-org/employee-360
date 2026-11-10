package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/ctx"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	empuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/employee"
)

// ErrorCodeProfileIncomplete is the machine-readable error code returned (with
// HTTP 403) while an employee has not completed their mandatory profile.
const ErrorCodeProfileIncomplete = "profile_incomplete"

// RequireCompleteProfile gates employee endpoints until the caller's profile is
// complete. Must run after Auth and Tenant. admin and super_admin are never
// gated. Apply it only to routes other than /me, /me/profile, the emergency
// contacts and logout, which stay reachable so the profile can be completed.
// A lookup failure fails closed with 503.
func RequireCompleteProfile(gate empuc.ProfileGateUseCase) gin.HandlerFunc {
	if gate == nil {
		panic("middleware.RequireCompleteProfile: gate is required")
	}
	return func(c *gin.Context) {
		if HasAnyRole(getRoles(c), entity.RoleAdmin, entity.RoleSuperAdmin) {
			c.Next()
			return
		}
		userID, err := getUserID(c)
		if err != nil {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}
		tenantRaw, _ := ctx.TenantIDFromContext(c.Request.Context())
		tenantID, err := uuid.Parse(tenantRaw)
		if err != nil || tenantID == uuid.Nil {
			response.Unauthorized(c, "tenant context required")
			c.Abort()
			return
		}
		complete, err := gate.Execute(c.Request.Context(), tenantID, userID)
		if err != nil {
			response.ServiceUnavailable(c, "unable to verify profile status")
			c.Abort()
			return
		}
		if !complete {
			response.Error(c, 403, ErrorCodeProfileIncomplete, "complete your profile to continue")
			c.Abort()
			return
		}
		c.Next()
	}
}
