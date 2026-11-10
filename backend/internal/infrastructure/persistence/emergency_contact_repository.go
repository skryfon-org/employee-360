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

type gormEmergencyContactRepository struct {
	db *gorm.DB
}

// NewGormEmergencyContactRepository constructs a GORM-backed EmergencyContactRepository.
func NewGormEmergencyContactRepository(db *gorm.DB) repository.EmergencyContactRepository {
	return &gormEmergencyContactRepository{db: db}
}

var _ repository.EmergencyContactRepository = (*gormEmergencyContactRepository)(nil)

func (r *gormEmergencyContactRepository) scoped(c context.Context, tenantID, userID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).
		Where("tenant_id = ? AND user_id = ? AND deleted_at IS NULL", tenantID, userID)
}

func (r *gormEmergencyContactRepository) ListByUser(c context.Context, tenantID, userID uuid.UUID) ([]*entity.EmergencyContact, error) {
	var out []*entity.EmergencyContact
	err := r.scoped(c, tenantID, userID).Order("is_primary DESC, created_at ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *gormEmergencyContactRepository) GetByID(c context.Context, tenantID, userID, id uuid.UUID) (*entity.EmergencyContact, error) {
	var ec entity.EmergencyContact
	err := r.scoped(c, tenantID, userID).Where("id = ?", id).First(&ec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrEmergencyContactNotFound
		}
		return nil, err
	}
	return &ec, nil
}

func (r *gormEmergencyContactRepository) CountByUser(c context.Context, tenantID, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.scoped(c, tenantID, userID).Model(&entity.EmergencyContact{}).Count(&n).Error
	return n, err
}

func (r *gormEmergencyContactRepository) Create(c context.Context, actorID uuid.UUID, ec *entity.EmergencyContact) error {
	if ec.TenantID == uuid.Nil || ec.UserID == uuid.Nil {
		return domainerrors.ErrEmergencyContactNotFound
	}
	if ec.ID == uuid.Nil {
		ec.ID = uuid.New()
	}
	now := time.Now().UTC()
	ec.CreatedAt, ec.UpdatedAt = now, now
	ec.CreatedBy, ec.UpdatedBy = &actorID, &actorID
	return database.DBFromContext(c, r.db).Create(ec).Error
}

func (r *gormEmergencyContactRepository) Update(c context.Context, tenantID, actorID uuid.UUID, ec *entity.EmergencyContact) error {
	ec.UpdatedAt = time.Now().UTC()
	ec.UpdatedBy = &actorID
	res := database.DBFromContext(c, r.db).
		Model(&entity.EmergencyContact{}).
		Where("id = ? AND tenant_id = ? AND user_id = ? AND deleted_at IS NULL", ec.ID, tenantID, ec.UserID).
		Select("*").
		Omit("ID", "TenantID", "UserID", "CreatedAt", "CreatedBy", "DeletedAt", "DeletedBy").
		Updates(ec)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrEmergencyContactNotFound
	}
	return nil
}

func (r *gormEmergencyContactRepository) Delete(c context.Context, tenantID, userID, id, actorID uuid.UUID) error {
	now := time.Now().UTC()
	res := r.scoped(c, tenantID, userID).
		Model(&entity.EmergencyContact{}).
		Where("id = ?", id).
		Updates(map[string]any{"deleted_at": now, "deleted_by": actorID, "updated_at": now, "updated_by": actorID, "is_primary": false})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrEmergencyContactNotFound
	}
	return nil
}

func (r *gormEmergencyContactRepository) ClearPrimary(c context.Context, tenantID, userID, exceptID, actorID uuid.UUID) error {
	return r.scoped(c, tenantID, userID).
		Model(&entity.EmergencyContact{}).
		Where("is_primary AND id <> ?", exceptID).
		Updates(map[string]any{"is_primary": false, "updated_at": time.Now().UTC(), "updated_by": actorID}).Error
}
