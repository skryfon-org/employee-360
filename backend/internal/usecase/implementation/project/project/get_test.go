package project

import (
	"context"
	"testing"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProject_CrossTenantNotFound(t *testing.T) {
	tenant, actor := uuid.New(), uuid.New()
	repo := newFakeProjects()
	p, err := NewCreateProjectUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil).Execute(context.Background(), tenant, actor, goodCreate())
	require.NoError(t, err)
	uc := NewGetProjectUseCase(repo)
	got, err := uc.Execute(context.Background(), tenant, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	_, err = uc.Execute(context.Background(), uuid.New(), p.ID)
	assert.ErrorIs(t, err, domainerrors.ErrProjectNotFound)
}
