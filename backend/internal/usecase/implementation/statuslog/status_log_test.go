package statuslog

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/types/statuslog"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
)

type fakeStatusLogRepo struct {
	entries []*entity.StatusLogEntry
	users   []*entity.User
}

func newFakeStatusLogRepo() *fakeStatusLogRepo {
	return &fakeStatusLogRepo{
		entries: []*entity.StatusLogEntry{},
		users:   []*entity.User{},
	}
}

func (f *fakeStatusLogRepo) LockUserDay(_ context.Context, tenantID, userID uuid.UUID, logDate time.Time) error {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	return nil
}

func (f *fakeStatusLogRepo) Create(_ context.Context, tenantID, actorID uuid.UUID, entry *entity.StatusLogEntry) error {
	entry.TenantID = tenantID
	entry.CreatedBy = &actorID
	entry.UpdatedBy = &actorID
	now := time.Now().UTC()
	entry.CreatedAt, entry.UpdatedAt = now, now
	cp := *entry
	f.entries = append(f.entries, &cp)
	return nil
}

func (f *fakeStatusLogRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.StatusLogEntry, error) {
	for _, e := range f.entries {
		if e.TenantID == tenantID && e.ID == id && e.DeletedAt == nil {
			cp := *e
			return &cp, nil
		}
	}
	return nil, domainerrors.ErrStatusLogNotFound
}

func (f *fakeStatusLogRepo) GetByIDWithProject(ctx context.Context, tenantID, id uuid.UUID) (*entity.StatusLogEntryWithProject, error) {
	e, err := f.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	out := &entity.StatusLogEntryWithProject{StatusLogEntry: *e}
	if e.ProjectID != nil {
		name := "Project " + e.ProjectID.String()[:8]
		code := "PRJ"
		out.ProjectName = &name
		out.ProjectCode = &code
	}
	return out, nil
}

func (f *fakeStatusLogRepo) Update(_ context.Context, tenantID, actorID uuid.UUID, entry *entity.StatusLogEntry) error {
	for i, e := range f.entries {
		if e.TenantID == tenantID && e.ID == entry.ID && e.DeletedAt == nil {
			entry.UpdatedBy = &actorID
			entry.UpdatedAt = time.Now().UTC()
			cp := *entry
			f.entries[i] = &cp
			return nil
		}
	}
	return domainerrors.ErrStatusLogNotFound
}

func (f *fakeStatusLogRepo) SoftDelete(_ context.Context, tenantID, id, actorID uuid.UUID) error {
	for i, e := range f.entries {
		if e.TenantID == tenantID && e.ID == id && e.DeletedAt == nil {
			now := time.Now().UTC()
			f.entries[i].DeletedAt = &now
			f.entries[i].DeletedBy = &actorID
			return nil
		}
	}
	return domainerrors.ErrStatusLogNotFound
}

func (f *fakeStatusLogRepo) GetDayTotalHours(_ context.Context, tenantID, userID uuid.UUID, logDate time.Time, excludeID uuid.UUID) (float64, error) {
	target := DateOnly(logDate)
	var total float64
	for _, e := range f.entries {
		if e.TenantID == tenantID && e.UserID == userID && e.DeletedAt == nil && e.ID != excludeID {
			if DateOnly(e.LogDate).Equal(target) {
				total += e.Hours
			}
		}
	}
	return RoundHours(total), nil
}

func (f *fakeStatusLogRepo) ListByUserAndDateRange(_ context.Context, tenantID, userID uuid.UUID, from, to time.Time, projectID *uuid.UUID) ([]*entity.StatusLogEntryWithProject, error) {
	var out []*entity.StatusLogEntryWithProject
	for _, e := range f.entries {
		if e.TenantID == tenantID && e.UserID == userID && e.DeletedAt == nil {
			d := DateOnly(e.LogDate)
			if (d.Equal(from) || d.After(from)) && (d.Equal(to) || d.Before(to)) {
				if projectID != nil && (e.ProjectID == nil || *e.ProjectID != *projectID) {
					continue
				}
				item := &entity.StatusLogEntryWithProject{StatusLogEntry: *e}
				if e.ProjectID != nil {
					name := "Project " + e.ProjectID.String()[:8]
					code := "PRJ"
					item.ProjectName = &name
					item.ProjectCode = &code
				}
				out = append(out, item)
			}
		}
	}
	return out, nil
}

func (f *fakeStatusLogRepo) MonthlySummary(_ context.Context, tenantID uuid.UUID, from, to time.Time, offDays []int, departmentID, projectID *uuid.UUID) ([]*entity.MonthlyEmployeeSummary, error) {
	var out []*entity.MonthlyEmployeeSummary
	for _, u := range f.users {
		if u.TenantID != tenantID || !u.IsActive || u.DeletedAt != nil {
			continue
		}
		if departmentID != nil && (u.DepartmentID == nil || *u.DepartmentID != *departmentID) {
			continue
		}
		row := &entity.MonthlyEmployeeSummary{
			UserID: u.ID, FirstName: u.FirstName, LastName: u.LastName, Email: u.Email, DepartmentID: u.DepartmentID,
			HoursByProject: []entity.ProjectHoursBreakdown{}, HoursByCategory: []entity.CategoryHoursBreakdown{},
		}
		days := map[string]bool{}
		cats := map[string]float64{}
		for _, e := range f.entries {
			d := DateOnly(e.LogDate)
			if e.TenantID != tenantID || e.UserID != u.ID || e.DeletedAt != nil || d.Before(from) || d.After(to) {
				continue
			}
			days[d.Format("2006-01-02")] = true
			row.TotalHours += e.Hours
			if e.ActivityCategory != nil {
				cats[*e.ActivityCategory] += e.Hours
			}
		}
		row.DaysLogged = len(days)
		for ds := range days {
			dt, _ := time.Parse("2006-01-02", ds)
			row.LoggedDates = append(row.LoggedDates, dt)
			for _, o := range offDays {
				if int(dt.Weekday()) == o {
					row.OffDaysWorked++
				}
			}
		}
		for c, h := range cats {
			row.HoursByCategory = append(row.HoursByCategory, entity.CategoryHoursBreakdown{Category: c, Hours: h})
		}
		out = append(out, row)
	}
	return out, nil
}

