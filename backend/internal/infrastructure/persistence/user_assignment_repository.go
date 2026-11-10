package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormUserAssignmentRepository struct {
	db *gorm.DB
}

// NewGormUserAssignmentRepository constructs a GORM-backed UserAssignmentRepository.
func NewGormUserAssignmentRepository(db *gorm.DB) repository.UserAssignmentRepository {
	return &gormUserAssignmentRepository{db: db}
}

var _ repository.UserAssignmentRepository = (*gormUserAssignmentRepository)(nil)

func (r *gormUserAssignmentRepository) scoped(c context.Context, tenantID, userID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).
		Where("tenant_id = ? AND user_id = ? AND deleted_at IS NULL", tenantID, userID)
}

func (r *gormUserAssignmentRepository) GetOpen(c context.Context, tenantID, userID uuid.UUID) (*entity.UserAssignment, error) {
	var a entity.UserAssignment
	err := r.scoped(c, tenantID, userID).Where("effective_to IS NULL").First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *gormUserAssignmentRepository) CloseOpen(c context.Context, tenantID, userID, actorID uuid.UUID, effectiveTo time.Time) error {
	return r.scoped(c, tenantID, userID).
		Model(&entity.UserAssignment{}).
		Where("effective_to IS NULL").
		Updates(map[string]any{"effective_to": effectiveTo, "updated_at": time.Now().UTC(), "updated_by": actorID}).Error
}

func (r *gormUserAssignmentRepository) Create(c context.Context, actorID uuid.UUID, a *entity.UserAssignment) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now().UTC()
	a.CreatedAt, a.UpdatedAt = now, now
	a.CreatedBy, a.UpdatedBy = &actorID, &actorID
	return database.DBFromContext(c, r.db).Create(a).Error
}

func (r *gormUserAssignmentRepository) ListByUser(c context.Context, tenantID, userID uuid.UUID) ([]*entity.UserAssignment, error) {
	var out []*entity.UserAssignment
	err := r.scoped(c, tenantID, userID).Order("effective_from DESC, created_at DESC, id DESC").Find(&out).Error
	return out, err
}
