package leavetype

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	e := newEnv()
	tenant := uuid.New()
	e.types.add(tenant, "A", true)
	e.types.add(tenant, "B", false)
	e.types.add(uuid.New(), "C", true)
	list := NewListLeaveTypesUseCase(e.types)

	active, err := list.Execute(bgctx, tenant, false)
	assert.NoError(t, err)
	assert.Len(t, active, 1)
	all, _ := list.Execute(bgctx, tenant, true)
	assert.Len(t, all, 2)
}
