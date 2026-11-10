ALTER TABLE organization_settings
    DROP CONSTRAINT IF EXISTS chk_organization_settings_weekly_off_days,
    DROP COLUMN IF EXISTS weekly_off_days;
