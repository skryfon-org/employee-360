package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestCancel(t *testing.T) {
	uc := func(e *env) *cancelLeave {
		return NewCancelLeaveUseCase(e.requests, e.audit, nil, clock).(*cancelLeave)
	}

	t.Run("pending", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		d, err := uc(e).Execute(bgctx, tenant, e.employee, id)
		require.NoError(t, err)
		assert.Equal(t, entity.LeaveStatusCancelled, d.Status)
		assert.Nil(t, d.DecidedBy, "a never-decided request stays undecided")
		assert.Equal(t, []string{"leave_request.apply", "leave_request.cancel"}, e.audit.actions)
		assert.Equal(t, "pending", e.audit.metas[1]["previous_status"])
	})
	t.Run("cancelling an approved request keeps the decision trail", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		_, err := NewApproveLeaveUseCase(e.requests, e.audit, nil, clock, e.events, testURLs).Execute(bgctx, tenant, e.mgr, false, id, "enjoy")
		require.NoError(t, err)
		decidedAt := *e.requests.rows[id].DecidedAt

		d, err := uc(e).Execute(bgctx, tenant, e.employee, id)
		require.NoError(t, err)
		assert.Equal(t, entity.LeaveStatusCancelled, d.Status)
		require.NotNil(t, d.DecidedBy)
		assert.Equal(t, e.mgr, *d.DecidedBy, "decided_by stays the approver")
		assert.Equal(t, decidedAt, *d.DecidedAt)
		assert.Equal(t, "enjoy", d.DecisionComment)
		last := len(e.audit.actions) - 1
		assert.Equal(t, "leave_request.cancel", e.audit.actions[last])
		assert.Equal(t, "approved", e.audit.metas[last]["previous_status"])
	})
	t.Run("approved not yet started", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		e.requests.rows[id].Status = entity.LeaveStatusApproved
		_, err := uc(e).Execute(bgctx, tenant, e.employee, id)
		require.NoError(t, err)
	})
	t.Run("approved starting today is still cancellable", func(t *testing.T) {
		e := newEnv()
		id := e.applyOK(t, e.in("2026-10-29", "2026-10-30")).ID
		e.requests.rows[id].Status = entity.LeaveStatusApproved
		_, err := uc(e).Execute(bgctx, tenant, e.employee, id)
		require.NoError(t, err)
	})
	t.Run("approved already started", func(t *testing.T) {
		e := newEnv()
		id := e.applyOK(t, e.in("2026-10-29", "2026-10-31")).ID
		// Apply forbids past starts, so simulate time passing by moving the start back.
		e.requests.rows[id].StartDate = day("2026-10-27")
		e.requests.rows[id].Status = entity.LeaveStatusApproved
		_, err := uc(e).Execute(bgctx, tenant, e.employee, id)
		assert.ErrorIs(t, err, domainerrors.ErrLeaveNotCancellable)
	})
	t.Run("rejected and cancelled are final", func(t *testing.T) {
		for _, st := range []string{entity.LeaveStatusRejected, entity.LeaveStatusCancelled} {
			e := newEnv()
			id := e.pending(t)
			e.requests.rows[id].Status = st
			_, err := uc(e).Execute(bgctx, tenant, e.employee, id)
			assert.ErrorIs(t, err, domainerrors.ErrLeaveNotCancellable, st)
		}
	})
	t.Run("someone else's request is not found", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		_, err := uc(e).Execute(bgctx, tenant, e.other, id)
		assert.ErrorIs(t, err, domainerrors.ErrLeaveRequestNotFound)
		_, err = uc(e).Execute(bgctx, tenant, e.mgr, id) // even the approver
		assert.ErrorIs(t, err, domainerrors.ErrLeaveRequestNotFound)
		_, err = uc(e).Execute(bgctx, uuid.New(), e.employee, id) // other tenant
		assert.ErrorIs(t, err, domainerrors.ErrLeaveRequestNotFound)
	})
}
