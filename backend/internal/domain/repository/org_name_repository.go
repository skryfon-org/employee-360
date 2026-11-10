package repository

import (
	"context"

	"github.com/google/uuid"
)

// OrgNameRepository resolves display names of organisation records (department,
// position) for read-only views. Lookups are tenant-scoped and exclude
// soft-deleted rows; found is false when no live row exists in the tenant.
type OrgNameRepository interface {
	DepartmentName(ctx context.Context, tenantID, id uuid.UUID) (name string, found bool, err error)
	PositionName(ctx context.Context, tenantID, id uuid.UUID) (name string, found bool, err error)
}
