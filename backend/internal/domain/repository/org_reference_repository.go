package repository

import (
	"context"

	"github.com/google/uuid"
)

// OrgReferenceRepository answers tenant-scoped existence checks for
// organisation records referenced by other aggregates (e.g. an invitation's
// department and position). tenantID always comes from the verified request
// context, never from client input.
type OrgReferenceRepository interface {
	DepartmentExists(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
	// LockDepartmentShared checks the department exists in the tenant and takes a
	// shared row lock (FOR SHARE) held until the transaction ends, so a
	// concurrent department delete (which takes FOR UPDATE) serialises with it.
	// It reports whether the department was found and whether it is active.
	// Must run inside a transaction.
	LockDepartmentShared(ctx context.Context, tenantID, id uuid.UUID) (found, active bool, err error)
	// LockPositionShared is the position counterpart of LockDepartmentShared:
	// FOR SHARE, tenant-scoped, excluding soft-deleted rows. Must run inside a
	// transaction.
	LockPositionShared(ctx context.Context, tenantID, id uuid.UUID) (found, active bool, err error)
}
