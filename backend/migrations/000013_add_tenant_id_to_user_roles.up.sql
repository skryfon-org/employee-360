-- user_roles was created without a tenant_id column (000007), relying on
-- both user_id and role_id already being tenant-scoped and matched at the
-- usecase layer. That leaves nothing at the DB level to stop a row pairing a
-- tenant-A user_id with a tenant-B role_id, violating the project's
-- multi-tenancy invariant (every tenant-owned table must carry tenant_id).
-- The table is empty at this point in migration history (pre-launch, no
-- data), so NOT NULL with no default is safe to add directly.
ALTER TABLE user_roles
    ADD COLUMN tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE;

CREATE INDEX idx_user_roles_tenant_id ON user_roles (tenant_id);
