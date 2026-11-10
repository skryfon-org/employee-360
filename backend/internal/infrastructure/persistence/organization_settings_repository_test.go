//go:build integration

package persistence

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
	orgimpl "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/organization"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
)

// newEmployeeCode takes the next counter value and inserts a profile holding the
// formatted code in one transaction, mirroring what the usecases do (the
// skip-taken/exhaustion logic itself is covered by the employee usecase tests).
func newEmployeeCode(t *testing.T, db *gorm.DB, tenantID uuid.UUID) string {
	t.Helper()
	u := createTestUser(t, db, tenantID, "e-"+uuid.NewString()+"@x.test")
	code, err := createProfileWithNextCode(context.Background(), db, tenantID, u.ID)
	require.NoError(t, err)
	return code
}

func createProfileWithNextCode(c context.Context, db *gorm.DB, tenantID, userID uuid.UUID) (string, error) {
	var code string
	err := database.NewGormTransactor(db).WithinTransaction(c, func(txc context.Context) error {
		counter, err := NewGormOrganizationSettingsRepository(db).NextEmployeeCodeNumber(txc, tenantID)
		if err != nil {
			return err
		}
		code = counter.Code()
		return NewGormEmployeeProfileRepository(db).Create(txc, userID, &entity.EmployeeProfile{TenantID: tenantID, UserID: userID, EmployeeCode: &code})
	})
	return code, err
}

func TestOrganizationSettingsRepository_BackfillAndTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	c := context.Background()
	repo := NewGormOrganizationSettingsRepository(db)
	a := createTestTenant(t, db, "org-a-"+uuid.NewString())
	b := createTestTenant(t, db, "org-b-"+uuid.NewString())

	// A tenant without a row: not found (usecases fall back to the defaults).
	_, err := repo.GetByTenantID(c, a.ID)
	assert.ErrorIs(t, err, domainerrors.ErrOrganizationSettingsNotFound)

	require.NoError(t, repo.EnsureDefaults(c, a.ID, uuid.New()))
	require.NoError(t, repo.EnsureDefaults(c, a.ID, uuid.New()), "idempotent")
	got, err := repo.GetByTenantID(c, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "EMP", got.EmployeeCodePrefix)
	assert.Equal(t, 4, got.EmployeeCodePadding)
	assert.EqualValues(t, 1, got.EmployeeCodeStartNumber)
	assert.EqualValues(t, 1, got.EmployeeCodeNextNumber)
	assert.EqualValues(t, 0, got.EmployeeCodeLastIssuedNumber)

	// Tenant B cannot read A's row, and updating through B's tenant id never touches A.
	_, err = repo.GetByTenantID(c, b.ID)
	assert.ErrorIs(t, err, domainerrors.ErrOrganizationSettingsNotFound)
	_, err = repo.LockByTenantID(c, b.ID)
	assert.ErrorIs(t, err, domainerrors.ErrOrganizationSettingsNotFound)
	hijack := *got
	hijack.EmployeeCodePrefix = "HIJACK"
	assert.ErrorIs(t, repo.Update(c, b.ID, uuid.New(), &hijack), domainerrors.ErrOrganizationSettingsNotFound)
	again, _ := repo.GetByTenantID(c, a.ID)
	assert.Equal(t, "EMP", again.EmployeeCodePrefix)

	// Independent counters.
	require.NoError(t, repo.EnsureDefaults(c, b.ID, uuid.New()))
	assert.Equal(t, "EMP0001", newEmployeeCode(t, db, a.ID))
	assert.Equal(t, "EMP0002", newEmployeeCode(t, db, a.ID))
	assert.Equal(t, "EMP0001", newEmployeeCode(t, db, b.ID), "tenant B counter is independent")
	ga, _ := repo.GetByTenantID(c, a.ID)
	gb, _ := repo.GetByTenantID(c, b.ID)
	assert.EqualValues(t, 3, ga.EmployeeCodeNextNumber)
	assert.EqualValues(t, 2, ga.EmployeeCodeLastIssuedNumber)
	assert.EqualValues(t, 2, gb.EmployeeCodeNextNumber)
}

func TestNextEmployeeCodeNumber_NoRowUsesDefaultsAndCreatesRow(t *testing.T) {
	db := setupTestDB(t)
	tn := createTestTenant(t, db, "org-norow-"+uuid.NewString())
	assert.Equal(t, "EMP0001", newEmployeeCode(t, db, tn.ID))
	assert.Equal(t, "EMP0002", newEmployeeCode(t, db, tn.ID))
	s, err := NewGormOrganizationSettingsRepository(db).GetByTenantID(context.Background(), tn.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 3, s.EmployeeCodeNextNumber)
	assert.EqualValues(t, 1, s.EmployeeCodeStartNumber)
}

