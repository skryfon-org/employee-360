package department

import (
	"context"
	"errors"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

type mockDepartmentRepo struct {
	createFn       func(ctx context.Context, tenantID, actorID uuid.UUID, department *entity.Department) error
	getByIDFn      func(ctx context.Context, tenantID, id uuid.UUID) (*entity.Department, error)
	listFn         func(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error)
	updateFn       func(ctx context.Context, tenantID, actorID uuid.UUID, department *entity.Department) error
	deleteFn       func(ctx context.Context, tenantID, id, actorID uuid.UUID) error
	existsByNameFn func(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)
	forUpdateCalls int
	isReferencedFn func(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
}

func (m *mockDepartmentRepo) Create(ctx context.Context, tenantID, actorID uuid.UUID, d *entity.Department) error {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, actorID, d)
	}
	return nil
}

func (m *mockDepartmentRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Department, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockDepartmentRepo) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*entity.Department, error) {
	m.forUpdateCalls++
	return m.GetByID(ctx, tenantID, id)
}
func (m *mockDepartmentRepo) List(ctx context.Context, tenantID uuid.UUID, isActive *bool, limit, offset int) ([]*entity.Department, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, isActive, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockDepartmentRepo) Update(ctx context.Context, tenantID, actorID uuid.UUID, d *entity.Department) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, actorID, d)
	}
	return nil
}

func (m *mockDepartmentRepo) Delete(ctx context.Context, tenantID, id, actorID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, id, actorID)
	}
	return nil
}

func (m *mockDepartmentRepo) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(ctx, tenantID, name)
	}
	return false, nil
}

func (m *mockDepartmentRepo) IsReferenced(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	if m.isReferencedFn != nil {
		return m.isReferencedFn(ctx, tenantID, id)
	}
	return false, nil
}

func TestCreateDepartmentUseCase(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	bg := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		repo := &mockDepartmentRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				assert.Equal(t, tenantID, tID)
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, d *entity.Department) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				assert.Equal(t, "Engineering", d.Name)
				assert.Equal(t, "Dev team", d.Description)
				assert.Equal(t, tenantID, d.TenantID)
				return nil
			},
		}

		uc := NewCreateDepartmentUseCase(repo, auditRepo, nil)
		dept, err := uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{
			Name:        "  Engineering  ",
			Description: "  Dev team  ",
		})
		require.NoError(t, err)
		assert.Equal(t, "Engineering", dept.Name)
		assert.Equal(t, "Dev team", dept.Description)
		assert.Equal(t, tenantID, dept.TenantID)

		require.Len(t, auditRepo.Logs, 1)
		assert.Equal(t, domainaudit.ActionDepartmentCreate, auditRepo.Logs[0].Action)
		assert.Equal(t, domainaudit.EntityDepartment, auditRepo.Logs[0].EntityType)
		assert.Equal(t, dept.ID, auditRepo.Logs[0].EntityID)
		assert.Equal(t, &actorID, auditRepo.Logs[0].ActorUserID)
		assert.Equal(t, tenantID, auditRepo.Logs[0].TenantID)
	})

	t.Run("is_active defaults to true when omitted", func(t *testing.T) {
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		var persisted bool
		repo := &mockDepartmentRepo{
			createFn: func(ctx context.Context, tID, aID uuid.UUID, d *entity.Department) error {
				persisted = d.IsActive
				return nil
			},
		}
		dept, err := NewCreateDepartmentUseCase(repo, auditRepo, nil).Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{Name: "Ops"})
		require.NoError(t, err)
		assert.True(t, dept.IsActive)
		assert.True(t, persisted)
		require.Len(t, auditRepo.Logs, 1)
		assert.Contains(t, auditRepo.Logs[0].Metadata, `"is_active":true`)
	})

	t.Run("explicit is_active false is honoured", func(t *testing.T) {
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		f := false
		var persisted = true
		repo := &mockDepartmentRepo{
			createFn: func(ctx context.Context, tID, aID uuid.UUID, d *entity.Department) error {
				persisted = d.IsActive
				return nil
			},
		}
		dept, err := NewCreateDepartmentUseCase(repo, auditRepo, nil).Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{Name: "Ops", IsActive: &f})
		require.NoError(t, err)
		assert.False(t, dept.IsActive)
		assert.False(t, persisted)
		require.Len(t, auditRepo.Logs, 1)
		assert.Contains(t, auditRepo.Logs[0].Metadata, `"is_active":false`)
	})

	t.Run("empty name returns validation error", func(t *testing.T) {
		uc := NewCreateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{Name: "   "})
		require.ErrorIs(t, err, ErrDepartmentNameRequired)
	})

	t.Run("name exceeds 100 characters", func(t *testing.T) {
		uc := NewCreateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{Name: strings.Repeat("a", 101)})
		require.ErrorIs(t, err, ErrDepartmentNameTooLong)
	})

	t.Run("description exceeds 500 characters", func(t *testing.T) {
		uc := NewCreateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{
			Name:        "Valid Name",
			Description: strings.Repeat("b", 501),
		})
		require.ErrorIs(t, err, ErrDescriptionTooLong)
	})

	t.Run("multibyte name and description are counted in characters", func(t *testing.T) {
		uc := NewCreateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		dept, err := uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{
			Name:        strings.Repeat("é", 100),
			Description: strings.Repeat("日", 500),
		})
		require.NoError(t, err)
		assert.Equal(t, strings.Repeat("é", 100), dept.Name)

		_, err = uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{Name: strings.Repeat("é", 101)})
		require.ErrorIs(t, err, ErrDepartmentNameTooLong)
		_, err = uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{
			Name:        "Valid",
			Description: strings.Repeat("日", 501),
		})
		require.ErrorIs(t, err, ErrDescriptionTooLong)
	})

	t.Run("name taken returns conflict", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				return true, nil
			},
		}
		uc := NewCreateDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{Name: "Existing"})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNameTaken)
	})

	t.Run("nil tenant or actor returns unauthorized", func(t *testing.T) {
		uc := NewCreateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, uuid.Nil, actorID, depttypes.CreateDepartmentInput{Name: "Engineering"})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(bg, tenantID, uuid.Nil, depttypes.CreateDepartmentInput{Name: "Engineering"})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockDepartmentRepo{
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, tID, aID uuid.UUID, d *entity.Department) error {
				return boom
			},
		}
		uc := NewCreateDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.CreateDepartmentInput{Name: "Engineering"})
		require.ErrorIs(t, err, boom)
	})
}
