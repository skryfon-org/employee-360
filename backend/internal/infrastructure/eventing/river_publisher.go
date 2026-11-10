package eventing

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// RiverPublisher implements service.EventPublisher by enqueuing River jobs in
// the caller's in-flight transaction (transactional outbox: river_job is the
// outbox table).
type RiverPublisher struct {
	client     *river.Client[*sql.Tx]
	dispatcher *Dispatcher
}

var _ service.EventPublisher = (*RiverPublisher)(nil)

// NewRiverPublisher constructs a RiverPublisher.
func NewRiverPublisher(client *river.Client[*sql.Tx], dispatcher *Dispatcher) *RiverPublisher {
	if dispatcher == nil {
		dispatcher = NewDispatcher()
	}
	return &RiverPublisher{client: client, dispatcher: dispatcher}
}

// Publish enqueues one job per event using the *sql.Tx carried by ctx. It
// fails if ctx is not inside a transaction, because publishing outside one
// would break the atomicity guarantee.
func (p *RiverPublisher) Publish(ctx context.Context, events ...event.Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := database.SQLTxFromContext(ctx)
	if err != nil {
		return fmt.Errorf("eventing: publish requires an active transaction: %w", err)
	}
	for _, evt := range events {
		args, err := p.dispatcher.Dispatch(evt)
		if err != nil {
			return err
		}
		if _, err := p.client.InsertTx(ctx, tx, args, nil); err != nil {
			return fmt.Errorf("eventing: enqueue %s: %w", evt.EventType, err)
		}
	}
	return nil
}
