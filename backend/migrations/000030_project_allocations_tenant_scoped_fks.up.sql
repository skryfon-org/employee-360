-- Make the DB guarantee that an allocation's project and user belong to the
-- allocation's own tenant. The single-column FKs from 000029 only prove the
-- referenced rows exist, not that they share tenant_id. Composite FKs
-- (tenant_id, project_id) / (tenant_id, user_id) close that gap.
-- The single-column FKs and indexes are kept (they still back ON DELETE CASCADE
-- and lookups by project_id / user_id).

-- Fail loudly (rather than with an opaque FK error) if existing rows violate
-- the invariant; such rows would be cross-tenant leaks that need manual repair.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM project_allocations a
        LEFT JOIN projects p ON p.id = a.project_id AND p.tenant_id = a.tenant_id
        LEFT JOIN users u ON u.id = a.user_id AND u.tenant_id = a.tenant_id
        WHERE p.id IS NULL OR u.id IS NULL
    ) THEN
        RAISE EXCEPTION 'project_allocations contains rows whose project or user belongs to a different tenant; repair them before applying this migration';
    END IF;
END $$;

-- FK targets: (tenant_id, id) must be unique. users only has (tenant_id, email).
ALTER TABLE projects ADD CONSTRAINT uq_projects_tenant_id_id UNIQUE (tenant_id, id);
ALTER TABLE users ADD CONSTRAINT uq_users_tenant_id_id UNIQUE (tenant_id, id);

ALTER TABLE project_allocations
    ADD CONSTRAINT fk_project_allocations_tenant_project
        FOREIGN KEY (tenant_id, project_id) REFERENCES projects (tenant_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_project_allocations_tenant_user
        FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id) ON DELETE CASCADE;
