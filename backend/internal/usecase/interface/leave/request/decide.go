package request

import (
	"context"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// DecideLeaveUseCase approves or rejects a pending request. isAdmin is the
// caller's tenant-admin status (resolved from the auth context by the handler).
type DecideLeaveUseCase interface {
	Execute(ctx context.Context, tenantID, actorID uuid.UUID, isAdmin bool, id uuid.UUID, comment string) (*entity.LeaveRequestDetail, error)
}
