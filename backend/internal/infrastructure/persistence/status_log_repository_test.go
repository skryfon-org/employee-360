//go:build integration

package persistence

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

func TestStatusLogRepository_TenantScopingAndCRUD(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormStatusLogRepository(e.db)
	actor := uuid.New()
	today := day(2026, 10, 27)

	cat := entity.ActivityCategoryMeeting
	entryA := &entity.StatusLogEntry{
		LogDate:          today,
		ActivityCategory: &cat,
		Hours:            3.5,
		Notes:            "Design sync",
		UserID:           e.uA.ID,
	}

	// 1. Create in Tenant A
	require.NoError(t, repo.Create(c, e.tA.ID, actor, entryA))
	assert.Equal(t, e.tA.ID, entryA.TenantID)
	assert.NotEqual(t, uuid.Nil, entryA.ID)

	// 2. Cross-tenant read must fail
	_, err := repo.GetByID(c, e.tB.ID, entryA.ID)
	assert.ErrorIs(t, err, domainerrors.ErrStatusLogNotFound)

	// 3. Read in Tenant A
	got, err := repo.GetByID(c, e.tA.ID, entryA.ID)
	require.NoError(t, err)
	assert.Equal(t, 3.5, got.Hours)
	assert.Equal(t, "Design sync", got.Notes)

	// 4. GetDayTotalHours
	total, err := repo.GetDayTotalHours(c, e.tA.ID, e.uA.ID, today, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, 3.5, total)

	// In Tenant B, total is 0
	totalB, err := repo.GetDayTotalHours(c, e.tB.ID, e.uA.ID, today, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, totalB)

	// 5. Update
	got.Hours = 5.0
	got.Notes = "Updated notes"
	require.NoError(t, repo.Update(c, e.tA.ID, actor, got))

	// Cross-tenant update must fail
	assert.ErrorIs(t, repo.Update(c, e.tB.ID, actor, got), domainerrors.ErrStatusLogNotFound)

	// Verify updated
	got2, err := repo.GetByID(c, e.tA.ID, entryA.ID)
	require.NoError(t, err)
	assert.Equal(t, 5.0, got2.Hours)
	assert.Equal(t, "Updated notes", got2.Notes)

	// 6. Cross-tenant delete must fail
	assert.ErrorIs(t, repo.SoftDelete(c, e.tB.ID, entryA.ID, actor), domainerrors.ErrStatusLogNotFound)

	// 7. Soft delete in Tenant A
	require.NoError(t, repo.SoftDelete(c, e.tA.ID, entryA.ID, actor))

	// After soft delete, GetByID returns ErrStatusLogNotFound
	_, err = repo.GetByID(c, e.tA.ID, entryA.ID)
	assert.ErrorIs(t, err, domainerrors.ErrStatusLogNotFound)

	// After soft delete, total hours is 0
	totalAfterDelete, err := repo.GetDayTotalHours(c, e.tA.ID, e.uA.ID, today, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, totalAfterDelete)
}

func TestStatusLogRepository_DailySummary(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormStatusLogRepository(e.db)
	actor := uuid.New()
	today := day(2026, 10, 27)

	// Create a status log entry for uA
	cat := entity.ActivityCategoryTraining
	entryA := &entity.StatusLogEntry{
		LogDate:          today,
		ActivityCategory: &cat,
		Hours:            6.0,
		Notes:            "Go training",
		UserID:           e.uA.ID,
	}
	require.NoError(t, repo.Create(c, e.tA.ID, actor, entryA))

	// Query daily summary for Tenant A (uA has entry, another active user in tA would have no entry)
	summaries, err := repo.DailySummary(c, e.tA.ID, today, nil, nil)
	require.NoError(t, err)
	require.NotEmpty(t, summaries)

	var uASummary *entity.DailyEmployeeSummary
	for _, s := range summaries {
		if s.UserID == e.uA.ID {
			uASummary = s
			break
		}
	}
	require.NotNil(t, uASummary)
	assert.True(t, uASummary.Logged)
	assert.Equal(t, 6.0, uASummary.TotalHours)
	require.Len(t, uASummary.HoursByCategory, 1)
	assert.Equal(t, "training", uASummary.HoursByCategory[0].Category)
	assert.Equal(t, 6.0, uASummary.HoursByCategory[0].Hours)
}

func TestStatusLogRepository_MonthlySummary(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormStatusLogRepository(e.db)
	actor := uuid.New()
	cat := entity.ActivityCategoryTraining

	for _, d := range []time.Time{day(2026, 10, 1), day(2026, 10, 27), day(2026, 11, 2)} {
		require.NoError(t, repo.Create(c, e.tA.ID, actor, &entity.StatusLogEntry{
			LogDate: d, ActivityCategory: &cat, Hours: 2, UserID: e.uA.ID,
		}))
	}

	rows, err := repo.MonthlySummary(c, e.tA.ID, day(2026, 10, 1), day(2026, 10, 31), []int{0, 6}, nil, nil)
	require.NoError(t, err)
	var got *entity.MonthlyEmployeeSummary
	for _, s := range rows {
		assert.NotEqual(t, e.uB.ID, s.UserID, "tenant B user must not leak")
		if s.UserID == e.uA.ID {
			got = s
		}
	}
	require.NotNil(t, got)
	assert.Equal(t, 2, got.DaysLogged) // Nov entry excluded
	assert.Equal(t, 4.0, got.TotalHours)
	require.Len(t, got.HoursByCategory, 1)

	// Tenant B sees none of tenant A's rows.
	rowsB, err := repo.MonthlySummary(c, e.tB.ID, day(2026, 10, 1), day(2026, 10, 31), []int{0, 6}, nil, nil)
	require.NoError(t, err)
	for _, s := range rowsB {
		assert.NotEqual(t, e.uA.ID, s.UserID)
		assert.Equal(t, 0, s.DaysLogged)
	}
}

func TestStatusLogRepository_LockUserDayConcurrency(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormStatusLogRepository(e.db)
	tx := database.NewGormTransactor(e.db)
	testDate := day(2026, 10, 27)

	concurrency := 2
	startSig := make(chan struct{})
	errs := make(chan error, concurrency)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startSig
			err := tx.WithinTransaction(c, func(txCtx context.Context) error {
				if err := repo.LockUserDay(txCtx, e.tA.ID, e.uA.ID, testDate); err != nil {
					return err
				}
				total, err := repo.GetDayTotalHours(txCtx, e.tA.ID, e.uA.ID, testDate, uuid.Nil)
				if err != nil {
					return err
				}
				if total+14.0 > 24.0 {
					return domainerrors.ErrDailyHoursExceeded
				}
				cat := entity.ActivityCategorySupport
				return repo.Create(txCtx, e.tA.ID, e.uA.ID, &entity.StatusLogEntry{
					LogDate:          testDate,
					ActivityCategory: &cat,
					Hours:            14.0,
					Notes:            "concurrent insert",
					UserID:           e.uA.ID,
				})
			})
			errs <- err
		}()
	}

	close(startSig)
	wg.Wait()
	close(errs)

	var successCount, exceededCount int
	for err := range errs {
		if err == nil {
			successCount++
		} else if errors.Is(err, domainerrors.ErrDailyHoursExceeded) {
			exceededCount++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	assert.Equal(t, 1, successCount, "exactly one concurrent 14h transaction must succeed")
	assert.Equal(t, 1, exceededCount, "the second concurrent 14h transaction must fail with ErrDailyHoursExceeded")

	finalTotal, err := repo.GetDayTotalHours(c, e.tA.ID, e.uA.ID, testDate, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, 14.0, finalTotal)
}
