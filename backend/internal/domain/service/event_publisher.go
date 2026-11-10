package service

import (
	"context"

	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
)

// EventPublisher defines the seam for publishing domain events within transactions.
// Usecases publish events instead of executing external side-effects directly.
type EventPublisher interface {
	Publish(ctx context.Context, events ...event.Event) error
}
