-- Add audit/soft-delete columns (created_by, updated_by, deleted_by, deleted_at)
-- to tables that lack them. Actor columns are plain nullable UUIDs with NO
-- foreign key (bootstrap/system/pre-auth writes have no user row, and an FK to
-- users would block hard-purging users). Skipped on purpose: audit_logs
-- (append-only), refresh_tokens / password_reset_tokens (ephemeral credential
-- rows, system-generated and purged), river_* (third-party schema).
-- user_roles is a join table: only created_by/updated_by (rows are hard-deleted;
-- revocation is recorded in audit_logs).

ALTER TABLE tenants
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE tenant_domains
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE departments
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE positions
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE users
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE roles
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE user_invitations
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID,
    ADD COLUMN deleted_by UUID,
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE user_roles
    ADD COLUMN created_by UUID,
    ADD COLUMN updated_by UUID;

-- REVERSAL SEMANTICS (see the down file): the down migration is lossless. Soft-
-- deleted users are NOT purged; their emails are rewritten to
-- "<email>+deleted-<id>" (tenant_id, id are unique) so the original
-- UNIQUE (tenant_id, email) constraint can be restored while the rows, their
-- user_roles/token rows and any user_invitations.invited_by references are
-- preserved. The audit/soft-delete columns themselves (and any values in them)
-- are dropped, so after a down the "deleted" marker on those users is lost:
-- they become ordinary (still is_active-controlled) rows with a renamed email.
--
-- Users are soft-deleted (revoking an invitation removes the pending user), so
-- email uniqueness must only apply to live rows or the email could never be re-invited.
ALTER TABLE users DROP CONSTRAINT uq_users_tenant_id_email;
CREATE UNIQUE INDEX uq_users_tenant_id_email ON users (tenant_id, email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
