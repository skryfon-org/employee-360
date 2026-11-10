package persistence

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormLeaveApproverRepository struct {
	db *gorm.DB
}

// NewGormLeaveApproverRepository constructs a GORM-backed LeaveApproverRepository.
func NewGormLeaveApproverRepository(db *gorm.DB) repository.LeaveApproverRepository {
	return &gormLeaveApproverRepository{db: db}
}

var _ repository.LeaveApproverRepository = (*gormLeaveApproverRepository)(nil)

type leaveApproverRow struct {
	UserID uuid.UUID
	Name   string
	Email  string
}

func (r *gormLeaveApproverRepository) ListActiveAdmins(c context.Context, tenantID uuid.UUID) ([]*entity.LeaveApprover, error) {
	out := []*entity.LeaveApprover{}
	if tenantID == uuid.Nil {
		return out, nil
	}
	var rows []leaveApproverRow
	err := database.DBFromContext(c, r.db).
		Table("users AS u").
		Select("u.id AS user_id, TRIM(u.first_name || ' ' || u.last_name) AS name, u.email AS email").
		Where("u.tenant_id = ? AND u.deleted_at IS NULL AND u.is_active = TRUE", tenantID).
		Where(`EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id AND ro.tenant_id = ur.tenant_id
			WHERE ur.user_id = u.id AND ur.tenant_id = u.tenant_id AND ro.deleted_at IS NULL
			AND ro.name IN ?)`, []string{entity.RoleAdmin, entity.RoleSuperAdmin}).
		Order("name ASC, u.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out = append(out, &entity.LeaveApprover{UserID: row.UserID, Name: row.Name, Email: row.Email, Kind: entity.LeaveApproverAdmin})
	}
	return out, nil
}

func (r *gormLeaveApproverRepository) GetActiveUser(c context.Context, tenantID, userID uuid.UUID) (*entity.LeaveApprover, bool, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return nil, false, nil
	}
	var rows []leaveApproverRow
	err := database.DBFromContext(c, r.db).
		Table("users AS u").
		Select("u.id AS user_id, TRIM(u.first_name || ' ' || u.last_name) AS name, u.email AS email").
		Where("u.tenant_id = ? AND u.id = ? AND u.deleted_at IS NULL AND u.is_active = TRUE", tenantID, userID).
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, false, err
	}
	return &entity.LeaveApprover{UserID: rows[0].UserID, Name: rows[0].Name, Email: rows[0].Email, Kind: entity.LeaveApproverManager}, true, nil
}
