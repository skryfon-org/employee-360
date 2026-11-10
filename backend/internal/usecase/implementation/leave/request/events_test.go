package request

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/event"
)

func TestApply_PublishesLeaveRequestedOnce(t *testing.T) {
	e := newEnv()
	in := e.in("2026-11-10", "2026-11-11")
	in.Reason = "Family trip"
	d := e.applyOK(t, in)
	require.Len(t, e.events.events, 1)
	evt := e.events.events[0]
	assert.Equal(t, event.EventTypeLeaveRequested, evt.EventType)
	assert.Equal(t, tenant, evt.TenantID)
	assert.Equal(t, d.ID.String(), evt.AggregateID)
	p := evt.Payload.(event.LeaveRequestedPayload)
	assert.Equal(t, e.mgr, p.ApproverID)
	assert.Equal(t, "mgr@example.com", p.ApproverEmail)
	assert.Equal(t, "Ann Lee", p.EmployeeName)
	assert.Equal(t, "CASUAL", p.LeaveType)
	assert.Equal(t, "Family trip", p.Reason)
	assert.Equal(t, 2.0, p.Days)
	assert.Equal(t, "https://emp.test/leave", p.ReviewURL)
}

func TestApply_AdminApproverGetsAdminReviewLink(t *testing.T) {
	e := newEnv()
	in := e.in("2026-11-10", "2026-11-11")
	in.ApproverID = e.admin
	e.applyOK(t, in)
	require.Len(t, e.events.events, 1)
	p := e.events.events[0].Payload.(event.LeaveRequestedPayload)
	assert.Equal(t, "admin@example.com", p.ApproverEmail)
	assert.Equal(t, "https://admin.test/leave", p.ReviewURL)
}

func TestApply_NoEventOnValidationFailure(t *testing.T) {
	e := newEnv()
	uc := e.applyUC()
	_, err := uc.Execute(bgctx, tenant, e.employee, e.in("2026-11-05", "2026-11-04"))
	require.Error(t, err)
	in := e.in("2026-11-10", "2026-11-11")
	in.ApproverID = e.other // not an allowed approver
	_, err = uc.Execute(bgctx, tenant, e.employee, in)
	require.Error(t, err)
	assert.Empty(t, e.events.events)
}

func TestApply_PublishFailureFailsTransaction(t *testing.T) {
	e := newEnv()
	e.events.err = errors.New("outbox down")
	_, err := e.applyUC().Execute(bgctx, tenant, e.employee, e.in("2026-11-10", "2026-11-11"))
	require.Error(t, err)
}

func TestDecide_PublishesDecisionEvent(t *testing.T) {
	cases := []struct {
		name    string
		uc      func(*env) *decideLeave
		evtType event.EventType
	}{
		{"approve", func(e *env) *decideLeave {
			return NewApproveLeaveUseCase(e.requests, e.audit, nil, clock, e.events, testURLs).(*decideLeave)
		}, event.EventTypeLeaveApproved},
		{"reject", func(e *env) *decideLeave {
			return NewRejectLeaveUseCase(e.requests, e.audit, nil, clock, e.events, testURLs).(*decideLeave)
		}, event.EventTypeLeaveRejected},
	}
	for _, tc := range cases {
		t.Run(tc.name+" with comment", func(t *testing.T) {
			e := newEnv()
			id := e.pending(t)
			e.events.events = nil // drop the LeaveRequested event
			_, err := tc.uc(e).Execute(bgctx, tenant, e.mgr, false, id, " see you soon ")
			require.NoError(t, err)
			require.Len(t, e.events.events, 1)
			evt := e.events.events[0]
			assert.Equal(t, tc.evtType, evt.EventType)
			p := evt.Payload.(event.LeaveDecisionPayload)
			assert.Equal(t, "ann@example.com", p.EmployeeEmail)
			assert.Equal(t, "Decider Name", p.ApproverName)
			assert.Equal(t, "see you soon", p.Comment)
			assert.Equal(t, "https://emp.test/leave", p.LeaveURL)
			assert.Equal(t, e.employee, p.EmployeeID)
		})
	}

	t.Run("no event when decision is invalid", func(t *testing.T) {
		e := newEnv()
		id := e.pending(t)
		e.events.events = nil
		uc := tc0(e)
		_, err := uc.Execute(bgctx, tenant, e.other, false, id, "")
		require.Error(t, err) // not the approver
		_, err = uc.Execute(bgctx, tenant, e.employee, true, id, "")
		require.Error(t, err) // self decision
		_, err = uc.Execute(bgctx, tenant, e.mgr, false, id, string(make([]rune, 501)))
		require.Error(t, err) // comment too long
		assert.Empty(t, e.events.events)
	})
}

func tc0(e *env) *decideLeave {
	return NewApproveLeaveUseCase(e.requests, e.audit, nil, clock, e.events, testURLs).(*decideLeave)
}

func TestCancel_PublishesNothing(t *testing.T) {
	e := newEnv()
	id := e.pending(t)
	e.events.events = nil
	_, err := NewCancelLeaveUseCase(e.requests, e.audit, nil, clock).Execute(bgctx, tenant, e.employee, id)
	require.NoError(t, err)
	assert.Empty(t, e.events.events)
	assert.Equal(t, entity.LeaveStatusCancelled, e.requests.rows[id].Status)
}
