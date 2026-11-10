-- Add an is_active flag to departments. Inactive departments stay visible and
-- referenced by existing users/invitations but cannot be used for new invitations.
ALTER TABLE departments
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;
