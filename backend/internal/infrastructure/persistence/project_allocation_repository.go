package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormProjectAllocationRepository struct {
	db *gorm.DB
}

// NewGormProjectAllocationRepository constructs a GORM-backed ProjectAllocationRepository.
func NewGormProjectAllocationRepository(db *gorm.DB) repository.ProjectAllocationRepository {
	return &gormProjectAllocationRepository{db: db}
}

var _ repository.ProjectAllocationRepository = (*gormProjectAllocationRepository)(nil)

func (r *gormProjectAllocationRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func (r *gormProjectAllocationRepository) Create(c context.Context, tenantID, actorID uuid.UUID, a *entity.ProjectAllocation) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrAllocationNotFound
	}
	now := time.Now().UTC()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.TenantID = tenantID
	a.CreatedAt, a.UpdatedAt = now, now
	a.CreatedBy, a.UpdatedBy = &actorID, &actorID
	return database.DBFromContext(c, r.db).Create(a).Error
}

func (r *gormProjectAllocationRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.ProjectAllocation, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrAllocationNotFound
	}
	var a entity.ProjectAllocation
	if err := r.scoped(c, tenantID).Where("id = ?", id).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrAllocationNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *gormProjectAllocationRepository) ListByProject(c context.Context, tenantID, projectID uuid.UUID, scope entity.AllocationScope, asOf time.Time) ([]*entity.AllocationWithUser, error) {
	out := []*entity.AllocationWithUser{}
	if tenantID == uuid.Nil {
		return out, nil
	}
	// The users join is deliberately not filtered on deleted_at so a removed
	// user's historical allocation still shows a name; it is tenant-scoped.
	q := database.DBFromContext(c, r.db).
		Table("project_allocations AS pa").
		Select("pa.*, u.first_name AS user_first_name, u.last_name AS user_last_name").
		Joins("JOIN users u ON u.id = pa.user_id AND u.tenant_id = pa.tenant_id").
		Where("pa.tenant_id = ? AND pa.deleted_at IS NULL AND pa.project_id = ?", tenantID, projectID)
	switch scope {
	case entity.AllocationScopeActive:
		q = q.Where("pa.start_date <= ? AND (pa.end_date IS NULL OR pa.end_date >= ?)", asOf, asOf)
	case entity.AllocationScopeUpcoming:
		q = q.Where("pa.start_date > ?", asOf)
	case entity.AllocationScopePast:
		q = q.Where("pa.end_date IS NOT NULL AND pa.end_date < ?", asOf)
	case entity.AllocationScopeAll:
	default:
		return out, nil
	}
	err := q.Order("pa.start_date DESC, pa.created_at DESC, pa.id DESC").Scan(&out).Error
	return out, err
}

func (r *gormProjectAllocationRepository) ListByUserScope(c context.Context, tenantID, userID uuid.UUID, scope entity.AllocationScope, asOf time.Time) ([]*entity.AllocationWithProject, error) {
	out := []*entity.AllocationWithProject{}
	if tenantID == uuid.Nil {
		return out, nil
	}
	q := database.DBFromContext(c, r.db).
		Table("project_allocations AS pa").
		Select("pa.*, p.name AS project_name, p.code AS project_code, p.status AS project_status").
		Joins("JOIN projects p ON p.id = pa.project_id AND p.tenant_id = pa.tenant_id AND p.deleted_at IS NULL").
		Where("pa.tenant_id = ? AND pa.user_id = ? AND pa.deleted_at IS NULL", tenantID, userID)
	switch scope {
	case entity.AllocationScopeActive:
		q = q.Where("pa.start_date <= ? AND (pa.end_date IS NULL OR pa.end_date >= ?)", asOf, asOf)
	case entity.AllocationScopeUpcoming:
		q = q.Where("pa.start_date > ?", asOf)
	case entity.AllocationScopePast:
		q = q.Where("pa.end_date IS NOT NULL AND pa.end_date < ?", asOf)
	case entity.AllocationScopeAll:
	default:
		return out, nil
	}
	err := q.Order("pa.start_date ASC, p.name ASC, pa.id ASC").Scan(&out).Error
	return out, err
}

func (r *gormProjectAllocationRepository) ListActiveByUser(c context.Context, tenantID, userID uuid.UUID, asOf time.Time) ([]*entity.AllocationWithProject, error) {
	return r.ListByUserScope(c, tenantID, userID, entity.AllocationScopeActive, asOf)
}

func (r *gormProjectAllocationRepository) CountByUserScopes(c context.Context, tenantID, userID uuid.UUID, asOf time.Time) (entity.AllocationScopeCounts, error) {
	var out entity.AllocationScopeCounts
	if tenantID == uuid.Nil {
		return out, nil
	}
	var row struct{ Active, Upcoming, Past int }
	err := database.DBFromContext(c, r.db).
		Table("project_allocations AS pa").
		Select(`COUNT(*) FILTER (WHERE pa.start_date <= @d AND (pa.end_date IS NULL OR pa.end_date >= @d) AND p.status IN ('active', 'on_hold')) AS active,
			COUNT(*) FILTER (WHERE pa.start_date > @d) AS upcoming,
			COUNT(*) FILTER (WHERE pa.end_date IS NOT NULL AND pa.end_date < @d) AS past`, sql.Named("d", asOf)).
		Joins("JOIN projects p ON p.id = pa.project_id AND p.tenant_id = pa.tenant_id AND p.deleted_at IS NULL").
		Where("pa.tenant_id = ? AND pa.user_id = ? AND pa.deleted_at IS NULL", tenantID, userID).
		Scan(&row).Error
	if err != nil {
		return out, err
	}
	return entity.AllocationScopeCounts{Active: row.Active, Upcoming: row.Upcoming, Past: row.Past}, nil
}

func (r *gormProjectAllocationRepository) HasOverlap(c context.Context, tenantID, projectID, userID uuid.UUID, start time.Time, end *time.Time, excludeID uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil {
		return false, nil
	}
	// Ranges [s1,e1] and [s2,e2] intersect iff s1 <= e2 and s2 <= e1 (null end = +inf).
	q := r.scoped(c, tenantID).Model(&entity.ProjectAllocation{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Where("(end_date IS NULL OR end_date >= ?)", start)
	if end != nil {
		q = q.Where("start_date <= ?", *end)
	}
	if excludeID != uuid.Nil {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *gormProjectAllocationRepository) Update(c context.Context, tenantID, actorID uuid.UUID, a *entity.ProjectAllocation) error {
	if tenantID == uuid.Nil || a.ID == uuid.Nil {
		return domainerrors.ErrAllocationNotFound
	}
	a.UpdatedAt = time.Now().UTC()
	a.UpdatedBy = &actorID
	res := r.scoped(c, tenantID).Model(&entity.ProjectAllocation{}).Where("id = ?", a.ID).
		Updates(map[string]any{
			"allocation_percent": a.AllocationPercent,
			"role_on_project":    a.RoleOnProject,
			"start_date":         a.StartDate,
			"end_date":           a.EndDate,
			"updated_at":         a.UpdatedAt,
			"updated_by":         actorID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrAllocationNotFound
	}
	return nil
}

func (r *gormProjectAllocationRepository) SoftDelete(c context.Context, tenantID, id, actorID uuid.UUID) error {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return domainerrors.ErrAllocationNotFound
	}
	now := time.Now().UTC()
	res := r.scoped(c, tenantID).Model(&entity.ProjectAllocation{}).Where("id = ?", id).
		Updates(map[string]any{"deleted_at": now, "deleted_by": actorID, "updated_at": now, "updated_by": actorID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrAllocationNotFound
	}
	return nil
}
