-- Supports the tenant-scoped invitation listing: filter by tenant, ordered
-- by created_at DESC, id DESC (deterministic pagination), live rows only.
CREATE INDEX idx_user_invitations_tenant_created
    ON user_invitations (tenant_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
