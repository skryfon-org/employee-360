-- Allocation of an employee to a project for a date range at a given percentage.
-- Total allocation across projects may exceed 100% (warn-only, computed at read
-- time). Overlapping ranges for the same (project, user) are rejected by the
-- usecase under a lock on the project row (no btree_gist extension required).
CREATE TABLE project_allocations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    project_id         UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    allocation_percent SMALLINT NOT NULL,
    role_on_project    TEXT,
    start_date         DATE NOT NULL,
    end_date           DATE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by         UUID,
    updated_by         UUID,
    deleted_by         UUID,
    deleted_at         TIMESTAMPTZ,
    CONSTRAINT chk_project_allocations_percent CHECK (allocation_percent BETWEEN 1 AND 100),
    CONSTRAINT chk_project_allocations_date_order CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_project_allocations_tenant_id ON project_allocations (tenant_id);
CREATE INDEX idx_project_allocations_project_id ON project_allocations (project_id);
CREATE INDEX idx_project_allocations_user_id ON project_allocations (user_id);
