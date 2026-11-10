package seeder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

// bootstrapLockKey is the Postgres advisory-lock key serializing concurrent
// bootstrap runs, so two simultaneous runs cannot both miss the tenant
// lookup and insert duplicates (tenants has no unique key on name).
const bootstrapLockKey int64 = 0x45333630 // "E360"

// PasswordPlaceholder is the literal value shipped in .env.example; it is
// rejected so it can never become a real credential.
const PasswordPlaceholder = "<CHANGE_ME>"

// Default names used when the corresponding option is left empty.
const (
	DefaultSystemTenantName = "System"
	superAdminFirstName     = "Platform"
	superAdminLastName      = "Super Admin"
)

// ErrSuperAdminEmailRequired and ErrSuperAdminPasswordRequired are returned
// when the bootstrap credentials are not configured. There are deliberately
// no defaults: credentials must come from configuration/environment.
var (
	ErrSuperAdminEmailRequired       = errors.New("bootstrap: super admin email is not set (BOOTSTRAP_SUPER_ADMIN_EMAIL)")
	ErrSuperAdminPasswordRequired    = errors.New("bootstrap: super admin password is not set (BOOTSTRAP_SUPER_ADMIN_PASSWORD)")
	ErrSuperAdminPasswordTooLong     = errors.New("bootstrap: super admin password must be at most 72 bytes (bcrypt limit)")
	ErrSuperAdminPasswordPlaceholder = errors.New("bootstrap: super admin password is still the .env.example placeholder; set a real password")
	ErrSuperAdminEmailInvalid        = errors.New("bootstrap: super admin email must contain exactly one '@' with a non-empty local part and domain")
)

// Options configures a bootstrap run.
type Options struct {
	SystemTenantName   string
	SuperAdminEmail    string
	SuperAdminPassword string
}

// Result reports what a bootstrap run did, for logging.
type Result struct {
	TenantID       uuid.UUID
	TenantCreated  bool
	RolesCreated   []string
	AdminID        uuid.UUID
	AdminCreated   bool
	AdminRoleAdded bool
	DomainCreated  bool
	// CategoriesCreated lists the default holiday category names inserted by
	// this run (empty when they already existed).
	CategoriesCreated []string
	// LeaveTypesCreated lists the default leave type codes inserted by this run.
	LeaveTypesCreated []string
}

// DefaultLeaveType is a leave type seeded for the system tenant.
type DefaultLeaveType struct {
	Name        string
	Code        string
	Description string
}

// DefaultLeaveTypes returns the leave types seeded per tenant (a fresh slice
// each call). Migration 000033 seeds the same set for pre-existing tenants.
func DefaultLeaveTypes() []DefaultLeaveType {
	return []DefaultLeaveType{
		{Name: "Casual", Code: "CASUAL", Description: "Short-notice personal leave."},
		{Name: "Sick", Code: "SICK", Description: "Leave for illness or medical appointments."},
		{Name: "Earned", Code: "EARNED", Description: "Planned annual or earned leave."},
		{Name: "Unpaid", Code: "UNPAID", Description: "Leave without pay."},
	}
}

// DefaultHolidayCategory is a holiday category seeded for the system tenant.
type DefaultHolidayCategory struct {
	Name        string
	Description string
	IsPublic    bool   // holidays in this category are company days off, excluded from leave
	Color       string // #RRGGBB (chk_holiday_categories_color_hex); design-system category dot colours
}

// DefaultHolidayCategories returns the holiday categories seeded for the
// system tenant (a fresh slice each call).
func DefaultHolidayCategories() []DefaultHolidayCategory {
	return []DefaultHolidayCategory{
		{Name: "Public Holiday", Description: "Company-wide holiday observed by all employees.", Color: "#15803D", IsPublic: true},
		{Name: "Optional Holiday", Description: "Holiday employees may choose to observe.", Color: "#B45309"},
	}
}