func (f *fakeStatusLogRepo) DailySummary(_ context.Context, tenantID uuid.UUID, date time.Time, departmentID, projectID *uuid.UUID) ([]*entity.DailyEmployeeSummary, error) {
	var out []*entity.DailyEmployeeSummary
	target := DateOnly(date)

	for _, u := range f.users {
		if u.TenantID != tenantID || !u.IsActive || u.DeletedAt != nil {
			continue
		}
		if departmentID != nil && (u.DepartmentID == nil || *u.DepartmentID != *departmentID) {
			continue
		}

		var userEntries []*entity.StatusLogEntry
		for _, e := range f.entries {
			if e.TenantID == tenantID && e.UserID == u.ID && e.DeletedAt == nil && DateOnly(e.LogDate).Equal(target) {
				if projectID != nil && (e.ProjectID == nil || *e.ProjectID != *projectID) {
					continue
				}
				userEntries = append(userEntries, e)
			}
		}

		if projectID != nil && len(userEntries) == 0 {
			// In fake, if filtering by project and no entries or allocations, skip
			continue
		}

		logged := len(userEntries) > 0
		var totalHours float64
		projMap := make(map[uuid.UUID]*entity.ProjectHoursBreakdown)
		catMap := make(map[string]float64)

		for _, e := range userEntries {
			totalHours += e.Hours
			if e.ProjectID != nil {
				pid := *e.ProjectID
				if pb, ok := projMap[pid]; ok {
					pb.Hours = RoundHours(pb.Hours + e.Hours)
				} else {
					projMap[pid] = &entity.ProjectHoursBreakdown{
						ProjectID:   pid,
						ProjectName: "Project " + pid.String()[:8],
						ProjectCode: "PRJ",
						Hours:       e.Hours,
					}
				}
			} else if e.ActivityCategory != nil {
				cat := *e.ActivityCategory
				catMap[cat] = RoundHours(catMap[cat] + e.Hours)
			}
		}

		hoursByProject := make([]entity.ProjectHoursBreakdown, 0, len(projMap))
		for _, pb := range projMap {
			hoursByProject = append(hoursByProject, *pb)
		}
		sort.Slice(hoursByProject, func(i, j int) bool {
			if hoursByProject[i].ProjectName != hoursByProject[j].ProjectName {
				return hoursByProject[i].ProjectName < hoursByProject[j].ProjectName
			}
			return hoursByProject[i].ProjectID.String() < hoursByProject[j].ProjectID.String()
		})

		hoursByCategory := make([]entity.CategoryHoursBreakdown, 0, len(catMap))
		for cat, h := range catMap {
			hoursByCategory = append(hoursByCategory, entity.CategoryHoursBreakdown{Category: cat, Hours: h})
		}
		sort.Slice(hoursByCategory, func(i, j int) bool {
			return hoursByCategory[i].Category < hoursByCategory[j].Category
		})

		var deptName *string
		if u.DepartmentID != nil {
			d := "Engineering"
			deptName = &d
		}

		out = append(out, &entity.DailyEmployeeSummary{
			UserID:          u.ID,
			FirstName:       u.FirstName,
			LastName:        u.LastName,
			Email:           u.Email,
			DepartmentID:    u.DepartmentID,
			DepartmentName:  deptName,
			Logged:          logged,
			TotalHours:      RoundHours(totalHours),
			HoursByProject:  hoursByProject,
			HoursByCategory: hoursByCategory,
		})
	}
	return out, nil
}

type fakeAllocations struct {
	allocs []*entity.AllocationWithProject
}

