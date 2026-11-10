package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormDashboardRepository is a GORM-backed, read-only DashboardRepository.
type gormDashboardRepository struct {
	db *gorm.DB
}

// NewGormDashboardRepository constructs a GORM-backed DashboardRepository.
func NewGormDashboardRepository(db *gorm.DB) repository.DashboardRepository {
	return &gormDashboardRepository{db: db}
}

var _ repository.DashboardRepository = (*gormDashboardRepository)(nil)

type invitationCountRow struct {
	Pending  int64
	Accepted int64
	Expired  int64
	Revoked  int64
}

func (r invitationCountRow) toEntity() entity.InvitationStatusCounts {
	return entity.InvitationStatusCounts{Pending: r.Pending, Accepted: r.Accepted, Expired: r.Expired, Revoked: r.Revoked}
}

// invitationCountSelect derives each status exactly like UserInvitation.Status
// (accepted wins over revoked wins over expired).
const invitationCountSelect = `
	COUNT(*) FILTER (WHERE accepted_at IS NULL AND revoked_at IS NULL AND expires_at > @now) AS pending,
	COUNT(*) FILTER (WHERE accepted_at IS NOT NULL) AS accepted,
	COUNT(*) FILTER (WHERE accepted_at IS NULL AND revoked_at IS NOT NULL) AS revoked,
	COUNT(*) FILTER (WHERE accepted_at IS NULL AND revoked_at IS NULL AND expires_at <= @now) AS expired`

func (r *gormDashboardRepository) invitationCounts(db *gorm.DB, tenantID *uuid.UUID, now time.Time) (entity.InvitationStatusCounts, error) {
	var row invitationCountRow
	q := db.Table("user_invitations").Select(invitationCountSelect, map[string]any{"now": now}).
		Where("deleted_at IS NULL")
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if err := q.Scan(&row).Error; err != nil {
		return entity.InvitationStatusCounts{}, err
	}
	return row.toEntity(), nil
}

func (r *gormDashboardRepository) countTenantRows(db *gorm.DB, table string, tenantID uuid.UUID) (int64, error) {
	var n int64
	err := db.Table(table).Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Count(&n).Error
	return n, err
}

func (r *gormDashboardRepository) TenantCounts(c context.Context, tenantID uuid.UUID, now time.Time) (*entity.TenantDashboardCounts, error) {
	db := database.DBFromContext(c, r.db)
	out := &entity.TenantDashboardCounts{}

	// pending_invited: an inactive user with no password that still has an open
	// (not deleted, accepted, revoked or expired) invitation for the same email
	// in the same tenant. Deactivated users without an open invitation are not
	// counted. Email match is exact: InviteUserUseCase lowercases/trims emails
	// before storing both the user and the invitation.
	var users struct {
		Total          int64
		Active         int64
		PendingInvited int64
	}
	if err := db.Table("users").Select(`
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE is_active) AS active,
		COUNT(*) FILTER (WHERE NOT is_active AND password_hash IS NULL AND EXISTS (
			SELECT 1 FROM user_invitations i
			WHERE i.tenant_id = users.tenant_id AND i.email = users.email
				AND i.deleted_at IS NULL AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > ?)) AS pending_invited`, now).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Scan(&users).Error; err != nil {
		return nil, err
	}
	out.UsersTotal, out.UsersActive, out.UsersPendingInvited = users.Total, users.Active, users.PendingInvited

	inv, err := r.invitationCounts(db, &tenantID, now)
	if err != nil {
		return nil, err
	}
	out.Invitations = inv

	if out.Departments, err = r.countTenantRows(db, "departments", tenantID); err != nil {
		return nil, err
	}
	if out.Positions, err = r.countTenantRows(db, "positions", tenantID); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *gormDashboardRepository) UsersByRole(c context.Context, tenantID uuid.UUID) ([]entity.RoleUserCount, error) {
	var rows []struct {
		Role     string
		Total    int64
		Active   int64
		Inactive int64
	}
	if err := database.DBFromContext(c, r.db).Table("user_roles AS ur").
		Select(`ro.name AS role, COUNT(DISTINCT u.id) AS total,
			COUNT(DISTINCT u.id) FILTER (WHERE u.is_active) AS active,
			COUNT(DISTINCT u.id) FILTER (WHERE NOT u.is_active) AS inactive`).
		Joins("JOIN users u ON u.id = ur.user_id AND u.tenant_id = ur.tenant_id AND u.deleted_at IS NULL").
		Joins("JOIN roles ro ON ro.id = ur.role_id AND ro.tenant_id = ur.tenant_id AND ro.deleted_at IS NULL").
		Where("ur.tenant_id = ?", tenantID).
		Group("ro.name").Order("ro.name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]entity.RoleUserCount, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.RoleUserCount{Role: row.Role, Total: row.Total, Active: row.Active, Inactive: row.Inactive})
	}
	return out, nil
}
