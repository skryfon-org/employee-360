-- Organization settings: exactly one row per tenant. A dedicated table because no
-- general tenant settings table exists (holiday_calendar_settings is feature-specific).
--
-- Employee code format: code = prefix + zero-padded number (padding is a minimum
-- width; longer numbers are never truncated). Defaults: prefix 'EMP', padding 4,
-- start number 1, next number 1 -> first code EMP0001. A tenant without a row is
-- treated by the application as having these defaults (the row is created lazily
-- on the first code allocation or settings update).
--
-- employee_code_next_number is the server-managed counter; it is only ever changed
-- by the code allocator and the settings usecase, never directly by clients.
-- employee_code_last_issued_number is the highest number the allocator has handed
-- out (0 = no code generated yet); the start number can never be set at or below it.
CREATE TABLE organization_settings (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                  UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    employee_code_prefix       VARCHAR(10) NOT NULL DEFAULT 'EMP',
    employee_code_padding      SMALLINT NOT NULL DEFAULT 4,
    employee_code_start_number BIGINT NOT NULL DEFAULT 1,
    employee_code_next_number  BIGINT NOT NULL DEFAULT 1,
    employee_code_last_issued_number BIGINT NOT NULL DEFAULT 0,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                 UUID,
    updated_by                 UUID,
    CONSTRAINT uq_organization_settings_tenant_id UNIQUE (tenant_id), -- also indexes the FK
    CONSTRAINT chk_organization_settings_code_prefix CHECK (employee_code_prefix ~ '^[A-Z0-9]{1,10}$' AND employee_code_prefix !~ '[0-9]$'),
    CONSTRAINT chk_organization_settings_code_padding CHECK (employee_code_padding BETWEEN 1 AND 10),
    CONSTRAINT chk_organization_settings_code_start CHECK (employee_code_start_number BETWEEN 1 AND 9999999999),
    CONSTRAINT chk_organization_settings_code_next CHECK (employee_code_next_number >= employee_code_start_number),
    CONSTRAINT chk_organization_settings_code_last_issued CHECK (employee_code_last_issued_number >= 0),
    CONSTRAINT chk_organization_settings_code_length CHECK (length(employee_code_prefix) + employee_code_padding <= 20)
);

-- Backfill existing tenants with the defaults.
INSERT INTO organization_settings (tenant_id)
SELECT id FROM tenants;
