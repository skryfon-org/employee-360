package persistence

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormAuditRepository implements repository.AuditRepository. Create persists
// the entity's TenantID (set by the usecase); reads are scoped by the explicit
// tenantID parameter.
type gormAuditRepository struct {
	db *gorm.DB
}

// NewGormAuditRepository constructs a GORM-backed AuditRepository.
func NewGormAuditRepository(db *gorm.DB) repository.AuditRepository {
	return &gormAuditRepository{db: db}
}

var _ repository.AuditRepository = (*gormAuditRepository)(nil)

func (r *gormAuditRepository) Create(c context.Context, log *entity.AuditLog) error {
	return database.DBFromContext(c, r.db).Create(log).Error
}

func (r *gormAuditRepository) list(q *gorm.DB, limit, offset int) ([]*entity.AuditLog, int64, error) {
	q = q.Session(&gorm.Session{})
	var total int64
	if err := q.Model(&entity.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []*entity.AuditLog
	page := q.Order("created_at DESC, id DESC")
	if limit > 0 {
		page = page.Limit(limit)
	}
	if offset > 0 {
		page = page.Offset(offset)
	}
	if err := page.Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *gormAuditRepository) tenantQuery(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ?", tenantID)
}

func (r *gormAuditRepository) ListByTenantID(c context.Context, tenantID uuid.UUID, limit, offset int) ([]*entity.AuditLog, int64, error) {
	return r.list(r.tenantQuery(c, tenantID), limit, offset)
}

func (r *gormAuditRepository) ListByEntity(c context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID, limit, offset int) ([]*entity.AuditLog, int64, error) {
	return r.list(r.tenantQuery(c, tenantID).Where("entity_type = ? AND entity_id = ?", entityType, entityID), limit, offset)
}
