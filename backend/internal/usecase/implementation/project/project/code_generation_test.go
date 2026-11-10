package project

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
)

func TestBaseCodeFromName(t *testing.T) {
	cases := map[string]string{
		"Apollo Mission Control":                 "APOLLO-MISSION-CONTROL",
		"  café  déjà--vu!! ":                    "CAFE-DEJA-VU",
		"???":                                    "PROJECT",
		"日本語":                                    "PROJECT",
		"Q3_2026 (Beta)":                         "Q3-2026-BETA",
		"Straße":                                 "STRASSE",
		strings.Repeat("A", 40):                  strings.Repeat("A", 30),
		"Alpha Bravo Charlie Delta Echo Foxtrot": "ALPHA-BRAVO-CHARLIE-DELTA-ECHO",
	}
	for in, want := range cases {
		got := baseCodeFromName(in, 30)
		assert.Equal(t, want, got, in)
		assert.LessOrEqual(t, len(got), 30)
	}
}

func TestCandidateCodeFitsWithSuffix(t *testing.T) {
	name := strings.Repeat("A", 40)
	got := candidateCode(name, 12)
	assert.Len(t, got, 30)
	assert.True(t, strings.HasSuffix(got, "-12"))
}

func TestCreateProject_GeneratedCode(t *testing.T) {
	tenant, tenantB, actor := uuid.New(), uuid.New(), uuid.New()
	repo := newFakeProjects()
	uc := NewCreateProjectUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
	ctx := context.Background()
	in := func(name, code string) prjtypes.CreateProjectInput {
		i := goodCreate()
		i.Name, i.Code = name, code
		return i
	}

	p, err := uc.Execute(ctx, tenant, actor, in("Apollo Mission Control", ""))
	require.NoError(t, err)
	assert.Equal(t, "APOLLO-MISSION-CONTROL", p.Code)

	for i, want := range []string{"APOLLO-MISSION-CONTROL-2", "APOLLO-MISSION-CONTROL-3"} {
		p, err = uc.Execute(ctx, tenant, actor, in("apollo mission control", "   "))
		require.NoError(t, err, i)
		assert.Equal(t, want, p.Code)
	}

	t.Run("tenant isolation", func(t *testing.T) {
		p, err := uc.Execute(ctx, tenantB, actor, in("Apollo Mission Control", ""))
		require.NoError(t, err)
		assert.Equal(t, "APOLLO-MISSION-CONTROL", p.Code)
	})
	t.Run("explicit code still honoured and 409 on duplicate", func(t *testing.T) {
		p, err := uc.Execute(ctx, tenant, actor, in("Whatever", "my-code"))
		require.NoError(t, err)
		assert.Equal(t, "MY-CODE", p.Code)
		_, err = uc.Execute(ctx, tenant, actor, in("Other", "MY-CODE"))
		assert.ErrorIs(t, err, domainerrors.ErrProjectCodeTaken)
	})
	t.Run("blank name rejected, symbol-only name falls back", func(t *testing.T) {
		_, err := uc.Execute(ctx, tenant, actor, in("  ", ""))
		assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
		p, err := uc.Execute(ctx, tenant, actor, in("!!!", ""))
		require.NoError(t, err)
		assert.Equal(t, "PROJECT", p.Code)
		p, err = uc.Execute(ctx, tenant, actor, in("???", ""))
		require.NoError(t, err)
		assert.Equal(t, "PROJECT-2", p.Code)
	})
	t.Run("long name truncated at word boundary", func(t *testing.T) {
		p, err := uc.Execute(ctx, tenant, actor, in("Alpha Bravo Charlie Delta Echo Foxtrot", ""))
		require.NoError(t, err)
		assert.Equal(t, "ALPHA-BRAVO-CHARLIE-DELTA-ECHO", p.Code)
	})
}

// racyProjects makes the first N Create calls fail with a unique violation,
// as if a concurrent request took the code between the check and the insert.
type racyProjects struct {
	*fakeProjects
	failures int
	created  []string
}

func (r *racyProjects) Create(c context.Context, tenantID, actorID uuid.UUID, p *entity.Project) error {
	if r.failures > 0 {
		r.failures--
		return domainerrors.ErrProjectCodeTaken
	}
	r.created = append(r.created, p.Code)
	return r.fakeProjects.Create(c, tenantID, actorID, p)
}

func TestCreateProject_GeneratedCodeRaceRetries(t *testing.T) {
	tenant, actor := uuid.New(), uuid.New()
	repo := &racyProjects{fakeProjects: newFakeProjects(), failures: 2}
	uc := NewCreateProjectUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
	in := goodCreate()
	in.Name, in.Code = "Apollo", ""
	p, err := uc.Execute(context.Background(), tenant, actor, in)
	require.NoError(t, err)
	assert.Equal(t, "APOLLO-3", p.Code)

	// explicit codes are never retried
	repo.failures = 1
	in.Code = "EXPLICIT"
	_, err = uc.Execute(context.Background(), tenant, actor, in)
	assert.ErrorIs(t, err, domainerrors.ErrProjectCodeTaken)
}

func TestSuggestProjectCode(t *testing.T) {
	tenant, tenantB, actor := uuid.New(), uuid.New(), uuid.New()
	repo := newFakeProjects()
	create := NewCreateProjectUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
	uc := NewSuggestProjectCodeUseCase(repo)
	ctx := context.Background()

	code, err := uc.Execute(ctx, tenant, "Apollo")
	require.NoError(t, err)
	assert.Equal(t, "APOLLO", code)

	in := goodCreate()
	in.Name, in.Code = "Apollo", ""
	_, err = create.Execute(ctx, tenant, actor, in)
	require.NoError(t, err)

	code, _ = uc.Execute(ctx, tenant, "Apollo")
	assert.Equal(t, "APOLLO-2", code)
	code, _ = uc.Execute(ctx, tenantB, "Apollo")
	assert.Equal(t, "APOLLO", code, "other tenant unaffected")

	_, err = uc.Execute(ctx, tenant, " ")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
	_, err = uc.Execute(ctx, uuid.Nil, "x")
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}
