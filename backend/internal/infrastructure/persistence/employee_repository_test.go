//go:build integration

package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	empimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/employee"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
)

func str(s string) *string { return &s }

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

type empEnv struct {
	db      *gorm.DB
	tA, tB  *entity.Tenant
	uA, uB  *entity.User
	profile *entity.EmployeeProfile
}

func newEmpEnv(t *testing.T) *empEnv {
	t.Helper()
	db := setupTestDB(t)
	e := &empEnv{db: db}
	e.tA = createTestTenant(t, db, "emp-a-"+uuid.NewString())
	e.tB = createTestTenant(t, db, "emp-b-"+uuid.NewString())
	e.uA = createTestUser(t, db, e.tA.ID, "a-"+uuid.NewString()+"@a.test")
	e.uB = createTestUser(t, db, e.tB.ID, "b-"+uuid.NewString()+"@b.test")
	return e
}

func TestEmployeeProfileRepository_TenantScopingAndSoftDelete(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormEmployeeProfileRepository(e.db)

	p := &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: e.uA.ID, EmployeeCode: str("EMP1"), Phone: str("12345"), DateOfBirth: ptrTime(day(1990, 4, 2)), AddressLine1: str("x")}
	require.NoError(t, repo.Create(c, e.uA.ID, p))

	got, err := repo.GetByUserID(c, e.tA.ID, e.uA.ID)
	require.NoError(t, err)
	assert.Equal(t, "EMP1", *got.EmployeeCode)
	assert.Equal(t, "active", got.EmploymentStatus, "default employment status")
	assert.Equal(t, day(1990, 4, 2), got.DateOfBirth.UTC())
	assert.Equal(t, "x", *got.AddressLine1, "address_line1 column maps")

	// Tenant B can neither read nor update tenant A's profile.
	_, err = repo.GetByUserID(c, e.tB.ID, e.uA.ID)
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeProfileNotFound)
	got.Phone = str("hijack")
	assert.ErrorIs(t, repo.Update(c, e.tB.ID, e.uB.ID, got), domainerrors.ErrEmployeeProfileNotFound)
	again, _ := repo.GetByUserID(c, e.tA.ID, e.uA.ID)
	assert.Equal(t, "12345", *again.Phone)

	// Code uniqueness is per tenant, among live rows.
	exists, err := repo.EmployeeCodeExists(c, e.tA.ID, "EMP1", uuid.New())
	require.NoError(t, err)
	assert.True(t, exists)
	exists, _ = repo.EmployeeCodeExists(c, e.tB.ID, "EMP1", uuid.New())
	assert.False(t, exists, "other tenant does not see the code")
	exists, _ = repo.EmployeeCodeExists(c, e.tA.ID, "EMP1", e.uA.ID)
	assert.False(t, exists, "own code is excluded")

	u2 := createTestUser(t, e.db, e.tA.ID, "a2-"+uuid.NewString()+"@a.test")
	err = repo.Create(c, u2.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: u2.ID, EmployeeCode: str("EMP1")})
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeCodeTaken)
	require.NoError(t, repo.Create(c, u2.ID, &entity.EmployeeProfile{TenantID: e.tB.ID, UserID: e.uB.ID, EmployeeCode: str("EMP1")}), "same code in another tenant is fine")

	// Second profile row for the same user is rejected.
	assert.Error(t, repo.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: e.uA.ID, EmployeeCode: str("EMP9")}))

	// Soft-deleted profile is invisible and frees its code.
	require.NoError(t, e.db.Exec("UPDATE employee_profiles SET deleted_at = NOW() WHERE user_id = ?", e.uA.ID).Error)
	_, err = repo.GetByUserID(c, e.tA.ID, e.uA.ID)
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeProfileNotFound)
	exists, _ = repo.EmployeeCodeExists(c, e.tA.ID, "EMP1", uuid.New())
	assert.False(t, exists)
	assert.NoError(t, repo.Create(c, u2.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: u2.ID, EmployeeCode: str("EMP1")}))
}

