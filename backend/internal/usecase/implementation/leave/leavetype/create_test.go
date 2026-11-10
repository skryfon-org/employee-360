package leavetype

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	leavetypes "github.com/skryfon-org/employee-360/backend/internal/types/leave"
)

func TestCreate(t *testing.T) {
	e := newEnv()
	tenant := uuid.New()
	e.types.add(tenant, "CASUAL", true)
	create := NewCreateLeaveTypeUseCase(e.types, e.audit, nil)

	lt, err := create.Execute(bgctx, tenant, e.admin, leavetypes.CreateLeaveTypeInput{Name: " Study ", Code: "study_leave"})
	require.NoError(t, err)
	assert.Equal(t, "STUDY_LEAVE", lt.Code)
	assert.Equal(t, "Study", lt.Name)
	assert.True(t, lt.IsActive)
	assert.Equal(t, []string{"leave_type.create"}, e.audit.actions)

	_, err = create.Execute(bgctx, tenant, e.admin, leavetypes.CreateLeaveTypeInput{Name: "Dup", Code: "casual"})
	assert.ErrorIs(t, err, domainerrors.ErrLeaveTypeCodeTaken)
	_, err = create.Execute(bgctx, tenant, e.admin, leavetypes.CreateLeaveTypeInput{Name: "", Code: "A"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)
	_, err = create.Execute(bgctx, tenant, e.admin, leavetypes.CreateLeaveTypeInput{Name: "Bad", Code: "has space"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidLeaveData)
	// Same code is fine in another tenant.
	_, err = create.Execute(bgctx, uuid.New(), e.admin, leavetypes.CreateLeaveTypeInput{Name: "Casual", Code: "CASUAL"})
	require.NoError(t, err)
}

func TestCreateIsActiveOptional(t *testing.T) {
	e := newEnv()
	tenant := uuid.New()
	create := NewCreateLeaveTypeUseCase(e.types, e.audit, nil)
	off := false
	lt, err := create.Execute(bgctx, tenant, e.admin, leavetypes.CreateLeaveTypeInput{Name: "Off", Code: "OFF", IsActive: &off})
	require.NoError(t, err)
	assert.False(t, lt.IsActive)
}
