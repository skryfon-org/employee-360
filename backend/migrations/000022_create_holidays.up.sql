-- Holidays: tenant-owned calendar entries, each belonging to a holiday category.
-- The category-belongs-to-same-tenant guard is enforced in the usecase layer.
CREATE TABLE holidays (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES holiday_categories (id) ON DELETE RESTRICT,
    name        VARCHAR(150) NOT NULL,
    description TEXT,
    date        DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID,
    updated_by  UUID,
    deleted_at  TIMESTAMPTZ,
    deleted_by  UUID
);

CREATE INDEX idx_holidays_tenant_id ON holidays (tenant_id);
CREATE INDEX idx_holidays_category_id ON holidays (category_id);
CREATE INDEX idx_holidays_tenant_id_date ON holidays (tenant_id, date);
CREATE INDEX idx_holidays_deleted_at ON holidays (deleted_at);
CREATE UNIQUE INDEX uq_holidays_tenant_id_date_lower_name
    ON holidays (tenant_id, date, LOWER(name))
    WHERE deleted_at IS NULL;