func TestEmployeeProfileRepository_CreateDistinguishesUniqueViolations(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormEmployeeProfileRepository(e.db)
	u2 := createTestUser(t, e.db, e.tA.ID, "second-"+uuid.NewString()+"@a.test")

	require.NoError(t, repo.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: e.uA.ID, EmployeeCode: str("DUP1")}))

	err := repo.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: e.uA.ID, EmployeeCode: str("DUP2")})
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeProfileExists, "same user twice is a retryable conflict")
	assert.NotErrorIs(t, err, domainerrors.ErrEmployeeCodeTaken)

	err = repo.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: u2.ID, EmployeeCode: str("DUP1")})
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeCodeTaken, "same code in tenant")

	// The repository never generates a code; the usecase must assign it.
	err = repo.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: u2.ID})
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeCodeRequired)
}

func TestEmployeeProfile_DBConstraints(t *testing.T) {
	e := newEmpEnv(t)
	repo := NewGormEmployeeProfileRepository(e.db)
	c := context.Background()
	bad := "robot"
	assert.Error(t, repo.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: e.uA.ID, EmployeeCode: str("C1"), Gender: &bad}), "gender CHECK")
	assert.Error(t, repo.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: e.uA.ID, EmployeeCode: str("C2"), JoiningDate: ptrTime(day(2024, 5, 1)), ExitDate: ptrTime(day(2024, 1, 1))}), "exit >= joining CHECK")
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestEmergencyContactRepository_SinglePrimaryScopingSoftDelete(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormEmergencyContactRepository(e.db)
	mk := func(name string, primary bool) *entity.EmergencyContact {
		ec := &entity.EmergencyContact{TenantID: e.tA.ID, UserID: e.uA.ID, Name: name, Relationship: "spouse", Phone: "12345", IsPrimary: primary}
		return ec
	}
	c1, c2 := mk("one", true), mk("two", false)
	require.NoError(t, repo.Create(c, e.uA.ID, c1))
	require.NoError(t, repo.Create(c, e.uA.ID, c2))

	// DB guarantees one primary: a second primary insert is rejected.
	assert.Error(t, repo.Create(c, e.uA.ID, mk("three", true)))

	// Switch primary: clear then set.
	require.NoError(t, repo.ClearPrimary(c, e.tA.ID, e.uA.ID, c2.ID, e.uA.ID))
	c2.IsPrimary = true
	require.NoError(t, repo.Update(c, e.tA.ID, e.uA.ID, c2))
	list, err := repo.ListByUser(c, e.tA.ID, e.uA.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "two", list[0].Name, "primary first")

	// Tenant isolation.
	other, err := repo.ListByUser(c, e.tB.ID, e.uA.ID)
	require.NoError(t, err)
	assert.Empty(t, other)
	_, err = repo.GetByID(c, e.tB.ID, e.uA.ID, c1.ID)
	assert.ErrorIs(t, err, domainerrors.ErrEmergencyContactNotFound)
	assert.ErrorIs(t, repo.Delete(c, e.tB.ID, e.uA.ID, c1.ID, e.uB.ID), domainerrors.ErrEmergencyContactNotFound)
	n, _ := repo.CountByUser(c, e.tB.ID, e.uA.ID)
	assert.Zero(t, n)

	// Soft delete hides the row and frees the primary slot.
	require.NoError(t, repo.Delete(c, e.tA.ID, e.uA.ID, c2.ID, e.uA.ID))
	_, err = repo.GetByID(c, e.tA.ID, e.uA.ID, c2.ID)
	assert.ErrorIs(t, err, domainerrors.ErrEmergencyContactNotFound)
	n, _ = repo.CountByUser(c, e.tA.ID, e.uA.ID)
	assert.EqualValues(t, 1, n)
	c1.IsPrimary = true
	assert.NoError(t, repo.Update(c, e.tA.ID, e.uA.ID, c1))
}

