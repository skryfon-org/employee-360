package persistence

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormOrgReferenceRepository implements repository.OrgReferenceRepository
// with tenant-scoped existence checks on departments and positions.
type gormOrgReferenceRepository struct {
	db *gorm.DB
}

// NewGormOrgReferenceRepository constructs a GORM-backed OrgReferenceRepository.
func NewGormOrgReferenceRepository(db *gorm.DB) repository.OrgReferenceRepository {
	return &gormOrgReferenceRepository{db: db}
}

var _ repository.OrgReferenceRepository = (*gormOrgReferenceRepository)(nil)

func (r *gormOrgReferenceRepository) exists(c context.Context, table string, tenantID, id uuid.UUID) (bool, error) {
	var n int64
	err := database.DBFromContext(c, r.db).
		Table(table).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Count(&n).Error
	return n > 0, err
}

func (r *gormOrgReferenceRepository) DepartmentExists(c context.Context, tenantID, id uuid.UUID) (bool, error) {
	return r.exists(c, "departments", tenantID, id)
}

func (r *gormOrgReferenceRepository) LockDepartmentShared(c context.Context, tenantID, id uuid.UUID) (found, active bool, err error) {
	var rows []bool
	err = database.DBFromContext(c, r.db).
		Table("departments").
		Clauses(clause.Locking{Strength: "SHARE"}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Pluck("is_active", &rows).Error
	if err != nil || len(rows) == 0 {
		return false, false, err
	}
	return true, rows[0], nil
}

func (r *gormOrgReferenceRepository) LockPositionShared(c context.Context, tenantID, id uuid.UUID) (found, active bool, err error) {
	var rows []bool
	err = database.DBFromContext(c, r.db).
		Table("positions").
		Clauses(clause.Locking{Strength: "SHARE"}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Pluck("is_active", &rows).Error
	if err != nil || len(rows) == 0 {
		return false, false, err
	}
	return true, rows[0], nil
}

// gormOrgNameRepository implements repository.OrgNameRepository.
type gormOrgNameRepository struct {
	db *gorm.DB
}

// NewGormOrgNameRepository constructs a GORM-backed OrgNameRepository.
func NewGormOrgNameRepository(db *gorm.DB) repository.OrgNameRepository {
	return &gormOrgNameRepository{db: db}
}

var _ repository.OrgNameRepository = (*gormOrgNameRepository)(nil)

func (r *gormOrgNameRepository) name(c context.Context, table string, tenantID, id uuid.UUID) (string, bool, error) {
	var names []string
	err := database.DBFromContext(c, r.db).
		Table(table).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Pluck("name", &names).Error
	if err != nil || len(names) == 0 {
		return "", false, err
	}
	return names[0], true, nil
}

func (r *gormOrgNameRepository) DepartmentName(c context.Context, tenantID, id uuid.UUID) (string, bool, error) {
	return r.name(c, "departments", tenantID, id)
}

func (r *gormOrgNameRepository) PositionName(c context.Context, tenantID, id uuid.UUID) (string, bool, error) {
	return r.name(c, "positions", tenantID, id)
}
