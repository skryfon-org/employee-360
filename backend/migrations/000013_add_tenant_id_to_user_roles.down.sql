DROP INDEX IF EXISTS idx_user_roles_tenant_id;
ALTER TABLE user_roles DROP COLUMN IF EXISTS tenant_id;
