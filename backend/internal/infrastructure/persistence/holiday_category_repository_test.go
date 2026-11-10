//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	holidayimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/holiday"
)

func TestGormHolidayCategoryRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormHolidayCategoryRepository(db)
	hRepo := NewGormHolidayRepository(db)

	tenantA := createTestTenant(t, db, "cat-tenant-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "cat-tenant-b-"+uuid.NewString())

	actorA := uuid.New()
	actorB := uuid.New()

	ctxA := context.Background()
	ctxB := context.Background()

	// 1. Create category in Tenant A
	catA := &entity.HolidayCategory{
		Name:        "Public Holiday",
		Description: "Official holidays",
		Color:       "#15803D",
		IsActive:    true,
		IsPublic:    true,
	}
	err := repo.Create(ctxA, tenantA.ID, actorA, catA)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, catA.ID)
	require.Equal(t, tenantA.ID, catA.TenantID)
	gotA, err := repo.GetByID(ctxA, tenantA.ID, catA.ID)
	require.NoError(t, err)
	require.True(t, gotA.IsPublic, "is_public persisted")
	require.Equal(t, &actorA, catA.CreatedBy)
	require.Equal(t, &actorA, catA.UpdatedBy)

	// 2. Tenant B can create category with same name (tenant isolation)
	catB := &entity.HolidayCategory{
		Name:        "Public Holiday",
		Description: "Tenant B holidays",
		Color:       "#15803D",
		IsActive:    true,
	}
	err = repo.Create(ctxB, tenantB.ID, actorB, catB)
	require.NoError(t, err)
	require.NotEqual(t, catA.ID, catB.ID)

	// 3. Duplicate category within Tenant A fails case-insensitively
	catADup := &entity.HolidayCategory{
		Name:  "public holiday",
		Color: "#3B82F6",
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, catADup)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryDuplicate)

	// 4. ExistsByName checks
	exists, err := repo.ExistsByName(ctxA, tenantA.ID, "PUBLIC HOLIDAY", nil)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = repo.ExistsByName(ctxA, tenantA.ID, "public holiday", &catA.ID)
	require.NoError(t, err)
	require.False(t, exists)

	// 5. Tenant B cannot read Tenant A's category
	_, err = repo.GetByID(ctxB, tenantB.ID, catA.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)

	// 6. Tenant A reads own category
	foundA, err := repo.GetByID(ctxA, tenantA.ID, catA.ID)
	require.NoError(t, err)
	assert.Equal(t, "Public Holiday", foundA.Name)
	assert.Equal(t, "#15803D", foundA.Color)

	// 7. Tenant B cannot update Tenant A's category
	catA.Name = "Hijacked"
	err = repo.Update(ctxB, tenantB.ID, actorB, catA)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)

	// 8. Tenant A updates own category
	catA.Name = "National Holiday"
	catA.Color = "#22C55E"
	err = repo.Update(ctxA, tenantA.ID, actorA, catA)
	require.NoError(t, err)

	updatedA, err := repo.GetByID(ctxA, tenantA.ID, catA.ID)
	require.NoError(t, err)
	assert.Equal(t, "National Holiday", updatedA.Name)
	assert.Equal(t, "#22C55E", updatedA.Color)
	assert.Equal(t, &actorA, updatedA.UpdatedBy)

	// 9. Create second category in Tenant A
	catA2 := &entity.HolidayCategory{
		Name:     "Optional Holiday",
		Color:    "#3B82F6",
		IsActive: false,
	}
	require.NoError(t, repo.Create(ctxA, tenantA.ID, actorA, catA2))

	// Attempting to rename catA2 to catA's name fails duplicate check
	catA2.Name = "national holiday"
	err = repo.Update(ctxA, tenantA.ID, actorA, catA2)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryDuplicate)

	// 10. List categories
	listAll, totalAll, err := repo.List(ctxA, tenantA.ID, nil, 20, 0)
	require.NoError(t, err)
	require.Len(t, listAll, 2)
	assert.Equal(t, int64(2), totalAll)

	activeOnly := true
	listActive, totalActive, err := repo.List(ctxA, tenantA.ID, &activeOnly, 20, 0)
	require.NoError(t, err)
	require.Len(t, listActive, 1)
	assert.Equal(t, int64(1), totalActive)
	assert.Equal(t, "National Holiday", listActive[0].Name)

	// Pagination: total ignores limit/offset; pages are disjoint and ordered by name.
	page1, totalP, err := repo.List(ctxA, tenantA.ID, nil, 1, 0)
	require.NoError(t, err)
	require.Len(t, page1, 1)
	assert.Equal(t, int64(2), totalP)
	page2, _, err := repo.List(ctxA, tenantA.ID, nil, 1, 1)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
	assert.Equal(t, listAll[0].ID, page1[0].ID)
	assert.Equal(t, listAll[1].ID, page2[0].ID)

	// 11. CountHolidaysByCategory: creates a holiday referencing catA
	count, err := repo.CountHolidaysByCategory(ctxA, tenantA.ID, catA.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	hA := &entity.Holiday{
		CategoryID: catA.ID,
		Name:       "Republic Day",
		Date:       time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, hRepo.Create(ctxA, tenantA.ID, actorA, hA))

	count, err = repo.CountHolidaysByCategory(ctxA, tenantA.ID, catA.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Tenant B count for same category ID is 0 (tenant isolation)
	countB, err := repo.CountHolidaysByCategory(ctxB, tenantB.ID, catA.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countB)

	// Soft-delete the holiday -> count becomes 0
	require.NoError(t, hRepo.SoftDelete(ctxA, tenantA.ID, hA.ID, actorA))
	countAfterDelete, err := repo.CountHolidaysByCategory(ctxA, tenantA.ID, catA.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countAfterDelete)

	// 12. Tenant B cannot delete Tenant A's category
	err = repo.SoftDelete(ctxB, tenantB.ID, catA.ID, actorB)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)

	// 13. Tenant A soft-deletes category
	err = repo.SoftDelete(ctxA, tenantA.ID, catA.ID, actorA)
	require.NoError(t, err)

	_, err = repo.GetByID(ctxA, tenantA.ID, catA.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)

	// 14. Can recreate category with same name after soft delete
	catARecreated := &entity.HolidayCategory{
		Name:  "National Holiday",
		Color: "#15803D",
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, catARecreated)
	require.NoError(t, err)
}