func (f *fakeAllocations) Create(_ context.Context, _, _ uuid.UUID, _ *entity.ProjectAllocation) error {
	return nil
}
func (f *fakeAllocations) GetByID(_ context.Context, _, _ uuid.UUID) (*entity.ProjectAllocation, error) {
	return nil, nil
}
func (f *fakeAllocations) ListByProject(_ context.Context, _, _ uuid.UUID, _ entity.AllocationScope, _ time.Time) ([]*entity.AllocationWithUser, error) {
	return nil, nil
}
func (f *fakeAllocations) ListActiveByUser(_ context.Context, tenantID, userID uuid.UUID, asOf time.Time) ([]*entity.AllocationWithProject, error) {
	var out []*entity.AllocationWithProject
	for _, a := range f.allocs {
		if a.TenantID == tenantID && a.UserID == userID && a.ActiveOn(asOf) {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *fakeAllocations) ListByUserScope(_ context.Context, _, _ uuid.UUID, _ entity.AllocationScope, _ time.Time) ([]*entity.AllocationWithProject, error) {
	return nil, nil
}
func (f *fakeAllocations) CountByUserScopes(_ context.Context, _, _ uuid.UUID, _ time.Time) (entity.AllocationScopeCounts, error) {
	return entity.AllocationScopeCounts{}, nil
}
func (f *fakeAllocations) HasOverlap(_ context.Context, _, _, _ uuid.UUID, _ time.Time, _ *time.Time, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakeAllocations) Update(_ context.Context, _, _ uuid.UUID, _ *entity.ProjectAllocation) error {
	return nil
}
func (f *fakeAllocations) SoftDelete(_ context.Context, _, _, _ uuid.UUID) error {
	return nil
}

type fakeUsers struct {
	users map[uuid.UUID]*entity.User
}

func (f *fakeUsers) Create(_ context.Context, u *entity.User) error {
	f.users[u.ID] = u
	return nil
}
func (f *fakeUsers) GetByID(_ context.Context, tenantID, id uuid.UUID) (*entity.User, error) {
	u, ok := f.users[id]
	if !ok || u.TenantID != tenantID {
		return nil, domainerrors.ErrUserNotFound
	}
	return u, nil
}
func (f *fakeUsers) GetByTenantAndEmail(_ context.Context, tenantID uuid.UUID, email string) (*entity.User, error) {
	for _, u := range f.users {
		if u.TenantID == tenantID && u.Email == email {
			return u, nil
		}
	}
	return nil, domainerrors.ErrUserNotFound
}
func (f *fakeUsers) GetByIDWithRoles(c context.Context, t, id uuid.UUID) (*entity.User, error) {
	return f.GetByID(c, t, id)
}
func (f *fakeUsers) GetByTenantAndEmailWithRoles(c context.Context, t uuid.UUID, email string) (*entity.User, error) {
	return f.GetByTenantAndEmail(c, t, email)
}
func (f *fakeUsers) Update(_ context.Context, tenantID uuid.UUID, user *entity.User) error {
	f.users[user.ID] = user
	return nil
}
func (f *fakeUsers) Delete(_ context.Context, tenantID, id, actorID uuid.UUID) error {
	delete(f.users, id)
	return nil
}
func (f *fakeUsers) List(_ context.Context, tenantID uuid.UUID, limit, offset int) ([]*entity.User, int64, error) {
	var out []*entity.User
	for _, u := range f.users {
		if u.TenantID == tenantID {
			out = append(out, u)
		}
	}
	return out, int64(len(out)), nil
}

type testEnv struct {
	tenantID uuid.UUID
	user1    uuid.UUID
	user2    uuid.UUID
	project1 uuid.UUID
	today    time.Time
	repo     *fakeStatusLogRepo
	allocs   *fakeAllocations
	users    *fakeUsers
	clock    Clock
}

func newTestEnv() *testEnv {
	tenantID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	project1 := uuid.New()
	today := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

	repo := newFakeStatusLogRepo()
	allocs := &fakeAllocations{
		allocs: []*entity.AllocationWithProject{
			{
				ProjectAllocation: entity.ProjectAllocation{
					ID:        uuid.New(),
					TenantID:  tenantID,
					ProjectID: project1,
					UserID:    user1,
					StartDate: today.AddDate(0, 0, -30),
					EndDate:   nil,
				},
				ProjectName:   "Alpha",
				ProjectCode:   "ALP",
				ProjectStatus: "active",
			},
		},
	}
	users := &fakeUsers{
		users: map[uuid.UUID]*entity.User{
			user1: {ID: user1, TenantID: tenantID, FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", IsActive: true},
			user2: {ID: user2, TenantID: tenantID, FirstName: "Bob", LastName: "Jones", Email: "bob@example.com", IsActive: true},
		},
	}
	repo.users = []*entity.User{users.users[user1], users.users[user2]}

	return &testEnv{
		tenantID: tenantID,
		user1:    user1,
		user2:    user2,
		project1: project1,
		today:    today,
		repo:     repo,
		allocs:   allocs,
		users:    users,
		clock:    func() time.Time { return today },
	}
}

// Tests: window boundaries (today, today−6 accepted; today−7 and tomorrow rejected)
func TestStatusLog_WindowBoundaries(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	cat := "meeting"

	// 1. Today accepted
	resToday, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            2.0,
		Notes:            "Today log",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-27", resToday.LogDate.UTC().Format("2006-01-02"))

	// 2. Today - 6 accepted
	resT6, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-21",
		ActivityCategory: &cat,
		Hours:            3.0,
		Notes:            "Today - 6 log",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-21", resT6.LogDate.UTC().Format("2006-01-02"))

	// 3. Today - 7 rejected with ErrLogWindowClosed
	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-20",
		ActivityCategory: &cat,
		Hours:            2.0,
		Notes:            "Too old",
	})
	require.ErrorIs(t, err, domainerrors.ErrLogWindowClosed)

	// 4. Tomorrow rejected with ErrFutureDate
	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-28",
		ActivityCategory: &cat,
		Hours:            2.0,
		Notes:            "Tomorrow",
	})
	require.ErrorIs(t, err, domainerrors.ErrFutureDate)

	// Update / Delete outside window
	// Put an entry directly into repo with log_date today - 7
	oldDate := e.today.AddDate(0, 0, -7)
	oldEntry := &entity.StatusLogEntry{
		ID:               uuid.New(),
		TenantID:         e.tenantID,
		UserID:           e.user1,
		LogDate:          oldDate,
		ActivityCategory: &cat,
		Hours:            4.0,
	}
	require.NoError(t, e.repo.Create(ctx, e.tenantID, e.user1, oldEntry))

	updateUC := NewUpdateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	newNotes := "try update"
	_, err = updateUC.Execute(ctx, e.tenantID, e.user1, oldEntry.ID, statuslog.UpdateStatusLogRequest{
		Notes: &newNotes,
	})
	require.ErrorIs(t, err, domainerrors.ErrLogWindowClosed)

	deleteUC := NewDeleteStatusLogUseCase(e.repo, transactor, e.clock)
	err = deleteUC.Execute(ctx, e.tenantID, e.user1, oldEntry.ID)
	require.ErrorIs(t, err, domainerrors.ErrLogWindowClosed)

	// Trying to update a valid entry to a future date
	futureDateStr := "2026-10-28"
	_, err = updateUC.Execute(ctx, e.tenantID, e.user1, resToday.ID, statuslog.UpdateStatusLogRequest{
		LogDate: &futureDateStr,
	})
	require.ErrorIs(t, err, domainerrors.ErrFutureDate)
}

// Tests: 24h cap
func TestStatusLog_DailyHours24Cap(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	cat := "meeting"

	// 1. First entry: 14 hours -> OK
	r1, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            14.0,
		Notes:            "14h meeting",
	})
	require.NoError(t, err)
	assert.Equal(t, 14.0, r1.Hours)

	// 2. Second entry on same day: 10 hours -> OK (14 + 10 = 24h)
	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            10.0,
		Notes:            "10h meeting",
	})
	require.NoError(t, err)

	// 3. Third entry on same day: 0.5 hours -> rejected with ErrDailyHoursExceeded (24.5 > 24)
	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            0.5,
		Notes:            "exceeds cap",
	})
	require.ErrorIs(t, err, domainerrors.ErrDailyHoursExceeded)

	// 4. Update entry 1 from 14h to 14.5h -> rejected with ErrDailyHoursExceeded
	updateUC := NewUpdateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	newHours := 14.5
	_, err = updateUC.Execute(ctx, e.tenantID, e.user1, r1.ID, statuslog.UpdateStatusLogRequest{
		Hours: &newHours,
	})
	require.ErrorIs(t, err, domainerrors.ErrDailyHoursExceeded)
}

// Tests: one-of project/category
func TestStatusLog_OneOfProjectOrCategory(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	cat := "meeting"

	// 1. Both project and category -> rejected
	_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ProjectID:        &e.project1,
		ActivityCategory: &cat,
		Hours:            4.0,
	})
	require.ErrorIs(t, err, domainerrors.ErrProjectOrCategoryRequired)

	// 2. Neither project nor category -> rejected
	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate: "2026-10-27",
		Hours:   4.0,
	})
	require.ErrorIs(t, err, domainerrors.ErrProjectOrCategoryRequired)

	// 3. Invalid category -> rejected
	invalidCat := "gaming"
	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &invalidCat,
		Hours:            4.0,
	})
	require.ErrorIs(t, err, domainerrors.ErrInvalidActivityCategory)

	// 4. Project only (user1 is allocated to project1) -> OK
	resProj, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:   "2026-10-27",
		ProjectID: &e.project1,
		Hours:     4.0,
	})
	require.NoError(t, err)
	assert.Equal(t, &e.project1, resProj.ProjectID)
	assert.Nil(t, resProj.ActivityCategory)

	// 5. Category only -> OK
	supportCat := "support"
	resCat, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-26",
		ActivityCategory: &supportCat,
		Hours:            3.0,
	})
	require.NoError(t, err)
	assert.Equal(t, &supportCat, resCat.ActivityCategory)
	assert.Nil(t, resCat.ProjectID)
}

