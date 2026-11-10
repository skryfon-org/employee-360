package persistence

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormAuditLogQueryRepository implements repository.AuditLogQueryRepository.
// Every query is scoped by the explicit tenantID, and the actor join is also
// constrained to the same tenant.
type gormAuditLogQueryRepository struct {
	db *gorm.DB
}

// NewGormAuditLogQueryRepository constructs a GORM-backed AuditLogQueryRepository.
func NewGormAuditLogQueryRepository(db *gorm.DB) repository.AuditLogQueryRepository {
	return &gormAuditLogQueryRepository{db: db}
}

var _ repository.AuditLogQueryRepository = (*gormAuditLogQueryRepository)(nil)

type auditLogRow struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	ActorUserID    *uuid.UUID
	Action         string
	EntityType     string
	EntityID       *uuid.UUID
	Metadata       string
	CreatedAt      time.Time
	ActorEmail     *string
	ActorFirstName *string
	ActorLastName  *string
}

func (r auditLogRow) toEntry() *entity.AuditLogEntry {
	e := &entity.AuditLogEntry{
		ID: r.ID, TenantID: r.TenantID, Action: r.Action, EntityType: r.EntityType,
		EntityID: r.EntityID, Metadata: r.Metadata, CreatedAt: r.CreatedAt,
	}
	if r.ActorUserID != nil {
		a := &entity.AuditActor{ID: *r.ActorUserID}
		if r.ActorEmail != nil {
			a.Email = *r.ActorEmail
		}
		if r.ActorFirstName != nil {
			a.FirstName = *r.ActorFirstName
		}
		if r.ActorLastName != nil {
			a.LastName = *r.ActorLastName
		}
		e.Actor = a
	}
	return e
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (r *gormAuditLogQueryRepository) base(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Table("audit_logs AS a").Where("a.tenant_id = ?", tenantID)
}

func applyAuditFilter(q *gorm.DB, f repository.AuditLogFilter) *gorm.DB {
	if f.Action != "" {
		if f.ActionPrefix {
			q = q.Where(`a.action LIKE ? ESCAPE '\'`, escapeLike(f.Action)+"%")
		} else {
			q = q.Where("a.action = ?", f.Action)
		}
	}
	if f.ActorUserID != nil {
		q = q.Where("a.actor_user_id = ?", *f.ActorUserID)
	}
	if f.EntityType != "" {
		q = q.Where("a.entity_type = ?", f.EntityType)
	}
	if f.From != nil {
		q = q.Where("a.created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("a.created_at <= ?", *f.To)
	}
	return q
}

const auditSelect = "a.id, a.tenant_id, a.actor_user_id, a.action, a.entity_type, a.entity_id, a.metadata::text AS metadata, a.created_at, " +
	"u.email AS actor_email, u.first_name AS actor_first_name, u.last_name AS actor_last_name"

const auditActorJoin = "LEFT JOIN users u ON u.id = a.actor_user_id AND u.tenant_id = a.tenant_id"

func (r *gormAuditLogQueryRepository) List(c context.Context, tenantID uuid.UUID, f repository.AuditLogFilter, limit, offset int) ([]*entity.AuditLogEntry, int64, error) {
	var total int64
	if err := applyAuditFilter(r.base(c, tenantID), f).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := applyAuditFilter(r.base(c, tenantID), f).
		Select(auditSelect).Joins(auditActorJoin).
		Order("a.created_at DESC, a.id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	var rows []auditLogRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*entity.AuditLogEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toEntry())
	}
	return out, total, nil
}

func (r *gormAuditLogQueryRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.AuditLogEntry, error) {
	var rows []auditLogRow
	if err := r.base(c, tenantID).Where("a.id = ?", id).
		Select(auditSelect).Joins(auditActorJoin).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domainerrors.ErrAuditLogNotFound
	}
	return rows[0].toEntry(), nil
}
