-- Users: covers super_admin, admin, and employee roles alike (role comes from
-- user_roles, not a column here). password_hash is nullable because a user
-- invited via the onboarding flow exists (is_active = false) before they've
-- accepted their invitation and set a password.
CREATE TABLE users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    department_id     UUID REFERENCES departments (id) ON DELETE SET NULL,
    position_id       UUID REFERENCES positions (id) ON DELETE SET NULL,
    email             TEXT NOT NULL,
    first_name        TEXT NOT NULL,
    last_name         TEXT NOT NULL,
    password_hash     TEXT,
    email_verified_at TIMESTAMPTZ,
    last_login_at     TIMESTAMPTZ,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_users_tenant_id_email UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant_id ON users (tenant_id);
CREATE INDEX idx_users_department_id ON users (department_id);
CREATE INDEX idx_users_position_id ON users (position_id);
