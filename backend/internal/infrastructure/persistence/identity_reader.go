package persistence

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormIdentityReader implements repository.IdentityReader with a single query
// joining tenants, users, user_roles and roles.
type gormIdentityReader struct {
	db *gorm.DB
}

// NewGormIdentityReader constructs a GORM-backed IdentityReader.
func NewGormIdentityReader(db *gorm.DB) repository.IdentityReader {
	return &gormIdentityReader{db: db}
}

var _ repository.IdentityReader = (*gormIdentityReader)(nil)

type identityRow struct {
	TenantActive bool
	UserID       *uuid.UUID
	UserActive   *bool
	RoleName     *string
}

// GetIdentityState returns one row per role (or a single row with NULL role
// when the user has none). The user join is constrained to the tenant, so a
// user id from another tenant yields UserFound == false.
func (r *gormIdentityReader) GetIdentityState(c context.Context, tenantID, userID uuid.UUID) (*repository.IdentityState, error) {
	var rows []identityRow
	err := database.DBFromContext(c, r.db).Raw(`
SELECT t.is_active AS tenant_active,
       u.id        AS user_id,
       u.is_active AS user_active,
       r.name      AS role_name
FROM tenants t
LEFT JOIN users u       ON u.id = ? AND u.tenant_id = t.id AND u.deleted_at IS NULL
LEFT JOIN user_roles ur ON ur.user_id = u.id AND ur.tenant_id = t.id
LEFT JOIN roles r       ON r.id = ur.role_id AND r.tenant_id = t.id AND r.deleted_at IS NULL
WHERE t.id = ? AND t.deleted_at IS NULL
ORDER BY r.name`, userID, tenantID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domainerrors.ErrTenantNotFound
	}

	state := &repository.IdentityState{
		TenantActive: rows[0].TenantActive,
		UserFound:    rows[0].UserID != nil,
		RoleNames:    []string{},
	}
	if state.UserFound {
		state.UserActive = rows[0].UserActive != nil && *rows[0].UserActive
		for _, row := range rows {
			if row.RoleName != nil {
				state.RoleNames = append(state.RoleNames, *row.RoleName)
			}
		}
	}
	return state, nil
}
