package persistence

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormStatusLogRepository struct {
	db *gorm.DB
}

// NewGormStatusLogRepository constructs a GORM-backed StatusLogRepository.
func NewGormStatusLogRepository(db *gorm.DB) repository.StatusLogRepository {
	return &gormStatusLogRepository{db: db}
}

var _ repository.StatusLogRepository = (*gormStatusLogRepository)(nil)

func (r *gormStatusLogRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func (r *gormStatusLogRepository) LockUserDay(c context.Context, tenantID, userID uuid.UUID, logDate time.Time) error {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	lockKey := fmt.Sprintf("status_log:%s:%s:%s", tenantID, userID, logDate.Format("2006-01-02"))
	return database.DBFromContext(c, r.db).Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error
}

func (r *gormStatusLogRepository) Create(c context.Context, tenantID, actorID uuid.UUID, entry *entity.StatusLogEntry) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrStatusLogNotFound
	}
	now := time.Now().UTC()
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	entry.TenantID = tenantID
	entry.CreatedAt, entry.UpdatedAt = now, now
	entry.CreatedBy, entry.UpdatedBy = &actorID, &actorID
	return database.DBFromContext(c, r.db).Create(entry).Error
}

func (r *gormStatusLogRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.StatusLogEntry, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrStatusLogNotFound
	}
	var entry entity.StatusLogEntry
	if err := r.scoped(c, tenantID).Where("id = ?", id).First(&entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrStatusLogNotFound
		}
		return nil, err
	}
	return &entry, nil
}

func (r *gormStatusLogRepository) GetByIDWithProject(c context.Context, tenantID, id uuid.UUID) (*entity.StatusLogEntryWithProject, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrStatusLogNotFound
	}
	var out entity.StatusLogEntryWithProject
	err := database.DBFromContext(c, r.db).
		Table("status_log_entries AS sle").
		Select("sle.*, p.name AS project_name, p.code AS project_code").
		Joins("LEFT JOIN projects p ON p.id = sle.project_id AND p.tenant_id = sle.tenant_id AND p.deleted_at IS NULL").
		Where("sle.tenant_id = ? AND sle.id = ? AND sle.deleted_at IS NULL", tenantID, id).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrStatusLogNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *gormStatusLogRepository) Update(c context.Context, tenantID, actorID uuid.UUID, entry *entity.StatusLogEntry) error {
	if tenantID == uuid.Nil || entry.ID == uuid.Nil {
		return domainerrors.ErrStatusLogNotFound
	}
	now := time.Now().UTC()
	entry.UpdatedAt = now
	entry.UpdatedBy = &actorID
	entry.TenantID = tenantID

	res := database.DBFromContext(c, r.db).
		Model(&entity.StatusLogEntry{}).
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, entry.ID).
		Updates(map[string]interface{}{
			"log_date":          entry.LogDate,
			"project_id":        entry.ProjectID,
			"activity_category": entry.ActivityCategory,
			"hours":             entry.Hours,
			"notes":             entry.Notes,
			"updated_at":        now,
			"updated_by":        actorID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrStatusLogNotFound
	}
	return nil
}

func (r *gormStatusLogRepository) SoftDelete(c context.Context, tenantID, id, actorID uuid.UUID) error {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return domainerrors.ErrStatusLogNotFound
	}
	now := time.Now().UTC()
	res := database.DBFromContext(c, r.db).
		Model(&entity.StatusLogEntry{}).
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"deleted_by": actorID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrStatusLogNotFound
	}
	return nil
}

func (r *gormStatusLogRepository) GetDayTotalHours(c context.Context, tenantID, userID uuid.UUID, logDate time.Time, excludeID uuid.UUID) (float64, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return 0, nil
	}
	var total float64
	q := database.DBFromContext(c, r.db).
		Table("status_log_entries").
		Select("COALESCE(SUM(hours), 0)").
		Where("tenant_id = ? AND user_id = ? AND log_date = ? AND deleted_at IS NULL", tenantID, userID, logDate)
	if excludeID != uuid.Nil {
		q = q.Where("id != ?", excludeID)
	}
	err := q.Scan(&total).Error
	return math.Round(total*100) / 100, err
}

