package leave

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestResponseIncludesDays(t *testing.T) {
	s := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	d := &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{
		DayPart: entity.LeaveDayPartFull, StartDate: s, EndDate: s.AddDate(0, 0, 2), Days: 3}}
	b, err := json.Marshal(ToLeaveRequestResponse(d))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	require.Equal(t, float64(3), m["days"])
}

func TestResponseDecidedByName(t *testing.T) {
	und := ToLeaveRequestResponse(&entity.LeaveRequestDetail{})
	require.Nil(t, und.DecidedByName)
	id := uuid.New()
	got := ToLeaveRequestResponse(&entity.LeaveRequestDetail{
		LeaveRequest: entity.LeaveRequest{DecidedBy: &id}, DecidedByName: "Mia Mgr"})
	require.NotNil(t, got.DecidedByName)
	require.Equal(t, "Mia Mgr", *got.DecidedByName)
	b, _ := json.Marshal(und)
	require.Contains(t, string(b), `"decided_by_name":null`)
}
