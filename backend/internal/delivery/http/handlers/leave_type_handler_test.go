package handlers

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

type listTypesFn func(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]*entity.LeaveType, error)

func (f listTypesFn) Execute(c context.Context, t uuid.UUID, inc bool) ([]*entity.LeaveType, error) {
	return f(c, t, inc)
}

func leaveTypeRouter(uc LeaveTypeUseCases, tenantID, userID uuid.UUID, roles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewLeaveTypeHandler(uc)
	r := gin.New()
	g := r.Group("/api/v1", withLeaveIdentity(tenantID, userID, roles))
	g.GET("/leave-types", h.ListLeaveTypes)
	return r
}

func TestLeaveHandler_IncludeInactiveOnlyForAdmins(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	var got bool
	uc := LeaveTypeUseCases{List: listTypesFn(func(_ context.Context, _ uuid.UUID, inc bool) ([]*entity.LeaveType, error) {
		got = inc
		return []*entity.LeaveType{}, nil
	})}
	do(leaveTypeRouter(uc, tenantID, userID, entity.RoleEmployee), "GET", "/api/v1/leave-types?include_inactive=true", "")
	assert.False(t, got)
	do(leaveTypeRouter(uc, tenantID, userID, entity.RoleAdmin), "GET", "/api/v1/leave-types?include_inactive=true", "")
	assert.True(t, got)
}
