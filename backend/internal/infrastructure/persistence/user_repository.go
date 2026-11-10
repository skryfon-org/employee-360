package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

// gormUserRepository is a GORM-backed adapter implementing repository.UserRepository.
//
// This repository never resolves tenant_id itself (no ctx-based lookup
// anywhere in this file): every method that needs tenant scoping takes it as
// an explicit parameter, supplied by the caller. Where that tenant_id comes
// from (request context via handler middleware, a token row, an already
// tenant-scoped lookup, ...) is entirely a usecase-layer concern.
type gormUserRepository struct {
	db *gorm.DB
}

// NewGormUserRepository constructs a GORM-backed UserRepository.
func NewGormUserRepository(db *gorm.DB) repository.UserRepository {
	return &gormUserRepository{db: db}
}

var _ repository.UserRepository = (*gormUserRepository)(nil)

// Create persists a new user, using user.TenantID exactly as the caller set
// it (Create takes no separate tenantID parameter -- the caller already
// resolved and set TenantID on the entity before calling this).
func (r *gormUserRepository) Create(c context.Context, user *entity.User) error {
	if err := database.DBFromContext(c, r.db).Create(user).Error; err != nil {
		// Map only the email index; uq_users_tenant_id_id (tenant_id, id) is also
		// unique on users, so a blanket mapping would misreport other violations.
		if isUniqueViolation(err) && violatesConstraint(err, "uq_users_tenant_id_email") {
			return domainerrors.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

// GetByID looks up a user by id, scoped to the given tenantID.
func (r *gormUserRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.User, error) {
	var user entity.User
	if err := database.DBFromContext(c, r.db).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByTenantAndEmail looks up a user by (tenant, email).
func (r *gormUserRepository) GetByTenantAndEmail(c context.Context, tenantID uuid.UUID, email string) (*entity.User, error) {
	var user entity.User
	if err := database.DBFromContext(c, r.db).
		Where("tenant_id = ? AND email = ? AND deleted_at IS NULL", tenantID, email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByIDWithRoles is GetByID plus the user's assigned roles.
func (r *gormUserRepository) GetByIDWithRoles(c context.Context, tenantID, id uuid.UUID) (*entity.User, error) {
	user, err := r.GetByID(c, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := r.loadRoles(c, user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetByTenantAndEmailWithRoles is GetByTenantAndEmail plus the user's assigned roles.
func (r *gormUserRepository) GetByTenantAndEmailWithRoles(c context.Context, tenantID uuid.UUID, email string) (*entity.User, error) {
	user, err := r.GetByTenantAndEmail(c, tenantID, email)
	if err != nil {
		return nil, err
	}
	if err := r.loadRoles(c, user); err != nil {
		return nil, err
	}
	return user, nil
}

// loadRoles populates user.Roles via the user_roles join table.
//
// This is a manual join rather than a GORM many2many association because
// entity.User carries no GORM tags by design (the domain layer stays
// framework-agnostic per Clean Architecture). The join additionally filters
// on user_roles.tenant_id (added by
// migrations/000013_add_tenant_id_to_user_roles.up.sql specifically to
// enforce this at the DB level) as defense-in-depth per CLAUDE.md
// Invariant 1, even though the caller has already scoped the user lookup
// itself.
func (r *gormUserRepository) loadRoles(c context.Context, user *entity.User) error {
	var roles []entity.Role
	if err := database.DBFromContext(c, r.db).
		Table("roles").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND user_roles.tenant_id = ? AND roles.deleted_at IS NULL", user.ID, user.TenantID).
		Find(&roles).Error; err != nil {
		return err
	}
	user.Roles = roles
	return nil
}

// Update persists changes to an existing user, scoped to the given tenantID.
// Select("*") + Omit(...) forces every field (including zero-valued ones,
// e.g. clearing PasswordHash) to be written, since GORM's struct-based
// Updates otherwise silently skips zero values.
func (r *gormUserRepository) Update(c context.Context, tenantID uuid.UUID, user *entity.User) error {
	result := database.DBFromContext(c, r.db).
		Model(&entity.User{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", user.ID, tenantID).
		Select("*").
		Omit("ID", "TenantID", "CreatedAt", "CreatedBy", "DeletedAt", "DeletedBy").
		Updates(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrUserNotFound
	}
	return nil
}

// Delete soft-deletes a user (sets deleted_at/deleted_by), scoped to the given
// tenantID. actorID is the admin performing the removal. Already-deleted rows
// are treated as not found.
func (r *gormUserRepository) Delete(c context.Context, tenantID, id, actorID uuid.UUID) error {
	now := time.Now().UTC()
	result := database.DBFromContext(c, r.db).
		Model(&entity.User{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(map[string]any{"deleted_at": now, "deleted_by": actorID, "updated_at": now, "updated_by": actorID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrUserNotFound
	}
	return nil
}

// List returns a page of users for the given tenantID, along with the total
// row count for that tenant.
func (r *gormUserRepository) List(c context.Context, tenantID uuid.UUID, limit, offset int) ([]*entity.User, int64, error) {
	var total int64
	if err := database.DBFromContext(c, r.db).
		Model(&entity.User{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []*entity.User
	if err := database.DBFromContext(c, r.db).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
