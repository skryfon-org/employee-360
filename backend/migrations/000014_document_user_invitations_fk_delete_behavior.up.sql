-- 000011_create_user_invitations left role_id/invited_by with the default
-- ON DELETE NO ACTION while department_id/position_id use ON DELETE SET
-- NULL. That's intentional, not an oversight -- document it via constraint
-- comments so a future migration author doesn't "fix" the inconsistency.
-- No schema or behavior change; comments only.
COMMENT ON CONSTRAINT user_invitations_role_id_fkey ON user_invitations IS
    'Intentionally NO ACTION, not SET NULL: a role with live invitations must not be deleted out from under them.';
COMMENT ON CONSTRAINT user_invitations_invited_by_fkey ON user_invitations IS
    'Intentionally NO ACTION, not SET NULL: a user with invitations they sent must not be deleted out from under them.';
