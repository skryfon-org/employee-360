package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormEmployeeProfileRepository implements repository.EmployeeProfileRepository.
// Every query is scoped by the explicit tenantID and excludes soft-deleted rows.
type gormEmployeeProfileRepository struct {
	db *gorm.DB
}

// NewGormEmployeeProfileRepository constructs a GORM-backed EmployeeProfileRepository.
func NewGormEmployeeProfileRepository(db *gorm.DB) repository.EmployeeProfileRepository {
	return &gormEmployeeProfileRepository{db: db}
}

var _ repository.EmployeeProfileRepository = (*gormEmployeeProfileRepository)(nil)

func (r *gormEmployeeProfileRepository) GetByUserID(c context.Context, tenantID, userID uuid.UUID) (*entity.EmployeeProfile, error) {
	var p entity.EmployeeProfile
	err := database.DBFromContext(c, r.db).
		Where("tenant_id = ? AND user_id = ? AND deleted_at IS NULL", tenantID, userID).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrEmployeeProfileNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *gormEmployeeProfileRepository) Create(c context.Context, actorID uuid.UUID, p *entity.EmployeeProfile) error {
	if p.TenantID == uuid.Nil || p.UserID == uuid.Nil {
		return domainerrors.ErrEmployeeProfileNotFound
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	p.CreatedBy, p.UpdatedBy = &actorID, &actorID
	if p.EmploymentStatus == "" {
		p.EmploymentStatus = entity.EmploymentStatusActive
	}
	if p.EmployeeCode == nil {
		// The usecase must assign the code (see EmployeeCodeAllocator); never insert NULL.
		return domainerrors.ErrEmployeeCodeRequired
	}
	db := database.DBFromContext(c, r.db)
	if err := db.Create(p).Error; err != nil {
		if isUniqueViolation(err) {
			// The partial unique index on (tenant_id, user_id) means a concurrent first
			// write won the race: surface a retryable conflict, not "code taken".
			if violatesConstraint(err, "uq_employee_profiles_tenant_user") {
				return domainerrors.ErrEmployeeProfileExists
			}
			return domainerrors.ErrEmployeeCodeTaken
		}
		return err
	}
	return nil
}

func (r *gormEmployeeProfileRepository) Update(c context.Context, tenantID, actorID uuid.UUID, p *entity.EmployeeProfile) error {
	p.UpdatedAt = time.Now().UTC()
	p.UpdatedBy = &actorID
	res := database.DBFromContext(c, r.db).
		Model(&entity.EmployeeProfile{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", p.ID, tenantID).
		Select("*").
		Omit("ID", "TenantID", "UserID", "CreatedAt", "CreatedBy", "DeletedAt", "DeletedBy").
		Updates(p)
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return domainerrors.ErrEmployeeCodeTaken
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrEmployeeProfileNotFound
	}
	return nil
}

func (r *gormEmployeeProfileRepository) EmployeeCodeExists(c context.Context, tenantID uuid.UUID, code string, excludeUserID uuid.UUID) (bool, error) {
	var n int64
	err := database.DBFromContext(c, r.db).
		Model(&entity.EmployeeProfile{}).
		Where("tenant_id = ? AND employee_code = ? AND user_id <> ? AND deleted_at IS NULL", tenantID, code, excludeUserID).
		Count(&n).Error
	return n > 0, err
}

// violatesConstraint reports whether err is a Postgres unique violation on the named constraint.
func violatesConstraint(err error, name string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName == name
	}
	return strings.Contains(err.Error(), name)
}
