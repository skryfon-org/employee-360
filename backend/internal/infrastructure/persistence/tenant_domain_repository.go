package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

// gormTenantDomainRepository is a GORM-backed adapter implementing
// repository.TenantDomainRepository.
//
// tenant_domains is the tenant-resolution table itself: FindTenantByDomain is
// the (only) pre-authentication lookup that derives a tenant from server-side
// data (the email's domain), so it is intentionally not scoped by a context
// tenant. No other methods exist: unscoped mutations/reads are deliberately
// not exposed.
type gormTenantDomainRepository struct {
	db *gorm.DB
}

// NewGormTenantDomainManager constructs the super_admin management adapter
// over tenant_domains (same underlying implementation).
func NewGormTenantDomainManager(db *gorm.DB) repository.TenantDomainManager {
	return &gormTenantDomainRepository{db: db}
}

// NewGormTenantDomainRepository constructs a GORM-backed TenantDomainRepository.
func NewGormTenantDomainRepository(db *gorm.DB) repository.TenantDomainRepository {
	return &gormTenantDomainRepository{db: db}
}

var _ repository.TenantDomainRepository = (*gormTenantDomainRepository)(nil)
var _ repository.TenantDomainManager = (*gormTenantDomainRepository)(nil)

// FindTenantByDomain returns the active tenant that owns the given email domain.
func (r *gormTenantDomainRepository) FindTenantByDomain(c context.Context, domain string) (*entity.Tenant, error) {
	var tenant entity.Tenant
	if err := database.DBFromContext(c, r.db).
		Table("tenants").
		Joins("JOIN tenant_domains ON tenant_domains.tenant_id = tenants.id").
		Where("tenant_domains.domain = ? AND tenants.is_active = TRUE AND tenants.deleted_at IS NULL AND tenant_domains.deleted_at IS NULL", strings.ToLower(domain)).
		First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// DomainBelongsToTenant reports whether domain is a live (not soft-deleted)
// domain of the given active, not soft-deleted tenant. tenant_domains has no
// is_active/verified column.
func (r *gormTenantDomainRepository) DomainBelongsToTenant(c context.Context, tenantID uuid.UUID, domain string) (bool, error) {
	var n int64
	if err := database.DBFromContext(c, r.db).
		Table("tenant_domains").
		Joins("JOIN tenants ON tenants.id = tenant_domains.tenant_id").
		Where("tenant_domains.tenant_id = ? AND tenant_domains.domain = ? AND tenant_domains.deleted_at IS NULL AND tenants.is_active = TRUE AND tenants.deleted_at IS NULL",
			tenantID, strings.ToLower(strings.TrimSpace(domain))).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// Create registers the domain. tenant_domains carries a UNIQUE (domain)
// constraint that also covers soft-deleted rows, so a previously removed
// domain is revived (re-pointed at the tenant) instead of inserted; this
// keeps "globally unique among live rows" without a schema change. A live
// owner (or a concurrent writer winning the race) yields ErrDomainAlreadyExists.
func (r *gormTenantDomainRepository) Create(c context.Context, d *entity.TenantDomain) error {
	db := database.DBFromContext(c, r.db)
	var revived []uuid.UUID
	if err := db.Raw(`UPDATE tenant_domains
SET tenant_id = ?, deleted_at = NULL, deleted_by = NULL,
    created_by = ?, updated_by = ?, created_at = ?, updated_at = ?
WHERE domain = ? AND deleted_at IS NOT NULL
RETURNING id`, d.TenantID, d.CreatedBy, d.CreatedBy, d.CreatedAt, d.UpdatedAt, d.Domain).
		Scan(&revived).Error; err != nil {
		return err
	}
	if len(revived) == 1 {
		d.ID = revived[0]
		return nil
	}
	res := db.Table("tenant_domains").Clauses(clause.OnConflict{DoNothing: true}).Create(d)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrDomainAlreadyExists
	}
	return nil
}

func (r *gormTenantDomainRepository) ListByTenantID(c context.Context, tenantID uuid.UUID) ([]*entity.TenantDomain, error) {
	var out []*entity.TenantDomain
	err := database.DBFromContext(c, r.db).Table("tenant_domains").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("created_at ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *gormTenantDomainRepository) CountByTenantID(c context.Context, tenantID uuid.UUID) (int64, error) {
	var n int64
	err := database.DBFromContext(c, r.db).Table("tenant_domains").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Count(&n).Error
	return n, err
}

func (r *gormTenantDomainRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.TenantDomain, error) {
	var d entity.TenantDomain
	if err := database.DBFromContext(c, r.db).Table("tenant_domains").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrDomainNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *gormTenantDomainRepository) SoftDelete(c context.Context, tenantID, id, actorID uuid.UUID, at time.Time) error {
	res := database.DBFromContext(c, r.db).Table("tenant_domains").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(map[string]any{"deleted_at": at, "deleted_by": actorID, "updated_at": at, "updated_by": actorID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrDomainNotFound
	}
	return nil
}

// CountUsersOnDomain counts the tenant's non-soft-deleted users (any
// is_active state) whose email domain, i.e. the part after the last '@',
// equals domain case-insensitively. Exact comparison, not a suffix match, so
// evilacme.com does not count against acme.com.
func (r *gormTenantDomainRepository) CountUsersOnDomain(c context.Context, tenantID uuid.UUID, domain string) (int64, error) {
	var n int64
	err := database.DBFromContext(c, r.db).Table("users").
		Where("tenant_id = ? AND deleted_at IS NULL AND lower(substring(email from '[^@]*$')) = ?",
			tenantID, strings.ToLower(strings.TrimSpace(domain))).
		Count(&n).Error
	return n, err
}

// tombstonePlaceholderSuffix marks a soft-deleted domain row whose value was
// handed to another row. "<domain>#deleted-<id>" is not a valid hostname, so
// it can never be matched by a lookup nor revived by Create.
const tombstonePlaceholderSuffix = "#deleted-"

// UpdateDomain changes the value of the tenant's own live domain. UNIQUE
// (domain) also covers soft-deleted rows, so a tombstone (another row, already
// soft-deleted) that still holds the target value is not destroyed but renamed
// to a non-colliding placeholder (domain || '#deleted-' || id), keeping its
// audit/history; this happens only once the caller's own live row is
// confirmed. A live owner surfaces as ErrDomainAlreadyExists. A row of another
// tenant, or a soft-deleted one, is reported as ErrDomainNotFound (existence
// not leaked).
func (r *gormTenantDomainRepository) UpdateDomain(c context.Context, tenantID, id uuid.UUID, domain string, actorID uuid.UUID, at time.Time) error {
	db := database.DBFromContext(c, r.db)
	if _, err := r.GetByID(c, tenantID, id); err != nil {
		return err
	}
	if err := db.Exec(`UPDATE tenant_domains SET domain = domain || '`+tombstonePlaceholderSuffix+`' || id::text
WHERE domain = ? AND deleted_at IS NOT NULL AND id <> ?`, domain, id).Error; err != nil {
		return err
	}
	res := db.Table("tenant_domains").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(map[string]any{"domain": domain, "updated_at": at, "updated_by": actorID})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return domainerrors.ErrDomainAlreadyExists
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrDomainNotFound
	}
	return nil
}
