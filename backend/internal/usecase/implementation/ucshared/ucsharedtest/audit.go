// Package ucsharedtest holds test doubles shared by usecase unit tests.
package ucsharedtest

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// RecordingAuditRecorder is a fake domain/service.AuditRecorder that captures
// entries as entity.AuditLog values (Metadata is the JSON-encoded meta).
type RecordingAuditRecorder struct {
	Logs []*entity.AuditLog
	// ErrFn, when set, is consulted on every call; a non-nil error is returned
	// and nothing is recorded.
	ErrFn func() error
	// OnRecord, when set, receives each accepted entry (e.g. to append it to a
	// transactional in-memory store). Logs is still populated.
	OnRecord func(*entity.AuditLog)
}

// Record implements domainservice.AuditRecorder.
func (r *RecordingAuditRecorder) Record(_ context.Context, tenantID, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, meta map[string]any) error {
	if r.ErrFn != nil {
		if err := r.ErrFn(); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	l := &entity.AuditLog{
		ID: uuid.New(), TenantID: tenantID, ActorUserID: &actorID,
		Action: action, EntityType: entityType, EntityID: entityID,
		Metadata: string(raw), CreatedAt: now, UpdatedAt: now,
	}
	r.Logs = append(r.Logs, l)
	if r.OnRecord != nil {
		r.OnRecord(l)
	}
	return nil
}
