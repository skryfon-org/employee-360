package project

import (
	"context"
	"testing"

	"github.com/google/uuid"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	emptypes "github.com/skryfon-org/employee-360/backend/internal/types/employee"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateProject_StatusChangeAndCrossTenant(t *testing.T) {
	tenant, other, actor := uuid.New(), uuid.New(), uuid.New()
	repo := newFakeProjects()
	rec := &ucsharedtest.RecordingAuditRecorder{}
	p, err := NewCreateProjectUseCase(repo, rec, nil).Execute(context.Background(), tenant, actor, goodCreate())
	require.NoError(t, err)
	other2, err := NewCreateProjectUseCase(repo, rec, nil).Execute(context.Background(), tenant, actor, prjtypes.CreateProjectInput{Name: "B", Code: "B", StartDate: d("2026-01-01")})
	require.NoError(t, err)
	uc := NewUpdateProjectUseCase(repo, rec, nil)
	ctx := context.Background()

	got, err := uc.Execute(ctx, tenant, actor, prjtypes.UpdateProjectInput{ID: p.ID, Status: opt("archived")})
	require.NoError(t, err)
	assert.Equal(t, "archived", got.Status)
	assert.Equal(t, domainaudit.ActionProjectStatusChange, rec.Logs[len(rec.Logs)-1].Action)

	_, err = uc.Execute(ctx, other, actor, prjtypes.UpdateProjectInput{ID: p.ID, Status: opt("active")})
	assert.ErrorIs(t, err, domainerrors.ErrProjectNotFound, "cross-tenant is a 404")

	_, err = uc.Execute(ctx, tenant, actor, prjtypes.UpdateProjectInput{ID: p.ID, Status: opt("nope")})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)

	_, err = uc.Execute(ctx, tenant, actor, prjtypes.UpdateProjectInput{ID: p.ID, Code: opt("b")})
	assert.ErrorIs(t, err, domainerrors.ErrProjectCodeTaken)
	_ = other2

	_, err = uc.Execute(ctx, tenant, actor, prjtypes.UpdateProjectInput{ID: p.ID, EndDate: optDate("2025-01-01")})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData, "end before start")

	got, err = uc.Execute(ctx, tenant, actor, prjtypes.UpdateProjectInput{ID: p.ID, ClientName: opt("ACME"), EndDate: optDate("2026-06-01")})
	require.NoError(t, err)
	require.NotNil(t, got.ClientName)
	got, err = uc.Execute(ctx, tenant, actor, prjtypes.UpdateProjectInput{ID: p.ID, ClientName: optNull[string](), EndDate: optNull[emptypes.Date]()})
	require.NoError(t, err)
	assert.Nil(t, got.ClientName)
	assert.Nil(t, got.EndDate)
}
