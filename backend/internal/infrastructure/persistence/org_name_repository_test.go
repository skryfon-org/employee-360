//go:build integration

package persistence

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestGormOrgNameRepository_TenantIsolationAndSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	names := NewGormOrgNameRepository(db)
	depts := NewGormDepartmentRepository(db)
	poss := NewGormPositionRepository(db)

	tenantA := createTestTenant(t, db, "orgname-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "orgname-b-"+uuid.NewString())
	actor := uuid.New()
	ctx := context.Background()

	deptB := &entity.Department{Name: "Engineering"}
	require.NoError(t, depts.Create(ctx, tenantB.ID, actor, deptB))
	posB := &entity.Position{Name: "Engineer"}
	require.NoError(t, poss.Create(ctx, tenantB.ID, actor, posB))

	// Owning tenant sees the names.
	n, found, err := names.DepartmentName(ctx, tenantB.ID, deptB.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "Engineering", n)
	n, found, err = names.PositionName(ctx, tenantB.ID, posB.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "Engineer", n)

	// Another tenant must not.
	_, found, err = names.DepartmentName(ctx, tenantA.ID, deptB.ID)
	require.NoError(t, err)
	require.False(t, found)
	_, found, err = names.PositionName(ctx, tenantA.ID, posB.ID)
	require.NoError(t, err)
	require.False(t, found)

	// Soft-deleted rows are not found.
	require.NoError(t, depts.Delete(ctx, tenantB.ID, deptB.ID, actor))
	require.NoError(t, poss.Delete(ctx, tenantB.ID, posB.ID, actor))
	_, found, err = names.DepartmentName(ctx, tenantB.ID, deptB.ID)
	require.NoError(t, err)
	require.False(t, found)
	_, found, err = names.PositionName(ctx, tenantB.ID, posB.ID)
	require.NoError(t, err)
	require.False(t, found)
}
