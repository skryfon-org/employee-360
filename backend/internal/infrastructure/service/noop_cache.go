package service

import (
	"context"
	"time"

	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

// NoopCache is the disabled-mode Cache: every operation returns
// domainservice.ErrCacheUnavailable so consumers apply their own fallback policy.
type NoopCache struct{}

var _ domainservice.Cache = NoopCache{}

// NewNoopCache returns the disabled-mode cache.
func NewNoopCache() NoopCache { return NoopCache{} }

func (NoopCache) Get(context.Context, string) (string, error) {
	return "", domainservice.ErrCacheUnavailable
}

func (NoopCache) Set(context.Context, string, string, time.Duration) error {
	return domainservice.ErrCacheUnavailable
}

func (NoopCache) Delete(context.Context, string) error {
	return domainservice.ErrCacheUnavailable
}

func (NoopCache) Incr(context.Context, string, time.Duration) (int64, error) {
	return 0, domainservice.ErrCacheUnavailable
}

func (NoopCache) Ping(context.Context) error {
	return domainservice.ErrCacheUnavailable
}
