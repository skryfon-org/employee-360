package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
)

type fakeDashRepo struct {
	gotTenant    uuid.UUID
	tenantCounts *entity.TenantDashboardCounts
	roles        []entity.RoleUserCount
	err          error
}

func (f *fakeDashRepo) TenantCounts(_ context.Context, tenantID uuid.UUID, _ time.Time) (*entity.TenantDashboardCounts, error) {
	f.gotTenant = tenantID
	return f.tenantCounts, f.err
}
func (f *fakeDashRepo) UsersByRole(_ context.Context, tenantID uuid.UUID) ([]entity.RoleUserCount, error) {
	f.gotTenant = tenantID
	return f.roles, f.err
}

type fakeTenantReader struct{ t *entity.Tenant }

func (f fakeTenantReader) GetByID(context.Context, uuid.UUID) (*entity.Tenant, error) {
	if f.t == nil {
		return nil, domainerrors.ErrTenantNotFound
	}
	return f.t, nil
}

type fakeInvRepo struct {
	repository.UserInvitationRepository
	gotTenant uuid.UUID
	gotFilter repository.InvitationListFilter
	items     []*entity.InvitationListItem
}

func (f *fakeInvRepo) List(_ context.Context, tenantID uuid.UUID, flt repository.InvitationListFilter) ([]*entity.InvitationListItem, int64, error) {
	f.gotTenant, f.gotFilter = tenantID, flt
	return f.items, int64(len(f.items)), nil
}

func TestAdminDashboard_ScopesToTenantAndLimitsRecent(t *testing.T) {
	tid := uuid.New()
	dr := &fakeDashRepo{tenantCounts: &entity.TenantDashboardCounts{UsersTotal: 3, Departments: 2}}
	ir := &fakeInvRepo{items: []*entity.InvitationListItem{{}}}
	res, err := NewAdminDashboardUseCase(dr, ir).Execute(context.Background(), tid)
	require.NoError(t, err)
	assert.Equal(t, tid, dr.gotTenant)
	assert.Equal(t, tid, ir.gotTenant)
	assert.Equal(t, 5, ir.gotFilter.Limit)
	assert.Equal(t, int64(3), res.Counts.UsersTotal)
	assert.Len(t, res.RecentInvitations, 1)
}

func TestAdminDashboard_NilTenantUnauthorized(t *testing.T) {
	_, err := NewAdminDashboardUseCase(&fakeDashRepo{}, &fakeInvRepo{}).Execute(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}

func TestAdminDashboard_RepoErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	_, err := NewAdminDashboardUseCase(&fakeDashRepo{err: boom}, &fakeInvRepo{}).Execute(context.Background(), uuid.New())
	assert.ErrorIs(t, err, boom)
}

func TestSuperAdminDashboard_ReusesAdminAndAddsTenantAndRoles(t *testing.T) {
	tid := uuid.New()
	dr := &fakeDashRepo{
		tenantCounts: &entity.TenantDashboardCounts{UsersTotal: 3},
		roles:        []entity.RoleUserCount{{Role: "admin", Total: 1, Active: 1}, {Role: "employee", Total: 2, Active: 1, Inactive: 1}},
	}
	ir := &fakeInvRepo{items: []*entity.InvitationListItem{{}}}
	admin := NewAdminDashboardUseCase(dr, ir)
	tenant := &entity.Tenant{ID: tid, Name: "Acme", IsActive: true}
	res, err := NewSuperAdminDashboardUseCase(admin, dr, fakeTenantReader{tenant}).Execute(context.Background(), tid)
	require.NoError(t, err)
	assert.Equal(t, tid, dr.gotTenant)
	assert.Equal(t, tid, ir.gotTenant)
	assert.Equal(t, tenant, res.Tenant)
	assert.Equal(t, int64(3), res.Counts.UsersTotal)
	assert.Len(t, res.UsersByRole, 2)
	assert.Len(t, res.RecentInvitations, 1)
}

func TestSuperAdminDashboard_Errors(t *testing.T) {
	admin := NewAdminDashboardUseCase(&fakeDashRepo{tenantCounts: &entity.TenantDashboardCounts{}}, &fakeInvRepo{})
	_, err := NewSuperAdminDashboardUseCase(admin, &fakeDashRepo{}, fakeTenantReader{}).Execute(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domainerrors.ErrTenantNotFound)
	_, err = NewSuperAdminDashboardUseCase(admin, &fakeDashRepo{}, fakeTenantReader{}).Execute(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}
