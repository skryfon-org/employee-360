package eventing

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/job"
)

// NewInsertOnlyClient builds a River client that can only enqueue jobs (used by
// the API process; it never fetches or works jobs).
func NewInsertOnlyClient(db *sql.DB) (*river.Client[*sql.Tx], error) {
	return river.NewClient(riverdatabasesql.New(db), &river.Config{})
}

// NewWorkerClient builds a River client that works send_email jobs (used by cmd/worker).
func NewWorkerClient(db *sql.DB, emailService service.EmailService, logger *slog.Logger, maxWorkers int) (*river.Client[*sql.Tx], error) {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, job.NewEmailWorker(emailService))
	return river.NewClient(riverdatabasesql.New(db), &river.Config{
		Logger: logger,
		// Completed jobs carry reset/invite URLs in their args; keep them briefly.
		CompletedJobRetentionPeriod: time.Hour,
		Queues:                      map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: maxWorkers}},
		Workers:                     workers,
	})
}
