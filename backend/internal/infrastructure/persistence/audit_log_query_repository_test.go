//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

func TestGormAuditLogQueryRepository(t *testing.T) {
	db := setupTestDB(t)
	writer := NewGormAuditRepository(db)
	repo := NewGormAuditLogQueryRepository(db)
	c := context.Background()

	a := createTestTenant(t, db, "auditq-a-"+uuid.NewString())
	b := createTestTenant(t, db, "auditq-b-"+uuid.NewString())
	actorA := createTestUser(t, db, a.ID, "actor-"+uuid.NewString()+"@a.test")
	actorB := createTestUser(t, db, b.ID, "actor-"+uuid.NewString()+"@b.test")

	base := time.Now().UTC().Truncate(time.Second).Add(-time.Hour)
	mk := func(tenant uuid.UUID, actor *uuid.UUID, action, etype string, at time.Time) *entity.AuditLog {
		l := &entity.AuditLog{ID: uuid.New(), TenantID: tenant, ActorUserID: actor, Action: action, EntityType: etype,
			EntityID: uuid.New(), Metadata: `{"k":"v"}`, CreatedAt: at, UpdatedAt: at}
		require.NoError(t, writer.Create(c, l))
		return l
	}
	l1 := mk(a.ID, &actorA.ID, "department.create", "department", base)
	l2 := mk(a.ID, &actorA.ID, "department.update", "department", base.Add(time.Minute))
	l3 := mk(a.ID, nil, "position.create", "position", base.Add(2*time.Minute)) // system action
	l4 := mk(a.ID, &actorA.ID, "departmentXother", "other", base.Add(3*time.Minute))
	lB := mk(b.ID, &actorB.ID, "department.create", "department", base.Add(time.Minute))

	t.Run("newest first, actor resolved, nil actor for system", func(t *testing.T) {
		got, total, err := repo.List(c, a.ID, repository.AuditLogFilter{}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 4, total)
		require.Equal(t, []uuid.UUID{l4.ID, l3.ID, l2.ID, l1.ID}, []uuid.UUID{got[0].ID, got[1].ID, got[2].ID, got[3].ID})
		require.Nil(t, got[1].Actor)
		require.NotNil(t, got[0].Actor)
		require.Equal(t, actorA.ID, got[0].Actor.ID)
		require.Equal(t, actorA.Email, got[0].Actor.Email)
		require.Equal(t, "Test User", got[0].Actor.DisplayName())
		require.JSONEq(t, `{"k":"v"}`, got[0].Metadata)
		for _, e := range got {
			require.Equal(t, a.ID, e.TenantID)
		}
	})

	t.Run("tenant isolation", func(t *testing.T) {
		got, total, err := repo.List(c, b.ID, repository.AuditLogFilter{}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Equal(t, lB.ID, got[0].ID)

		_, err = repo.GetByID(c, b.ID, l1.ID)
		require.ErrorIs(t, err, domainerrors.ErrAuditLogNotFound)
		e, err := repo.GetByID(c, a.ID, l1.ID)
		require.NoError(t, err)
		require.Equal(t, l1.ID, e.ID)
		require.Equal(t, actorA.ID, e.Actor.ID)

		// Filtering by another tenant's actor never leaks rows.
		got, total, err = repo.List(c, a.ID, repository.AuditLogFilter{ActorUserID: &actorB.ID}, 10, 0)
		require.NoError(t, err)
		require.Empty(t, got)
		require.EqualValues(t, 0, total)

		got, total, err = repo.List(c, uuid.Nil, repository.AuditLogFilter{}, 10, 0)
		require.NoError(t, err)
		require.Empty(t, got)
		require.EqualValues(t, 0, total)
	})

	t.Run("action exact and prefix (LIKE-escaped)", func(t *testing.T) {
		got, total, err := repo.List(c, a.ID, repository.AuditLogFilter{Action: "department.create"}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Equal(t, l1.ID, got[0].ID)

		// Prefix "department." matches the two department.* rows but not "departmentXother".
		got, total, err = repo.List(c, a.ID, repository.AuditLogFilter{Action: "department.", ActionPrefix: true}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		require.Len(t, got, 2)

		// "_" is a literal, not a wildcard.
		_, total, err = repo.List(c, a.ID, repository.AuditLogFilter{Action: "department_", ActionPrefix: true}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 0, total)
	})

	t.Run("actor, entity_type and time range filters", func(t *testing.T) {
		_, total, err := repo.List(c, a.ID, repository.AuditLogFilter{ActorUserID: &actorA.ID}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 3, total)

		_, total, err = repo.List(c, a.ID, repository.AuditLogFilter{EntityType: "position"}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)

		from, to := base.Add(time.Minute), base.Add(2*time.Minute) // inclusive both ends
		got, total, err := repo.List(c, a.ID, repository.AuditLogFilter{From: &from, To: &to}, 10, 0)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		require.Equal(t, []uuid.UUID{l3.ID, l2.ID}, []uuid.UUID{got[0].ID, got[1].ID})
	})

	t.Run("pagination keeps total", func(t *testing.T) {
		got, total, err := repo.List(c, a.ID, repository.AuditLogFilter{}, 2, 2)
		require.NoError(t, err)
		require.EqualValues(t, 4, total)
		require.Equal(t, []uuid.UUID{l2.ID, l1.ID}, []uuid.UUID{got[0].ID, got[1].ID})
	})
}
