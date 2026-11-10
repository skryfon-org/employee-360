// Package shared holds cross-cutting constants and utility types.
package shared

const (
	// AppName is the application service name.
	AppName = "employee360"

	// APIVersionPrefix is the route prefix for the versioned REST API.
	APIVersionPrefix = "/api/v1"

	// RequestIDHeader is the HTTP header for request tracing.
	RequestIDHeader = "X-Request-ID"

	// TenantIDHeader is the HTTP header for tenant resolution.
	TenantIDHeader = "X-Tenant-ID"
)