// Tests: project entries require an allocation to that project active on log_date
func TestStatusLog_ProjectAllocationRequired(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)

	// user2 is not allocated to project1
	_, err := createUC.Execute(ctx, e.tenantID, e.user2, statuslog.CreateStatusLogRequest{
		LogDate:   "2026-10-27",
		ProjectID: &e.project1,
		Hours:     4.0,
	})
	require.ErrorIs(t, err, domainerrors.ErrNoActiveAllocation)

	// Unknown project for user1
	unknownProj := uuid.New()
	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:   "2026-10-27",
		ProjectID: &unknownProj,
		Hours:     4.0,
	})
	require.ErrorIs(t, err, domainerrors.ErrNoActiveAllocation)
}

// Tests: cannot edit another user's entry
func TestStatusLog_CannotEditAnotherUserEntry(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	cat := "training"

	// user1 creates an entry
	entry, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            2.0,
		Notes:            "user1 training",
	})
	require.NoError(t, err)

	// user2 attempts to update user1's entry -> not found (prevents id leakage)
	updateUC := NewUpdateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	newNotes := "hacked notes"
	_, err = updateUC.Execute(ctx, e.tenantID, e.user2, entry.ID, statuslog.UpdateStatusLogRequest{
		Notes: &newNotes,
	})
	require.ErrorIs(t, err, domainerrors.ErrStatusLogNotFound)

	// user2 attempts to delete user1's entry -> not found (prevents id leakage)
	deleteUC := NewDeleteStatusLogUseCase(e.repo, transactor, e.clock)
	err = deleteUC.Execute(ctx, e.tenantID, e.user2, entry.ID)
	require.ErrorIs(t, err, domainerrors.ErrStatusLogNotFound)

	// user1 can update own entry
	updated, err := updateUC.Execute(ctx, e.tenantID, e.user1, entry.ID, statuslog.UpdateStatusLogRequest{
		Notes: &newNotes,
	})
	require.NoError(t, err)
	assert.Equal(t, "hacked notes", updated.Notes)

	// user1 can delete own entry
	err = deleteUC.Execute(ctx, e.tenantID, e.user1, entry.ID)
	require.NoError(t, err)
}

// Tests: daily view lists non-loggers
func TestStatusLog_DailyViewListsNonLoggers(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	cat := "meeting"

	// Only user1 logs hours on 2026-10-27
	_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            5.5,
		Notes:            "Sync",
	})
	require.NoError(t, err)

	dailyUC := NewGetDailyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{}, &fakeHolidayRange{}, e.clock)
	summaries, err := dailyUC.Execute(ctx, e.tenantID, "2026-10-27", "", "")
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	var u1Sum, u2Sum *entity.DailyEmployeeSummary
	for _, s := range summaries {
		if s.UserID == e.user1 {
			u1Sum = s
		} else if s.UserID == e.user2 {
			u2Sum = s
		}
	}

	require.NotNil(t, u1Sum)
	require.NotNil(t, u2Sum)

	// user1 logged
	assert.True(t, u1Sum.Logged)
	assert.Equal(t, 5.5, u1Sum.TotalHours)
	assert.Len(t, u1Sum.HoursByCategory, 1)

	// user2 non-logger: listed with logged=false and total_hours=0
	assert.False(t, u2Sum.Logged)
	assert.Equal(t, 0.0, u2Sum.TotalHours)
	assert.Empty(t, u2Sum.HoursByProject)
	assert.Empty(t, u2Sum.HoursByCategory)
}

func TestStatusLog_ListMyGrouped(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	cat := "meeting"

	// Create 2 entries on 2026-10-27 and 1 on 2026-10-26
	_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-27",
		ActivityCategory: &cat,
		Hours:            3.0,
	})
	require.NoError(t, err)

	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:   "2026-10-27",
		ProjectID: &e.project1,
		Hours:     4.5,
	})
	require.NoError(t, err)

	_, err = createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:          "2026-10-26",
		ActivityCategory: &cat,
		Hours:            2.0,
	})
	require.NoError(t, err)

	listUC := NewListMyStatusLogsUseCase(e.repo, e.clock)
	groups, err := listUC.Execute(ctx, e.tenantID, e.user1, "2026-10-21", "2026-10-27", "")
	require.NoError(t, err)
	require.Len(t, groups, 2)

	// Newest first: 2026-10-27 has 2 entries and 7.5h total
	assert.Equal(t, "2026-10-27", groups[0].Date.UTC().Format("2006-01-02"))
	assert.Equal(t, 7.5, groups[0].TotalHours)
	assert.Len(t, groups[0].Entries, 2)

	// 2026-10-26 has 1 entry and 2.0h total
	assert.Equal(t, "2026-10-26", groups[1].Date.UTC().Format("2006-01-02"))
	assert.Equal(t, 2.0, groups[1].TotalHours)
	assert.Len(t, groups[1].Entries, 1)
}

func TestStatusLogUseCase_DateRangeSpanCap(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()

	listMyUC := NewListMyStatusLogsUseCase(e.repo, e.clock)
	listEmpUC := NewListEmployeeStatusLogsUseCase(e.repo, e.users, e.clock)

	// 1. Span > 366 days in list_my -> ErrStatusLogDateRangeExceeded
	_, err := listMyUC.Execute(ctx, e.tenantID, e.user1, "2024-01-01", "2025-01-03", "")
	require.ErrorIs(t, err, domainerrors.ErrStatusLogDateRangeExceeded)

	// Unbounded range like 0001-01-01 to 9999-12-31 -> ErrStatusLogDateRangeExceeded
	_, err = listMyUC.Execute(ctx, e.tenantID, e.user1, "0001-01-01", "9999-12-31", "")
	require.ErrorIs(t, err, domainerrors.ErrStatusLogDateRangeExceeded)

	// Span <= 366 days -> OK
	_, err = listMyUC.Execute(ctx, e.tenantID, e.user1, "2024-01-01", "2024-12-31", "")
	require.NoError(t, err)

	// 2. Span > 366 days in list_employee -> ErrStatusLogDateRangeExceeded
	_, err = listEmpUC.Execute(ctx, e.tenantID, e.user1, "2024-01-01", "2025-01-03", "")
	require.ErrorIs(t, err, domainerrors.ErrStatusLogDateRangeExceeded)

	// Span <= 366 days in list_employee -> OK
	_, err = listEmpUC.Execute(ctx, e.tenantID, e.user1, "2024-01-01", "2024-12-31", "")
	require.NoError(t, err)
}

