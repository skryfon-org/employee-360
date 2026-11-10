-- Tenant domains: one tenant may own many email domains. Login resolution
-- matches the domain portion of a user's email against this table to find
-- their tenant_id, so `domain` must be unique across ALL tenants, not just
-- within one.
CREATE TABLE tenant_domains (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    domain     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_domains_domain UNIQUE (domain)
);

CREATE INDEX idx_tenant_domains_tenant_id ON tenant_domains (tenant_id);
