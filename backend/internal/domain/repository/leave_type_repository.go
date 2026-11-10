package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// LeaveTypeRepository persists tenant-scoped leave types. Every method takes
// tenantID explicitly (resolved by the handler from the auth context) and
// excludes soft-deleted rows.
type LeaveTypeRepository interface {
	// Create returns ErrLeaveTypeCodeTaken on a duplicate (tenant, code).
	Create(ctx context.Context, tenantID, actorID uuid.UUID, lt *entity.LeaveType) error
	// GetByID returns ErrLeaveTypeNotFound when missing or in another tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.LeaveType, error)
	// List returns leave types ordered by name; activeOnly hides inactive ones.
	List(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*entity.LeaveType, error)
	// Update returns ErrLeaveTypeNotFound / ErrLeaveTypeCodeTaken.
	Update(ctx context.Context, tenantID, actorID uuid.UUID, lt *entity.LeaveType) error
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string, excludeID uuid.UUID) (bool, error)
}