func TestStatusLogUseCase_PatchNilUUIDProjectID(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	transactor := ucshared.NewNopTransactor()

	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)
	updateUC := NewUpdateStatusLogUseCase(e.repo, e.allocs, transactor, e.clock)

	// Create project-based status log entry
	entry, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{
		LogDate:   "2026-10-27",
		ProjectID: &e.project1,
		Hours:     3.0,
		Notes:     "initial project log",
	})
	require.NoError(t, err)
	require.NotNil(t, entry.ProjectID)

	// 1. Sending nil UUID project_id alone -> rejected with ErrProjectOrCategoryRequired
	nilUUID := uuid.Nil
	_, err = updateUC.Execute(ctx, e.tenantID, e.user1, entry.ID, statuslog.UpdateStatusLogRequest{
		ProjectID: &nilUUID,
	})
	require.ErrorIs(t, err, domainerrors.ErrProjectOrCategoryRequired)

	// 2. Sending empty string activity_category alone -> rejected with ErrProjectOrCategoryRequired
	emptyCat := "   "
	_, err = updateUC.Execute(ctx, e.tenantID, e.user1, entry.ID, statuslog.UpdateStatusLogRequest{
		ActivityCategory: &emptyCat,
	})
	require.ErrorIs(t, err, domainerrors.ErrProjectOrCategoryRequired)

	// 3. Sending nil UUID project_id WITH valid category -> switches to category successfully
	cat := "training"
	updated, err := updateUC.Execute(ctx, e.tenantID, e.user1, entry.ID, statuslog.UpdateStatusLogRequest{
		ProjectID:        &nilUUID,
		ActivityCategory: &cat,
	})
	require.NoError(t, err)
	assert.Nil(t, updated.ProjectID)
	require.NotNil(t, updated.ActivityCategory)
	assert.Equal(t, "training", *updated.ActivityCategory)

	// 4. Switching back to project by sending ProjectID alone
	updated2, err := updateUC.Execute(ctx, e.tenantID, e.user1, entry.ID, statuslog.UpdateStatusLogRequest{
		ProjectID: &e.project1,
	})
	require.NoError(t, err)
	require.NotNil(t, updated2.ProjectID)
	assert.Equal(t, e.project1, *updated2.ProjectID)
	assert.Nil(t, updated2.ActivityCategory)
}

func TestStatusLogUseCase_DailySummary_DeterministicSorting(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()

	d27 := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

	// Seed multiple entries with different categories for user1 on same day
	e.repo.entries = []*entity.StatusLogEntry{
		{
			ID:               uuid.New(),
			TenantID:         e.tenantID,
			UserID:           e.user1,
			LogDate:          d27,
			ActivityCategory: strPtr(entity.ActivityCategorySupport),
			Hours:            2.0,
		},
		{
			ID:               uuid.New(),
			TenantID:         e.tenantID,
			UserID:           e.user1,
			LogDate:          d27,
			ActivityCategory: strPtr(entity.ActivityCategoryInternal),
			Hours:            1.5,
		},
		{
			ID:               uuid.New(),
			TenantID:         e.tenantID,
			UserID:           e.user1,
			LogDate:          d27,
			ActivityCategory: strPtr(entity.ActivityCategoryMeeting),
			Hours:            1.0,
		},
	}

	dailyUC := NewGetDailyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{}, &fakeHolidayRange{}, e.clock)
	summaries, err := dailyUC.Execute(ctx, e.tenantID, "2026-10-27", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, summaries)

	var u1Summary *entity.DailyEmployeeSummary
	for _, s := range summaries {
		if s.UserID == e.user1 {
			u1Summary = s
			break
		}
	}
	require.NotNil(t, u1Summary)
	require.Len(t, u1Summary.HoursByCategory, 3)

	// Check alphabetical ordering of categories: internal, meeting, support
	assert.Equal(t, "internal", u1Summary.HoursByCategory[0].Category)
	assert.Equal(t, "meeting", u1Summary.HoursByCategory[1].Category)
	assert.Equal(t, "support", u1Summary.HoursByCategory[2].Category)
}

func strPtr(s string) *string {
	return &s
}

func TestStatusLog_MonthlyReport(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, ucshared.NewNopTransactor(), e.clock)
	cat := "meeting"
	for _, d := range []string{"2026-10-27", "2026-10-26"} {
		_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{LogDate: d, ActivityCategory: &cat, Hours: 4, Notes: "x"})
		require.NoError(t, err)
	}

	uc := NewGetMonthlyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{}, &fakeHolidayRange{}, e.clock)
	rep, err := uc.Execute(ctx, e.tenantID, "2026-10", "", "")
	require.NoError(t, err)
	assert.Equal(t, "2026-10", rep.Month)
	assert.Equal(t, 22, rep.WorkingDays)
	assert.Equal(t, 2, rep.TotalEmployees)
	assert.Equal(t, 1, rep.LoggedEmployees)
	// user1 logged 2 of 22 expected days, so both employees have missing days.
	assert.Equal(t, 2, rep.NotLoggedEmployees)
	assert.Equal(t, 8.0, rep.TotalHours)
	for _, s := range rep.Employees {
		if s.UserID == e.user2 {
			assert.Equal(t, 0, s.DaysLogged)
			assert.Equal(t, 0.0, s.TotalHours)
		} else {
			assert.Equal(t, 2, s.DaysLogged)
		}
	}

	// Other month: everyone appears with zeros.
	rep, err = uc.Execute(ctx, e.tenantID, "2026-02", "", "")
	require.NoError(t, err)
	assert.Equal(t, 20, rep.WorkingDays)
	assert.Equal(t, 2, rep.NotLoggedEmployees)

	_, err = uc.Execute(ctx, e.tenantID, "2026-13", "", "")
	require.ErrorIs(t, err, domainerrors.ErrInvalidStatusLogDate)
	_, err = uc.Execute(ctx, e.tenantID, "2026-10", "nope", "")
	require.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	_, err = uc.Execute(ctx, uuid.Nil, "2026-10", "", "")
	require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

type fakeOffDays struct{ days []int }

func (f fakeOffDays) WeeklyOffDays(context.Context, uuid.UUID) ([]int, error) { return f.days, nil }

