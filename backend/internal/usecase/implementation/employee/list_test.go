package employee

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList_ScopedToTenantAndPaginated(t *testing.T) {
	f := newFixture()
	other := f.w.tenant("Other")
	f.w.user(other, "x@other.test", true, entity.RoleEmployee)
	gone := f.w.user(f.tenant, "gone@acme.test", true, entity.RoleEmployee)
	now := time.Now()
	gone.DeletedAt = &now

	res, err := f.w.newListEmployees().Execute(bg, f.tenant, emptypes.ListEmployeesQuery{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.Total, "only live users of this tenant")
	assert.Len(t, res.Employees, 2)

	_, err = f.w.newListEmployees().Execute(bg, f.tenant, emptypes.ListEmployeesQuery{Filter: entity.EmployeeFilter{EmploymentStatus: "bogus"}})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidEmployeeData)
	_, err = f.w.newListEmployees().Execute(bg, uuid.Nil, emptypes.ListEmployeesQuery{})
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

func TestList_PageSizeIsCappedAndFilterForwarded(t *testing.T) {
	f := newFixture()
	res, err := f.w.newListEmployees().Execute(bg, f.tenant, emptypes.ListEmployeesQuery{PageSize: 100000, Filter: entity.EmployeeFilter{Search: "x"}})
	require.NoError(t, err)
	assert.LessOrEqual(t, res.PageSize, 100)
	assert.Equal(t, 1, res.Page)
}
