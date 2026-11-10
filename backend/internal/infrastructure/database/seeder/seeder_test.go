//go:build integration

package seeder

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/persistence"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/service"
	"github.com/skryfon-org/employee-360/backend/internal/testutil/pgtest"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// seeder -> database -> infrastructure -> internal -> backend -> repo root
	return pgtest.Open(t, "../../../../..")
}

// setupIsolatedDB returns a handle inside a transaction that is rolled back
// when the test ends, with the tenants table emptied inside it. Run reuses any
// existing tenant, so tests that expect a fresh tenant must not see (or modify)
// real ones.
func setupIsolatedDB(t *testing.T) *gorm.DB {
	t.Helper()
	tx := setupTestDB(t).Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	require.NoError(t, tx.Exec("DELETE FROM tenants").Error)
	return tx
}

func count(t *testing.T, db *gorm.DB, model any, where string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(model).Where(where, args...).Count(&n).Error)
	return n
}

func countCategories(t *testing.T, db *gorm.DB, tenantID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table("holiday_categories").Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Count(&n).Error)
	return n
}

func TestRun_SeedsHolidayCategoriesWithoutOverwriting(t *testing.T) {
	db := setupIsolatedDB(t)
	name := "seeder-test-" + uuid.NewString()
	domain := "seeder-" + uuid.NewString() + ".example.com"
	t.Cleanup(func() { db.Where("name = ?", name).Delete(&entity.Tenant{}) })

	s := New(db, service.NewHashService(bcrypt.MinCost))
	opts := Options{SystemTenantName: name, SuperAdminEmail: "root@" + domain, SuperAdminPassword: "s3cret-pass"}
	first, err := s.Run(context.Background(), opts)
	require.NoError(t, err)

	// An admin recolours a default and lower-cases its name; the case-variant must not be duplicated.
	require.NoError(t, db.Exec("UPDATE holiday_categories SET color = '#000000', name = LOWER(name) WHERE tenant_id = ? AND name = 'Public Holiday'", first.TenantID).Error)
	second, err := s.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.Empty(t, second.CategoriesCreated)
	assert.EqualValues(t, len(DefaultHolidayCategories()), countCategories(t, db, first.TenantID))
	var color string
	require.NoError(t, db.Raw("SELECT color FROM holiday_categories WHERE tenant_id = ? AND name = 'public holiday'", first.TenantID).Scan(&color).Error)
	assert.Equal(t, "#000000", color)
}

func TestRun_IsIdempotent(t *testing.T) {
	db := setupIsolatedDB(t)
	name := "seeder-test-" + uuid.NewString()
	domain := "seeder-" + uuid.NewString() + ".example.com"
	email := "root@" + domain
	t.Cleanup(func() { db.Where("name = ?", name).Delete(&entity.Tenant{}) }) // cascades

	s := New(db, service.NewHashService(bcrypt.MinCost))
	opts := Options{SystemTenantName: name, SuperAdminEmail: email, SuperAdminPassword: "s3cret-pass"}

	first, err := s.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.True(t, first.TenantCreated)
	assert.True(t, first.AdminCreated)
	assert.True(t, first.DomainCreated)
	assert.ElementsMatch(t, DefaultRoles(), first.RolesCreated)
	assert.Len(t, first.CategoriesCreated, len(DefaultHolidayCategories()))
	assert.ElementsMatch(t, []string{"CASUAL", "SICK", "EARNED", "UNPAID"}, first.LeaveTypesCreated)

	second, err := s.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, first.TenantID, second.TenantID)
	assert.False(t, second.TenantCreated)
	assert.False(t, second.AdminCreated)
	assert.False(t, second.AdminRoleAdded)
	assert.False(t, second.DomainCreated)
	assert.EqualValues(t, 1, count(t, db, &entity.TenantDomain{}, "domain = ?", domain))
	assert.Empty(t, second.RolesCreated)
	assert.Empty(t, second.CategoriesCreated)
	assert.Empty(t, second.LeaveTypesCreated)
	assert.EqualValues(t, len(DefaultHolidayCategories()), countCategories(t, db, first.TenantID))
	assert.EqualValues(t, len(DefaultLeaveTypes()), count(t, db, &entity.LeaveType{}, "tenant_id = ?", first.TenantID))

	assert.EqualValues(t, 1, count(t, db, &entity.Tenant{}, "name = ?", name))
	assert.EqualValues(t, 3, count(t, db, &entity.Role{}, "tenant_id = ?", first.TenantID))
	assert.EqualValues(t, 1, count(t, db, &entity.User{}, "tenant_id = ? AND email = ?", first.TenantID, email))
	assert.EqualValues(t, 1, count(t, db, &entity.UserRole{}, "tenant_id = ? AND user_id = ?", first.TenantID, first.AdminID))

	// Password is stored hashed, never plaintext.
	var u entity.User
	require.NoError(t, db.First(&u, "id = ?", first.AdminID).Error)
	require.NotNil(t, u.PasswordHash)
	assert.NotEqual(t, "s3cret-pass", *u.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte("s3cret-pass")))
}

