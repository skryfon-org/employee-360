package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestListApprovers(t *testing.T) {
	e := newEnv()
	uc := NewListApproversUseCase(e.profiles, e.approvers)

	out, err := uc.Execute(bgctx, tenant, e.employee)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, e.mgr, out[0].UserID)
	assert.Equal(t, entity.LeaveApproverManager, out[0].Kind)
	assert.Equal(t, e.admin, out[1].UserID)

	// Manager who is also an admin appears once, as manager.
	e.approvers.admins = []uuid.UUID{e.admin, e.mgr}
	out, _ = uc.Execute(bgctx, tenant, e.employee)
	require.Len(t, out, 2)
	assert.Equal(t, entity.LeaveApproverManager, out[0].Kind)

	// The caller never sees themselves; no profile -> admins only.
	out, _ = uc.Execute(bgctx, tenant, e.admin)
	for _, a := range out {
		assert.NotEqual(t, e.admin, a.UserID)
	}
	e.profiles.managers = map[uuid.UUID]uuid.UUID{}
	out, _ = uc.Execute(bgctx, tenant, e.employee)
	require.Len(t, out, 2) // admin + mgr(as admin)
}
