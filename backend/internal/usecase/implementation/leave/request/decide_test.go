package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestDecide(t *testing.T) {
	approve := func(e *env) *decideLeave {
		return NewApproveLeaveUseCase(e.requests, e.audit, nil, clock, e.events, testURLs).(*decideLeave)
	}
	reject := func(e *env) *decideLeave {
		return NewRejectLeaveUseCase(e.requests, e.audit, nil, clock, e.events, testURLs).(*decideLeave)
	}

	t.Run("chosen approver approves", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		d, err := approve(e).Execute(bgctx, tenant, e.mgr, false, id, " ok ")
		require.NoError(t, err)
		assert.Equal(t, entity.LeaveStatusApproved, d.Status)
		assert.Equal(t, e.mgr, *d.DecidedBy)
		assert.Equal(t, "ok", d.DecisionComment)
		assert.NotNil(t, d.DecidedAt)
		assert.Equal(t, []string{"leave_request.apply", "leave_request.approve"}, e.audit.actions)
	})
	t.Run("chosen approver rejects", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		d, err := reject(e).Execute(bgctx, tenant, e.mgr, false, id, "busy period")
		require.NoError(t, err)
		assert.Equal(t, entity.LeaveStatusRejected, d.Status)
	})
	t.Run("tenant admin who was not chosen may decide", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		d, err := approve(e).Execute(bgctx, tenant, e.admin, true, id, "")
		require.NoError(t, err)
		assert.Equal(t, e.admin, *d.DecidedBy)
	})
	t.Run("applicant can never decide, even as admin", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		_, err := approve(e).Execute(bgctx, tenant, e.employee, true, id, "")
		assert.ErrorIs(t, err, domainerrors.ErrLeaveSelfDecision)
		_, err = reject(e).Execute(bgctx, tenant, e.employee, false, id, "")
		assert.ErrorIs(t, err, domainerrors.ErrLeaveSelfDecision)
		assert.Equal(t, entity.LeaveStatusPending, e.requests.rows[id].Status)
	})
	t.Run("non-approver employee gets not found", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		_, err := approve(e).Execute(bgctx, tenant, e.other, false, id, "")
		assert.ErrorIs(t, err, domainerrors.ErrLeaveRequestNotFound)
		assert.Equal(t, entity.LeaveStatusPending, e.requests.rows[id].Status)
	})
	t.Run("cross-tenant is not found even for an admin", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		_, err := approve(e).Execute(bgctx, uuid.New(), e.admin, true, id, "")
		assert.ErrorIs(t, err, domainerrors.ErrLeaveRequestNotFound)
	})
	t.Run("only from pending", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		_, err := approve(e).Execute(bgctx, tenant, e.mgr, false, id, "")
		require.NoError(t, err)
		_, err = reject(e).Execute(bgctx, tenant, e.mgr, false, id, "")
		assert.ErrorIs(t, err, domainerrors.ErrLeaveNotPending)
		_, err = approve(e).Execute(bgctx, tenant, e.mgr, false, id, "")
		assert.ErrorIs(t, err, domainerrors.ErrLeaveNotPending)
	})
	t.Run("comment length", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		long := make([]byte, 501)
		for i := range long {
			long[i] = 'a'
		}
		_, err := reject(e).Execute(bgctx, tenant, e.mgr, false, id, string(long))
		assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)
	})
}
