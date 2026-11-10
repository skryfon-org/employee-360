package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/skryfon-org/employee-360/backend/config"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/container"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/pkg/logger"
)

// Command worker runs the River background worker that drains the email outbox.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.LogLevel)

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

	log.Info().Msg("employee360 worker: connected to database")

	wc, err := container.NewWorkerContainer(cfg, db)
	if err != nil {
		log.Error().Err(err).Msg("failed to wire worker dependencies")
		os.Exit(1)
	}

	// Signal context controls process lifetime; River's Start context is kept
	// separate so in-flight jobs can finish during graceful Stop.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := wc.RiverClient.Start(context.Background()); err != nil {
		log.Error().Err(err).Msg("employee360 worker: failed to start river client")
		os.Exit(1)
	}
	log.Info().Msg("employee360 worker: started")

	<-sigCtx.Done()
	log.Info().Msg("employee360 worker: shutdown signal received")
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := wc.RiverClient.Stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("employee360 worker: graceful shutdown failed")
		os.Exit(1)
	}
	log.Info().Msg("employee360 worker: shutdown complete")
}
