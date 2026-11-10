//go:build integration

package persistence

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
)

func TestLeaveTypeRepository_TenantScopingAndUniqueness(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormLeaveTypeRepository(e.db)
	actor := uuid.New()

	a := &entity.LeaveType{Name: "Casual", Code: "CASUAL", IsActive: true}
	require.NoError(t, repo.Create(c, e.tA.ID, actor, a))
	assert.ErrorIs(t, repo.Create(c, e.tA.ID, actor, &entity.LeaveType{Name: "Dup", Code: "CASUAL", IsActive: true}), domainerrors.ErrLeaveTypeCodeTaken)
	require.NoError(t, repo.Create(c, e.tB.ID, actor, &entity.LeaveType{Name: "Casual", Code: "CASUAL", IsActive: true}))

	_, err := repo.GetByID(c, e.tB.ID, a.ID)
	assert.ErrorIs(t, err, domainerrors.ErrLeaveTypeNotFound)

	hijack := *a
	hijack.Name = "hijacked"
	assert.ErrorIs(t, repo.Update(c, e.tB.ID, actor, &hijack), domainerrors.ErrLeaveTypeNotFound)
	got, _ := repo.GetByID(c, e.tA.ID, a.ID)
	assert.Equal(t, "Casual", got.Name)

	a.IsActive = false
	require.NoError(t, repo.Update(c, e.tA.ID, actor, a))
	active, err := repo.List(c, e.tA.ID, true)
	require.NoError(t, err)
	assert.Empty(t, active)
	all, _ := repo.List(c, e.tA.ID, false)
	assert.Len(t, all, 1)

	taken, _ := repo.ExistsByCode(c, e.tA.ID, "CASUAL", uuid.Nil)
	assert.True(t, taken)
	taken, _ = repo.ExistsByCode(c, e.tA.ID, "CASUAL", a.ID)
	assert.False(t, taken, "own row excluded")

	// A soft-deleted code can be reused (partial unique index).
	require.NoError(t, e.db.Exec("UPDATE leave_types SET deleted_at = NOW() WHERE id = ?", a.ID).Error)
	require.NoError(t, repo.Create(c, e.tA.ID, actor, &entity.LeaveType{Name: "Casual2", Code: "CASUAL", IsActive: true}))
}
