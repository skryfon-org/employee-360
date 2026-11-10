//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	holidaytypes "github.com/skryfon-org/employee-360/backend/internal/types/holiday"
	holidayimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/holiday"
)

func createTestHolidayCategory(t *testing.T, db *gorm.DB, tenantID uuid.UUID, name string) *entity.HolidayCategory {
	t.Helper()
	cat := &entity.HolidayCategory{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Color:     "#15803D",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(cat).Error; err != nil {
		t.Fatalf("failed to create test holiday category: %v", err)
	}
	return cat
}

func TestGormHolidayRepository_CRUDAndTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormHolidayRepository(db)

	tenantA := createTestTenant(t, db, "holiday-tenant-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "holiday-tenant-b-"+uuid.NewString())

	actorA := uuid.New()
	actorB := uuid.New()

	catA := createTestHolidayCategory(t, db, tenantA.ID, "Public")
	catB := createTestHolidayCategory(t, db, tenantB.ID, "Public")

	ctxA := context.Background()
	ctxB := context.Background()

	date2026_01_01 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// 1. Create holiday in Tenant A
	holidayA := &entity.Holiday{
		CategoryID:  catA.ID,
		Name:        "New Year's Day",
		Description: "First day of 2026",
		Date:        date2026_01_01,
	}
	err := repo.Create(ctxA, tenantA.ID, actorA, holidayA)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, holidayA.ID)
	require.Equal(t, tenantA.ID, holidayA.TenantID)
	require.Equal(t, &actorA, holidayA.CreatedBy)
	require.Equal(t, &actorA, holidayA.UpdatedBy)

	// 2. Tenant B can create holiday on same date with same name (tenant isolation)
	holidayB := &entity.Holiday{
		CategoryID:  catB.ID,
		Name:        "New Year's Day",
		Description: "Tenant B New Year",
		Date:        date2026_01_01,
	}
	err = repo.Create(ctxB, tenantB.ID, actorB, holidayB)
	require.NoError(t, err)
	require.NotEqual(t, holidayA.ID, holidayB.ID)

	// 3. Duplicate holiday within Tenant A fails case-insensitively
	holidayADup := &entity.Holiday{
		CategoryID: catA.ID,
		Name:       "new year's day",
		Date:       date2026_01_01,
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, holidayADup)
	require.ErrorIs(t, err, domainerrors.ErrHolidayDuplicate)

	// 4. ExistsDuplicate checks
	exists, err := repo.ExistsDuplicate(ctxA, tenantA.ID, date2026_01_01, "NEW YEAR'S DAY", nil)
	require.NoError(t, err)
	require.True(t, exists)

	// ExistsDuplicate excluding holidayA.ID returns false
	exists, err = repo.ExistsDuplicate(ctxA, tenantA.ID, date2026_01_01, "new year's day", &holidayA.ID)
	require.NoError(t, err)
	require.False(t, exists)

	// Different date returns false
	exists, err = repo.ExistsDuplicate(ctxA, tenantA.ID, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), "New Year's Day", nil)
	require.NoError(t, err)
	require.False(t, exists)

	// 5. Tenant B cannot read Tenant A's holiday
	_, err = repo.GetByID(ctxB, tenantB.ID, holidayA.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)

	// 6. Tenant A reads own holiday
	foundA, err := repo.GetByID(ctxA, tenantA.ID, holidayA.ID)
	require.NoError(t, err)
	require.Equal(t, "New Year's Day", foundA.Name)
	require.Equal(t, date2026_01_01, foundA.Date)

	// 7. Tenant B cannot update Tenant A's holiday
	holidayA.Name = "Hijacked Name"
	err = repo.Update(ctxB, tenantB.ID, actorB, holidayA)
	require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)

	// 8. Tenant A updates own holiday
	holidayA.Name = "Global New Year"
	err = repo.Update(ctxA, tenantA.ID, actorA, holidayA)
	require.NoError(t, err)

	updatedA, err := repo.GetByID(ctxA, tenantA.ID, holidayA.ID)
	require.NoError(t, err)
	require.Equal(t, "Global New Year", updatedA.Name)
	require.Equal(t, &actorA, updatedA.UpdatedBy)

	// 9. Create a second holiday in Tenant A
	holidayA2 := &entity.Holiday{
		CategoryID: catA.ID,
		Name:       "Holiday Two",
		Date:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, holidayA2)
	require.NoError(t, err)

	// Attempt to update holidayA2 onto holidayA's date and name triggers unique constraint error
	holidayA2.Name = "global new year"
	holidayA2.Date = date2026_01_01
	err = repo.Update(ctxA, tenantA.ID, actorA, holidayA2)
	require.ErrorIs(t, err, domainerrors.ErrHolidayDuplicate)

	// 10. Tenant B cannot delete Tenant A's holiday
	err = repo.SoftDelete(ctxB, tenantB.ID, holidayA.ID, actorB)
	require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)

	// 11. Tenant A soft-deletes holiday
	err = repo.SoftDelete(ctxA, tenantA.ID, holidayA.ID, actorA)
	require.NoError(t, err)

	// 12. Soft-deleted holiday is no longer returned by GetByID
	_, err = repo.GetByID(ctxA, tenantA.ID, holidayA.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)

	// 13. ExistsDuplicate no longer reports duplicate for soft-deleted holiday
	exists, err = repo.ExistsDuplicate(ctxA, tenantA.ID, date2026_01_01, "Global New Year", nil)
	require.NoError(t, err)
	require.False(t, exists)

	// 14. Can recreate holiday with same date and name after soft deletion
	holidayARecreated := &entity.Holiday{
		CategoryID: catA.ID,
		Name:       "Global New Year",
		Date:       date2026_01_01,
	}
	err = repo.Create(ctxA, tenantA.ID, actorA, holidayARecreated)
	require.NoError(t, err)
}