// Validate normalizes the options and checks required credentials.
func (o *Options) Validate() error {
	o.SystemTenantName = strings.TrimSpace(o.SystemTenantName)
	if o.SystemTenantName == "" {
		o.SystemTenantName = DefaultSystemTenantName
	}
	o.SuperAdminEmail = strings.ToLower(strings.TrimSpace(o.SuperAdminEmail))
	if o.SuperAdminEmail == "" {
		return ErrSuperAdminEmailRequired
	}
	if _, err := emailDomain(o.SuperAdminEmail); err != nil {
		return err
	}
	if o.SuperAdminPassword == "" {
		return ErrSuperAdminPasswordRequired
	}
	if strings.TrimSpace(o.SuperAdminPassword) == PasswordPlaceholder {
		return ErrSuperAdminPasswordPlaceholder
	}
	if len(o.SuperAdminPassword) > 72 {
		return ErrSuperAdminPasswordTooLong
	}
	return nil
}

// emailDomain returns the lowercased domain part of an email that has exactly
// one '@' and non-empty local and domain parts.
func emailDomain(email string) (string, error) {
	if strings.Count(email, "@") != 1 {
		return "", ErrSuperAdminEmailInvalid
	}
	local, domain, _ := strings.Cut(email, "@")
	domain = strings.ToLower(strings.TrimSpace(domain))
	if strings.TrimSpace(local) == "" || domain == "" {
		return "", ErrSuperAdminEmailInvalid
	}
	return domain, nil
}

// Seeder bootstraps the system tenant, default roles and platform Super Admin.
type Seeder struct {
	db     *gorm.DB
	hasher service.HashService
}

// New constructs a Seeder.
func New(db *gorm.DB, hasher service.HashService) *Seeder {
	return &Seeder{db: db, hasher: hasher}
}

// DefaultRoles are the fixed roles seeded for the system tenant.
func DefaultRoles() []string {
	return []string{entity.RoleSuperAdmin, entity.RoleAdmin, entity.RoleEmployee}
}

