// Package handlers holds HTTP request handlers.
package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	healthtypes "github.com/skryfon-org/employee-360/backend/internal/types/health"
	usecaseinterface "github.com/skryfon-org/employee-360/backend/internal/usecase/interface"
)

// HealthHandler reports application and database health status.
type HealthHandler struct {
	healthUseCase usecaseinterface.HealthUseCase
}

// NewHealthHandler constructs a HealthHandler with the provided usecase.
func NewHealthHandler(healthUseCase usecaseinterface.HealthUseCase) *HealthHandler {
	return &HealthHandler{healthUseCase: healthUseCase}
}

// HealthResponse aliases the health response DTO; the alias keeps the swagger definition name stable.
type HealthResponse = healthtypes.HealthResponse

// Health handles health check requests and returns system status.
//
// @Summary      Report application and database health
// @Description  Returns application, database and (optional) Redis connectivity status. Redis reports ok, unreachable or disabled and never changes the HTTP status. Used by load balancers, Kubernetes probes, monitoring, and client SDK smoke tests.
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.Envelope{data=handlers.HealthResponse}
// @Router       /health [get]
// @Router       /healthz [get]
// @Router       /api/v1/health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	result := h.healthUseCase.Execute(c.Request.Context())

	response.Success(c, HealthResponse{
		Status:   "ok",
		App:      result.App,
		Database: result.Database,
		Redis:    result.Redis,
	})
}