func TestUserAssignmentRepository_OpenCloseOrderAndConstraints(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormUserAssignmentRepository(e.db)

	open, err := repo.GetOpen(c, e.tA.ID, e.uA.ID)
	require.NoError(t, err)
	assert.Nil(t, open)

	a1 := &entity.UserAssignment{TenantID: e.tA.ID, UserID: e.uA.ID, EffectiveFrom: day(2024, 1, 1)}
	require.NoError(t, repo.Create(c, e.uA.ID, a1))

	// Only one open stint per user (DB-enforced).
	dup := &entity.UserAssignment{TenantID: e.tA.ID, UserID: e.uA.ID, EffectiveFrom: day(2024, 2, 1)}
	assert.Error(t, repo.Create(c, e.uA.ID, dup), "second open row must violate the partial unique index")

	// Date order CHECK: effective_to before effective_from.
	assert.Error(t, e.db.Exec("UPDATE user_assignments SET effective_to = '2023-01-01' WHERE id = ?", a1.ID).Error)

	// Close then open.
	require.NoError(t, repo.CloseOpen(c, e.tA.ID, e.uA.ID, e.uA.ID, day(2024, 6, 1)))
	a2 := &entity.UserAssignment{TenantID: e.tA.ID, UserID: e.uA.ID, EffectiveFrom: day(2024, 6, 1)}
	require.NoError(t, repo.Create(c, e.uA.ID, a2))
	open, err = repo.GetOpen(c, e.tA.ID, e.uA.ID)
	require.NoError(t, err)
	assert.Equal(t, a2.ID, open.ID)

	list, err := repo.ListByUser(c, e.tA.ID, e.uA.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, a2.ID, list[0].ID, "newest first")

	// Tenant isolation: tenant B sees nothing and cannot close tenant A's stint.
	bl, _ := repo.ListByUser(c, e.tB.ID, e.uA.ID)
	assert.Empty(t, bl)
	require.NoError(t, repo.CloseOpen(c, e.tB.ID, e.uA.ID, e.uB.ID, day(2024, 7, 1)))
	open, _ = repo.GetOpen(c, e.tA.ID, e.uA.ID)
	assert.NotNil(t, open, "still open: wrong-tenant close is a no-op")

	// Soft-deleted stints are hidden.
	require.NoError(t, e.db.Exec("UPDATE user_assignments SET deleted_at = NOW() WHERE user_id = ?", e.uA.ID).Error)
	list, _ = repo.ListByUser(c, e.tA.ID, e.uA.ID)
	assert.Empty(t, list)
}