// Run seeds everything in a single transaction. It is idempotent: existing
// rows are reused and never overwritten (an existing super admin keeps its
// current password), so re-running creates no duplicates.
func (s *Seeder) Run(ctx context.Context, opts Options) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// Hash before opening the transaction (bcrypt is slow) and before any
	// DB work, so a hashing failure changes nothing.
	hash, err := s.hasher.HashPassword(opts.SuperAdminPassword)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: hash super admin password: %w", err)
	}

	// Login resolves the tenant from the email domain, so the super admin's
	// domain must map to the system tenant.
	domain, _ := emailDomain(opts.SuperAdminEmail) // validated above

	res := &Result{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", bootstrapLockKey).Error; err != nil {
			return fmt.Errorf("acquire bootstrap lock: %w", err)
		}

		tenant, created, err := s.ensureTenant(tx, opts.SystemTenantName, domain)
		if err != nil {
			return err
		}
		res.TenantID, res.TenantCreated = tenant.ID, created

		domainCreated, err := s.ensureTenantDomain(tx, tenant.ID, domain)
		if err != nil {
			return err
		}
		res.DomainCreated = domainCreated

		roleIDs := make(map[string]uuid.UUID, 3)
		for _, name := range DefaultRoles() {
			role, created, err := s.ensureRole(tx, tenant.ID, name)
			if err != nil {
				return err
			}
			roleIDs[name] = role.ID
			if created {
				res.RolesCreated = append(res.RolesCreated, name)
			}
		}

		user, created, err := s.ensureSuperAdmin(tx, tenant.ID, opts.SuperAdminEmail, hash)
		if err != nil {
			return err
		}
		res.AdminID, res.AdminCreated = user.ID, created

		added, err := s.ensureUserRole(tx, tenant.ID, user.ID, roleIDs[entity.RoleSuperAdmin])
		if err != nil {
			return err
		}
		res.AdminRoleAdded = added

		for _, c := range DefaultHolidayCategories() {
			created, err := s.ensureHolidayCategory(tx, tenant.ID, c)
			if err != nil {
				return err
			}
			if created {
				res.CategoriesCreated = append(res.CategoriesCreated, c.Name)
			}
		}
		for _, lt := range DefaultLeaveTypes() {
			created, err := s.ensureLeaveType(tx, tenant.ID, lt)
			if err != nil {
				return err
			}
			if created {
				res.LeaveTypesCreated = append(res.LeaveTypesCreated, lt.Code)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ensureTenant finds the system tenant. tenants.name is not unique, so among
// tenants with the configured name it prefers the one that owns the super
// admin's email domain in tenant_domains (what login resolves through), then
// the oldest. If none match, it reuses an existing tenant of any name only when
// the tenants table holds exactly one (never seed a second tenant); with more
// than one unmatched tenant it returns an error asking for explicit config, so
// the super admin and roles are never attached to an arbitrary customer tenant.
// It creates a tenant only when the table is empty.
func (s *Seeder) ensureTenant(tx *gorm.DB, name, domain string) (*entity.Tenant, bool, error) {
	var t entity.Tenant
	// Prefer the same-named tenant that owns the domain. Session gives a clean
	// statement rather than one carrying state from earlier calls on tx.
	err := tx.Session(&gorm.Session{}).
		Where("name = ? AND EXISTS (SELECT 1 FROM tenant_domains d WHERE d.tenant_id = tenants.id AND d.domain = ?)", name, domain).
		Order("created_at ASC").First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = tx.Session(&gorm.Session{}).Where("name = ?", name).Order("created_at ASC").First(&t).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var total int64
		if cerr := tx.Session(&gorm.Session{}).Model(&entity.Tenant{}).Count(&total).Error; cerr != nil {
			return nil, false, fmt.Errorf("lookup system tenant: %w", cerr)
		}
		switch {
		case total > 1:
			return nil, false, fmt.Errorf("system tenant %q not found and %d other tenants exist: set the system tenant name (or super admin email domain) to match the intended tenant explicitly", name, total)
		case total == 1:
			// Never seed a second tenant: reuse the only one, whatever its name.
			err = tx.Session(&gorm.Session{}).First(&t).Error
		}
	}
	if err == nil {
		return &t, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("lookup system tenant: %w", err)
	}
	now := time.Now().UTC()
	t = entity.Tenant{ID: uuid.New(), Name: name, IsActive: true, CreatedAt: now, UpdatedAt: now}
	if err := tx.Create(&t).Error; err != nil {
		return nil, false, fmt.Errorf("create system tenant: %w", err)
	}
	return &t, true, nil
}

// ensureTenantDomain registers domain on the tenant. The domain is unique
// across all tenants, so if another tenant already owns it, login would not
// resolve to this tenant and we fail loudly instead of silently succeeding.
func (s *Seeder) ensureTenantDomain(tx *gorm.DB, tenantID uuid.UUID, domain string) (bool, error) {
	now := time.Now().UTC()
	td := entity.TenantDomain{ID: uuid.New(), TenantID: tenantID, Domain: domain, CreatedAt: now, UpdatedAt: now}
	res := tx.Table("tenant_domains").Clauses(clause.OnConflict{DoNothing: true}).Create(&td)
	if res.Error != nil {
		return false, fmt.Errorf("create tenant domain %q: %w", domain, res.Error)
	}
	if res.RowsAffected == 1 {
		return true, nil
	}
	var owner entity.TenantDomain
	if err := tx.Table("tenant_domains").Where("domain = ?", domain).First(&owner).Error; err != nil {
		return false, fmt.Errorf("lookup tenant domain %q: %w", domain, err)
	}
	if owner.TenantID != tenantID {
		return false, fmt.Errorf("bootstrap: domain %s already belongs to another tenant", domain)
	}
	return false, nil
}

func (s *Seeder) ensureRole(tx *gorm.DB, tenantID uuid.UUID, name string) (*entity.Role, bool, error) {
	now := time.Now().UTC()
	role := entity.Role{ID: uuid.New(), TenantID: tenantID, Name: name, CreatedAt: now, UpdatedAt: now}
	// Unique key: (tenant_id, name).
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&role)
	if res.Error != nil {
		return nil, false, fmt.Errorf("create role %q: %w", name, res.Error)
	}
	if res.RowsAffected == 1 {
		return &role, true, nil
	}
	var existing entity.Role
	if err := tx.Where("tenant_id = ? AND name = ?", tenantID, name).First(&existing).Error; err != nil {
		return nil, false, fmt.Errorf("lookup role %q: %w", name, err)
	}
	return &existing, false, nil
}

