// Package response provides the standardized JSON envelope and error
// helpers every handler must use instead of calling c.JSON directly.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the standard JSON response shape returned by every endpoint.
// Exactly one of Data or Error is populated depending on Success.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo describes a failed request in a client-agnostic, machine
// readable way.
type ErrorInfo struct {
	// Code is a stable machine-readable error code. 409 responses use CONFLICT
	// (e.g. duplicate leave-type code) or a domain-specific code.
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Meta carries auxiliary response metadata, e.g. pagination.
type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// Success writes a 200 OK envelope carrying data.
func Success(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, data)
}

// Created writes a 201 Created envelope carrying the created resource.
func Created(c *gin.Context, data interface{}) {
	JSON(c, http.StatusCreated, data)
}

// JSON writes a successful envelope with the given HTTP status code.
func JSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, Envelope{
		Success: true,
		Data:    data,
	})
}

// Paginated writes a 200 OK envelope carrying a page of data plus
// pagination metadata.
func Paginated(c *gin.Context, data interface{}, meta Meta) {
	c.JSON(http.StatusOK, Envelope{
		Success: true,
		Data:    data,
		Meta:    &meta,
	})
}

// Error writes a failed envelope with the given HTTP status code, a
// stable machine-readable error code, and a human-readable message.
func Error(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, Envelope{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails is like Error but also attaches structured details
// (e.g. field-level validation errors).
func ErrorWithDetails(c *gin.Context, statusCode int, code, message string, details interface{}) {
	c.JSON(statusCode, Envelope{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest writes a 400 error envelope.
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// NotFound writes a 404 error envelope.
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

// Unauthorized writes a 401 error envelope.
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden writes a 403 error envelope.
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

// Internal writes a 500 error envelope.
func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// ServiceUnavailable writes a 503 error envelope, used e.g. by the health
// endpoint when a dependency (such as the database) is unreachable.
func ServiceUnavailable(c *gin.Context, message string) {
	Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message)
}
