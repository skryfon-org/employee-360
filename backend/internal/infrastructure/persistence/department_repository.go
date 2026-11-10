package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormDepartmentRepository implements repository.DepartmentRepository.
// Every operation is scoped to the tenantID explicitly provided by the caller.
type gormDepartmentRepository struct {
	db *gorm.DB
}

// NewGormDepartmentRepository constructs a GORM-backed DepartmentRepository.
func NewGormDepartmentRepository(db *gorm.DB) repository.DepartmentRepository {
	return &gormDepartmentRepository{db: db}
}

var _ repository.DepartmentRepository = (*gormDepartmentRepository)(nil)

// scoped returns a DB handle restricted to tenantID and non-deleted rows.
func (r *gormDepartmentRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func isUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Create persists a new department row scoped to the caller's tenant and records created_by and updated_by.
func (r *gormDepartmentRepository) Create(c context.Context, tenantID, actorID uuid.UUID, department *entity.Department) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrDepartmentNotFound
	}
	department.TenantID = tenantID
	department.CreatedBy = &actorID
	department.UpdatedBy = &actorID
	now := time.Now().UTC()
	if department.ID == uuid.Nil {
		department.ID = uuid.New()
	}
	if department.CreatedAt.IsZero() {
		department.CreatedAt = now
	}
	if department.UpdatedAt.IsZero() {
		department.UpdatedAt = now
	}

	err := database.DBFromContext(c, r.db).Create(department).Error
	if err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrDepartmentNameTaken
		}
		return err
	}
	return nil
}

// GetByIDForUpdate looks up a department within the caller's tenant and locks the
// row (FOR UPDATE) until the surrounding transaction ends.
func (r *gormDepartmentRepository) GetByIDForUpdate(c context.Context, tenantID, id uuid.UUID) (*entity.Department, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrDepartmentNotFound
	}

	var dept entity.Department
	err := r.scoped(c, tenantID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&dept).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

// GetByID looks up a department by ID within the caller's tenant.
func (r *gormDepartmentRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.Department, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrDepartmentNotFound
	}

	var dept entity.Department
	err := r.scoped(c, tenantID).Where("id = ?", id).First(&dept).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrDepartmentNotFound
		}
		return nil, err
	}
	return &dept, nil
}

// List returns a paginated slice of departments for the caller's tenant.
func (r *gormDepartmentRepository) List(c context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error) {
	if tenantID == uuid.Nil {
		return []*entity.Department{}, 0, nil
	}

	var (
		departments []*entity.Department
		total       int64
	)

	db := r.scoped(c, tenantID).Model(&entity.Department{})
	if isActive != nil {
		db = db.Where("is_active = ?", *isActive)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		db = db.Limit(limit)
	}
	if offset > 0 {
		db = db.Offset(offset)
	}

	if err := db.Order("name ASC, id ASC").Find(&departments).Error; err != nil {
		return nil, 0, err
	}

	return departments, total, nil
}

// Update modifies department details within the caller's tenant and records updated_by.
func (r *gormDepartmentRepository) Update(c context.Context, tenantID, actorID uuid.UUID, department *entity.Department) error {
	if tenantID == uuid.Nil || department.ID == uuid.Nil {
		return domainerrors.ErrDepartmentNotFound
	}

	department.TenantID = tenantID
	department.UpdatedBy = &actorID
	department.UpdatedAt = time.Now().UTC()

	result := r.scoped(c, tenantID).
		Model(&entity.Department{}).
		Where("id = ?", department.ID).
		Updates(map[string]interface{}{
			"name":        department.Name,
			"description": department.Description,
			"is_active":   department.IsActive,
			"updated_at":  department.UpdatedAt,
			"updated_by":  actorID,
		})

	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return domainerrors.ErrDepartmentNameTaken
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrDepartmentNotFound
	}

	return nil
}

// Delete soft-deletes a department by ID within the caller's tenant.
// actorID is the admin performing the removal.
func (r *gormDepartmentRepository) Delete(c context.Context, tenantID, id, actorID uuid.UUID) error {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return domainerrors.ErrDepartmentNotFound
	}

	now := time.Now().UTC()
	result := database.DBFromContext(c, r.db).
		Model(&entity.Department{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(map[string]any{
			"deleted_at": now,
			"deleted_by": actorID,
			"updated_at": now,
			"updated_by": actorID,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrDepartmentNotFound
	}
	return nil
}

// ExistsByName checks if another department with the same name exists in the caller's tenant.
func (r *gormDepartmentRepository) ExistsByName(c context.Context, tenantID uuid.UUID, name string) (bool, error) {
	if tenantID == uuid.Nil {
		return false, nil
	}

	var count int64
	err := r.scoped(c, tenantID).
		Model(&entity.Department{}).
		Where("LOWER(name) = LOWER(?)", strings.TrimSpace(name)).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsReferenced checks whether any active users or pending user_invitations
// (not accepted, revoked or expired) reference this department.
func (r *gormDepartmentRepository) IsReferenced(c context.Context, tenantID, id uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return false, nil
	}

	db := database.DBFromContext(c, r.db)

	var userCount int64
	if err := db.Table("users").
		Where("department_id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Count(&userCount).Error; err != nil {
		return false, err
	}
	if userCount > 0 {
		return true, nil
	}

	var invitationCount int64
	if err := db.Table("user_invitations").
		Where("department_id = ? AND tenant_id = ? AND deleted_at IS NULL AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?",
			id, tenantID, time.Now().UTC()).
		Count(&invitationCount).Error; err != nil {
		return false, err
	}
	return invitationCount > 0, nil
}
