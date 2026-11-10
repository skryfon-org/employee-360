// Package middleware holds standard Gin HTTP middleware.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/shared"
)

// ContextKeyRequestID is the Gin context key where request ID is stored.
const ContextKeyRequestID = "request_id"

// RequestID assigns or propagates a unique X-Request-ID for every request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(shared.RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(ContextKeyRequestID, requestID)
		c.Writer.Header().Set(shared.RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID returns the request id stored on the Gin context by
// RequestID, or "" if it hasn't run (or ran in a different chain).
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(ContextKeyRequestID); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}
