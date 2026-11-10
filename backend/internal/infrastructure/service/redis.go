package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/skryfon-org/employee-360/backend/config"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

// incrScript atomically increments a counter and sets its expiry on creation.
var incrScript = redis.NewScript(`
local v = redis.call('INCR', KEYS[1])
if v == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return v
`)

// NewRedisClient builds a go-redis client from cfg (URL, if set, overrides the
// individual fields) and verifies connectivity with a ping. The returned error
// never includes the password.
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	var opts *redis.Options
	if cfg.URL != "" {
		parsed, err := redis.ParseURL(cfg.URL)
		if err != nil {
			// Do not wrap err: it may echo the URL (and its credentials).
			return nil, errors.New("cache: invalid redis url")
		}
		opts = parsed
	} else {
		opts = &redis.Options{
			Addr:     cfg.Addr(),
			Password: cfg.Password,
			DB:       cfg.DB,
		}
		if cfg.TLS {
			opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: cfg.Host}
		}
	}
	opts.DialTimeout = cfg.DialTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout
	opts.PoolSize = cfg.PoolSize
	// Fail fast rather than retrying with backoff; callers decide the policy.
	opts.MaxRetries = 1

	client := redis.NewClient(opts)

	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("cache: redis ping failed: %w", err)
	}
	return client, nil
}

// RedisCache implements domainservice.Cache and domainservice.CachePinger on go-redis.
type RedisCache struct {
	client *redis.Client
	log    zerolog.Logger
}

var (
	_ domainservice.Cache       = (*RedisCache)(nil)
	_ domainservice.CachePinger = (*RedisCache)(nil)
)

// NewRedisCache wraps an already-connected client.
func NewRedisCache(client *redis.Client, log zerolog.Logger) *RedisCache {
	return &RedisCache{client: client, log: log}
}

func (c *RedisCache) fail(op string, err error) error {
	c.log.Warn().Err(err).Str("op", op).Msg("redis cache operation failed")
	return wrapCacheErr(op, err)
}

// wrapCacheErr wraps the cause with ErrCacheUnavailable while preserving the
// cause chain, so errors.Is works for both (e.g. context.Canceled).
func wrapCacheErr(op string, err error) error {
	return fmt.Errorf("%w: %s: %w", domainservice.ErrCacheUnavailable, op, err)
}

// pingTimeout bounds a single health Ping so /health stays responsive when
// Redis is down.
const pingTimeout = 2 * time.Second

// Get returns the value at key or domainservice.ErrCacheMiss.
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	v, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", domainservice.ErrCacheMiss
	}
	if err != nil {
		return "", c.fail("get", err)
	}
	return v, nil
}

// Set stores value with a mandatory positive ttl.
func (c *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl <= 0 {
		return domainservice.ErrInvalidTTL
	}
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return c.fail("set", err)
	}
	return nil
}

// Delete removes key.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return c.fail("delete", err)
	}
	return nil
}

// Incr atomically increments the counter, applying ttl when it is created.
func (c *RedisCache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if ttl <= 0 {
		return 0, domainservice.ErrInvalidTTL
	}
	ms := ttl.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	n, err := incrScript.Run(ctx, c.client, []string{key}, ms).Int64()
	if err != nil {
		return 0, c.fail("incr", err)
	}
	return n, nil
}

// Ping checks Redis reachability.
func (c *RedisCache) Ping(ctx context.Context) error {
	timeout := pingTimeout
	if rt := c.client.Options().ReadTimeout; rt > 0 && rt < timeout {
		timeout = rt
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := c.client.Ping(ctx).Err(); err != nil {
		// Debug: health probes hit this repeatedly while Redis is down.
		c.log.Debug().Err(err).Str("op", "ping").Msg("redis ping failed")
		return wrapCacheErr("ping", err)
	}
	return nil
}

// Close releases the connection pool; call during shutdown.
func (c *RedisCache) Close() error {
	return c.client.Close()
}
