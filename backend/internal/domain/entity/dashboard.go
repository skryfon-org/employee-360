package entity

// InvitationStatusCounts is the number of invitations in each derived
// lifecycle state (soft-deleted invitations are never counted).
type InvitationStatusCounts struct {
	Pending  int64
	Accepted int64
	Expired  int64
	Revoked  int64
}

// TenantDashboardCounts are the aggregate counts behind the tenant admin
// dashboard, all scoped to a single tenant.
type TenantDashboardCounts struct {
	UsersTotal  int64
	UsersActive int64
	// UsersPendingInvited counts invited users who have not yet completed
	// onboarding (inactive, no password set, never verified, never logged in).
	UsersPendingInvited int64
	Invitations         InvitationStatusCounts
	Departments         int64
	Positions           int64
}

// RoleUserCount is the number of (non-deleted) users holding a role within a
// tenant. A user holding several roles is counted once under each.
type RoleUserCount struct {
	Role     string
	Total    int64
	Active   int64
	Inactive int64
}