func TestCategoryDeleteUseCase_InUseRejection(t *testing.T) {
	db := setupTestDB(t)
	cRepo := NewGormHolidayCategoryRepository(db)
	hRepo := NewGormHolidayRepository(db)
	deleteUC := holidayimpl.NewDeleteCategoryUseCase(cRepo, nil, nil)

	tenant := createTestTenant(t, db, "delete-cat-"+uuid.NewString())
	actor := uuid.New()
	ctx := context.Background()

	cat := &entity.HolidayCategory{
		Name:     "Religious",
		Color:    "#7C3AED",
		IsActive: true,
	}
	require.NoError(t, cRepo.Create(ctx, tenant.ID, actor, cat))

	// Add an active holiday referencing this category
	holiday := &entity.Holiday{
		CategoryID: cat.ID,
		Name:       "Good Friday",
		Date:       time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, hRepo.Create(ctx, tenant.ID, actor, holiday))

	// Attempting to delete in-use category is rejected with ErrHolidayCategoryInUse
	err := deleteUC.Execute(ctx, tenant.ID, actor, holidaytypes.DeleteCategoryInput{ID: cat.ID})
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryInUse)

	// Soft-delete the holiday
	require.NoError(t, hRepo.SoftDelete(ctx, tenant.ID, holiday.ID, actor))

	// Now deleting the unused category succeeds
	err = deleteUC.Execute(ctx, tenant.ID, actor, holidaytypes.DeleteCategoryInput{ID: cat.ID})
	require.NoError(t, err)

	// Confirm category is soft-deleted
	_, err = cRepo.GetByID(ctx, tenant.ID, cat.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCategoryNotFound)
}
