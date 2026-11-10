-- Tenant-scoped leave types (Casual, Sick, Earned, Unpaid, ...). Types are never
-- hard-deleted by the API: is_active=false deactivates them.
CREATE TABLE leave_types (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    code        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID,
    updated_by  UUID,
    deleted_by  UUID,
    deleted_at  TIMESTAMPTZ,
    -- Target of the tenant-scoped composite FK from leave_requests.
    CONSTRAINT uq_leave_types_tenant_id_id UNIQUE (tenant_id, id)
);

CREATE INDEX idx_leave_types_tenant_id ON leave_types (tenant_id);
-- Code is unique per tenant among non-deleted leave types.
CREATE UNIQUE INDEX uq_leave_types_tenant_code ON leave_types (tenant_id, code) WHERE deleted_at IS NULL;

-- Seed the default leave types for every existing tenant (idempotent).
-- New tenants are seeded by cmd/bootstrap (seeder.DefaultLeaveTypes).
INSERT INTO leave_types (tenant_id, name, code, description)
SELECT t.id, d.name, d.code, d.description
FROM tenants t
CROSS JOIN (VALUES
    ('Casual', 'CASUAL', 'Short-notice personal leave.'),
    ('Sick',   'SICK',   'Leave for illness or medical appointments.'),
    ('Earned', 'EARNED', 'Planned annual or earned leave.'),
    ('Unpaid', 'UNPAID', 'Leave without pay.')
) AS d (name, code, description)
WHERE t.deleted_at IS NULL
ON CONFLICT DO NOTHING;
