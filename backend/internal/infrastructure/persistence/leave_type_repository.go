package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormLeaveTypeRepository struct {
	db *gorm.DB
}

// NewGormLeaveTypeRepository constructs a GORM-backed LeaveTypeRepository.
func NewGormLeaveTypeRepository(db *gorm.DB) repository.LeaveTypeRepository {
	return &gormLeaveTypeRepository{db: db}
}

var _ repository.LeaveTypeRepository = (*gormLeaveTypeRepository)(nil)

func (r *gormLeaveTypeRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func (r *gormLeaveTypeRepository) Create(c context.Context, tenantID, actorID uuid.UUID, lt *entity.LeaveType) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrLeaveTypeNotFound
	}
	now := time.Now().UTC()
	if lt.ID == uuid.Nil {
		lt.ID = uuid.New()
	}
	lt.TenantID = tenantID
	lt.CreatedAt, lt.UpdatedAt = now, now
	lt.CreatedBy, lt.UpdatedBy = &actorID, &actorID
	if err := database.DBFromContext(c, r.db).Create(lt).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrLeaveTypeCodeTaken
		}
		return err
	}
	return nil
}

func (r *gormLeaveTypeRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.LeaveType, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrLeaveTypeNotFound
	}
	var lt entity.LeaveType
	if err := r.scoped(c, tenantID).Where("id = ?", id).First(&lt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrLeaveTypeNotFound
		}
		return nil, err
	}
	return &lt, nil
}

func (r *gormLeaveTypeRepository) List(c context.Context, tenantID uuid.UUID, activeOnly bool) ([]*entity.LeaveType, error) {
	out := []*entity.LeaveType{}
	if tenantID == uuid.Nil {
		return out, nil
	}
	q := r.scoped(c, tenantID)
	if activeOnly {
		q = q.Where("is_active = TRUE")
	}
	if err := q.Order("name ASC, id ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *gormLeaveTypeRepository) Update(c context.Context, tenantID, actorID uuid.UUID, lt *entity.LeaveType) error {
	if tenantID == uuid.Nil || lt.ID == uuid.Nil {
		return domainerrors.ErrLeaveTypeNotFound
	}
	lt.UpdatedAt = time.Now().UTC()
	lt.UpdatedBy = &actorID
	res := r.scoped(c, tenantID).Model(&entity.LeaveType{}).Where("id = ?", lt.ID).
		Updates(map[string]any{
			"name":        lt.Name,
			"code":        lt.Code,
			"description": lt.Description,
			"is_active":   lt.IsActive,
			"updated_at":  lt.UpdatedAt,
			"updated_by":  actorID,
		})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return domainerrors.ErrLeaveTypeCodeTaken
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrLeaveTypeNotFound
	}
	return nil
}

func (r *gormLeaveTypeRepository) ExistsByCode(c context.Context, tenantID uuid.UUID, code string, excludeID uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil {
		return false, nil
	}
	q := r.scoped(c, tenantID).Model(&entity.LeaveType{}).Where("code = ?", code)
	if excludeID != uuid.Nil {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}
