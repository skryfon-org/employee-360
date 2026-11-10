package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// UserRoleRepository defines the data access methods for user-role associations.
// Reads and deletes take tenantID explicitly; AssignRole persists userRole.TenantID.
type UserRoleRepository interface {
	AssignRole(ctx context.Context, userRole *entity.UserRole) error
	RemoveRole(ctx context.Context, tenantID, userID, roleID uuid.UUID) error
	GetRolesByUserID(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.Role, error)
	GetUserRolesByUserID(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.UserRole, error)
	DeleteByUserID(ctx context.Context, tenantID, userID uuid.UUID) error
}