func TestNextEmployeeCodeNumber_ConfiguredFormatPrefixChangeAndOverflow(t *testing.T) {
	db := setupTestDB(t)
	c := context.Background()
	tn := createTestTenant(t, db, "org-fmt-"+uuid.NewString())
	actor := uuid.New()
	repo := NewGormOrganizationSettingsRepository(db)
	update := orgimpl.NewUpdateSettingsUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, database.NewGormTransactor(db))
	get := orgimpl.NewGetSettingsUseCase(repo)

	_, err := update.Execute(c, tn.ID, actor, orgtypes.UpdateSettingsInput{EmployeeCodePrefix: "skryfon", EmployeeCodePadding: 4, EmployeeCodeStartNumber: 9998})
	require.NoError(t, err)
	assert.Equal(t, "SKRYFON9998", newEmployeeCode(t, db, tn.ID))
	assert.Equal(t, "SKRYFON9999", newEmployeeCode(t, db, tn.ID))
	assert.Equal(t, "SKRYFON10000", newEmployeeCode(t, db, tn.ID), "padding is a minimum width; never truncated")

	// Start number at or below the highest issued number is rejected; the counter is not rewound.
	for _, start := range []int64{1, 9997, 9999, 10000} {
		_, err = update.Execute(c, tn.ID, actor, orgtypes.UpdateSettingsInput{EmployeeCodePrefix: "SKRYFON", EmployeeCodePadding: 4, EmployeeCodeStartNumber: start})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidOrganizationSettings, "start %d", start)
	}

	// Prefix + padding change affects new codes only; old codes are never rewritten.
	old := entity.EmployeeProfile{}
	require.NoError(t, db.Where("tenant_id = ? AND employee_code = ?", tn.ID, "SKRYFON9998").First(&old).Error)
	s, err := update.Execute(c, tn.ID, actor, orgtypes.UpdateSettingsInput{EmployeeCodePrefix: "ACME", EmployeeCodePadding: 6, EmployeeCodeStartNumber: 9998})
	require.NoError(t, err)
	assert.EqualValues(t, 10001, s.EmployeeCodeNextNumber, "counter keeps going")
	assert.Equal(t, "ACME010001", newEmployeeCode(t, db, tn.ID))
	var n int64
	db.Model(&entity.EmployeeProfile{}).Where("tenant_id = ? AND employee_code LIKE 'SKRYFON%'", tn.ID).Count(&n)
	assert.EqualValues(t, 3, n, "existing codes untouched")

	// Raising the start number above the counter is allowed and the counter follows.
	s, err = update.Execute(c, tn.ID, actor, orgtypes.UpdateSettingsInput{EmployeeCodePrefix: "ACME", EmployeeCodePadding: 6, EmployeeCodeStartNumber: 50000})
	require.NoError(t, err)
	assert.Equal(t, "ACME050000", s.NextEmployeeCode())
	assert.Equal(t, "ACME050000", newEmployeeCode(t, db, tn.ID))
	got, err := get.Execute(c, tn.ID)
	require.NoError(t, err)
	assert.True(t, got.CodesIssued())
}

func TestNextEmployeeCodeNumber_ConcurrentCreatesGetUniqueSequentialCodes(t *testing.T) {
	db := setupTestDB(t)
	tn := createTestTenant(t, db, "org-conc-"+uuid.NewString())
	other := createTestTenant(t, db, "org-conc-other-"+uuid.NewString())

	const workers = 25
	users := make([]*entity.User, workers)
	for i := range users {
		users[i] = createTestUser(t, db, tn.ID, fmt.Sprintf("c%d-%s@x.test", i, uuid.NewString()))
	}
	codes := make([]string, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			codes[i], errs[i] = createProfileWithNextCode(context.Background(), db, tn.ID, users[i].ID)
		}(i)
	}
	// Another tenant allocates concurrently without interfering.
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		_ = newEmployeeCode(t, db, other.ID)
	}()
	close(start)
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "worker %d", i)
	}

	sort.Strings(codes)
	for i, code := range codes {
		assert.Equal(t, fmt.Sprintf("EMP%04d", i+1), code, "codes are unique and gapless")
	}
	s, err := NewGormOrganizationSettingsRepository(db).GetByTenantID(context.Background(), tn.ID)
	require.NoError(t, err)
	assert.EqualValues(t, workers+1, s.EmployeeCodeNextNumber)
}

func TestNextEmployeeCodeNumber_RollbackReleasesCounter(t *testing.T) {
	db := setupTestDB(t)
	c := context.Background()
	tn := createTestTenant(t, db, "org-rb-"+uuid.NewString())
	tx := database.NewGormTransactor(db)
	err := tx.WithinTransaction(c, func(txc context.Context) error {
		if _, err := NewGormOrganizationSettingsRepository(db).NextEmployeeCodeNumber(txc, tn.ID); err != nil {
			return err
		}
		return assert.AnError
	})
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, "EMP0001", newEmployeeCode(t, db, tn.ID), "rolled-back creation did not burn a number")
}

func TestEmployeeCode_ManualDuplicateMapsToCodeTaken(t *testing.T) {
	db := setupTestDB(t)
	c := context.Background()
	tn := createTestTenant(t, db, "org-dup-"+uuid.NewString())
	repo := NewGormEmployeeProfileRepository(db)
	u1 := createTestUser(t, db, tn.ID, "d1-"+uuid.NewString()+"@x.test")
	u2 := createTestUser(t, db, tn.ID, "d2-"+uuid.NewString()+"@x.test")
	code := "EMP0777"
	require.NoError(t, repo.Create(c, u1.ID, &entity.EmployeeProfile{TenantID: tn.ID, UserID: u1.ID, EmployeeCode: &code}))
	err := repo.Create(c, u2.ID, &entity.EmployeeProfile{TenantID: tn.ID, UserID: u2.ID, EmployeeCode: &code})
	assert.ErrorIs(t, err, domainerrors.ErrEmployeeCodeTaken)
}
