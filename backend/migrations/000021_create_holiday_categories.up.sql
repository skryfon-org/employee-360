-- Holiday categories: tenant-owned labels (e.g. Public, Optional) used to group and colour-code holidays.
CREATE TABLE holiday_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    color       VARCHAR(7) NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID,
    updated_by  UUID,
    deleted_at  TIMESTAMPTZ,
    deleted_by  UUID,
    CONSTRAINT chk_holiday_categories_color_hex CHECK (color ~ '^#[0-9A-Fa-f]{6}$')
);

CREATE INDEX idx_holiday_categories_tenant_id ON holiday_categories (tenant_id);
CREATE INDEX idx_holiday_categories_deleted_at ON holiday_categories (deleted_at);
CREATE UNIQUE INDEX uq_holiday_categories_tenant_id_lower_name
    ON holiday_categories (tenant_id, LOWER(name))
    WHERE deleted_at IS NULL;
