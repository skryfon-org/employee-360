// Package health holds API DTOs and usecase results for the health check.
package health

// HealthResult contains the status of the application and its dependencies.
type HealthResult struct {
	App      string
	Database string
	// Redis is "ok", "unreachable", or "disabled" (Redis is optional).
	Redis string
}

// HealthResponse is the health endpoint's response body.
type HealthResponse struct {
	Status   string `json:"status"`
	App      string `json:"app"`
	Database string `json:"database"`
	// Redis is "ok", "unreachable", or "disabled"; it never changes the HTTP status.
	Redis string `json:"redis"`
}
