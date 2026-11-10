package persistence

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormEmployeeDirectoryRepository struct {
	db *gorm.DB
}

// NewGormEmployeeDirectoryRepository constructs a GORM-backed EmployeeDirectoryRepository.
func NewGormEmployeeDirectoryRepository(db *gorm.DB) repository.EmployeeDirectoryRepository {
	return &gormEmployeeDirectoryRepository{db: db}
}

var _ repository.EmployeeDirectoryRepository = (*gormEmployeeDirectoryRepository)(nil)

type employeeRow struct {
	UserID             uuid.UUID
	Email              string
	FirstName          string
	LastName           string
	IsActive           bool
	LastLoginAt        *time.Time
	DepartmentID       *uuid.UUID
	DepartmentName     *string
	PositionID         *uuid.UUID
	PositionName       *string
	EmployeeCode       *string
	EmploymentType     *string
	EmploymentStatus   *string
	JoiningDate        *time.Time
	WorkLocation       *string
	ManagerID          *uuid.UUID
	Phone              *string
	ProfileCompletedAt *time.Time
	CreatedAt          time.Time
}

const employeeSelect = `u.id AS user_id, u.email, u.first_name, u.last_name, u.is_active, u.last_login_at,
	u.department_id, d.name AS department_name, u.position_id, p.name AS position_name,
	ep.employee_code, ep.employment_type, ep.employment_status, ep.joining_date, ep.work_location,
	ep.manager_id, ep.phone, ep.profile_completed_at, u.created_at`

// base builds the tenant-scoped, soft-delete-aware joined query. Joined tables
// are tenant-scoped in the ON clause as defence in depth.
func (r *gormEmployeeDirectoryRepository) base(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).
		Table("users u").
		Joins("LEFT JOIN employee_profiles ep ON ep.user_id = u.id AND ep.tenant_id = u.tenant_id AND ep.deleted_at IS NULL").
		Joins("LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = u.tenant_id AND d.deleted_at IS NULL").
		Joins("LEFT JOIN positions p ON p.id = u.position_id AND p.tenant_id = u.tenant_id AND p.deleted_at IS NULL").
		Where("u.tenant_id = ? AND u.deleted_at IS NULL", tenantID)
}

func (r *gormEmployeeDirectoryRepository) List(c context.Context, tenantID uuid.UUID, f entity.EmployeeFilter, limit, offset int) ([]*entity.EmployeeSummary, int64, error) {
	q := r.base(c, tenantID)
	// Collapse runs of whitespace so "alice   smith" still matches "Alice Smith".
	if s := strings.Join(strings.Fields(f.Search), " "); s != "" {
		like := "%" + escapeLike(strings.ToLower(s)) + "%"
		q = q.Where(`(LOWER(u.first_name || ' ' || u.last_name) LIKE ? OR LOWER(u.email) LIKE ? OR LOWER(COALESCE(ep.employee_code, '')) LIKE ?)`, like, like, like)
	}
	if f.IsActive != nil {
		q = q.Where("u.is_active = ?", *f.IsActive)
	}
	if f.DepartmentID != nil {
		q = q.Where("u.department_id = ?", *f.DepartmentID)
	}
	if f.PositionID != nil {
		q = q.Where("u.position_id = ?", *f.PositionID)
	}
	if f.EmploymentStatus != "" {
		q = q.Where("ep.employment_status = ?", f.EmploymentStatus)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []employeeRow
	if err := q.Session(&gorm.Session{}).
		Select(employeeSelect).
		Order("LOWER(u.first_name) ASC, LOWER(u.last_name) ASC, u.id ASC").
		Limit(limit).Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	out := make([]*entity.EmployeeSummary, 0, len(rows))
	ids := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toSummary())
		ids = append(ids, rows[i].UserID)
	}
	if err := r.loadRoles(c, tenantID, out, ids); err != nil {
		return nil, 0, err
	}
	if f.IncludeAllocation {
		if err := r.loadAllocated(c, tenantID, out, ids, f.AllocationAsOf); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

func (r *gormEmployeeDirectoryRepository) GetSummary(c context.Context, tenantID, userID uuid.UUID) (*entity.EmployeeSummary, error) {
	var rows []employeeRow
	if err := r.base(c, tenantID).Where("u.id = ?", userID).Select(employeeSelect).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domainerrors.ErrUserNotFound
	}
	s := rows[0].toSummary()
	if err := r.loadRoles(c, tenantID, []*entity.EmployeeSummary{s}, []uuid.UUID{userID}); err != nil {
		return nil, err
	}
	return s, nil
}