func TestStatusLog_MonthlyWorkingDaysAndOffDaysWorked(t *testing.T) {
	// Oct 2026: 31 days, starts Thursday. Sundays=4, Mondays=4, Saturdays=5.
	e := newTestEnv()
	ctx := context.Background()
	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, ucshared.NewNopTransactor(), e.clock)
	cat := "meeting"
	// 2026-10-26 is Monday, 2026-10-25 Sunday, 2026-10-24 Saturday.
	for _, d := range []string{"2026-10-26", "2026-10-25", "2026-10-24"} {
		_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{LogDate: d, ActivityCategory: &cat, Hours: 2, Notes: "x"})
		require.NoError(t, err)
	}
	cases := []struct {
		name    string
		off     []int
		working int
		worked  int
	}{
		{"sun+sat", []int{0, 6}, 22, 2},
		{"sun+mon", []int{0, 1}, 23, 2},
		{"nil falls back to default", nil, 22, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rep, err := NewGetMonthlyStatusLogsUseCase(e.repo, fakeOffDays{c.off}, &fakeLeaveList{}, &fakeHolidayRange{}, e.clock).Execute(ctx, e.tenantID, "2026-10", "", "")
			require.NoError(t, err)
			assert.Equal(t, c.working, rep.WorkingDays)
			for _, s := range rep.Employees {
				if s.UserID == e.user1 {
					assert.Equal(t, 3, s.DaysLogged)
					assert.Equal(t, c.worked, s.OffDaysWorked)
					assert.Equal(t, 6.0, s.TotalHours)
				} else {
					assert.Equal(t, 0, s.OffDaysWorked)
				}
			}
		})
	}
}

type fakeLeaveList struct {
	repository.LeaveRequestRepository
	reqs []*entity.LeaveRequestDetail
}

// filterLeave mimics the repository query: tenant, status and overlap with [From, To].
func filterLeave(reqs []*entity.LeaveRequestDetail, tenantID uuid.UUID, f repository.LeaveRequestFilter) []*entity.LeaveRequestDetail {
	var out []*entity.LeaveRequestDetail
	for _, r := range reqs {
		if r.TenantID != tenantID || (f.Status != "" && r.Status != f.Status) {
			continue
		}
		if (f.From != nil && r.EndDate.Before(*f.From)) || (f.To != nil && r.StartDate.After(*f.To)) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (f *fakeLeaveList) List(_ context.Context, tenantID uuid.UUID, flt repository.LeaveRequestFilter, _, _ int) ([]*entity.LeaveRequestDetail, int64, error) {
	out := filterLeave(f.reqs, tenantID, flt)
	return out, int64(len(out)), nil
}

type fakeHolidayRange struct {
	repository.HolidayRepository
	hols []*entity.Holiday
}

func (f *fakeHolidayRange) ListByDateRange(context.Context, uuid.UUID, time.Time, time.Time) ([]*entity.Holiday, error) {
	return f.hols, nil
}

func TestStatusLog_MonthlyLeaveDays(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	req := func(tenant, user uuid.UUID, status, s, en, part string) *entity.LeaveRequestDetail {
		return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{TenantID: tenant, UserID: user, Status: status, StartDate: d(s), EndDate: d(en), DayPart: part, Days: 99}}
	}
	hol := func(date string, public bool) *entity.Holiday {
		return &entity.Holiday{Date: d(date), CategoryIsPublic: public}
	}
	other := uuid.New()
	cases := []struct {
		name string
		reqs []*entity.LeaveRequestDetail
		hols []*entity.Holiday
		want float64
	}{
		{"fully in month", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-10-05", "2026-10-07", "full")}, nil, 3},
		{"straddles month end clipped", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-10-29", "2026-11-05", "full")}, nil, 2},
		{"straddles month start clipped", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-09-28", "2026-10-02", "full")}, nil, 2},
		{"outside month", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-11-02", "2026-11-03", "full")}, nil, 0},
		{"weekend excluded", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-10-09", "2026-10-12", "full")}, nil, 2},
		{"public holiday excluded", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-10-05", "2026-10-07", "full")}, []*entity.Holiday{hol("2026-10-06", true)}, 2},
		{"optional holiday counted", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-10-05", "2026-10-07", "full")}, []*entity.Holiday{hol("2026-10-06", false)}, 3},
		{"half day", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-10-06", "2026-10-06", "first_half")}, nil, 0.5},
		{"half day on weekend", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "approved", "2026-10-10", "2026-10-10", "first_half")}, nil, 0},
		{"pending and rejected ignored", []*entity.LeaveRequestDetail{req(e.tenantID, e.user1, "pending", "2026-10-05", "2026-10-07", "full"), req(e.tenantID, e.user1, "rejected", "2026-10-05", "2026-10-07", "full"), req(e.tenantID, e.user1, "cancelled", "2026-10-05", "2026-10-07", "full")}, nil, 0},
		{"other tenant ignored", []*entity.LeaveRequestDetail{req(other, e.user1, "approved", "2026-10-05", "2026-10-07", "full")}, nil, 0},
		{"other employee not attributed", []*entity.LeaveRequestDetail{req(e.tenantID, e.user2, "approved", "2026-10-05", "2026-10-07", "full")}, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			uc := NewGetMonthlyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{reqs: c.reqs}, &fakeHolidayRange{hols: c.hols}, e.clock)
			rep, err := uc.Execute(ctx, e.tenantID, "2026-10", "", "")
			require.NoError(t, err)
			for _, s := range rep.Employees {
				if s.UserID == e.user1 {
					assert.Equal(t, c.want, s.LeaveDays)
				}
			}
			wantWD := 22
			for _, h := range c.hols {
				if h.CategoryIsPublic {
					wantWD--
				}
			}
			assert.Equal(t, wantWD, rep.WorkingDays)
		})
	}
}

func TestStatusLog_MonthlyExpectedAndMissingDays(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, ucshared.NewNopTransactor(), e.clock)
	cat := "meeting"
	// user1 logs 2 days; user2 logs none but is on approved leave for 3 working days.
	for _, day := range []string{"2026-10-26", "2026-10-27"} {
		_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{LogDate: day, ActivityCategory: &cat, Hours: 4, Notes: "x"})
		require.NoError(t, err)
	}
	reqs := []*entity.LeaveRequestDetail{{LeaveRequest: entity.LeaveRequest{TenantID: e.tenantID, UserID: e.user2, Status: "approved", StartDate: d("2026-10-05"), EndDate: d("2026-10-07"), DayPart: "full"}}}
	hols := []*entity.Holiday{{Date: d("2026-10-02"), CategoryIsPublic: true}, {Date: d("2026-10-09"), CategoryIsPublic: false}}
	uc := NewGetMonthlyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{reqs: reqs}, &fakeHolidayRange{hols: hols}, e.clock)
	rep, err := uc.Execute(ctx, e.tenantID, "2026-10", "", "")
	require.NoError(t, err)
	assert.Equal(t, 21, rep.WorkingDays)
	for _, s := range rep.Employees {
		switch s.UserID {
		case e.user1:
			assert.Equal(t, 21.0, s.ExpectedDays)
			assert.Equal(t, 19.0, s.MissingDays)
		case e.user2:
			assert.Equal(t, 18.0, s.ExpectedDays)
			assert.Equal(t, 18.0, s.MissingDays)
		}
	}
	assert.Equal(t, 1, rep.LoggedEmployees)
	assert.Equal(t, 2, rep.NotLoggedEmployees)
}

func TestStatusLog_MonthlyNotLoggedZeroWhenFullyCovered(t *testing.T) {
	e := newTestEnv()
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	// Whole month approved as leave for user2, and user1 on leave too: expected 0, none missing.
	reqs := []*entity.LeaveRequestDetail{
		{LeaveRequest: entity.LeaveRequest{TenantID: e.tenantID, UserID: e.user1, Status: "approved", StartDate: d("2026-10-01"), EndDate: d("2026-10-31"), DayPart: "full"}},
		{LeaveRequest: entity.LeaveRequest{TenantID: e.tenantID, UserID: e.user2, Status: "approved", StartDate: d("2026-10-01"), EndDate: d("2026-10-31"), DayPart: "full"}},
	}
	uc := NewGetMonthlyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{reqs: reqs}, &fakeHolidayRange{}, e.clock)
	rep, err := uc.Execute(context.Background(), e.tenantID, "2026-10", "", "")
	require.NoError(t, err)
	assert.Equal(t, 0, rep.NotLoggedEmployees)
	for _, s := range rep.Employees {
		assert.Equal(t, 0.0, s.ExpectedDays)
		assert.Equal(t, 0.0, s.MissingDays)
	}
}

func TestStatusLog_DailyDayStatus(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	lr := func(e *testEnv, user uuid.UUID, status, part string) *entity.LeaveRequestDetail {
		return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{TenantID: e.tenantID, UserID: user, Status: status, StartDate: d("2026-10-26"), EndDate: d("2026-10-28"), DayPart: part}}
	}
	cases := []struct {
		name     string
		date     string // Mon 2026-10-26, Sat 2026-10-24
		reqs     func(e *testEnv) []*entity.LeaveRequestDetail
		hols     []*entity.Holiday
		logUser1 bool
		want1    string
		half1    bool
		want2    string
	}{
		{name: "missing", date: "2026-10-26", want1: "missing", want2: "missing"},
		{name: "logged", date: "2026-10-26", logUser1: true, want1: "logged", want2: "missing"},
		{name: "approved full leave", date: "2026-10-27", reqs: func(e *testEnv) []*entity.LeaveRequestDetail {
			return []*entity.LeaveRequestDetail{lr(e, e.user1, "approved", "full")}
		}, want1: "on_leave", want2: "missing"},
		{name: "approved half leave", date: "2026-10-27", reqs: func(e *testEnv) []*entity.LeaveRequestDetail {
			return []*entity.LeaveRequestDetail{lr(e, e.user1, "approved", "first_half")}
		}, want1: "missing", half1: true, want2: "missing"},
		{name: "approved half leave with a log is logged", date: "2026-10-27", logUser1: true, reqs: func(e *testEnv) []*entity.LeaveRequestDetail {
			return []*entity.LeaveRequestDetail{lr(e, e.user1, "approved", "second_half")}
		}, want1: "logged", want2: "missing"},
		{name: "pending and rejected do not excuse", date: "2026-10-27", reqs: func(e *testEnv) []*entity.LeaveRequestDetail {
			return []*entity.LeaveRequestDetail{lr(e, e.user1, "pending", "full"), lr(e, e.user1, "rejected", "full")}
		}, want1: "missing", want2: "missing"},
		{name: "log on leave day is logged", date: "2026-10-27", logUser1: true, reqs: func(e *testEnv) []*entity.LeaveRequestDetail {
			return []*entity.LeaveRequestDetail{lr(e, e.user1, "approved", "full")}
		}, want1: "logged", want2: "missing"},
		{name: "public holiday", date: "2026-10-27", hols: []*entity.Holiday{{Date: d("2026-10-27"), CategoryIsPublic: true}}, want1: "holiday", want2: "holiday"},
		{name: "optional holiday is not a holiday", date: "2026-10-27", hols: []*entity.Holiday{{Date: d("2026-10-27"), CategoryIsPublic: false}}, want1: "missing", want2: "missing"},
		{name: "weekly off", date: "2026-10-24", want1: "weekly_off", want2: "weekly_off"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newTestEnv()
			ctx := context.Background()
			if c.logUser1 {
				createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, ucshared.NewNopTransactor(), e.clock)
				cat := "meeting"
				_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{LogDate: c.date, ActivityCategory: &cat, Hours: 4, Notes: "x"})
				require.NoError(t, err)
			}
			var reqs []*entity.LeaveRequestDetail
			if c.reqs != nil {
				reqs = c.reqs(e)
			}
			uc := NewGetDailyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{reqs: reqs}, &fakeHolidayRange{hols: c.hols}, e.clock)
			rows, err := uc.Execute(ctx, e.tenantID, c.date, "", "")
			require.NoError(t, err)
			for _, r := range rows {
				if r.UserID == e.user1 {
					assert.Equal(t, c.want1, r.DayStatus)
					assert.Equal(t, c.half1, r.HalfDay)
				} else if r.UserID == e.user2 {
					assert.Equal(t, c.want2, r.DayStatus)
				}
			}
		})
	}
}

