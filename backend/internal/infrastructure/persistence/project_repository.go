package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormProjectRepository struct {
	db *gorm.DB
}

// NewGormProjectRepository constructs a GORM-backed ProjectRepository.
func NewGormProjectRepository(db *gorm.DB) repository.ProjectRepository {
	return &gormProjectRepository{db: db}
}

var _ repository.ProjectRepository = (*gormProjectRepository)(nil)

func (r *gormProjectRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

func (r *gormProjectRepository) Create(c context.Context, tenantID, actorID uuid.UUID, p *entity.Project) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrProjectNotFound
	}
	now := time.Now().UTC()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.TenantID = tenantID
	p.CreatedAt, p.UpdatedAt = now, now
	p.CreatedBy, p.UpdatedBy = &actorID, &actorID
	if err := database.DBFromContext(c, r.db).Create(p).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrProjectCodeTaken
		}
		return err
	}
	return nil
}

func (r *gormProjectRepository) get(c context.Context, tenantID, id uuid.UUID, lock bool) (*entity.Project, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrProjectNotFound
	}
	q := r.scoped(c, tenantID)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var p entity.Project
	if err := q.Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrProjectNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *gormProjectRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.Project, error) {
	return r.get(c, tenantID, id, false)
}

func (r *gormProjectRepository) GetByIDForUpdate(c context.Context, tenantID, id uuid.UUID) (*entity.Project, error) {
	return r.get(c, tenantID, id, true)
}

func (r *gormProjectRepository) List(c context.Context, tenantID uuid.UUID, f repository.ProjectFilter, limit, offset int) ([]*entity.Project, int64, error) {
	if tenantID == uuid.Nil {
		return []*entity.Project{}, 0, nil
	}
	db := r.scoped(c, tenantID).Model(&entity.Project{})
	if f.Status != "" {
		db = db.Where("status = ?", f.Status)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + escapeLike(strings.ToLower(s)) + "%"
		db = db.Where("(LOWER(name) LIKE ? OR LOWER(code) LIKE ? OR LOWER(COALESCE(client_name, '')) LIKE ?)", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if offset > 0 {
		db = db.Offset(offset)
	}
	var out []*entity.Project
	if err := db.Order("name ASC, id ASC").Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *gormProjectRepository) Update(c context.Context, tenantID, actorID uuid.UUID, p *entity.Project) error {
	if tenantID == uuid.Nil || p.ID == uuid.Nil {
		return domainerrors.ErrProjectNotFound
	}
	p.UpdatedAt = time.Now().UTC()
	p.UpdatedBy = &actorID
	res := r.scoped(c, tenantID).Model(&entity.Project{}).Where("id = ?", p.ID).
		Updates(map[string]any{
			"name":        p.Name,
			"code":        p.Code,
			"description": p.Description,
			"client_name": p.ClientName,
			"status":      p.Status,
			"start_date":  p.StartDate,
			"end_date":    p.EndDate,
			"updated_at":  p.UpdatedAt,
			"updated_by":  actorID,
		})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return domainerrors.ErrProjectCodeTaken
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrProjectNotFound
	}
	return nil
}

func (r *gormProjectRepository) ExistsByCode(c context.Context, tenantID uuid.UUID, code string) (bool, error) {
	if tenantID == uuid.Nil {
		return false, nil
	}
	var n int64
	err := r.scoped(c, tenantID).Model(&entity.Project{}).Where("code = ?", code).Count(&n).Error
	return n > 0, err
}
