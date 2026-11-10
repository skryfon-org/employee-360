package employee

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEmployeesRequest_ToFilter_PickerParams(t *testing.T) {
	f, err := ListEmployeesRequest{Search: "  ada lovelace ", Status: "active"}.ToFilter()
	require.NoError(t, err)
	assert.Equal(t, "ada lovelace", f.Search)
	require.NotNil(t, f.IsActive)
	assert.True(t, *f.IsActive)

	f, err = ListEmployeesRequest{}.ToFilter()
	require.NoError(t, err)
	assert.Nil(t, f.IsActive)

	_, err = ListEmployeesRequest{Status: "bogus"}.ToFilter()
	assert.Error(t, err)
}

func TestListEmployeesRequest_Include(t *testing.T) {
	f, err := ListEmployeesRequest{Include: "allocation"}.ToFilter()
	if err != nil || !f.IncludeAllocation {
		t.Fatalf("include=allocation should enable allocation, got %v %v", f.IncludeAllocation, err)
	}
	f, err = ListEmployeesRequest{}.ToFilter()
	if err != nil || f.IncludeAllocation {
		t.Fatalf("default must not include allocation")
	}
	if _, err = (ListEmployeesRequest{Include: "bogus"}).ToFilter(); err == nil {
		t.Fatal("bogus include must be rejected")
	}
}
