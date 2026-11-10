-- User invitations: admin-driven onboarding. Only the token hash is
-- persisted -- the raw token is emailed and never stored. department_id and
-- position_id are optional at invite time (a user can be invited before
-- being assigned either).
CREATE TABLE user_invitations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    role_id       UUID NOT NULL REFERENCES roles (id),
    department_id UUID REFERENCES departments (id) ON DELETE SET NULL,
    position_id   UUID REFERENCES positions (id) ON DELETE SET NULL,
    invited_by    UUID NOT NULL REFERENCES users (id),
    token_hash    TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    accepted_at   TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_invitations_token_hash UNIQUE (token_hash)
);

CREATE INDEX idx_user_invitations_tenant_id ON user_invitations (tenant_id);
CREATE INDEX idx_user_invitations_role_id ON user_invitations (role_id);
CREATE INDEX idx_user_invitations_department_id ON user_invitations (department_id);
CREATE INDEX idx_user_invitations_position_id ON user_invitations (position_id);
CREATE INDEX idx_user_invitations_invited_by ON user_invitations (invited_by);
