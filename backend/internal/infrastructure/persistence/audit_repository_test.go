//go:build integration

package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestGormAuditRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormAuditRepository(db)
	a := createTestTenant(t, db, "audit-a-"+uuid.NewString())
	b := createTestTenant(t, db, "audit-b-"+uuid.NewString())
	c := context.Background()

	now := time.Now().UTC()
	entityID := uuid.New()
	mk := func(tenant uuid.UUID, eid uuid.UUID) *entity.AuditLog {
		return &entity.AuditLog{ID: uuid.New(), TenantID: tenant, Action: "update", EntityType: "role",
			EntityID: eid, Metadata: "{}", CreatedAt: now, UpdatedAt: now}
	}
	require.NoError(t, repo.Create(c, mk(a.ID, entityID)))
	require.NoError(t, repo.Create(c, mk(a.ID, uuid.New())))
	// Tenant B has a log on the SAME entity ID; it must not leak into A's results.
	require.NoError(t, repo.Create(c, mk(b.ID, entityID)))

	// Happy path in own tenant.
	logs, total, err := repo.ListByTenantID(c, a.ID, 10, 0)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, logs, 2)
	for _, l := range logs {
		require.Equal(t, a.ID, l.TenantID)
	}
	logs, total, err = repo.ListByEntity(c, a.ID, "role", entityID, 10, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, logs, 1)
	require.Equal(t, a.ID, logs[0].TenantID)

	// Tenant B sees only its own row.
	logs, total, err = repo.ListByTenantID(c, b.ID, 10, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, b.ID, logs[0].TenantID)

	// The nil tenant matches nothing.
	logs, total, err = repo.ListByTenantID(c, uuid.Nil, 10, 0)
	require.NoError(t, err)
	require.Empty(t, logs)
	require.EqualValues(t, 0, total)
	logs, total, err = repo.ListByEntity(c, uuid.Nil, "role", entityID, 10, 0)
	require.NoError(t, err)
	require.Empty(t, logs)
	require.EqualValues(t, 0, total)
}