func TestGormHolidayRepository_ListByYear(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormHolidayRepository(db)

	tenantA := createTestTenant(t, db, "holiday-list-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "holiday-list-b-"+uuid.NewString())
	actor := uuid.New()

	catA1 := createTestHolidayCategory(t, db, tenantA.ID, "National")
	catA2 := createTestHolidayCategory(t, db, tenantA.ID, "Religious")
	catB := createTestHolidayCategory(t, db, tenantB.ID, "National")

	ctxA := context.Background()
	ctxB := context.Background()

	// Tenant A: 3 holidays in 2026, 1 in 2027
	h2026_Jan := &entity.Holiday{
		CategoryID: catA1.ID,
		Name:       "New Year's Day",
		Date:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	h2026_May := &entity.Holiday{
		CategoryID: catA1.ID,
		Name:       "Labor Day",
		Date:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	h2026_Dec := &entity.Holiday{
		CategoryID: catA2.ID,
		Name:       "Christmas Day",
		Date:       time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC),
	}
	h2027_Jan := &entity.Holiday{
		CategoryID: catA1.ID,
		Name:       "Next Year Day",
		Date:       time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	require.NoError(t, repo.Create(ctxA, tenantA.ID, actor, h2026_Dec))
	require.NoError(t, repo.Create(ctxA, tenantA.ID, actor, h2026_Jan))
	require.NoError(t, repo.Create(ctxA, tenantA.ID, actor, h2026_May))
	require.NoError(t, repo.Create(ctxA, tenantA.ID, actor, h2027_Jan))

	// Tenant B: 1 holiday in 2026
	hB := &entity.Holiday{
		CategoryID: catB.ID,
		Name:       "Tenant B Holiday",
		Date:       time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, repo.Create(ctxB, tenantB.ID, actor, hB))

	// List year 2026 for Tenant A: returns 3 holidays ordered by date
	listA, err := repo.ListByYear(ctxA, tenantA.ID, 2026, nil, nil)
	require.NoError(t, err)
	require.Len(t, listA, 3)
	assert.Equal(t, "New Year's Day", listA[0].Name)
	assert.Equal(t, "Labor Day", listA[1].Name)
	assert.Equal(t, "Christmas Day", listA[2].Name)

	// Month filter: month=5 returns only May holiday
	monthMay := 5
	listMay, err := repo.ListByYear(ctxA, tenantA.ID, 2026, nil, &monthMay)
	require.NoError(t, err)
	require.Len(t, listMay, 1)
	assert.Equal(t, "Labor Day", listMay[0].Name)

	// Category filter: catA2 returns only Christmas
	listCat2, err := repo.ListByYear(ctxA, tenantA.ID, 2026, &catA2.ID, nil)
	require.NoError(t, err)
	require.Len(t, listCat2, 1)
	assert.Equal(t, "Christmas Day", listCat2[0].Name)

	// Category from another tenant (or a nonexistent one) yields an empty list, never leaking rows
	crossCat, err := repo.ListByYear(ctxA, tenantA.ID, 2026, &catB.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, crossCat)
	unknownCat := uuid.New()
	missingCat, err := repo.ListByYear(ctxA, tenantA.ID, 2026, &unknownCat, nil)
	require.NoError(t, err)
	assert.Empty(t, missingCat)
	crossCatB, err := repo.ListByYear(ctxB, tenantB.ID, 2026, &catA1.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, crossCatB)

	// Cross-tenant isolation: Tenant B only sees Tenant B's holiday
	listB, err := repo.ListByYear(ctxB, tenantB.ID, 2026, nil, nil)
	require.NoError(t, err)
	require.Len(t, listB, 1)
	assert.Equal(t, "Tenant B Holiday", listB[0].Name)

	// Soft-deleted holiday not returned in ListByYear
	require.NoError(t, repo.SoftDelete(ctxA, tenantA.ID, h2026_Jan.ID, actor))
	listAAfterDelete, err := repo.ListByYear(ctxA, tenantA.ID, 2026, nil, nil)
	require.NoError(t, err)
	require.Len(t, listAAfterDelete, 2)
}

func TestGormHolidayCalendarSettingsRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormHolidayCalendarSettingsRepository(db)

	tenantA := createTestTenant(t, db, "settings-tenant-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "settings-tenant-b-"+uuid.NewString())
	actor := uuid.New()

	ctxA := context.Background()
	ctxB := context.Background()

	// Initial Get returns not found
	_, err := repo.GetByTenantID(ctxA, tenantA.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsNotFound)

	// Upsert settings for Tenant A
	defaultYear := 2026
	settingsA := &entity.HolidayCalendarSettings{
		TenantID:     tenantA.ID,
		DefaultView:  "calendar",
		DefaultYear:  &defaultYear,
		WeekStartDay: 1, // Monday
	}
	err = repo.Upsert(ctxA, tenantA.ID, actor, settingsA)
	require.NoError(t, err)

	// GetByTenantID retrieves settings
	foundA, err := repo.GetByTenantID(ctxA, tenantA.ID)
	require.NoError(t, err)
	assert.Equal(t, "calendar", foundA.DefaultView)
	require.NotNil(t, foundA.DefaultYear)
	assert.Equal(t, 2026, *foundA.DefaultYear)
	assert.Equal(t, 1, foundA.WeekStartDay)

	// Cross-tenant isolation: Tenant B cannot get Tenant A's settings
	_, err = repo.GetByTenantID(ctxB, tenantB.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsNotFound)

	// Update settings for Tenant A via Upsert (clearing default_year)
	settingsA.DefaultView = "list"
	settingsA.DefaultYear = nil
	settingsA.WeekStartDay = 0 // Sunday
	err = repo.Upsert(ctxA, tenantA.ID, actor, settingsA)
	require.NoError(t, err)

	foundAUpdated, err := repo.GetByTenantID(ctxA, tenantA.ID)
	require.NoError(t, err)
	assert.Equal(t, "list", foundAUpdated.DefaultView)
	assert.Nil(t, foundAUpdated.DefaultYear, "default_year must be cleared to NULL")
	assert.Equal(t, 0, foundAUpdated.WeekStartDay)
}

func TestHolidayCalendarSettingsUseCase_RoundTripAndDefaults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormHolidayCalendarSettingsRepository(db)
	getUC := holidayimpl.NewGetSettingsUseCase(repo)
	updateUC := holidayimpl.NewUpdateSettingsUseCase(repo, nil, nil)

	tenantA := createTestTenant(t, db, "settings-uc-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "settings-uc-b-"+uuid.NewString())
	actor := uuid.New()
	ctx := context.Background()

	// 1. Get with no row returns defaults, does NOT write a row
	defaults, err := getUC.Execute(ctx, tenantA.ID)
	require.NoError(t, err)
	assert.Equal(t, tenantA.ID, defaults.TenantID)
	assert.Equal(t, "calendar", defaults.DefaultView)
	assert.Nil(t, defaults.DefaultYear)
	assert.Equal(t, 1, defaults.WeekStartDay)

	// Verify no row was written
	_, err = repo.GetByTenantID(ctx, tenantA.ID)
	require.ErrorIs(t, err, domainerrors.ErrHolidayCalendarSettingsNotFound)

	// 2. Update then Get returns the same values (round-trip)
	year := 2027
	updated, err := updateUC.Execute(ctx, tenantA.ID, actor, holidaytypes.UpdateSettingsInput{
		DefaultView:  "list",
		DefaultYear:  &year,
		WeekStartDay: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, "list", updated.DefaultView)
	require.NotNil(t, updated.DefaultYear)
	assert.Equal(t, 2027, *updated.DefaultYear)
	assert.Equal(t, 0, updated.WeekStartDay)

	fetched, err := getUC.Execute(ctx, tenantA.ID)
	require.NoError(t, err)
	assert.Equal(t, "list", fetched.DefaultView)
	require.NotNil(t, fetched.DefaultYear)
	assert.Equal(t, 2027, *fetched.DefaultYear)
	assert.Equal(t, 0, fetched.WeekStartDay)

	// 3. Update clearing default_year
	cleared, err := updateUC.Execute(ctx, tenantA.ID, actor, holidaytypes.UpdateSettingsInput{
		DefaultView:  "calendar",
		DefaultYear:  nil,
		WeekStartDay: 6,
	})
	require.NoError(t, err)
	assert.Nil(t, cleared.DefaultYear)

	fetchedCleared, err := getUC.Execute(ctx, tenantA.ID)
	require.NoError(t, err)
	assert.Equal(t, "calendar", fetchedCleared.DefaultView)
	assert.Nil(t, fetchedCleared.DefaultYear)
	assert.Equal(t, 6, fetchedCleared.WeekStartDay)

	// 4. Tenant isolation: Tenant B gets defaults, not Tenant A's settings
	defaultsB, err := getUC.Execute(ctx, tenantB.ID)
	require.NoError(t, err)
	assert.Equal(t, "calendar", defaultsB.DefaultView)
	assert.Nil(t, defaultsB.DefaultYear)
	assert.Equal(t, 1, defaultsB.WeekStartDay)
}

func TestHolidayDateNormalization_UseCaseIntegration(t *testing.T) {
	db := setupTestDB(t)
	hRepo := NewGormHolidayRepository(db)
	cRepo := NewGormHolidayCategoryRepository(db)
	createUC := holidayimpl.NewCreateHolidayUseCase(hRepo, cRepo, nil, nil)
	updateUC := holidayimpl.NewUpdateHolidayUseCase(hRepo, cRepo, nil, nil)

	tenant := createTestTenant(t, db, "holiday-date-norm-"+uuid.NewString())
	cat := createTestHolidayCategory(t, db, tenant.ID, "National")
	actor := uuid.New()
	ctx := context.Background()

	// 1. Create with non-midnight time and non-UTC timezone normalizes to UTC midnight
	loc := time.FixedZone("IST", 5*3600+30*60)
	inputDate := time.Date(2026, 8, 15, 17, 45, 30, 500, loc)
	expectedMidnight := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	created, err := createUC.Execute(ctx, tenant.ID, actor, holidaytypes.CreateHolidayInput{
		CategoryID:  cat.ID,
		Name:        "Independence Day",
		Description: "National celebration",
		Date:        inputDate,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, expectedMidnight, created.Date)

	// Verify persistence in DB is also normalized to UTC midnight
	persisted, err := hRepo.GetByID(ctx, tenant.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedMidnight, persisted.Date)

	// 2. Duplicate detection works across different time components on the same date
	differentTimeSameDay := time.Date(2026, 8, 15, 23, 59, 59, 0, time.UTC)
	_, err = createUC.Execute(ctx, tenant.ID, actor, holidaytypes.CreateHolidayInput{
		CategoryID:  cat.ID,
		Name:        "independence day", // case-insensitive duplicate check
		Description: "Duplicate entry",
		Date:        differentTimeSameDay,
	})
	require.ErrorIs(t, err, domainerrors.ErrHolidayDuplicate)

	// 3. Update with non-midnight time normalizes to UTC midnight
	updateInputDate := time.Date(2026, 8, 16, 21, 15, 0, 0, time.UTC)
	expectedUpdateMidnight := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)

	updated, err := updateUC.Execute(ctx, tenant.ID, actor, holidaytypes.UpdateHolidayInput{
		ID:          created.ID,
		CategoryID:  cat.ID,
		Name:        "Independence Day Observed",
		Description: "Updated holiday date",
		Date:        updateInputDate,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, expectedUpdateMidnight, updated.Date)

	persistedUpdated, err := hRepo.GetByID(ctx, tenant.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedUpdateMidnight, persistedUpdated.Date)

	// 4. Validation errors: zero date rejected
	_, err = createUC.Execute(ctx, tenant.ID, actor, holidaytypes.CreateHolidayInput{
		CategoryID: cat.ID,
		Name:       "Zero Date Holiday",
		Date:       time.Time{},
	})
	require.ErrorIs(t, err, domainerrors.ErrHolidayDateInvalid)

	// 5. Validation errors: year out of range [2000, 2100] rejected
	_, err = createUC.Execute(ctx, tenant.ID, actor, holidaytypes.CreateHolidayInput{
		CategoryID: cat.ID,
		Name:       "Past Holiday",
		Date:       time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)

	_, err = createUC.Execute(ctx, tenant.ID, actor, holidaytypes.CreateHolidayInput{
		CategoryID: cat.ID,
		Name:       "Far Future Holiday",
		Date:       time.Date(2101, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)

	// 6. Update validation: zero date and out-of-range year rejected
	_, err = updateUC.Execute(ctx, tenant.ID, actor, holidaytypes.UpdateHolidayInput{
		ID:         created.ID,
		CategoryID: cat.ID,
		Name:       "Invalid Date Update",
		Date:       time.Time{},
	})
	require.ErrorIs(t, err, domainerrors.ErrHolidayDateInvalid)

	_, err = updateUC.Execute(ctx, tenant.ID, actor, holidaytypes.UpdateHolidayInput{
		ID:         created.ID,
		CategoryID: cat.ID,
		Name:       "Invalid Year Update",
		Date:       time.Date(1999, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	require.ErrorIs(t, err, domainerrors.ErrHolidayYearInvalid)
}

func TestGormHolidayRepository_ListByDateRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormHolidayRepository(db)
	ctx := context.Background()
	tenantA := createTestTenant(t, db, "holiday-range-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "holiday-range-b-"+uuid.NewString())
	actor := uuid.New()
	catA := createTestHolidayCategory(t, db, tenantA.ID, "National")
	catB := createTestHolidayCategory(t, db, tenantB.ID, "National")
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

	mk := func(tid, cat uuid.UUID, name string, dt time.Time) *entity.Holiday {
		h := &entity.Holiday{CategoryID: cat, Name: name, Date: dt}
		require.NoError(t, repo.Create(ctx, tid, actor, h))
		return h
	}
	mk(tenantA.ID, catA.ID, "Before", day(2026, 10, 23))
	mk(tenantA.ID, catA.ID, "AtFrom", day(2026, 10, 24))
	mk(tenantA.ID, catA.ID, "Zeta", day(2026, 12, 25))
	mk(tenantA.ID, catA.ID, "Alpha", day(2026, 12, 25))
	mk(tenantA.ID, catA.ID, "NextYear", day(2027, 12, 31))
	mk(tenantA.ID, catA.ID, "AtTo", day(2028, 1, 1))
	gone := mk(tenantA.ID, catA.ID, "Deleted", day(2026, 11, 1))
	require.NoError(t, repo.SoftDelete(ctx, tenantA.ID, gone.ID, actor))
	mk(tenantB.ID, catB.ID, "TenantB", day(2026, 11, 2))

	got, err := repo.ListByDateRange(ctx, tenantA.ID, day(2026, 10, 24), day(2028, 1, 1))
	require.NoError(t, err)
	var names []string
	for _, h := range got {
		names = append(names, h.Name)
	}
	assert.Equal(t, []string{"AtFrom", "Alpha", "Zeta", "NextYear"}, names, "from inclusive, to exclusive, date then name order, no deleted/other tenant")

	gotB, err := repo.ListByDateRange(ctx, tenantB.ID, day(2026, 1, 1), day(2030, 1, 1))
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, "TenantB", gotB[0].Name)

	none, err := repo.ListByDateRange(ctx, uuid.Nil, day(2026, 1, 1), day(2030, 1, 1))
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestGormHolidayRepository_CategoryIsPublicFlag(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormHolidayRepository(db)
	ctx := context.Background()
	tenantA := createTestTenant(t, db, "holiday-pub-a-"+uuid.NewString())
	tenantB := createTestTenant(t, db, "holiday-pub-b-"+uuid.NewString())
	actor := uuid.New()
	pub := createTestHolidayCategory(t, db, tenantA.ID, "Public")
	require.NoError(t, db.Model(pub).Update("is_public", true).Error)
	opt := createTestHolidayCategory(t, db, tenantA.ID, "Optional")
	catB := createTestHolidayCategory(t, db, tenantB.ID, "Public")
	require.NoError(t, db.Model(catB).Update("is_public", true).Error)
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }

	hp := &entity.Holiday{CategoryID: pub.ID, Name: "P", Date: day(12, 25)}
	ho := &entity.Holiday{CategoryID: opt.ID, Name: "O", Date: day(12, 26)}
	require.NoError(t, repo.Create(ctx, tenantA.ID, actor, hp))
	require.NoError(t, repo.Create(ctx, tenantA.ID, actor, ho))
	require.NoError(t, repo.Create(ctx, tenantB.ID, actor, &entity.Holiday{CategoryID: catB.ID, Name: "B", Date: day(12, 25)}))

	flags := func(hs []*entity.Holiday) map[string]bool {
		m := map[string]bool{}
		for _, h := range hs {
			m[h.Name] = h.CategoryIsPublic
		}
		return m
	}
	byRange, err := repo.ListByDateRange(ctx, tenantA.ID, day(1, 1), day(12, 31))
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"P": true, "O": false}, flags(byRange), "other tenant's public holiday never appears")
	byYear, err := repo.ListByYear(ctx, tenantA.ID, 2026, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"P": true, "O": false}, flags(byYear))
	got, err := repo.GetByID(ctx, tenantA.ID, hp.ID)
	require.NoError(t, err)
	assert.True(t, got.CategoryIsPublic)
	_, err = repo.GetByID(ctx, tenantB.ID, hp.ID)
	assert.ErrorIs(t, err, domainerrors.ErrHolidayNotFound)
}
