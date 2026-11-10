package service

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss is returned by Cache.Get when the key does not exist.
	ErrCacheMiss = errors.New("cache: key not found")
	// ErrCacheUnavailable is returned when no cache backend is configured
	// (Redis disabled) or the backend cannot be reached.
	ErrCacheUnavailable = errors.New("cache: unavailable")
	// ErrInvalidTTL is returned when a write is attempted without a positive TTL.
	ErrInvalidTTL = errors.New("cache: ttl must be greater than zero")
)

// Cache is the domain port for a key/value store with expiry. Every write
// requires a positive TTL so entries can never live forever by accident.
//
// Keys are raw strings for now; a typed key is deferred to the first consumer
// ticket.
//
// Keys must be built with CacheKey / GlobalCacheKey so tenant data is
// namespaced. Failure policy is decided by each consumer (for example rate
// limiting falls back to in-memory; identity caching fails closed).
type Cache interface {
	// Get returns the value for key, or ErrCacheMiss.
	Get(ctx context.Context, key string) (string, error)
	// Set stores value under key with the given ttl (ttl <= 0 returns ErrInvalidTTL).
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Delete removes key; deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
	// Incr atomically increments the counter at key and returns the new value.
	// The ttl is applied when the counter is created (first increment).
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	// Ping checks backend reachability.
	Ping(ctx context.Context) error
}

// CachePinger is the health-check port for the cache backend. It is satisfied
// by Cache implementations; a nil CachePinger means Redis is disabled.
type CachePinger interface {
	Ping(ctx context.Context) error
}
