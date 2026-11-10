package leavetype

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

func TestUpdate(t *testing.T) {
	e := newEnv()
	tenant := uuid.New()
	e.types.add(tenant, "CASUAL", true)
	lt := e.types.add(tenant, "STUDY", true)
	update := NewUpdateLeaveTypeUseCase(e.types, e.audit, nil)

	off := false
	upd, err := update.Execute(bgctx, tenant, e.admin, lt.ID, leavetypes.UpdateLeaveTypeInput{IsActive: &off})
	require.NoError(t, err)
	assert.False(t, upd.IsActive)
	assert.Equal(t, []string{"leave_type.update"}, e.audit.actions)

	taken := "CASUAL"
	_, err = update.Execute(bgctx, tenant, e.admin, lt.ID, leavetypes.UpdateLeaveTypeInput{Code: &taken})
	assert.ErrorIs(t, err, domainerrors.ErrLeaveTypeCodeTaken)
	_, err = update.Execute(bgctx, uuid.New(), e.admin, lt.ID, leavetypes.UpdateLeaveTypeInput{IsActive: &off})
	assert.ErrorIs(t, err, domainerrors.ErrLeaveTypeNotFound, "cross-tenant")
}
