//go:build integration

package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/eventing"
)

// recordingEmailService records every message it is asked to send.
type recordingEmailService struct {
	mu   sync.Mutex
	msgs []service.EmailMessage
}

func (r *recordingEmailService) Send(_ context.Context, msg service.EmailMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, msg)
	return nil
}

func (r *recordingEmailService) to(addr string) []service.EmailMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []service.EmailMessage
	for _, m := range r.msgs {
		if m.To == addr {
			out = append(out, m)
		}
	}
	return out
}

// TestWorker_DeliversAfterCommitOnly proves a job published through
// RiverPublisher inside GormTransactor.WithinTransaction is picked up by the
// worker client and delivered via EmailService.Send - only after the commit.
func TestWorker_DeliversAfterCommitOnly(t *testing.T) {
	db, pub, tx := setup(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// The worker drains the shared default queue; refuse to run if unrelated
	// pending send_email jobs exist, otherwise this test would swallow them.
	var pending int64
	require.NoError(t, db.Raw("SELECT count(*) FROM river_job WHERE kind = 'send_email' AND state IN ('available','retryable','scheduled')").Scan(&pending).Error)
	if pending > 0 {
		t.Skipf("skipping: %d unrelated pending send_email jobs in river_job would be consumed by the test worker", pending)
	}

	to := fmt.Sprintf("worker-%s@example.test", uuid.NewString())
	tenant := &entity.Tenant{ID: uuid.New(), Name: "worker-it-" + uuid.NewString(), IsActive: true}
	t.Cleanup(func() {
		db.Exec("DELETE FROM river_job WHERE args->>'to' = ?", to)
		db.Unscoped().Where("id = ?", tenant.ID).Delete(&entity.Tenant{})
	})

	rec := &recordingEmailService{}
	worker, err := eventing.NewWorkerClient(sqlDB, rec, nil, 2)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, worker.Start(ctx))
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = worker.Stop(stopCtx)
	})

	evt := event.Event{
		ID: uuid.New(), TenantID: tenant.ID, EventType: event.EventTypePasswordResetRequested,
		Payload: event.PasswordResetRequestedPayload{
			UserID: uuid.New(), TenantID: tenant.ID, Email: to, UserName: "Worker Test",
			ResetURL: "http://frontend.test/reset-password?token=abc", ExpiresAt: time.Now().Add(time.Hour),
		},
		OccurredAt: time.Now(),
	}

	err = tx.WithinTransaction(context.Background(), func(txCtx context.Context) error {
		require.NoError(t, database.DBFromContext(txCtx, db).Create(tenant).Error)
		require.NoError(t, pub.Publish(txCtx, evt))
		// Well beyond River's 1s fetch poll interval: an uncommitted job must
		// never be delivered.
		time.Sleep(3 * time.Second)
		require.Empty(t, rec.to(to), "email must not be delivered before commit")
		return nil
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool { return len(rec.to(to)) == 1 }, 15*time.Second, 200*time.Millisecond,
		"worker must deliver the email after commit")

	got := rec.to(to)[0]
	require.Equal(t, service.EmailTemplatePasswordReset, got.TemplateName)
	require.Equal(t, "http://frontend.test/reset-password?token=abc", got.TemplateData["reset_url"])
	require.NotContains(t, got.TemplateData, "plain_token")
}
