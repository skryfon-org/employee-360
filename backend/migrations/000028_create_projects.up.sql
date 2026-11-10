-- Tenant-scoped projects that employees can be allocated to.
-- Projects are never hard-deleted by the API: status moves to 'archived' instead.
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    code        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    client_name TEXT,
    status      TEXT NOT NULL DEFAULT 'active',
    start_date  DATE NOT NULL,
    end_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID,
    updated_by  UUID,
    deleted_by  UUID,
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT chk_projects_status CHECK (status IN ('active', 'on_hold', 'completed', 'archived')),
    CONSTRAINT chk_projects_date_order CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_projects_tenant_id ON projects (tenant_id);
CREATE INDEX idx_projects_tenant_status ON projects (tenant_id, status);
-- Project code is unique per tenant among non-deleted projects.
CREATE UNIQUE INDEX uq_projects_tenant_code ON projects (tenant_id, code) WHERE deleted_at IS NULL;