func TestRun_MissingPasswordLeavesDBUntouched(t *testing.T) {
	db := setupIsolatedDB(t)
	name := "seeder-test-" + uuid.NewString()

	_, err := New(db, service.NewHashService(bcrypt.MinCost)).Run(context.Background(),
		Options{SystemTenantName: name, SuperAdminEmail: "x@example.com"})
	require.ErrorIs(t, err, ErrSuperAdminPasswordRequired)
	assert.EqualValues(t, 0, count(t, db, &entity.Tenant{}, "name = ?", name))
}

func TestRun_DomainOwnedByOtherTenantErrors(t *testing.T) {
	db := setupIsolatedDB(t)
	domain := "seeder-" + uuid.NewString() + ".example.com"
	sysName := "seeder-test-" + uuid.NewString()
	other := entity.Tenant{ID: uuid.New(), Name: "seeder-other-" + uuid.NewString(), IsActive: true}
	sys := entity.Tenant{ID: uuid.New(), Name: sysName, IsActive: true}
	require.NoError(t, db.Create(&other).Error)
	require.NoError(t, db.Create(&sys).Error)
	t.Cleanup(func() {
		db.Where("id = ?", other.ID).Delete(&entity.Tenant{})
		db.Where("name = ?", sysName).Delete(&entity.Tenant{})
	})
	require.NoError(t, db.Exec("INSERT INTO tenant_domains (id, tenant_id, domain) VALUES (?, ?, ?)",
		uuid.New(), other.ID, domain).Error)

	_, err := New(db, service.NewHashService(bcrypt.MinCost)).Run(context.Background(),
		Options{SystemTenantName: sysName, SuperAdminEmail: "root@" + domain, SuperAdminPassword: "pw"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain "+domain+" already belongs to another tenant")
	// Transaction rolled back: nothing was seeded into the system tenant.
	assert.EqualValues(t, 1, count(t, db, &entity.Tenant{}, "name = ?", sysName))
	assert.EqualValues(t, 0, count(t, db, &entity.Role{}, "tenant_id = ?", sys.ID))
}

func TestRun_ExistingBootstrapWithoutDomainGetsBackfilled(t *testing.T) {
	db := setupIsolatedDB(t)
	name := "seeder-test-" + uuid.NewString()
	domain := "seeder-" + uuid.NewString() + ".example.com"
	email := "Root@" + domain // mixed case: domain must be lowercased

	s := New(db, service.NewHashService(bcrypt.MinCost))
	opts := Options{SystemTenantName: name, SuperAdminEmail: email, SuperAdminPassword: "s3cret-pass"}
	first, err := s.Run(context.Background(), opts)
	require.NoError(t, err)

	// Simulate a database bootstrapped before domain registration existed.
	require.NoError(t, db.Exec("DELETE FROM tenant_domains WHERE tenant_id = ?", first.TenantID).Error)
	var before entity.User
	require.NoError(t, db.First(&before, "id = ?", first.AdminID).Error)

	// Re-run with a different password: domain is added, password untouched.
	opts.SuperAdminPassword = "different-pass"
	second, err := s.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.True(t, second.DomainCreated)
	assert.False(t, second.AdminCreated)
	var after entity.User
	require.NoError(t, db.First(&after, "id = ?", first.AdminID).Error)
	assert.Equal(t, before.PasswordHash, after.PasswordHash)

	// The bootstrapped domain resolves to the system tenant for login.
	got, err := persistence.NewGormTenantDomainRepository(db).FindTenantByDomain(context.Background(), domain)
	require.NoError(t, err)
	assert.Equal(t, first.TenantID, got.ID)
}

func TestRun_PrefersDuplicateNamedTenantOwningDomain(t *testing.T) {
	db := setupIsolatedDB(t)
	name := "seeder-test-" + uuid.NewString()
	domain := "seeder-" + uuid.NewString() + ".example.com"
	older := entity.Tenant{ID: uuid.New(), Name: name, IsActive: true, CreatedAt: time.Now().Add(-time.Hour)}
	owner := entity.Tenant{ID: uuid.New(), Name: name, IsActive: true, CreatedAt: time.Now()}
	require.NoError(t, db.Create(&older).Error)
	require.NoError(t, db.Create(&owner).Error)
	t.Cleanup(func() { db.Where("name = ?", name).Delete(&entity.Tenant{}) })
	require.NoError(t, db.Exec("INSERT INTO tenant_domains (id, tenant_id, domain) VALUES (?, ?, ?)",
		uuid.New(), owner.ID, domain).Error)

	res, err := New(db, service.NewHashService(bcrypt.MinCost)).Run(context.Background(),
		Options{SystemTenantName: name, SuperAdminEmail: "root@" + domain, SuperAdminPassword: "pw"})
	require.NoError(t, err)
	assert.False(t, res.TenantCreated)
	assert.Equal(t, owner.ID, res.TenantID)
	assert.EqualValues(t, 2, count(t, db, &entity.Tenant{}, "name = ?", name))
}

func TestRun_EmptyTenantsTableCreatesSystemTenant(t *testing.T) {
	db := setupIsolatedDB(t)
	name := "seeder-test-" + uuid.NewString()
	res, err := New(db, service.NewHashService(bcrypt.MinCost)).Run(context.Background(),
		Options{SystemTenantName: name, SuperAdminEmail: "root@seeder-" + uuid.NewString() + ".example.com", SuperAdminPassword: "pw"})
	require.NoError(t, err)
	assert.True(t, res.TenantCreated)
	assert.EqualValues(t, 1, count(t, db, &entity.Tenant{}, "name = ?", name))
}

func TestRun_SingleOtherNamedTenantIsReused(t *testing.T) {
	db := setupIsolatedDB(t)
	existing := entity.Tenant{ID: uuid.New(), Name: "seeder-existing-" + uuid.NewString(), IsActive: true}
	require.NoError(t, db.Create(&existing).Error)

	res, err := New(db, service.NewHashService(bcrypt.MinCost)).Run(context.Background(),
		Options{SystemTenantName: "seeder-test-" + uuid.NewString(), SuperAdminEmail: "root@seeder-" + uuid.NewString() + ".example.com", SuperAdminPassword: "pw"})
	require.NoError(t, err)
	assert.False(t, res.TenantCreated)
	assert.Equal(t, existing.ID, res.TenantID)
	assert.EqualValues(t, 1, count(t, db, &entity.Tenant{}, "1 = 1"))
}

func TestRun_MultipleUnmatchedTenantsErrors(t *testing.T) {
	db := setupIsolatedDB(t)
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&entity.Tenant{ID: uuid.New(), Name: "seeder-customer-" + uuid.NewString(), IsActive: true}).Error)
	}
	sysName := "seeder-test-" + uuid.NewString()

	_, err := New(db, service.NewHashService(bcrypt.MinCost)).Run(context.Background(),
		Options{SystemTenantName: sysName, SuperAdminEmail: "root@seeder-" + uuid.NewString() + ".example.com", SuperAdminPassword: "pw"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.EqualValues(t, 2, count(t, db, &entity.Tenant{}, "1 = 1"))
	assert.EqualValues(t, 0, count(t, db, &entity.Role{}, "1 = 1"))
}

func TestRun_LeaveTypesKeepAdminEdits(t *testing.T) {
	db := setupIsolatedDB(t)
	name := "seeder-test-" + uuid.NewString()
	domain := "seeder-" + uuid.NewString() + ".example.com"
	t.Cleanup(func() { db.Where("name = ?", name).Delete(&entity.Tenant{}) })

	s := New(db, service.NewHashService(bcrypt.MinCost))
	opts := Options{SystemTenantName: name, SuperAdminEmail: "root@" + domain, SuperAdminPassword: "s3cret-pass"}
	first, err := s.Run(context.Background(), opts)
	require.NoError(t, err)

	// A deactivated/renamed default must not be recreated or overwritten.
	require.NoError(t, db.Exec("UPDATE leave_types SET is_active = FALSE, name = 'Sick day' WHERE tenant_id = ? AND code = 'SICK'", first.TenantID).Error)
	second, err := s.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.Empty(t, second.LeaveTypesCreated)
	assert.EqualValues(t, 1, count(t, db, &entity.LeaveType{}, "tenant_id = ? AND code = 'SICK' AND is_active = FALSE AND name = 'Sick day'", first.TenantID))
	assert.EqualValues(t, len(DefaultLeaveTypes()), count(t, db, &entity.LeaveType{}, "tenant_id = ?", first.TenantID))
}
