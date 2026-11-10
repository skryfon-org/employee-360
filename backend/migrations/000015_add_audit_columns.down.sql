-- Reverse of 000015 up. Non-destructive: soft-deleted users are kept (so FKs
-- such as user_invitations.invited_by and cascades to user_roles/tokens are not
-- touched). Their emails are disambiguated, and they are deactivated so a
-- renamed leftover cannot log in, before the original unique constraint is
-- restored. Soft-deleted users that never collided are renamed too, keeping the
-- rule simple and deterministic.

DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS uq_users_tenant_id_email;
UPDATE users
SET email = email || '+deleted-' || id::text,
    is_active = FALSE
WHERE deleted_at IS NOT NULL;
ALTER TABLE users ADD CONSTRAINT uq_users_tenant_id_email UNIQUE (tenant_id, email);

ALTER TABLE user_roles
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE user_invitations
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE roles
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE users
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE positions
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE departments
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE tenant_domains
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;