func TestEmployeeDirectoryRepository_FiltersScopingSoftDelete(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormEmployeeDirectoryRepository(e.db)
	profiles := NewGormEmployeeProfileRepository(e.db)

	deptID, posID := uuid.New(), uuid.New()
	require.NoError(t, e.db.Exec("INSERT INTO departments (id, tenant_id, name) VALUES (?, ?, 'Eng')", deptID, e.tA.ID).Error)
	require.NoError(t, e.db.Exec("INSERT INTO positions (id, tenant_id, name) VALUES (?, ?, 'Dev')", posID, e.tA.ID).Error)
	require.NoError(t, e.db.Exec("UPDATE users SET department_id = ?, position_id = ?, first_name = 'Alice', last_name = 'Smith' WHERE id = ?", deptID, posID, e.uA.ID).Error)
	require.NoError(t, profiles.Create(c, e.uA.ID, &entity.EmployeeProfile{TenantID: e.tA.ID, UserID: e.uA.ID, EmployeeCode: str("ZED100"), EmploymentStatus: "probation", ProfileCompletedAt: ptrTime(time.Now())}))

	inactive := createTestUser(t, e.db, e.tA.ID, "inactive-"+uuid.NewString()+"@a.test")
	require.NoError(t, e.db.Exec("UPDATE users SET is_active = false WHERE id = ?", inactive.ID).Error)
	deleted := createTestUser(t, e.db, e.tA.ID, "deleted-"+uuid.NewString()+"@a.test")
	require.NoError(t, e.db.Exec("UPDATE users SET deleted_at = NOW() WHERE id = ?", deleted.ID).Error)

	roleID := uuid.New()
	require.NoError(t, e.db.Exec("INSERT INTO roles (id, tenant_id, name) VALUES (?, ?, 'admin')", roleID, e.tA.ID).Error)
	require.NoError(t, e.db.Exec("INSERT INTO user_roles (id, tenant_id, user_id, role_id) VALUES (gen_random_uuid(), ?, ?, ?)", e.tA.ID, e.uA.ID, roleID).Error)

	ids := func(f entity.EmployeeFilter) []uuid.UUID {
		rows, _, err := repo.List(c, e.tA.ID, f, 50, 0)
		require.NoError(t, err)
		var out []uuid.UUID
		for _, r := range rows {
			out = append(out, r.UserID)
		}
		return out
	}

	all := ids(entity.EmployeeFilter{})
	assert.ElementsMatch(t, []uuid.UUID{e.uA.ID, inactive.ID}, all, "soft-deleted and other-tenant users excluded")
	assert.NotContains(t, all, e.uB.ID)

	assert.Equal(t, []uuid.UUID{e.uA.ID}, ids(entity.EmployeeFilter{Search: "alice sm"}), "search by name")
	assert.Equal(t, []uuid.UUID{e.uA.ID}, ids(entity.EmployeeFilter{Search: "zed1"}), "search by employee code")
	assert.Equal(t, []uuid.UUID{inactive.ID}, ids(entity.EmployeeFilter{Search: inactive.Email}), "search by email")
	assert.Empty(t, ids(entity.EmployeeFilter{Search: "100%"}), "LIKE wildcards are escaped")
	active, off := true, false
	assert.Equal(t, []uuid.UUID{e.uA.ID}, ids(entity.EmployeeFilter{IsActive: &active}))
	assert.Equal(t, []uuid.UUID{inactive.ID}, ids(entity.EmployeeFilter{IsActive: &off}))
	assert.Equal(t, []uuid.UUID{e.uA.ID}, ids(entity.EmployeeFilter{DepartmentID: &deptID}))
	assert.Equal(t, []uuid.UUID{e.uA.ID}, ids(entity.EmployeeFilter{PositionID: &posID}))
	assert.Equal(t, []uuid.UUID{e.uA.ID}, ids(entity.EmployeeFilter{EmploymentStatus: "probation"}))
	assert.Empty(t, ids(entity.EmployeeFilter{EmploymentStatus: "resigned"}))

	// Results are ordered by name ascending (case-insensitive): Alice Smith first.
	require.NoError(t, e.db.Exec("UPDATE users SET first_name = 'bob', last_name = 'Jones' WHERE id = ?", inactive.ID).Error)
	assert.Equal(t, []uuid.UUID{e.uA.ID, inactive.ID}, ids(entity.EmployeeFilter{}), "ordered by name ASC")

	// Pagination keeps the total.
	rows, total, err := repo.List(c, e.tA.ID, entity.EmployeeFilter{}, 1, 0)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.EqualValues(t, 2, total)

	s, err := repo.GetSummary(c, e.tA.ID, e.uA.ID)
	require.NoError(t, err)
	assert.Equal(t, "Eng", *s.DepartmentName)
	assert.Equal(t, "Dev", *s.PositionName)
	assert.Equal(t, []string{"admin"}, s.Roles)
	assert.True(t, s.ProfileCompleted)

	_, err = repo.GetSummary(c, e.tB.ID, e.uA.ID)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound, "cross-tenant summary is 404")
	_, err = repo.GetSummary(c, e.tA.ID, deleted.ID)
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound, "soft-deleted summary is 404")

	// Role lock helper: scoped + active only.
	require.NoError(t, e.db.Transaction(func(tx *gorm.DB) error {
		lockIDs, err := repo.LockActiveUserIDsWithRole(database.WithTx(c, tx), e.tA.ID, "admin")
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{e.uA.ID}, lockIDs)
		lockIDs, err = repo.LockActiveUserIDsWithRole(database.WithTx(c, tx), e.tB.ID, "admin")
		require.NoError(t, err)
		assert.Empty(t, lockIDs)
		return nil
	}))
}

