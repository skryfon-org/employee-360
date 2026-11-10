package organization

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	orgtypes "github.com/skryfon-org/employee-360/backend/internal/types/organization"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
)

// fakeRepo is an in-memory, tenant-keyed OrganizationSettingsRepository.
type fakeRepo struct {
	rows map[uuid.UUID]*entity.OrganizationSettings
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[uuid.UUID]*entity.OrganizationSettings{}} }

func (f *fakeRepo) GetByTenantID(_ context.Context, t uuid.UUID) (*entity.OrganizationSettings, error) {
	if s, ok := f.rows[t]; ok {
		c := *s
		return &c, nil
	}
	return nil, domainerrors.ErrOrganizationSettingsNotFound
}

func (f *fakeRepo) EnsureDefaults(_ context.Context, t, _ uuid.UUID) error {
	if _, ok := f.rows[t]; !ok {
		s := entity.DefaultOrganizationSettings(t)
		s.ID = uuid.New()
		f.rows[t] = s
	}
	return nil
}

func (f *fakeRepo) LockByTenantID(c context.Context, t uuid.UUID) (*entity.OrganizationSettings, error) {
	return f.GetByTenantID(c, t)
}

func (f *fakeRepo) Update(_ context.Context, t, _ uuid.UUID, s *entity.OrganizationSettings) error {
	if _, ok := f.rows[t]; !ok {
		return domainerrors.ErrOrganizationSettingsNotFound
	}
	c := *s
	f.rows[t] = &c
	return nil
}

var bg = context.Background()

func in(prefix string, padding int, start int64) orgtypes.UpdateSettingsInput {
	return orgtypes.UpdateSettingsInput{EmployeeCodePrefix: prefix, EmployeeCodePadding: padding, EmployeeCodeStartNumber: start}
}