type fakeMyLeave struct {
	repository.LeaveRequestRepository
	reqs []*entity.LeaveRequestDetail
}

func (f *fakeMyLeave) ListByUser(_ context.Context, tenantID, userID uuid.UUID, flt repository.LeaveRequestFilter, _, _ int) ([]*entity.LeaveRequestDetail, int64, error) {
	var out []*entity.LeaveRequestDetail
	for _, r := range filterLeave(f.reqs, tenantID, flt) {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, int64(len(out)), nil
}

func TestStatusLog_ListMyDayStatuses(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	e := newTestEnv() // today = Tue 2026-10-27
	ctx := context.Background()
	createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, ucshared.NewNopTransactor(), e.clock)
	cat := "meeting"
	_, err := createUC.Execute(ctx, e.tenantID, e.user1, statuslog.CreateStatusLogRequest{LogDate: "2026-10-26", ActivityCategory: &cat, Hours: 4, Notes: "x"})
	require.NoError(t, err)

	leave := func(status, start, end, part, name string) *entity.LeaveRequestDetail {
		return &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{TenantID: e.tenantID, UserID: e.user1, Status: status, StartDate: d(start), EndDate: d(end), DayPart: part}, LeaveTypeName: name}
	}
	uc := NewListMyDayStatusesUseCase(e.repo, fakeOffDays{},
		&fakeMyLeave{reqs: []*entity.LeaveRequestDetail{
			leave("approved", "2026-10-22", "2026-10-22", "first_half", "Casual"),
			leave("approved", "2026-10-23", "2026-10-26", "full", "Annual"),
			leave("pending", "2026-10-27", "2026-10-27", "full", "Annual"),
		}},
		&fakeHolidayRange{hols: []*entity.Holiday{
			{Name: "Diwali", Date: d("2026-10-21"), CategoryIsPublic: true},
			{Name: "Optional", Date: d("2026-10-27"), CategoryIsPublic: false},
		}}, e.clock)

	// to is in the future: days after today are omitted.
	rows, err := uc.Execute(ctx, e.tenantID, e.user1, "2026-10-21", "2026-10-30")
	require.NoError(t, err)
	require.Len(t, rows, 7) // 21..27
	by := map[string]*entity.MyDayStatus{}
	for _, r := range rows {
		by[r.Date.Format("2006-01-02")] = r
	}
	assert.Equal(t, "holiday", by["2026-10-21"].DayStatus)
	assert.Equal(t, "Diwali", by["2026-10-21"].HolidayName)
	assert.Equal(t, "missing", by["2026-10-22"].DayStatus) // half-day leave, nothing logged
	assert.True(t, by["2026-10-22"].HalfDay)
	assert.Equal(t, "Casual", by["2026-10-22"].LeaveTypeName)
	assert.Equal(t, "on_leave", by["2026-10-23"].DayStatus) // Fri
	assert.False(t, by["2026-10-23"].HalfDay)
	assert.Equal(t, "Annual", by["2026-10-23"].LeaveTypeName)
	assert.Equal(t, "weekly_off", by["2026-10-24"].DayStatus) // leave on weekly off is not on_leave
	assert.Equal(t, "weekly_off", by["2026-10-25"].DayStatus)
	assert.Equal(t, "logged", by["2026-10-26"].DayStatus) // logging beats leave
	assert.Equal(t, 4.0, by["2026-10-26"].TotalHours)
	assert.Equal(t, "missing", by["2026-10-27"].DayStatus) // pending leave and optional holiday do not excuse

	// Entirely in the future: empty.
	rows, err = uc.Execute(ctx, e.tenantID, e.user1, "2026-10-28", "2026-10-30")
	require.NoError(t, err)
	assert.Empty(t, rows)

	// Same validation as ListMyStatusLogs.
	_, err = uc.Execute(ctx, e.tenantID, e.user1, "2026-10-30", "2026-10-21")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidStatusLogDate)
	_, err = uc.Execute(ctx, e.tenantID, e.user1, "2024-01-01", "2026-10-27")
	assert.ErrorIs(t, err, domainerrors.ErrStatusLogDateRangeExceeded)
	_, err = uc.Execute(ctx, uuid.Nil, e.user1, "", "")
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)

	// Other user's log is not counted.
	rows, err = uc.Execute(ctx, e.tenantID, e.user2, "2026-10-26", "2026-10-26")
	require.NoError(t, err)
	assert.Equal(t, "missing", rows[0].DayStatus)
}

