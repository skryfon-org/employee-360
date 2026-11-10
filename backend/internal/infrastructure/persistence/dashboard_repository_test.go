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

func TestGormDashboardRepository_TenantScopedCounts(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormDashboardRepository(db)
	c := context.Background()
	a := createTestTenant(t, db, "dash-a-"+uuid.NewString())
	b := createTestTenant(t, db, "dash-b-"+uuid.NewString())
	now := time.Now().UTC()

	adminRole := uuid.New()
	empRole := uuid.New()
	require.NoError(t, db.Create(&entity.Role{ID: adminRole, TenantID: a.ID, Name: "admin"}).Error)
	require.NoError(t, db.Create(&entity.Role{ID: empRole, TenantID: a.ID, Name: "employee"}).Error)

	admin := createTestUser(t, db, a.ID, "adm+"+uuid.NewString()+"@example.com")
	emp := createTestUser(t, db, a.ID, "emp+"+uuid.NewString()+"@example.com")
	require.NoError(t, db.Model(&entity.User{}).Where("id = ?", emp.ID).Update("is_active", false).Error)
	// Pending invited user: inactive, no password.
	pending := createTestUser(t, db, a.ID, "pend+"+uuid.NewString()+"@example.com")
	require.NoError(t, db.Exec("UPDATE users SET is_active = false, password_hash = NULL WHERE id = ?", pending.ID).Error)
	// Deactivated never-logged-in user without an open invitation: not counted.
	deact := createTestUser(t, db, a.ID, "deact+"+uuid.NewString()+"@example.com")
	require.NoError(t, db.Exec("UPDATE users SET is_active = false, password_hash = NULL WHERE id = ?", deact.ID).Error)
	// Deactivated no-password user whose only invitation is expired: not counted.
	expiredUser := createTestUser(t, db, a.ID, "expd+"+uuid.NewString()+"@example.com")
	require.NoError(t, db.Exec("UPDATE users SET is_active = false, password_hash = NULL WHERE id = ?", expiredUser.ID).Error)
	// Same shape in tenant B with an open invitation in tenant A for its email: not counted for A or B.
	crossEmail := "cross+" + uuid.NewString() + "@example.com"
	cross := createTestUser(t, db, b.ID, crossEmail)
	require.NoError(t, db.Exec("UPDATE users SET is_active = false, password_hash = NULL WHERE id = ?", cross.ID).Error)
	// Soft-deleted user must not count.
	gone := createTestUser(t, db, a.ID, "gone+"+uuid.NewString()+"@example.com")
	require.NoError(t, db.Exec("UPDATE users SET deleted_at = NOW() WHERE id = ?", gone.ID).Error)
	// Other tenant's user must not leak.
	createTestUser(t, db, b.ID, "other+"+uuid.NewString()+"@example.com")

	require.NoError(t, db.Create(&entity.UserRole{ID: uuid.New(), TenantID: a.ID, UserID: admin.ID, RoleID: adminRole}).Error)
	require.NoError(t, db.Create(&entity.UserRole{ID: uuid.New(), TenantID: a.ID, UserID: emp.ID, RoleID: empRole}).Error)

	mkInv := func(tenant uuid.UUID, mod func(*entity.UserInvitation)) {
		inv := &entity.UserInvitation{ID: uuid.New(), TenantID: tenant, Email: uuid.NewString() + "@example.com",
			RoleID: empRole, InvitedBy: admin.ID, TokenHash: uuid.NewString(), ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}
		mod(inv)
		require.NoError(t, db.Table("user_invitations").Create(inv).Error)
	}
	mkInv(a.ID, func(i *entity.UserInvitation) { i.Email = pending.Email })
	mkInv(a.ID, func(i *entity.UserInvitation) { i.Email = crossEmail })
	mkInv(a.ID, func(i *entity.UserInvitation) { i.Email = deact.Email; i.RevokedAt = &now })
	mkInv(a.ID, func(i *entity.UserInvitation) { i.AcceptedAt = &now })
	mkInv(a.ID, func(i *entity.UserInvitation) { i.RevokedAt = &now })
	mkInv(a.ID, func(i *entity.UserInvitation) { i.ExpiresAt = now.Add(-time.Hour) })
	mkInv(a.ID, func(i *entity.UserInvitation) { i.Email = expiredUser.Email; i.ExpiresAt = now.Add(-time.Hour) })

	counts, err := repo.TenantCounts(c, a.ID, now)
	require.NoError(t, err)
	require.EqualValues(t, 5, counts.UsersTotal)
	require.EqualValues(t, 1, counts.UsersActive)
	require.EqualValues(t, 1, counts.UsersPendingInvited)
	require.Equal(t, entity.InvitationStatusCounts{Pending: 2, Accepted: 1, Expired: 2, Revoked: 2}, counts.Invitations)

	other, err := repo.TenantCounts(c, b.ID, now)
	require.NoError(t, err)
	require.EqualValues(t, 2, other.UsersTotal)
	require.EqualValues(t, 0, other.UsersPendingInvited)
	require.Equal(t, entity.InvitationStatusCounts{}, other.Invitations)

	roles, err := repo.UsersByRole(c, a.ID)
	require.NoError(t, err)
	require.Equal(t, []entity.RoleUserCount{
		{Role: "admin", Total: 1, Active: 1},
		{Role: "employee", Total: 1, Inactive: 1},
	}, roles)
	none, err := repo.UsersByRole(c, b.ID)
	require.NoError(t, err)
	require.Empty(t, none)
}