func (s *Seeder) ensureSuperAdmin(tx *gorm.DB, tenantID uuid.UUID, email, hash string) (*entity.User, bool, error) {
	now := time.Now().UTC()
	user := entity.User{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Email:           email,
		FirstName:       superAdminFirstName,
		LastName:        superAdminLastName,
		PasswordHash:    &hash,
		EmailVerifiedAt: &now,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	// Unique key: (tenant_id, email).
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&user)
	if res.Error != nil {
		return nil, false, fmt.Errorf("create super admin: %w", res.Error)
	}
	if res.RowsAffected == 1 {
		return &user, true, nil
	}
	var existing entity.User
	if err := tx.Where("tenant_id = ? AND email = ?", tenantID, email).First(&existing).Error; err != nil {
		return nil, false, fmt.Errorf("lookup super admin: %w", err)
	}
	return &existing, false, nil
}

func (s *Seeder) ensureUserRole(tx *gorm.DB, tenantID, userID, roleID uuid.UUID) (bool, error) {
	now := time.Now().UTC()
	ur := entity.UserRole{ID: uuid.New(), TenantID: tenantID, UserID: userID, RoleID: roleID, CreatedAt: now, UpdatedAt: now}
	// Unique key: (user_id, role_id).
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ur)
	if res.Error != nil {
		return false, fmt.Errorf("assign super_admin role: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ensureHolidayCategory inserts a default category unless the tenant already
// has a non-deleted one with the same name (case-insensitive). A default an admin
// soft-deleted is therefore recreated on the next run. Existing rows
// are never overwritten, so admin edits survive re-runs.
func (s *Seeder) ensureHolidayCategory(tx *gorm.DB, tenantID uuid.UUID, c DefaultHolidayCategory) (bool, error) {
	// Unique key: (tenant_id, LOWER(name)) WHERE deleted_at IS NULL. A
	// conflict target-less DO NOTHING covers that partial expression index.
	res := tx.Exec(`INSERT INTO holiday_categories (id, tenant_id, name, description, color, is_public)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
		uuid.New(), tenantID, c.Name, c.Description, c.Color, c.IsPublic)
	if res.Error != nil {
		return false, fmt.Errorf("create holiday category %q: %w", c.Name, res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ensureLeaveType inserts a default leave type unless the tenant already has a
// non-deleted one with the same code. Existing rows are never overwritten, so
// admin edits (and deactivation) survive re-runs.
func (s *Seeder) ensureLeaveType(tx *gorm.DB, tenantID uuid.UUID, lt DefaultLeaveType) (bool, error) {
	// Unique key: (tenant_id, code) WHERE deleted_at IS NULL (partial index).
	res := tx.Exec(`INSERT INTO leave_types (id, tenant_id, name, code, description)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
		uuid.New(), tenantID, lt.Name, lt.Code, lt.Description)
	if res.Error != nil {
		return false, fmt.Errorf("create leave type %q: %w", lt.Code, res.Error)
	}
	return res.RowsAffected == 1, nil
}
