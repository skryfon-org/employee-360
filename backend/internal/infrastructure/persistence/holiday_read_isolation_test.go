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
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	holidayimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/holiday"
)

// Real usecases + real GORM repositories (no HTTP layer): a tenant B user
// listing holidays, categories and calendar settings must see none of tenant
// A's data. The HTTP layer passes the token tenant straight into these
// usecases (see routes_holiday_test.go), so this closes the loop end to end
// below the transport.
func TestHolidayReads_CrossTenantIsolation_RealUsecasesAndRepos(t *testing.T) {
	db := setupTestDB(t)
	holidayRepo := NewGormHolidayRepository(db)
	categoryRepo := NewGormHolidayCategoryRepository(db)
	settingsRepo := NewGormHolidayCalendarSettingsRepository(db)

	listHolidays := holidayimpl.NewListHolidaysByYearUseCase(holidayRepo)
	listCategories := holidayimpl.NewListCategoriesUseCase(categoryRepo)
	getSettings := holidayimpl.NewGetSettingsUseCase(settingsRepo)
	updateSettings := holidayimpl.NewUpdateSettingsUseCase(settingsRepo, nil, nil)

	tenantA := createTestTenant(t, db, "read-iso-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "read-iso-b-"+uuid.NewString())
	actor := uuid.New()
	ctx := context.Background()

	catA := createTestHolidayCategory(t, db, tenantA.ID, "A-Only-Category")
	require.NoError(t, holidayRepo.Create(ctx, tenantA.ID, actor, &entity.Holiday{
		CategoryID: catA.ID,
		Name:       "A-Only-Holiday",
		Date:       time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
	}))
	year := 2031
	_, err := updateSettings.Execute(ctx, tenantA.ID, actor, holidaytypes.UpdateSettingsInput{
		DefaultView: "list", DefaultYear: &year, WeekStartDay: 0,
	})
	require.NoError(t, err)

	// Sanity: tenant A sees its own data through the same usecases.
	aHolidays, err := listHolidays.Execute(ctx, tenantA.ID, holidaytypes.ListHolidaysByYearQuery{Year: 2026})
	require.NoError(t, err)
	require.Len(t, aHolidays, 1)
	aCats, err := listCategories.Execute(ctx, tenantA.ID, holidaytypes.ListCategoriesQuery{Page: 1, PageSize: 50})
	require.NoError(t, err)
	require.NotEmpty(t, aCats.Categories)

	// Tenant B sees none of it.
	bHolidays, err := listHolidays.Execute(ctx, tenantB.ID, holidaytypes.ListHolidaysByYearQuery{Year: 2026})
	require.NoError(t, err)
	assert.Empty(t, bHolidays, "tenant B must not see tenant A holidays")

	bCats, err := listCategories.Execute(ctx, tenantB.ID, holidaytypes.ListCategoriesQuery{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.Empty(t, bCats.Categories, "tenant B must not see tenant A categories")
	assert.Zero(t, bCats.Total)

	bSettings, err := getSettings.Execute(ctx, tenantB.ID)
	require.NoError(t, err)
	assert.Equal(t, tenantB.ID, bSettings.TenantID)
	assert.Equal(t, "calendar", bSettings.DefaultView, "tenant B must get defaults, not tenant A settings")
	assert.Nil(t, bSettings.DefaultYear)
	assert.Equal(t, 1, bSettings.WeekStartDay)
}
