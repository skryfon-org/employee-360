//go:build integration

package integration

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
)

func liveRedisConfig(t *testing.T) config.RedisConfig {
	t.Helper()
	port := 6379
	if p := os.Getenv("REDIS_PORT"); p != "" {
		n, err := strconv.Atoi(p)
		require.NoError(t, err)
		port = n
	}
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	return config.RedisConfig{
		Host: host, Port: port, Password: os.Getenv("REDIS_PASSWORD"), PoolSize: 5,
		DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second,
	}
}

func liveCache(t *testing.T) *infraservice.RedisCache {
	t.Helper()
	client, err := infraservice.NewRedisClient(liveRedisConfig(t))
	require.NoError(t, err, "a live Redis is required (REDIS_HOST/REDIS_PORT/REDIS_PASSWORD)")
	c := infraservice.NewRedisCache(client, zerolog.Nop())
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestRedisIntegration_GetSetTTLIncrPing(t *testing.T) {
	c := liveCache(t)
	ctx := context.Background()
	key, err := service.CacheKey(uuid.New(), "it", "k")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Delete(ctx, key) })

	require.NoError(t, c.Ping(ctx))

	_, err = c.Get(ctx, key)
	require.ErrorIs(t, err, service.ErrCacheMiss)
	require.NoError(t, c.Set(ctx, key, "v", 500*time.Millisecond))
	v, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, "v", v)
	require.Eventually(t, func() bool {
		_, err := c.Get(ctx, key)
		return err == service.ErrCacheMiss
	}, 3*time.Second, 100*time.Millisecond, "key should expire")

	ctr, err := service.CacheKey(uuid.New(), "it", "ctr")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Delete(ctx, ctr) })
	for want := int64(1); want <= 3; want++ {
		got, err := c.Incr(ctx, ctr, time.Minute)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestRedisIntegration_NoCrossTenantReads(t *testing.T) {
	c := liveCache(t)
	ctx := context.Background()
	ka, err := service.CacheKey(uuid.New(), "it", "same")
	require.NoError(t, err)
	kb, err := service.CacheKey(uuid.New(), "it", "same")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Delete(ctx, ka); _ = c.Delete(ctx, kb) })

	require.NoError(t, c.Set(ctx, ka, "tenant-a", time.Minute))
	_, err = c.Get(ctx, kb)
	require.ErrorIs(t, err, service.ErrCacheMiss)
	require.NoError(t, c.Set(ctx, kb, "tenant-b", time.Minute))
	v, err := c.Get(ctx, ka)
	require.NoError(t, err)
	assert.Equal(t, "tenant-a", v)
}

func TestRedisIntegration_DownRedisReturnsErrorNotHang(t *testing.T) {
	cfg := config.RedisConfig{Host: "127.0.0.1", Port: 1, PoolSize: 1,
		DialTimeout: 500 * time.Millisecond, ReadTimeout: 500 * time.Millisecond, WriteTimeout: 500 * time.Millisecond}
	done := make(chan error, 1)
	go func() { _, err := infraservice.NewRedisClient(cfg); done <- err }()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("connecting to an unreachable Redis hung")
	}
}
