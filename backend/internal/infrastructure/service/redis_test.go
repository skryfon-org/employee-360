package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/config"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

func newTestCache(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client, err := NewRedisClient(config.RedisConfig{
		Host: mr.Host(), Port: mr.Server().Addr().Port, PoolSize: 2,
		DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second,
	})
	require.NoError(t, err)
	c := NewRedisCache(client, zerolog.Nop())
	t.Cleanup(func() { _ = c.Close() })
	return c, mr
}

func TestRedisCache_SetGetDelete(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	_, err := c.Get(ctx, "k")
	require.ErrorIs(t, err, domainservice.ErrCacheMiss)

	require.NoError(t, c.Set(ctx, "k", "v", time.Minute))
	v, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", v)

	require.NoError(t, c.Delete(ctx, "k"))
	require.NoError(t, c.Delete(ctx, "k"), "deleting a missing key is not an error")
	_, err = c.Get(ctx, "k")
	require.ErrorIs(t, err, domainservice.ErrCacheMiss)
}

func TestRedisCache_SetRejectsNonPositiveTTL(t *testing.T) {
	c, _ := newTestCache(t)
	require.ErrorIs(t, c.Set(context.Background(), "k", "v", 0), domainservice.ErrInvalidTTL)
	require.ErrorIs(t, c.Set(context.Background(), "k", "v", -time.Second), domainservice.ErrInvalidTTL)
	_, err := c.Incr(context.Background(), "k", 0)
	require.ErrorIs(t, err, domainservice.ErrInvalidTTL)
}

func TestRedisCache_TTLExpiry(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", "v", 10*time.Second))
	mr.FastForward(11 * time.Second)
	_, err := c.Get(ctx, "k")
	require.ErrorIs(t, err, domainservice.ErrCacheMiss)
}

func TestRedisCache_IncrSetsTTLOnceAndCounts(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	for want := int64(1); want <= 3; want++ {
		got, err := c.Incr(ctx, "ctr", 10*time.Second)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	assert.Equal(t, 10*time.Second, mr.TTL("ctr"))

	mr.FastForward(11 * time.Second)
	got, err := c.Incr(ctx, "ctr", 10*time.Second)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got, "counter restarts after expiry")
}

func TestRedisCache_TenantKeysIsolated(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	ka, err := domainservice.CacheKey(uuid.New(), "feat", "k")
	require.NoError(t, err)
	kb, err := domainservice.CacheKey(uuid.New(), "feat", "k")
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, ka, "a", time.Minute))
	_, err = c.Get(ctx, kb)
	require.ErrorIs(t, err, domainservice.ErrCacheMiss)
}

func TestRedisCache_PingAndOutage(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()
	require.NoError(t, c.Ping(ctx))

	mr.Close()
	require.ErrorIs(t, c.Ping(ctx), domainservice.ErrCacheUnavailable)
	_, err := c.Get(ctx, "k")
	require.ErrorIs(t, err, domainservice.ErrCacheUnavailable)
	require.ErrorIs(t, c.Set(ctx, "k", "v", time.Minute), domainservice.ErrCacheUnavailable)
	_, err = c.Incr(ctx, "k", time.Minute)
	require.ErrorIs(t, err, domainservice.ErrCacheUnavailable)
}

func TestNewRedisClient_Unreachable(t *testing.T) {
	start := time.Now()
	_, err := NewRedisClient(config.RedisConfig{
		Host: "127.0.0.1", Port: 1, PoolSize: 1,
		DialTimeout: 500 * time.Millisecond, ReadTimeout: time.Second, WriteTimeout: time.Second,
	})
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestNewRedisClient_URLAndPasswordNotLeaked(t *testing.T) {
	_, err := NewRedisClient(config.RedisConfig{URL: "http://:topsecret@host", PoolSize: 1})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "topsecret")
}

func TestNewRedisClient_URL(t *testing.T) {
	mr := miniredis.RunT(t)
	client, err := NewRedisClient(config.RedisConfig{URL: "redis://" + mr.Addr() + "/0", PoolSize: 2, DialTimeout: time.Second})
	require.NoError(t, err)
	require.NoError(t, client.Close())
}

func TestNewRedisClient_Password(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("pw")
	cfg := config.RedisConfig{Host: mr.Host(), Port: mr.Server().Addr().Port, PoolSize: 1, DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second}
	_, err := NewRedisClient(cfg)
	require.Error(t, err)
	cfg.Password = "pw"
	client, err := NewRedisClient(cfg)
	require.NoError(t, err)
	_ = client.Close()
}

func TestNoopCache(t *testing.T) {
	n := NewNoopCache()
	ctx := context.Background()
	_, err := n.Get(ctx, "k")
	require.ErrorIs(t, err, domainservice.ErrCacheUnavailable)
	require.ErrorIs(t, n.Set(ctx, "k", "v", time.Second), domainservice.ErrCacheUnavailable)
	require.ErrorIs(t, n.Delete(ctx, "k"), domainservice.ErrCacheUnavailable)
	_, err = n.Incr(ctx, "k", time.Second)
	require.ErrorIs(t, err, domainservice.ErrCacheUnavailable)
	require.ErrorIs(t, n.Ping(ctx), domainservice.ErrCacheUnavailable)
}

func TestRedisCache_ErrorChainPreservesCause(t *testing.T) {
	c, _ := newTestCache(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Get(ctx, "k")
	require.Error(t, err)
	assert.ErrorIs(t, err, domainservice.ErrCacheUnavailable)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRedisCache_PingBoundedWhenDown(t *testing.T) {
	c, mr := newTestCache(t)
	mr.Close()
	start := time.Now()
	err := c.Ping(context.Background())
	require.ErrorIs(t, err, domainservice.ErrCacheUnavailable)
	assert.Less(t, time.Since(start), 3*time.Second)
}