// The department/position change (users update + close + open + audit) must be
// one transaction: a failing audit write leaves everything untouched.
func TestUpdateEmployeeUseCase_AssignmentChangeIsAtomic(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	d1, d2 := uuid.New(), uuid.New()
	for _, d := range []uuid.UUID{d1, d2} {
		require.NoError(t, e.db.Exec("INSERT INTO departments (id, tenant_id, name) VALUES (?, ?, ?)", d, e.tA.ID, d.String()).Error)
	}
	admin := createTestUser(t, e.db, e.tA.ID, "adm-"+uuid.NewString()+"@a.test")

	rec := &ucsharedtest.RecordingAuditRecorder{}
	profiles := NewGormEmployeeProfileRepository(e.db)
	uc := empimpl.NewUpdateEmployeeUseCase(
		NewGormUserRepository(e.db), profiles, NewGormEmergencyContactRepository(e.db),
		NewGormUserAssignmentRepository(e.db), NewGormEmployeeDirectoryRepository(e.db),
		NewGormOrgReferenceRepository(e.db), NewGormOrganizationSettingsRepository(e.db), rec, database.NewGormTransactor(e.db),
	)
	actor := emptypes.Actor{TenantID: e.tA.ID, UserID: admin.ID, Roles: []string{entity.RoleAdmin}}
	body := func(s string) emptypes.UpdateEmployeeInput {
		var in emptypes.UpdateEmployeeInput
		require.NoError(t, json.Unmarshal([]byte(s), &in))
		return in
	}

	_, err := uc.Execute(c, actor, e.uA.ID, body(`{"department_id":"`+d1.String()+`","joining_date":"2024-01-01"}`))
	require.NoError(t, err)

	// Force the audit write to fail on the next change -> full rollback.
	boom := errors.New("audit down")
	rec.ErrFn = func() error { return boom }
	_, err = uc.Execute(c, actor, e.uA.ID, body(`{"department_id":"`+d2.String()+`","effective_date":"2024-06-01"}`))
	require.ErrorIs(t, err, boom)

	var cur string
	require.NoError(t, e.db.Raw("SELECT department_id::text FROM users WHERE id = ?", e.uA.ID).Row().Scan(&cur))
	assert.Equal(t, d1.String(), cur, "users.department_id rolled back")
	assigns, _ := NewGormUserAssignmentRepository(e.db).ListByUser(c, e.tA.ID, e.uA.ID)
	require.Len(t, assigns, 1, "no new stint committed")
	assert.Nil(t, assigns[0].EffectiveTo, "open stint was not closed")

	// And the happy path commits close + open together with exactly one open row.
	rec.ErrFn = nil
	_, err = uc.Execute(c, actor, e.uA.ID, body(`{"department_id":"`+d2.String()+`","effective_date":"2024-06-01"}`))
	require.NoError(t, err)
	assigns, _ = NewGormUserAssignmentRepository(e.db).ListByUser(c, e.tA.ID, e.uA.ID)
	require.Len(t, assigns, 2)
	open := 0
	for _, a := range assigns {
		if a.EffectiveTo == nil {
			open++
		}
	}
	assert.Equal(t, 1, open)

	// Cross-tenant admin gets 404 semantics.
	adminB := createTestUser(t, e.db, e.tB.ID, "admb-"+uuid.NewString()+"@b.test")
	_, err = uc.Execute(c, emptypes.Actor{TenantID: e.tB.ID, UserID: adminB.ID, Roles: []string{entity.RoleAdmin}}, e.uA.ID, body(`{"first_name":"X"}`))
	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
}

