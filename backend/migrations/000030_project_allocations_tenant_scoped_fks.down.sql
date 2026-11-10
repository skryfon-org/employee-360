ALTER TABLE project_allocations
    DROP CONSTRAINT IF EXISTS fk_project_allocations_tenant_user,
    DROP CONSTRAINT IF EXISTS fk_project_allocations_tenant_project;

ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_users_tenant_id_id;
ALTER TABLE projects DROP CONSTRAINT IF EXISTS uq_projects_tenant_id_id;
