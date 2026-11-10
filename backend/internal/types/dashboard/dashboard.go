// Package dashboard holds result and response types for the dashboard usecases.
// None carry a client-supplied tenant ID.
package dashboard

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	invtypes "github.com/skryfon-org/employee-360/backend/internal/types/invitation"
)

// RecentLimit is the number of "recent" rows each dashboard returns.
const RecentLimit = 5

// AdminDashboardResult is the usecase output for the tenant admin dashboard.
type AdminDashboardResult struct {
	Counts            *entity.TenantDashboardCounts
	RecentInvitations []*entity.InvitationListItem
}

// SuperAdminDashboardResult is the usecase output for the super admin
// dashboard: the admin overview plus tenant info and a per-role user breakdown.
type SuperAdminDashboardResult struct {
	AdminDashboardResult
	Tenant      *entity.Tenant
	UsersByRole []entity.RoleUserCount
}

// InvitationCountsResponse is the invitation count per derived status.
type InvitationCountsResponse struct {
	Pending  int64 `json:"pending"`
	Accepted int64 `json:"accepted"`
	Expired  int64 `json:"expired"`
	Revoked  int64 `json:"revoked"`
}

// UserCountsResponse is the tenant user breakdown.
type UserCountsResponse struct {
	Total          int64 `json:"total"`
	Active         int64 `json:"active"`
	Inactive       int64 `json:"inactive"`
	PendingInvited int64 `json:"pending_invited"`
}

// AdminDashboardResponse is the tenant admin dashboard payload.
type AdminDashboardResponse struct {
	Users             UserCountsResponse                    `json:"users"`
	Invitations       InvitationCountsResponse              `json:"invitations"`
	Departments       int64                                 `json:"departments"`
	Positions         int64                                 `json:"positions"`
	RecentInvitations []invtypes.InvitationListItemResponse `json:"recent_invitations"`
}

// TenantInfoResponse identifies the organisation the dashboard describes.
type TenantInfoResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// RoleUserCountResponse is the user count for one role.
type RoleUserCountResponse struct {
	Role     string `json:"role"`
	Total    int64  `json:"total"`
	Active   int64  `json:"active"`
	Inactive int64  `json:"inactive"`
}

// SuperAdminDashboardResponse is the super admin dashboard payload: everything
// in the admin dashboard plus tenant info and users by role.
type SuperAdminDashboardResponse struct {
	AdminDashboardResponse
	Tenant      TenantInfoResponse      `json:"tenant"`
	UsersByRole []RoleUserCountResponse `json:"users_by_role"`
}

func toInvitationCounts(c entity.InvitationStatusCounts) InvitationCountsResponse {
	return InvitationCountsResponse{Pending: c.Pending, Accepted: c.Accepted, Expired: c.Expired, Revoked: c.Revoked}
}

// ToAdminDashboardResponse maps the usecase result to its API representation.
func ToAdminDashboardResponse(r *AdminDashboardResult) AdminDashboardResponse {
	recent := make([]invtypes.InvitationListItemResponse, 0, len(r.RecentInvitations))
	for _, i := range r.RecentInvitations {
		recent = append(recent, invtypes.ToInvitationListItemResponse(i))
	}
	c := r.Counts
	return AdminDashboardResponse{
		Users:             UserCountsResponse{Total: c.UsersTotal, Active: c.UsersActive, Inactive: c.UsersTotal - c.UsersActive, PendingInvited: c.UsersPendingInvited},
		Invitations:       toInvitationCounts(c.Invitations),
		Departments:       c.Departments,
		Positions:         c.Positions,
		RecentInvitations: recent,
	}
}

// ToSuperAdminDashboardResponse maps the usecase result to its API representation.
func ToSuperAdminDashboardResponse(r *SuperAdminDashboardResult) SuperAdminDashboardResponse {
	roles := make([]RoleUserCountResponse, 0, len(r.UsersByRole))
	for _, rc := range r.UsersByRole {
		roles = append(roles, RoleUserCountResponse{Role: rc.Role, Total: rc.Total, Active: rc.Active, Inactive: rc.Inactive})
	}
	return SuperAdminDashboardResponse{
		AdminDashboardResponse: ToAdminDashboardResponse(&r.AdminDashboardResult),
		Tenant:                 TenantInfoResponse{ID: r.Tenant.ID, Name: r.Tenant.Name, IsActive: r.Tenant.IsActive, CreatedAt: r.Tenant.CreatedAt},
		UsersByRole:            roles,
	}
}
