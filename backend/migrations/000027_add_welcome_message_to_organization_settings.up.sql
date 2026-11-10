-- Optional per-tenant welcome message included in the Welcome email sent when an
-- invited user accepts their invitation. Plain text; NULL means "use the built-in
-- default message". Tenant scoping is inherited from the organization_settings row.
ALTER TABLE organization_settings
    ADD COLUMN welcome_message TEXT,
    ADD CONSTRAINT chk_organization_settings_welcome_message_length
        CHECK (welcome_message IS NULL OR char_length(welcome_message) <= 2000);
