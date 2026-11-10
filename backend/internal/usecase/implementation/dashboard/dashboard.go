// Package dashboard implements the read-only dashboard usecases.
package dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	dashtypes "github.com/skryfon-org/employee-360/backend/internal/types/dashboard"
	dashusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/dashboard"
)

// AdminDashboardUseCaseImpl implements dashusecase.AdminDashboardUseCase.
type AdminDashboardUseCaseImpl struct {
	dashRepo       repository.DashboardRepository
	invitationRepo repository.UserInvitationRepository
}

var _ dashusecase.AdminDashboardUseCase = (*AdminDashboardUseCaseImpl)(nil)

// NewAdminDashboardUseCase constructs an AdminDashboardUseCaseImpl.
func NewAdminDashboardUseCase(dashRepo repository.DashboardRepository, invitationRepo repository.UserInvitationRepository) *AdminDashboardUseCaseImpl {
	return &AdminDashboardUseCaseImpl{dashRepo: dashRepo, invitationRepo: invitationRepo}
}

// Execute aggregates the caller's tenant only.
func (u *AdminDashboardUseCaseImpl) Execute(c context.Context, tenantID uuid.UUID) (*dashtypes.AdminDashboardResult, error) {
	if tenantID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	now := time.Now()
	counts, err := u.dashRepo.TenantCounts(c, tenantID, now)
	if err != nil {
		return nil, err
	}
	recent, _, err := u.invitationRepo.List(c, tenantID, repository.InvitationListFilter{Now: now, Limit: dashtypes.RecentLimit})
	if err != nil {
		return nil, err
	}
	return &dashtypes.AdminDashboardResult{Counts: counts, RecentInvitations: recent}, nil
}

// SuperAdminDashboardUseCaseImpl implements dashusecase.SuperAdminDashboardUseCase.
// It reuses the admin dashboard and adds tenant info and a per-role user breakdown.
type SuperAdminDashboardUseCaseImpl struct {
	admin      dashusecase.AdminDashboardUseCase
	dashRepo   repository.DashboardRepository
	tenantRepo repository.TenantReader
}

var _ dashusecase.SuperAdminDashboardUseCase = (*SuperAdminDashboardUseCaseImpl)(nil)

// NewSuperAdminDashboardUseCase constructs a SuperAdminDashboardUseCaseImpl.
func NewSuperAdminDashboardUseCase(admin dashusecase.AdminDashboardUseCase, dashRepo repository.DashboardRepository, tenantRepo repository.TenantReader) *SuperAdminDashboardUseCaseImpl {
	return &SuperAdminDashboardUseCaseImpl{admin: admin, dashRepo: dashRepo, tenantRepo: tenantRepo}
}

// Execute scopes everything to tenantID; no cross-tenant aggregates are exposed.
func (u *SuperAdminDashboardUseCaseImpl) Execute(c context.Context, tenantID uuid.UUID) (*dashtypes.SuperAdminDashboardResult, error) {
	base, err := u.admin.Execute(c, tenantID)
	if err != nil {
		return nil, err
	}
	tenant, err := u.tenantRepo.GetByID(c, tenantID)
	if err != nil {
		return nil, err
	}
	roles, err := u.dashRepo.UsersByRole(c, tenantID)
	if err != nil {
		return nil, err
	}
	return &dashtypes.SuperAdminDashboardResult{AdminDashboardResult: *base, Tenant: tenant, UsersByRole: roles}, nil
}
