// Package service defines domain service interfaces (ports).
package service

import "context"

// DatabasePinger is a domain service port for checking database reachability.
type DatabasePinger interface {
	Ping(ctx context.Context) error
}
