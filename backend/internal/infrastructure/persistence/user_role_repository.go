package persistence

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormUserRoleRepository implements repository.UserRoleRepository. Writes use
// the TenantID set on the entity by the usecase; reads and deletes are scoped
// by the explicit tenantID parameter.
type gormUserRoleRepository struct {
	db *gorm.DB
}

// NewGormUserRoleRepository constructs a GORM-backed UserRoleRepository.
func NewGormUserRoleRepository(db *gorm.DB) repository.UserRoleRepository {
	return &gormUserRoleRepository{db: db}
}

var _ repository.UserRoleRepository = (*gormUserRoleRepository)(nil)

func (r *gormUserRoleRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ?", tenantID)
}

func (r *gormUserRoleRepository) AssignRole(c context.Context, ur *entity.UserRole) error {
	return database.DBFromContext(c, r.db).Omit("User", "Role").Create(ur).Error
}

func (r *gormUserRoleRepository) RemoveRole(c context.Context, tenantID, userID, roleID uuid.UUID) error {
	return r.scoped(c, tenantID).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&entity.UserRole{}).Error
}

func (r *gormUserRoleRepository) GetRolesByUserID(c context.Context, tenantID, userID uuid.UUID) ([]*entity.Role, error) {
	var roles []*entity.Role
	err := database.DBFromContext(c, r.db).
		Joins("JOIN user_roles ur ON ur.role_id = roles.id AND ur.tenant_id = roles.tenant_id").
		Where("ur.user_id = ? AND ur.tenant_id = ? AND roles.deleted_at IS NULL", userID, tenantID).
		Find(&roles).Error
	return roles, err
}

func (r *gormUserRoleRepository) GetUserRolesByUserID(c context.Context, tenantID, userID uuid.UUID) ([]*entity.UserRole, error) {
	var urs []*entity.UserRole
	err := r.scoped(c, tenantID).Where("user_id = ?", userID).Find(&urs).Error
	return urs, err
}

func (r *gormUserRoleRepository) DeleteByUserID(c context.Context, tenantID, userID uuid.UUID) error {
	return r.scoped(c, tenantID).Where("user_id = ?", userID).Delete(&entity.UserRole{}).Error
}