// loadAllocated sets AllocatedPercent for the page in one aggregate query:
// the sum of allocations covering asOf on live (active/on_hold) projects.
func (r *gormEmployeeDirectoryRepository) loadAllocated(c context.Context, tenantID uuid.UUID, list []*entity.EmployeeSummary, ids []uuid.UUID, asOf time.Time) error {
	for _, s := range list {
		zero := 0
		s.AllocatedPercent = &zero
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []struct {
		UserID uuid.UUID
		Total  int
	}
	if err := database.DBFromContext(c, r.db).
		Table("project_allocations pa").
		Select("pa.user_id AS user_id, COALESCE(SUM(pa.allocation_percent), 0) AS total").
		Joins("JOIN projects pr ON pr.id = pa.project_id AND pr.tenant_id = pa.tenant_id AND pr.deleted_at IS NULL AND pr.status IN ('active', 'on_hold')").
		Where("pa.tenant_id = ? AND pa.user_id IN ? AND pa.deleted_at IS NULL", tenantID, ids).
		Where("pa.start_date <= ? AND (pa.end_date IS NULL OR pa.end_date >= ?)", asOf, asOf).
		Group("pa.user_id").
		Scan(&rows).Error; err != nil {
		return err
	}
	byID := make(map[uuid.UUID]*entity.EmployeeSummary, len(list))
	for _, s := range list {
		byID[s.UserID] = s
	}
	for _, row := range rows {
		if s := byID[row.UserID]; s != nil {
			t := row.Total
			s.AllocatedPercent = &t
		}
	}
	return nil
}

func (r *gormEmployeeDirectoryRepository) loadRoles(c context.Context, tenantID uuid.UUID, list []*entity.EmployeeSummary, ids []uuid.UUID) error {
	for _, s := range list {
		s.Roles = []string{}
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []struct {
		UserID uuid.UUID
		Name   string
	}
	if err := database.DBFromContext(c, r.db).
		Table("user_roles ur").
		Select("ur.user_id AS user_id, ro.name AS name").
		Joins("JOIN roles ro ON ro.id = ur.role_id AND ro.deleted_at IS NULL").
		Where("ur.tenant_id = ? AND ur.user_id IN ?", tenantID, ids).
		Order("ro.name").
		Scan(&rows).Error; err != nil {
		return err
	}
	byID := make(map[uuid.UUID]*entity.EmployeeSummary, len(list))
	for _, s := range list {
		byID[s.UserID] = s
	}
	for _, row := range rows {
		if s := byID[row.UserID]; s != nil {
			s.Roles = append(s.Roles, row.Name)
		}
	}
	return nil
}

func (e employeeRow) toSummary() *entity.EmployeeSummary {
	return &entity.EmployeeSummary{
		UserID: e.UserID, Email: e.Email, FirstName: e.FirstName, LastName: e.LastName,
		IsActive: e.IsActive, LastLoginAt: e.LastLoginAt,
		DepartmentID: e.DepartmentID, DepartmentName: e.DepartmentName,
		PositionID: e.PositionID, PositionName: e.PositionName,
		EmployeeCode: e.EmployeeCode, EmploymentType: e.EmploymentType, EmploymentStatus: e.EmploymentStatus,
		JoiningDate: e.JoiningDate, WorkLocation: e.WorkLocation, ManagerID: e.ManagerID, Phone: e.Phone,
		ProfileCompleted: e.ProfileCompletedAt != nil, CreatedAt: e.CreatedAt,
	}
}

func (r *gormEmployeeDirectoryRepository) LockActiveUserIDsWithRole(c context.Context, tenantID uuid.UUID, roleName string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := database.DBFromContext(c, r.db).
		Table("users").
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "users"}}).
		Select("users.id").
		Joins("JOIN user_roles ur ON ur.user_id = users.id AND ur.tenant_id = users.tenant_id").
		Joins("JOIN roles ro ON ro.id = ur.role_id AND ro.deleted_at IS NULL").
		Where("users.tenant_id = ? AND users.deleted_at IS NULL AND users.is_active AND ro.name = ?", tenantID, roleName).
		Order("users.id").
		Pluck("users.id", &ids).Error
	return ids, err
}
