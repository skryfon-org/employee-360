package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
)

// Recovery returns panic-recovery middleware with structured error responses.
func Recovery(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Str("request_id", GetRequestID(c)).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Interface("panic", r).
					Bytes("stack", debug.Stack()).
					Msg("panic recovered")

				if !c.Writer.Written() {
					response.Internal(c, "an unexpected error occurred")
				}
				c.Abort()
			}
		}()

		c.Next()
	}
}