// The "Add allocation" employee picker relies on GET /employees?search=&status=active:
// case-insensitive partial match on full name, code and email; tenant-scoped; paginated.
func TestEmployeeDirectoryRepository_PickerSearch(t *testing.T) {
	e := newEmpEnv(t)
	c := context.Background()
	repo := NewGormEmployeeDirectoryRepository(e.db)
	profiles := NewGormEmployeeProfileRepository(e.db)

	mk := func(tenant uuid.UUID, first, last, email, code string, active bool) *entity.User {
		u := createTestUser(t, e.db, tenant, email)
		require.NoError(t, e.db.Exec("UPDATE users SET first_name = ?, last_name = ?, is_active = ? WHERE id = ?", first, last, active, u.ID).Error)
		if code != "" {
			require.NoError(t, profiles.Create(c, u.ID, &entity.EmployeeProfile{TenantID: tenant, UserID: u.ID, EmployeeCode: str(code), EmploymentStatus: "active"}))
		}
		return u
	}
	suffix := uuid.NewString()[:8]
	ada := mk(e.tA.ID, "Ada", "Lovelace", "ada."+suffix+"@a.test", "PK"+suffix+"1", true)
	grace := mk(e.tA.ID, "Grace", "Hopper", "grace."+suffix+"@a.test", "PK"+suffix+"2", true)
	gone := mk(e.tA.ID, "Adam", "Inactive", "adam."+suffix+"@a.test", "PK"+suffix+"3", false)
	_ = mk(e.tB.ID, "Ada", "Othertenant", "ada.other."+suffix+"@b.test", "PK"+suffix+"9", true)
	pct := mk(e.tA.ID, "Per", "100%_Cent", "pct."+suffix+"@a.test", "", true)

	search := func(q string, active *bool) []uuid.UUID {
		rows, _, err := repo.List(c, e.tA.ID, entity.EmployeeFilter{Search: q, IsActive: active}, 20, 0)
		require.NoError(t, err)
		var out []uuid.UUID
		for _, r := range rows {
			out = append(out, r.UserID)
		}
		return out
	}
	yes := true

	assert.Equal(t, []uuid.UUID{ada.ID}, search("LOVEL", nil), "last name, case-insensitive")
	assert.Equal(t, []uuid.UUID{ada.ID}, search("ada lovelace", nil), "combined first last")
	assert.Equal(t, []uuid.UUID{ada.ID}, search("  ada   LOVE ", nil), "whitespace normalised")
	assert.Equal(t, []uuid.UUID{grace.ID}, search("pk"+suffix+"2", nil), "employee code")
	assert.Equal(t, []uuid.UUID{grace.ID}, search("GRACE."+suffix, nil), "email")
	assert.Equal(t, []uuid.UUID{gone.ID}, search("adam inactive", nil))
	assert.Empty(t, search("adam inactive", &yes), "active-only excludes inactive")
	assert.Equal(t, []uuid.UUID{ada.ID}, search("ada lovelace", &yes))
	assert.Empty(t, search("othertenant", nil), "other tenant never returned")
	assert.Empty(t, search("ada.other."+suffix, nil), "other tenant email not matched")

	// Wildcards are literal.
	assert.Equal(t, []uuid.UUID{pct.ID}, search("100%_", nil))
	assert.Equal(t, []uuid.UUID{pct.ID}, search("%", nil), "only the literal %")
	assert.Equal(t, []uuid.UUID{pct.ID}, search("_", nil), "only the literal _")

	// Pagination with a small limit keeps the total.
	rows, total, err := repo.List(c, e.tA.ID, entity.EmployeeFilter{Search: "a.test"}, 2, 0)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.GreaterOrEqual(t, total, int64(4))
	rows2, _, err := repo.List(c, e.tA.ID, entity.EmployeeFilter{Search: "a.test"}, 2, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, rows2)
	assert.NotEqual(t, rows[0].UserID, rows2[0].UserID)
}
