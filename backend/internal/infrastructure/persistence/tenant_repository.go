package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormTenantReader is a GORM-backed adapter implementing repository.TenantReader.
// The tenants table is the tenant boundary itself, so it carries no tenant_id
// and is looked up by primary key.
type gormTenantReader struct {
	db *gorm.DB
}

// NewGormTenantReader constructs a GORM-backed TenantReader.
func NewGormTenantReader(db *gorm.DB) repository.TenantReader {
	return &gormTenantReader{db: db}
}

var _ repository.TenantReader = (*gormTenantReader)(nil)

// GetByID returns the non-soft-deleted tenant with the given id.
func (r *gormTenantReader) GetByID(c context.Context, id uuid.UUID) (*entity.Tenant, error) {
	var tenant entity.Tenant
	if err := database.DBFromContext(c, r.db).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// gormTenantRepository is the GORM-backed repository.TenantRepository. tenants
// is the tenant boundary itself (no tenant_id), so every method addresses rows
// by primary key and always filters deleted_at IS NULL.
type gormTenantRepository struct {
	gormTenantReader
}

// NewGormTenantRepository constructs a GORM-backed TenantRepository.
func NewGormTenantRepository(db *gorm.DB) repository.TenantRepository {
	return &gormTenantRepository{gormTenantReader{db: db}}
}

var _ repository.TenantRepository = (*gormTenantRepository)(nil)

func (r *gormTenantRepository) LockByID(c context.Context, id uuid.UUID) (*entity.Tenant, error) {
	var tenant entity.Tenant
	if err := database.DBFromContext(c, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *gormTenantRepository) update(c context.Context, id uuid.UUID, updates map[string]any) error {
	res := database.DBFromContext(c, r.db).Model(&entity.Tenant{}).
		Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrTenantNotFound
	}
	return nil
}

func (r *gormTenantRepository) UpdateName(c context.Context, id uuid.UUID, name string, actorID uuid.UUID, at time.Time) error {
	return r.update(c, id, map[string]any{"name": name, "updated_at": at, "updated_by": actorID})
}