func (r *gormStatusLogRepository) ListByUserAndDateRange(c context.Context, tenantID, userID uuid.UUID, from, to time.Time, projectID *uuid.UUID) ([]*entity.StatusLogEntryWithProject, error) {
	out := []*entity.StatusLogEntryWithProject{}
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return out, nil
	}
	q := database.DBFromContext(c, r.db).
		Table("status_log_entries AS sle").
		Select("sle.*, p.name AS project_name, p.code AS project_code").
		Joins("LEFT JOIN projects p ON p.id = sle.project_id AND p.tenant_id = sle.tenant_id AND p.deleted_at IS NULL").
		Where("sle.tenant_id = ? AND sle.user_id = ? AND sle.deleted_at IS NULL", tenantID, userID).
		Where("sle.log_date >= ? AND sle.log_date <= ?", from, to)
	if projectID != nil && *projectID != uuid.Nil {
		q = q.Where("sle.project_id = ?", *projectID)
	}
	err := q.Order("sle.log_date DESC, sle.created_at DESC, sle.id DESC").Scan(&out).Error
	return out, err
}

type userSummaryRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	FirstName      string     `gorm:"column:first_name"`
	LastName       string     `gorm:"column:last_name"`
	Email          string     `gorm:"column:email"`
	DepartmentID   *uuid.UUID `gorm:"column:department_id"`
	DepartmentName *string    `gorm:"column:department_name"`
}

type dailyEntryRow struct {
	UserID           uuid.UUID  `gorm:"column:user_id"`
	LogDate          time.Time  `gorm:"column:log_date"`
	ProjectID        *uuid.UUID `gorm:"column:project_id"`
	ProjectName      *string    `gorm:"column:project_name"`
	ProjectCode      *string    `gorm:"column:project_code"`
	ActivityCategory *string    `gorm:"column:activity_category"`
	Hours            float64    `gorm:"column:hours"`
}

func (r *gormStatusLogRepository) DailySummary(c context.Context, tenantID uuid.UUID, date time.Time, departmentID, projectID *uuid.UUID) ([]*entity.DailyEmployeeSummary, error) {
	rows, err := r.summarizeRange(c, tenantID, date, date, departmentID, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.DailyEmployeeSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, &row.DailyEmployeeSummary)
	}
	return out, nil
}

func (r *gormStatusLogRepository) MonthlySummary(c context.Context, tenantID uuid.UUID, from, to time.Time, offDays []int, departmentID, projectID *uuid.UUID) ([]*entity.MonthlyEmployeeSummary, error) {
	rows, err := r.summarizeRange(c, tenantID, from, to, departmentID, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.MonthlyEmployeeSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.MonthlyEmployeeSummary{
			UserID:          row.UserID,
			FirstName:       row.FirstName,
			LastName:        row.LastName,
			Email:           row.Email,
			DepartmentID:    row.DepartmentID,
			DepartmentName:  row.DepartmentName,
			DaysLogged:      row.daysLogged,
			OffDaysWorked:   countOffDays(row.loggedDays, offDays),
			LoggedDates:     row.loggedDays,
			TotalHours:      row.TotalHours,
			HoursByProject:  row.HoursByProject,
			HoursByCategory: row.HoursByCategory,
		})
	}
	return out, nil
}

type rangeSummary struct {
	entity.DailyEmployeeSummary
	daysLogged int
	loggedDays []time.Time // distinct logged dates (UTC midnight)
}

// countOffDays counts the dates whose weekday is in offDays.
func countOffDays(days []time.Time, offDays []int) int {
	off := make(map[time.Weekday]bool, len(offDays))
	for _, d := range offDays {
		off[time.Weekday(d)] = true
	}
	n := 0
	for _, d := range days {
		if off[d.Weekday()] {
			n++
		}
	}
	return n
}

