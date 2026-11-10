ALTER TABLE organization_settings
    DROP CONSTRAINT IF EXISTS chk_organization_settings_welcome_message_length,
    DROP COLUMN IF EXISTS welcome_message;
