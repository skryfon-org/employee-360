// Package usecaseimpl contains application usecase implementations.
package usecaseimpl

import (
	"context"

	healthtypes "github.com/skryfon-org/employee-360/backend/internal/types/health"

	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	usecaseinterface "github.com/skryfon-org/employee-360/backend/internal/usecase/interface"
	"github.com/skryfon-org/employee-360/backend/shared"
)

// HealthUseCaseImpl implements usecaseinterface.HealthUseCase.
type HealthUseCaseImpl struct {
	pinger      service.DatabasePinger
	cachePinger service.CachePinger // nil when Redis is disabled
}

var _ usecaseinterface.HealthUseCase = (*HealthUseCaseImpl)(nil)

// NewHealthUseCase constructs a HealthUseCaseImpl with the given database pinger
// and an optional cache pinger (nil means Redis is disabled).
func NewHealthUseCase(pinger service.DatabasePinger, cachePinger service.CachePinger) *HealthUseCaseImpl {
	return &HealthUseCaseImpl{pinger: pinger, cachePinger: cachePinger}
}

// Execute checks application and database health status.
func (u *HealthUseCaseImpl) Execute(ctx context.Context) healthtypes.HealthResult {
	database := "ok"
	if err := u.pinger.Ping(ctx); err != nil {
		database = "unreachable"
	}

	// Redis is optional: its state is reported but never changes the overall status.
	redis := "disabled"
	if u.cachePinger != nil {
		redis = "ok"
		if err := u.cachePinger.Ping(ctx); err != nil {
			redis = "unreachable"
		}
	}

	return healthtypes.HealthResult{
		App:      shared.AppName,
		Database: database,
		Redis:    redis,
	}
}
