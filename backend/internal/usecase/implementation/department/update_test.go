package department

import (
	"context"
	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared/ucsharedtest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	depttypes "github.com/skryfon-org/employee-360/backend/internal/types/department"
)

func TestUpdateDepartmentUseCase(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	deptID := uuid.New()
	bg := context.Background()

	existing := &entity.Department{
		ID:          deptID,
		TenantID:    tenantID,
		Name:        "Old Name",
		Description: "Old Desc",
		CreatedAt:   time.Now().UTC().Add(-time.Hour),
		UpdatedAt:   time.Now().UTC().Add(-time.Hour),
	}

	t.Run("successful update with name change", func(t *testing.T) {
		auditRepo := &ucsharedtest.RecordingAuditRecorder{}
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				assert.Equal(t, tenantID, tID)
				return &entity.Department{
					ID:       deptID,
					TenantID: tenantID,
					Name:     "Old Name",
				}, nil
			},
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, "New Name", name)
				return false, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, d *entity.Department) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				assert.Equal(t, "New Name", d.Name)
				assert.Equal(t, "New Desc", d.Description)
				return nil
			},
		}

		uc := NewUpdateDepartmentUseCase(repo, auditRepo, nil)
		updated, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:          deptID,
			Name:        "  New Name  ",
			Description: "  New Desc  ",
		})
		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.Name)
		assert.Equal(t, "New Desc", updated.Description)

		require.Len(t, auditRepo.Logs, 1)
		assert.Equal(t, domainaudit.ActionDepartmentUpdate, auditRepo.Logs[0].Action)
		assert.Equal(t, domainaudit.EntityDepartment, auditRepo.Logs[0].EntityType)
		assert.Equal(t, deptID, auditRepo.Logs[0].EntityID)
		assert.Equal(t, &actorID, auditRepo.Logs[0].ActorUserID)
		assert.Equal(t, tenantID, auditRepo.Logs[0].TenantID)
	})

	t.Run("successful update keeping same name (case-insensitive)", func(t *testing.T) {
		existsChecked := false
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				assert.Equal(t, tenantID, tID)
				return &entity.Department{
					ID:       deptID,
					TenantID: tenantID,
					Name:     "Old Name",
				}, nil
			},
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				existsChecked = true
				return true, nil
			},
			updateFn: func(ctx context.Context, tID, aID uuid.UUID, d *entity.Department) error {
				assert.Equal(t, tenantID, tID)
				assert.Equal(t, actorID, aID)
				return nil
			},
		}

		uc := NewUpdateDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		updated, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:          deptID,
			Name:        "old name",
			Description: "Updated desc only",
		})
		require.NoError(t, err)
		assert.False(t, existsChecked, "should not check ExistsByName if name unchanged")
		assert.Equal(t, "old name", updated.Name)
	})

	t.Run("is_active toggle persists and logs a distinct audit action", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			before     bool
			input      *bool
			wantActive bool
			wantAction string
		}{
			{"deactivate", true, boolPtr(false), false, domainaudit.ActionDepartmentDeactivate},
			{"activate", false, boolPtr(true), true, domainaudit.ActionDepartmentActivate},
			{"nil keeps active", true, nil, true, ""},
			{"nil keeps inactive", false, nil, false, ""},
			{"same value is not a toggle", true, boolPtr(true), true, ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				auditRepo := &ucsharedtest.RecordingAuditRecorder{}
				var persisted bool
				repo := &mockDepartmentRepo{
					getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
						return &entity.Department{ID: deptID, TenantID: tenantID, Name: "Same", IsActive: tc.before}, nil
					},
					updateFn: func(ctx context.Context, tID, aID uuid.UUID, d *entity.Department) error {
						persisted = d.IsActive
						return nil
					},
				}
				out, err := NewUpdateDepartmentUseCase(repo, auditRepo, nil).Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{ID: deptID, Name: "Same", IsActive: tc.input})
				require.NoError(t, err)
				assert.Equal(t, tc.wantActive, out.IsActive)
				assert.Equal(t, tc.wantActive, persisted)
				require.NotEmpty(t, auditRepo.Logs)
				assert.Equal(t, domainaudit.ActionDepartmentUpdate, auditRepo.Logs[0].Action)
				if tc.wantAction == "" {
					assert.Len(t, auditRepo.Logs, 1)
				} else {
					require.Len(t, auditRepo.Logs, 2)
					assert.Equal(t, tc.wantAction, auditRepo.Logs[1].Action)
				}
			})
		}
	})

	t.Run("name taken returns conflict", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return existing, nil
			},
			existsByNameFn: func(ctx context.Context, tID uuid.UUID, name string) (bool, error) {
				return true, nil
			},
		}

		uc := NewUpdateDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:   deptID,
			Name: "Taken Name",
		})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNameTaken)
	})

	t.Run("nil uuid returns ErrDepartmentNotFound", func(t *testing.T) {
		uc := NewUpdateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:   uuid.Nil,
			Name: "Some Name",
		})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	})

	t.Run("not found in repo returns ErrDepartmentNotFound", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return nil, domainerrors.ErrDepartmentNotFound
			},
		}
		uc := NewUpdateDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:   deptID,
			Name: "Some Name",
		})
		require.ErrorIs(t, err, domainerrors.ErrDepartmentNotFound)
	})

	t.Run("empty name returns validation error", func(t *testing.T) {
		uc := NewUpdateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:   deptID,
			Name: "  ",
		})
		require.ErrorIs(t, err, ErrDepartmentNameRequired)
	})

	t.Run("name too long returns validation error", func(t *testing.T) {
		uc := NewUpdateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:   deptID,
			Name: strings.Repeat("x", 101),
		})
		require.ErrorIs(t, err, ErrDepartmentNameTooLong)
	})

	t.Run("multibyte name and description are counted in characters", func(t *testing.T) {
		repo := &mockDepartmentRepo{
			getByIDFn: func(ctx context.Context, tID, id uuid.UUID) (*entity.Department, error) {
				return &entity.Department{ID: deptID, TenantID: tenantID, Name: "Old"}, nil
			},
		}
		uc := NewUpdateDepartmentUseCase(repo, &ucsharedtest.RecordingAuditRecorder{}, nil)
		dept, err := uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{
			ID:          deptID,
			Name:        strings.Repeat("é", 100),
			Description: strings.Repeat("日", 500),
		})
		require.NoError(t, err)
		assert.Equal(t, strings.Repeat("é", 100), dept.Name)

		_, err = uc.Execute(bg, tenantID, actorID, depttypes.UpdateDepartmentInput{ID: deptID, Name: strings.Repeat("é", 101)})
		require.ErrorIs(t, err, ErrDepartmentNameTooLong)
	})

	t.Run("nil tenant or actor returns unauthorized", func(t *testing.T) {
		uc := NewUpdateDepartmentUseCase(&mockDepartmentRepo{}, &ucsharedtest.RecordingAuditRecorder{}, nil)
		_, err := uc.Execute(bg, uuid.Nil, actorID, depttypes.UpdateDepartmentInput{
			ID:   deptID,
			Name: "Valid Name",
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)

		_, err = uc.Execute(bg, tenantID, uuid.Nil, depttypes.UpdateDepartmentInput{
			ID:   deptID,
			Name: "Valid Name",
		})
		require.ErrorIs(t, err, domainerrors.ErrUnauthorized)
	})
}

func boolPtr(b bool) *bool { return &b }
