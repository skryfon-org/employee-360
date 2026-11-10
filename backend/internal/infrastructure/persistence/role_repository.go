package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormRoleRepository implements repository.RoleRepository. Every query is
// scoped by the explicit tenantID parameter; the tenant is never read from
// context.Context.
type gormRoleRepository struct {
	db *gorm.DB
}

// NewGormRoleRepository constructs a GORM-backed RoleRepository.
func NewGormRoleRepository(db *gorm.DB) repository.RoleRepository {
	return &gormRoleRepository{db: db}
}

var _ repository.RoleRepository = (*gormRoleRepository)(nil)

// scoped returns a DB handle restricted to tenantID.
func (r *gormRoleRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func (r *gormRoleRepository) Create(c context.Context, role *entity.Role) error {
	return database.DBFromContext(c, r.db).Create(role).Error
}

func (r *gormRoleRepository) first(c context.Context, tenantID uuid.UUID, where string, arg any) (*entity.Role, error) {
	var role entity.Role
	if err := r.scoped(c, tenantID).Where(where, arg).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *gormRoleRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.Role, error) {
	return r.first(c, tenantID, "id = ?", id)
}

func (r *gormRoleRepository) GetByName(c context.Context, tenantID uuid.UUID, name string) (*entity.Role, error) {
	return r.first(c, tenantID, "name = ?", name)
}

func (r *gormRoleRepository) List(c context.Context, tenantID uuid.UUID) ([]*entity.Role, error) {
	var roles []*entity.Role
	if err := r.scoped(c, tenantID).Order("name").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *gormRoleRepository) Update(c context.Context, tenantID uuid.UUID, role *entity.Role) error {
	res := r.scoped(c, tenantID).Model(&entity.Role{}).Where("id = ?", role.ID).
		Updates(map[string]any{"name": role.Name, "updated_at": role.UpdatedAt})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrRoleNotFound
	}
	return nil
}

func (r *gormRoleRepository) Delete(c context.Context, tenantID, id uuid.UUID) error {
	res := r.scoped(c, tenantID).Where("id = ?", id).Delete(&entity.Role{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrRoleNotFound
	}
	return nil
}