// summarizeRange aggregates every active employee's entries over the inclusive
// [from, to] range. Employees with no entries are returned with zero values.
func (r *gormStatusLogRepository) summarizeRange(c context.Context, tenantID uuid.UUID, from, to time.Time, departmentID, projectID *uuid.UUID) ([]*rangeSummary, error) {
	out := []*rangeSummary{}
	if tenantID == uuid.Nil {
		return out, nil
	}

	uq := database.DBFromContext(c, r.db).
		Table("users AS u").
		Select("u.id, u.first_name, u.last_name, u.email, u.department_id, d.name AS department_name").
		Joins("LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = u.tenant_id AND d.deleted_at IS NULL").
		Where("u.tenant_id = ? AND u.is_active = TRUE AND u.deleted_at IS NULL", tenantID)

	if departmentID != nil && *departmentID != uuid.Nil {
		uq = uq.Where("u.department_id = ?", *departmentID)
	}

	if projectID != nil && *projectID != uuid.Nil {
		uq = uq.Where(`u.id IN (
			SELECT pa.user_id FROM project_allocations pa 
			WHERE pa.tenant_id = ? AND pa.project_id = ? AND pa.deleted_at IS NULL
			  AND pa.start_date <= ? AND (pa.end_date IS NULL OR pa.end_date >= ?)
			UNION
			SELECT sle.user_id FROM status_log_entries sle
			WHERE sle.tenant_id = ? AND sle.project_id = ? AND sle.log_date BETWEEN ? AND ? AND sle.deleted_at IS NULL
		)`, tenantID, *projectID, to, from, tenantID, *projectID, from, to)
	}

	var users []userSummaryRow
	if err := uq.Order("u.first_name ASC, u.last_name ASC, u.id ASC").Scan(&users).Error; err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return out, nil
	}

	eq := database.DBFromContext(c, r.db).
		Table("status_log_entries AS sle").
		Select("sle.user_id, sle.log_date, sle.project_id, sle.activity_category, sle.hours, p.name AS project_name, p.code AS project_code").
		Joins("LEFT JOIN projects p ON p.id = sle.project_id AND p.tenant_id = sle.tenant_id AND p.deleted_at IS NULL").
		Where("sle.tenant_id = ? AND sle.log_date BETWEEN ? AND ? AND sle.deleted_at IS NULL", tenantID, from, to)

	if projectID != nil && *projectID != uuid.Nil {
		eq = eq.Where("sle.project_id = ?", *projectID)
	}

	var entries []dailyEntryRow
	if err := eq.Scan(&entries).Error; err != nil {
		return nil, err
	}

	entriesByUser := make(map[uuid.UUID][]dailyEntryRow)
	for _, e := range entries {
		entriesByUser[e.UserID] = append(entriesByUser[e.UserID], e)
	}

	for _, u := range users {
		userEntries := entriesByUser[u.ID]
		var totalHours float64
		projMap := make(map[uuid.UUID]*entity.ProjectHoursBreakdown)
		catMap := make(map[string]float64)
		days := make(map[string]struct{})

		for _, e := range userEntries {
			totalHours += e.Hours
			days[e.LogDate.UTC().Format("2006-01-02")] = struct{}{}
			if e.ProjectID != nil && *e.ProjectID != uuid.Nil {
				pid := *e.ProjectID
				pName := ""
				if e.ProjectName != nil {
					pName = *e.ProjectName
				}
				pCode := ""
				if e.ProjectCode != nil {
					pCode = *e.ProjectCode
				}
				if existing, ok := projMap[pid]; ok {
					existing.Hours = math.Round((existing.Hours+e.Hours)*100) / 100
				} else {
					projMap[pid] = &entity.ProjectHoursBreakdown{
						ProjectID:   pid,
						ProjectName: pName,
						ProjectCode: pCode,
						Hours:       math.Round(e.Hours*100) / 100,
					}
				}
			} else if e.ActivityCategory != nil && *e.ActivityCategory != "" {
				cat := *e.ActivityCategory
				catMap[cat] = math.Round((catMap[cat]+e.Hours)*100) / 100
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
			hoursByCategory = append(hoursByCategory, entity.CategoryHoursBreakdown{
				Category: cat,
				Hours:    h,
			})
		}
		sort.Slice(hoursByCategory, func(i, j int) bool {
			return hoursByCategory[i].Category < hoursByCategory[j].Category
		})

		out = append(out, &rangeSummary{
			DailyEmployeeSummary: entity.DailyEmployeeSummary{
				UserID:          u.ID,
				FirstName:       u.FirstName,
				LastName:        u.LastName,
				Email:           u.Email,
				DepartmentID:    u.DepartmentID,
				DepartmentName:  u.DepartmentName,
				Logged:          len(userEntries) > 0,
				TotalHours:      math.Round(totalHours*100) / 100,
				HoursByProject:  hoursByProject,
				HoursByCategory: hoursByCategory,
			},
			daysLogged: len(days),
			loggedDays: daysToTimes(days),
		})
	}

	return out, nil
}

func daysToTimes(days map[string]struct{}) []time.Time {
	out := make([]time.Time, 0, len(days))
	for s := range days {
		if t, err := time.ParseInLocation("2006-01-02", s, time.UTC); err == nil {
			out = append(out, t)
		}
	}
	return out
}