func TestStatusLog_MonthlyMissingDaysIgnoresOffDayAndLeaveDayLogs(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	// Oct 2026: 22 working days (Sat/Sun off); the log window is the 7 days up to today (27th). 2026-10-22 is a public holiday => 21.
	hols := []*entity.Holiday{{Date: d("2026-10-22"), CategoryIsPublic: true}}
	logDays := func(t *testing.T, e *testEnv, days ...string) {
		createUC := NewCreateStatusLogUseCase(e.repo, e.allocs, ucshared.NewNopTransactor(), e.clock)
		cat := "meeting"
		for _, day := range days {
			_, err := createUC.Execute(context.Background(), e.tenantID, e.user1, statuslog.CreateStatusLogRequest{LogDate: day, ActivityCategory: &cat, Hours: 2, Notes: "x"})
			require.NoError(t, err)
		}
	}
	missing := func(t *testing.T, e *testEnv, reqs []*entity.LeaveRequestDetail) *entity.MonthlyEmployeeSummary {
		uc := NewGetMonthlyStatusLogsUseCase(e.repo, fakeOffDays{}, &fakeLeaveList{reqs: reqs}, &fakeHolidayRange{hols: hols}, e.clock)
		rep, err := uc.Execute(context.Background(), e.tenantID, "2026-10", "", "")
		require.NoError(t, err)
		for _, s := range rep.Employees {
			if s.UserID == e.user1 {
				return s
			}
		}
		t.Fatal("user1 not in report")
		return nil
	}

	t.Run("off-day work does not hide missing working days", func(t *testing.T) {
		e := newTestEnv()
		// Saturday 24th and Sunday 25th plus one real working day (Mon 26th).
		logDays(t, e, "2026-10-24", "2026-10-25", "2026-10-26")
		s := missing(t, e, nil)
		assert.Equal(t, 3, s.DaysLogged)
		assert.Equal(t, 2, s.OffDaysWorked)
		assert.Equal(t, 21.0, s.ExpectedDays)
		assert.Equal(t, 20.0, s.MissingDays) // 21 expected - 1 working day logged
	})

	t.Run("log on a public holiday does not hide a missing day", func(t *testing.T) {
		e := newTestEnv()
		logDays(t, e, "2026-10-22")
		assert.Equal(t, 21.0, missing(t, e, nil).MissingDays)
	})

	t.Run("log on a full leave day is not double counted", func(t *testing.T) {
		e := newTestEnv()
		leave := &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{TenantID: e.tenantID, UserID: e.user1, Status: "approved", StartDate: d("2026-10-23"), EndDate: d("2026-10-26"), DayPart: "full"}}
		logDays(t, e, "2026-10-23") // logged while on leave
		s := missing(t, e, []*entity.LeaveRequestDetail{leave})
		assert.Equal(t, 2.0, s.LeaveDays)
		assert.Equal(t, 19.0, s.ExpectedDays)
		// 21 working days: 2 on leave (one also logged) => 19 still to log, none logged on those.
		assert.Equal(t, 19.0, s.MissingDays)
	})

	t.Run("half-day leave with a log leaves no missing time that day", func(t *testing.T) {
		e := newTestEnv()
		leave := &entity.LeaveRequestDetail{LeaveRequest: entity.LeaveRequest{TenantID: e.tenantID, UserID: e.user1, Status: "approved", StartDate: d("2026-10-23"), EndDate: d("2026-10-23"), DayPart: "first_half"}}
		s := missing(t, e, []*entity.LeaveRequestDetail{leave})
		assert.Equal(t, 0.5, s.LeaveDays)
		assert.Equal(t, 20.5, s.ExpectedDays)
		assert.Equal(t, 20.5, s.MissingDays)
		logDays(t, e, "2026-10-23")
		s = missing(t, e, []*entity.LeaveRequestDetail{leave})
		assert.Equal(t, 20.0, s.MissingDays)
	})
}
