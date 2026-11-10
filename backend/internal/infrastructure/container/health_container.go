package container

import (
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/handlers"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	usecaseimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation"
	usecaseinterface "github.com/skryfon-org/employee-360/backend/internal/usecase/interface"
	"gorm.io/gorm"
)

// HealthContainer encapsulates dependencies, usecases, and handlers for health checks.
type HealthContainer struct {
	Handler *handlers.HealthHandler
	UseCase usecaseinterface.HealthUseCase
}

// NewHealthContainer initializes and wires the health check usecase and handler.
//
// cache may be nil (Redis disabled); health then reports redis as "disabled".
func NewHealthContainer(db *gorm.DB, cache domainservice.Cache) *HealthContainer {
	dbPinger := database.NewGormDatabasePinger(db)
	// Pass an untyped nil when disabled so the usecase sees a nil interface.
	var cachePinger domainservice.CachePinger
	if cache != nil {
		cachePinger = cache
	}
	healthUseCase := usecaseimpl.NewHealthUseCase(dbPinger, cachePinger)
	healthHandler := handlers.NewHealthHandler(healthUseCase)

	return &HealthContainer{
		Handler: healthHandler,
		UseCase: healthUseCase,
	}
}
