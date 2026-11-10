package project

import (
	"context"
	"testing"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects(t *testing.T) {
	tenant, actor := uuid.New(), uuid.New()
	repo := newFakeProjects()
	_, err := NewCreateProjectUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil).Execute(context.Background(), tenant, actor, goodCreate())
	require.NoError(t, err)

	list := NewListProjectsUseCase(repo)
	res, err := list.Execute(context.Background(), tenant, prjtypes.ListProjectsQuery{Page: 0, PageSize: 1000, Search: "apo"})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Page)
	assert.Equal(t, 100, res.PageSize)
	assert.EqualValues(t, 1, res.Total)
	_, err = list.Execute(context.Background(), tenant, prjtypes.ListProjectsQuery{Status: "bad"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidProjectData)
	_, err = list.Execute(context.Background(), uuid.Nil, prjtypes.ListProjectsQuery{})
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}
