-- Holiday calendar settings: exactly one row per tenant. A dedicated table because
-- Cycle 2 (migrations 000001-000020) created no tenant settings table to reuse.
CREATE TABLE holiday_calendar_settings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    default_view   VARCHAR(10) NOT NULL DEFAULT 'calendar',
    default_year   SMALLINT,
    week_start_day SMALLINT NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     UUID,
    updated_by     UUID,
    CONSTRAINT uq_holiday_calendar_settings_tenant_id UNIQUE (tenant_id), -- also indexes the FK
    CONSTRAINT chk_holiday_calendar_settings_default_view CHECK (default_view IN ('calendar', 'list')),
    CONSTRAINT chk_holiday_calendar_settings_default_year CHECK (default_year BETWEEN 2000 AND 2100),
    CONSTRAINT chk_holiday_calendar_settings_week_start_day CHECK (week_start_day BETWEEN 0 AND 6)
);
