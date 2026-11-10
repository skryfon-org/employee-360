package allocation

import (
	"context"
	"testing"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteAllocation(t *testing.T) {
	e := newAllocEnv(t)
	ctx := context.Background()
	r, err := e.create().Execute(ctx, e.tenant, e.actor, e.in(e.p1, 60, "2026-01-01", nil))
	require.NoError(t, err)
	id := r.Allocation.ID

	del := NewDeleteAllocationUseCase(e.allocs, e.rec, nil)
	assert.ErrorIs(t, del.Execute(ctx, uuid.New(), e.actor, id), domainerrors.ErrAllocationNotFound, "cross-tenant")
	require.NoError(t, del.Execute(ctx, e.tenant, e.actor, id))
	assert.Equal(t, domainaudit.ActionProjectAllocationDelete, e.rec.Logs[len(e.rec.Logs)-1].Action)
	assert.ErrorIs(t, del.Execute(ctx, e.tenant, e.actor, id), domainerrors.ErrAllocationNotFound)
}
