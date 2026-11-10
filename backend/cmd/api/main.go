package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/skryfon-org/employee-360/backend/config"
	deliveryhttp "github.com/skryfon-org/employee-360/backend/internal/delivery/http"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/server"
	infraservice "github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	"github.com/skryfon-org/employee-360/backend/pkg/logger"
)

// @title       Employee360 API
// @version     1.0
// @description REST API for Employee360, an open-source, self-hostable, multi-tenant employee platform. Includes unversioned operational endpoints (health checks) and the versioned client-facing API under /api/v1.
// @BasePath    /
// @schemes     http https
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.LogLevel)

	// Connect to database (fail fast if unreachable).
	db, err := database.Connect(cfg.Database)
	if err != nil {
		database.FailFast(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get underlying database connection: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	log.Info().Msg("employee360 api: connected to database")

	// Connect to Redis only when enabled (fail fast if unreachable); otherwise
	// the container falls back to a no-op cache and health reports "disabled".
	var redisCache *infraservice.RedisCache
	var appCache domainservice.Cache
	if cfg.Redis.Enabled {
		redisClient, err := infraservice.NewRedisClient(cfg.Redis)
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis unreachable: %v\n", err)
			os.Exit(1)
		}
		redisCache = infraservice.NewRedisCache(redisClient, log)
		appCache = redisCache
		log.Info().Msg("employee360 api: connected to redis")
	} else {
		log.Info().Msg("employee360 api: redis disabled (REDIS_ENABLED=false)")
	}

	// Wire dependencies.
	c, err := container.New(cfg, db, appCache, log)
	if err != nil {
		log.Error().Err(err).Msg("failed to wire dependencies")
		os.Exit(1)
	}

	srv := server.New(cfg.Server, deliveryhttp.SetupRouter(cfg, log, c))

	// Run the server in the background so the main goroutine is free to
	// wait for either a shutdown signal or a server startup failure.
	serverErrCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr()).Msg("employee360 api: starting server")
		serverErrCh <- srv.Start()
	}()

	// Wait for SIGINT/SIGTERM to trigger a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrCh:
		if err != nil {
			log.Error().Err(err).Msg("employee360 api: server failed")
			os.Exit(1)
		}
	case <-ctx.Done():
		log.Info().Msg("employee360 api: shutdown signal received")
		stop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("employee360 api: graceful shutdown failed")
			os.Exit(1)
		}

		if redisCache != nil {
			if err := redisCache.Close(); err != nil {
				log.Warn().Err(err).Msg("employee360 api: closing redis failed")
			}
		}

		log.Info().Msg("employee360 api: shutdown complete")
	}
}
