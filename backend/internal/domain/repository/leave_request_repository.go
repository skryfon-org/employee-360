package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// LeaveRequestFilter narrows a leave request listing. Zero fields are ignored.
// From/To select requests intersecting the inclusive [From, To] date range.
type LeaveRequestFilter struct {
	Status string
	// ExcludeStatus, when set, omits requests with this status.
	ExcludeStatus string
	UserID        *uuid.UUID
	ApproverID    *uuid.UUID
	LeaveTypeID   *uuid.UUID
	From          *time.Time
	To            *time.Time
	// OldestFirst orders by start_date ASC instead of the default newest first.
	OldestFirst bool
}

// LeaveRequestRepository persists leave requests, tenant scoped and soft-delete aware.
type LeaveRequestRepository interface {
	// LockUser takes a transaction-scoped advisory lock serializing a user's
	// leave applications (overlap check + insert); it must run in a transaction.
	LockUser(ctx context.Context, tenantID, userID uuid.UUID) error
	Create(ctx context.Context, tenantID, actorID uuid.UUID, r *entity.LeaveRequest) error
	// GetByID returns ErrLeaveRequestNotFound when missing or in another tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.LeaveRequest, error)
	// GetByIDForUpdate is GetByID with a row lock; it must run in a transaction.
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*entity.LeaveRequest, error)
	// GetDetail is GetByID joined with display names.
	GetDetail(ctx context.Context, tenantID, id uuid.UUID) (*entity.LeaveRequestDetail, error)
	// UpdateStatus persists only the status (+ updated_at/by); the decision
	// trail (decided_by/at, decision_comment) is left untouched. Used by cancel.
	UpdateStatus(ctx context.Context, tenantID, actorID uuid.UUID, r *entity.LeaveRequest) error
	// RecordDecision persists status together with decided_by/at and
	// decision_comment. Used by approve and reject.
	RecordDecision(ctx context.Context, tenantID, actorID uuid.UUID, r *entity.LeaveRequest) error
	// HasOverlap reports whether the user has a pending/approved request
	// intersecting [start, end]. Two half-day requests on the same single day
	// with different parts do not overlap. excludeID is skipped.
	HasOverlap(ctx context.Context, tenantID, userID uuid.UUID, start, end time.Time, dayPart string, excludeID uuid.UUID) (bool, error)
	// ListByUser lists the user's requests, newest start first, with the total
	// count. limit <= 0 means no limit.
	ListByUser(ctx context.Context, tenantID, userID uuid.UUID, f LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error)
	// ListByApprover lists requests addressed to approverID, newest start first,
	// with the total count. limit <= 0 means no limit.
	ListByApprover(ctx context.Context, tenantID, approverID uuid.UUID, f LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error)
	// ApproverSummary reports, in one query, how many pending requests are
	// addressed to approverID and whether any request (any status) ever was.
	// Tenant scoped and soft-delete aware, like ListByApprover.
	ApproverSummary(ctx context.Context, tenantID, approverID uuid.UUID) (pending int64, everAddressed bool, err error)
	// List lists all tenant requests with pagination plus the total count.
	List(ctx context.Context, tenantID uuid.UUID, f LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error)
}

// LeaveApproverRepository resolves who may be chosen as approver.
type LeaveApproverRepository interface {
	// ListActiveAdmins returns active, non-deleted users of the tenant holding
	// the admin or super_admin role, ordered by name.
	ListActiveAdmins(ctx context.Context, tenantID uuid.UUID) ([]*entity.LeaveApprover, error)
	// GetActiveUser returns the user as an approver candidate when they are an
	// active, non-deleted user of the tenant; ok=false otherwise.
	GetActiveUser(ctx context.Context, tenantID, userID uuid.UUID) (a *entity.LeaveApprover, ok bool, err error)
}