func TestValidateCodeFormat(t *testing.T) {
	cases := []struct {
		name    string
		prefix  string
		padding int
		start   int64
		want    string
		err     error
	}{
		{"default", "EMP", 4, 1, "EMP", nil},
		{"trimmed and uppercased", "  skryfon ", 4, 40001, "SKRYFON", nil},
		{"digit inside is fine", "A1B", 4, 1, "A1B", nil},
		{"max prefix", "ABCDEFGHIJ", 10, 1, "ABCDEFGHIJ", nil},
		{"empty", "", 4, 1, "", ErrPrefixRequired},
		{"blank", "   ", 4, 1, "", ErrPrefixRequired},
		{"too long", "ABCDEFGHIJK", 4, 1, "", ErrPrefixTooLong},
		{"separator", "EMP-", 4, 1, "", ErrPrefixChars},
		{"inner space", "E MP", 4, 1, "", ErrPrefixChars},
		{"symbol", "EMP_", 4, 1, "", ErrPrefixChars},
		{"unicode", "ÉMP", 4, 1, "", ErrPrefixChars},
		{"ends in digit", "EMP1", 4, 1, "", ErrPrefixEndsInDigit},
		{"padding zero", "EMP", 0, 1, "", ErrPaddingRange},
		{"padding negative", "EMP", -1, 1, "", ErrPaddingRange},
		{"padding too big", "EMP", 11, 1, "", ErrPaddingRange},
		{"start zero", "EMP", 4, 0, "", ErrStartNumberRange},
		{"start negative", "EMP", 4, -5, "", ErrStartNumberRange},
		{"start 11 digits", "EMP", 4, 10000000000, "", ErrStartNumberRange},
		{"start 10 digits ok", "EMP", 4, 9999999999, "EMP", nil},
		{"total length 20 ok", "ABCDEFGHIJ", 10, 1, "ABCDEFGHIJ", nil},
		{"total length 21", "ABCDEFGHIJ", 11, 1, "", ErrPaddingRange}, // padding rule fires first
		{"total length over", "ABCDEFGHIJK", 10, 1, "", ErrPrefixTooLong},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ValidateCodeFormat(c.prefix, c.padding, c.start)
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
				assert.ErrorIs(t, err, domainerrors.ErrInvalidOrganizationSettings)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestValidateCodeFormat_TotalLengthLimit(t *testing.T) {
	// prefix 10 + padding 10 = 20 passes; the combined limit cannot otherwise be
	// exceeded while both individual limits hold, but the rule is still enforced.
	_, err := ValidateCodeFormat("ABCDEFGHIJ", 10, 1)
	assert.NoError(t, err)
	assert.Equal(t, entity.MaxEmployeeCodePrefixLen+entity.MaxEmployeeCodePadding, entity.MaxEmployeeCodeLen)
}

func TestGetSettings_DefaultsWithoutRow_NoWrite(t *testing.T) {
	repo, tenant := newFakeRepo(), uuid.New()
	s, err := NewGetSettingsUseCase(repo).Execute(bg, tenant)
	require.NoError(t, err)
	assert.Equal(t, "EMP", s.EmployeeCodePrefix)
	assert.Equal(t, 4, s.EmployeeCodePadding)
	assert.EqualValues(t, 1, s.EmployeeCodeStartNumber)
	assert.Equal(t, "EMP0001", s.NextEmployeeCode())
	assert.Empty(t, repo.rows, "reading must not create a row")

	_, err = NewGetSettingsUseCase(repo).Execute(bg, uuid.Nil)
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

func TestUpdateSettings_SavesNormalizedAndSetsCounterToStart(t *testing.T) {
	repo, tenant, actor := newFakeRepo(), uuid.New(), uuid.New()
	audit := &ucsharedtest.RecordingAuditRecorder{}
	uc := NewUpdateSettingsUseCase(repo, audit, nil)

	s, err := uc.Execute(bg, tenant, actor, in(" skryfon", 4, 40001))
	require.NoError(t, err)
	assert.Equal(t, "SKRYFON", s.EmployeeCodePrefix)
	assert.EqualValues(t, 40001, s.EmployeeCodeNextNumber)
	assert.Equal(t, "SKRYFON40001", s.NextEmployeeCode())
	assert.False(t, s.CodesIssued())
	assert.Equal(t, "SKRYFON", repo.rows[tenant].EmployeeCodePrefix)
	assert.Len(t, audit.Logs, 1)
}

func TestUpdateSettings_RejectsInvalidWithoutWriting(t *testing.T) {
	repo, tenant, actor := newFakeRepo(), uuid.New(), uuid.New()
	_, err := NewUpdateSettingsUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil).Execute(bg, tenant, actor, in("EMP1", 4, 1))
	assert.ErrorIs(t, err, ErrPrefixEndsInDigit)
	assert.Empty(t, repo.rows)

	_, err = NewUpdateSettingsUseCase(repo, nil, nil).Execute(bg, uuid.Nil, actor, in("EMP", 4, 1))
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	_, err = NewUpdateSettingsUseCase(repo, nil, nil).Execute(bg, tenant, uuid.Nil, in("EMP", 4, 1))
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

func TestUpdateSettings_StartNumberBelowIssuedRejected(t *testing.T) {
	repo, tenant, actor := newFakeRepo(), uuid.New(), uuid.New()
	uc := NewUpdateSettingsUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
	_, err := uc.Execute(bg, tenant, actor, in("EMP", 4, 1))
	require.NoError(t, err)

	// Simulate five generated codes: EMP0001..EMP0005 issued, counter at 6.
	row := repo.rows[tenant]
	row.EmployeeCodeLastIssuedNumber, row.EmployeeCodeNextNumber = 5, 6

	for _, start := range []int64{2, 3, 5} {
		_, err = uc.Execute(bg, tenant, actor, in("EMP", 4, start))
		require.ErrorIs(t, err, domainerrors.ErrInvalidOrganizationSettings, "start %d", start)
		assert.True(t, strings.Contains(err.Error(), "6 or higher"), err.Error())
	}
	assert.EqualValues(t, 1, repo.rows[tenant].EmployeeCodeStartNumber, "rejected edits change nothing")

	// Unchanged start number is always fine (prefix/padding edit only); counter untouched.
	s, err := uc.Execute(bg, tenant, actor, in("ACME", 6, 1))
	require.NoError(t, err)
	assert.EqualValues(t, 6, s.EmployeeCodeNextNumber)
	assert.Equal(t, "ACME000006", s.NextEmployeeCode())

	// Strictly above the highest issued number is accepted and the counter follows.
	s, err = uc.Execute(bg, tenant, actor, in("ACME", 6, 100))
	require.NoError(t, err)
	assert.EqualValues(t, 100, s.EmployeeCodeNextNumber)
	assert.True(t, s.CodesIssued())

	// The counter never moves backwards when the start is lowered to a still-valid value.
	s, err = uc.Execute(bg, tenant, actor, in("ACME", 6, 50))
	require.NoError(t, err)
	assert.EqualValues(t, 100, s.EmployeeCodeNextNumber)
}

func TestUpdateSettings_CounterIsNotClientWritable(t *testing.T) {
	// The input type has no counter field; the stored counter survives an update.
	repo, tenant, actor := newFakeRepo(), uuid.New(), uuid.New()
	uc := NewUpdateSettingsUseCase(repo, nil, nil)
	_, err := uc.Execute(bg, tenant, actor, in("EMP", 4, 1))
	require.NoError(t, err)
	repo.rows[tenant].EmployeeCodeNextNumber, repo.rows[tenant].EmployeeCodeLastIssuedNumber = 8, 7
	s, err := uc.Execute(bg, tenant, actor, in("NEW", 4, 1))
	require.NoError(t, err)
	assert.EqualValues(t, 8, s.EmployeeCodeNextNumber)
}

func TestUpdateSettings_TenantIsolation(t *testing.T) {
	repo, a, b, actor := newFakeRepo(), uuid.New(), uuid.New(), uuid.New()
	uc := NewUpdateSettingsUseCase(repo, nil, nil)
	_, err := uc.Execute(bg, a, actor, in("AAA", 4, 100))
	require.NoError(t, err)
	got, err := NewGetSettingsUseCase(repo).Execute(bg, b)
	require.NoError(t, err)
	assert.Equal(t, "EMP", got.EmployeeCodePrefix, "tenant B still sees defaults")
	_, err = uc.Execute(bg, b, actor, in("BBB", 3, 5))
	require.NoError(t, err)
	assert.Equal(t, "AAA", repo.rows[a].EmployeeCodePrefix)
	assert.EqualValues(t, 100, repo.rows[a].EmployeeCodeStartNumber)
}

func inWelcome(m *string) orgtypes.UpdateSettingsInput {
	i := in("EMP", 4, 1)
	i.WelcomeMessage = m
	return i
}

func TestUpdateSettings_WelcomeMessage(t *testing.T) {
	repo, tenant, actor := newFakeRepo(), uuid.New(), uuid.New()
	uc := NewUpdateSettingsUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
	str := func(s string) *string { return &s }

	// Set (trimmed).
	s, err := uc.Execute(bg, tenant, actor, inWelcome(str("  Welcome aboard!\nEnjoy.  ")))
	require.NoError(t, err)
	require.NotNil(t, s.WelcomeMessage)
	assert.Equal(t, "Welcome aboard!\nEnjoy.", *s.WelcomeMessage)

	// Omitted leaves it unchanged.
	s, err = uc.Execute(bg, tenant, actor, inWelcome(nil))
	require.NoError(t, err)
	require.NotNil(t, s.WelcomeMessage)
	assert.Equal(t, "Welcome aboard!\nEnjoy.", *repo.rows[tenant].WelcomeMessage)

	// Exactly the max length is accepted; one more is rejected without changing the stored value.
	_, err = uc.Execute(bg, tenant, actor, inWelcome(str(strings.Repeat("é", entity.MaxWelcomeMessageLen))))
	require.NoError(t, err)
	_, err = uc.Execute(bg, tenant, actor, inWelcome(str(strings.Repeat("é", entity.MaxWelcomeMessageLen+1))))
	assert.ErrorIs(t, err, ErrWelcomeMessageTooLong)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidOrganizationSettings)
	assert.Len(t, []rune(*repo.rows[tenant].WelcomeMessage), entity.MaxWelcomeMessageLen)

	// Empty and blank strings clear it.
	for _, blank := range []string{"", "  \n "} {
		_, err = uc.Execute(bg, tenant, actor, inWelcome(str("x")))
		require.NoError(t, err)
		s, err = uc.Execute(bg, tenant, actor, inWelcome(str(blank)))
		require.NoError(t, err)
		assert.Nil(t, s.WelcomeMessage)
		assert.Nil(t, repo.rows[tenant].WelcomeMessage)
	}
}

func (f *fakeRepo) NextEmployeeCodeNumber(context.Context, uuid.UUID) (entity.EmployeeCodeCounter, error) {
	return entity.EmployeeCodeCounter{}, errors.New("not used by organization usecases")
}

func TestValidateWeeklyOffDays(t *testing.T) {
	got, err := ValidateWeeklyOffDays([]int{6, 0})
	require.NoError(t, err)
	assert.Equal(t, []int{0, 6}, got)
	for _, bad := range [][]int{nil, {}, {1, 1}, {-1}, {7}, {0, 1, 2, 3, 4, 5, 6}} {
		_, err := ValidateWeeklyOffDays(bad)
		assert.ErrorIs(t, err, ErrWeeklyOffDaysInvalid, "%v", bad)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidOrganizationSettings)
	}
}

func TestWeeklyOffDays_DefaultAndUpdate(t *testing.T) {
	repo, tenant, actor := newFakeRepo(), uuid.New(), uuid.New()
	s, err := NewGetSettingsUseCase(repo).Execute(bg, tenant)
	require.NoError(t, err)
	assert.Equal(t, []int{0, 6}, s.WeeklyOffDays)
	assert.Equal(t, []int{0, 6}, orgtypes.ToSettingsResponse(s).WeeklyOffDays)

	uc := NewUpdateSettingsUseCase(repo, nil, nil)
	req := in("EMP", 4, 1)
	req.WeeklyOffDays = []int{5, 0}
	s, err = uc.Execute(bg, tenant, actor, req)
	require.NoError(t, err)
	assert.Equal(t, []int{0, 5}, s.WeeklyOffDays)

	req.WeeklyOffDays = []int{1, 1}
	_, err = uc.Execute(bg, tenant, actor, req)
	assert.ErrorIs(t, err, ErrWeeklyOffDaysInvalid)

	// Omitted leaves unchanged.
	req.WeeklyOffDays = nil
	s, err = uc.Execute(bg, tenant, actor, req)
	require.NoError(t, err)
	assert.Equal(t, []int{0, 5}, s.WeeklyOffDays)
}
