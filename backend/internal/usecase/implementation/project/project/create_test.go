package project

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProject(t *testing.T) {
	tenant, tenantB, actor := uuid.New(), uuid.New(), uuid.New()
	repo := newFakeProjects()
	rec := &ucsharedtest.RecordingAuditRecorder{}
	uc := NewCreateProjectUseCase(repo, rec, nil)
	ctx := context.Background()

	p, err := uc.Execute(ctx, tenant, actor, goodCreate())
	require.NoError(t, err)
	assert.Equal(t, "APO-1", p.Code, "code is normalised to upper case")
	assert.Equal(t, entity.ProjectStatusActive, p.Status)
	require.Len(t, rec.Logs, 1)
	assert.Equal(t, domainaudit.ActionProjectCreate, rec.Logs[0].Action)

	t.Run("code unique per tenant", func(t *testing.T) {
		_, err := uc.Execute(ctx, tenant, actor, goodCreate())
		assert.ErrorIs(t, err, domainerrors.ErrProjectCodeTaken)
		_, err = uc.Execute(ctx, tenantB, actor, goodCreate())
		assert.NoError(t, err, "same code is fine in another tenant")
	})
	t.Run("validation", func(t *testing.T) {
		cases := map[string]func(*prjtypes.CreateProjectInput){
			"name":   func(i *prjtypes.CreateProjectInput) { i.Name = " " },
			"code":   func(i *prjtypes.CreateProjectInput) { i.Code = "has space" },
			"long":   func(i *prjtypes.CreateProjectInput) { i.Code = strings.Repeat("A", 31) },
			"status": func(i *prjtypes.CreateProjectInput) { i.Status = "bogus" },
			"start":  func(i *prjtypes.CreateProjectInput) { i.StartDate = time.Time{} },
			"order":  func(i *prjtypes.CreateProjectInput) { i.EndDate = dp("2025-12-31") },
		}
		for name, mut := range cases {
			in := goodCreate()
			in.Code = "OK-" + name
			mut(&in)
			_, err := uc.Execute(ctx, tenant, actor, in)
			assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData, name)
		}
	})
	t.Run("end equal to start is allowed", func(t *testing.T) {
		in := goodCreate()
		in.Code = "SAME"
		in.EndDate = dp("2026-01-01")
		_, err := uc.Execute(ctx, tenant, actor, in)
		assert.NoError(t, err)
	})
	t.Run("nil identity", func(t *testing.T) {
		_, err := uc.Execute(ctx, uuid.Nil, actor, goodCreate())
		assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})
}
