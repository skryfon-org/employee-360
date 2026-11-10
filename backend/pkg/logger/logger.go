// Package logger provides structured zerolog logger construction.
package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// New constructs a structured JSON zerolog.Logger for the specified log level.
func New(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(strings.TrimSpace(level)))
	if err != nil || level == "" {
		parsedLevel = zerolog.InfoLevel
	}

	return zerolog.New(os.Stdout).
		Level(parsedLevel).
		With().
		Timestamp().
		Logger()
}

// Nop returns a no-op logger that discards all log output.
func Nop() zerolog.Logger {
	return zerolog.Nop()
}
