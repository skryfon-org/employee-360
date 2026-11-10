// Package server manages HTTP server lifecycle and graceful shutdown.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/skryfon-org/employee-360/backend/config"
)

// Server wraps the standard HTTP server with lifecycle controls.
type Server struct {
	httpServer *http.Server
}

// New constructs an HTTP server with the provided configuration and handler.
func New(cfg config.ServerConfig, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Address(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}
}

// Start begins listening and serving incoming HTTP requests.
func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Addr returns the address the server is configured to listen on.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Shutdown gracefully stops the server: it stops accepting new
// connections and waits for in-flight requests to complete, up to the
// deadline carried by ctx (callers should derive ctx from
// config.ServerConfig.ShutdownTimeout).
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
