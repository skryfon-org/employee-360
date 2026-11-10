-- Department/position history ("internal experience"), many per user.
-- users.department_id / position_id stay as the denormalised CURRENT value; this
-- table records every stint. effective_to NULL = the current (open) stint.
CREATE TABLE user_assignments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id        UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    department_id  UUID REFERENCES departments (id) ON DELETE SET NULL,
    position_id    UUID REFERENCES positions (id) ON DELETE SET NULL,
    effective_from DATE NOT NULL,
    effective_to   DATE,
    change_reason  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     UUID,
    updated_by     UUID,
    deleted_by     UUID,
    deleted_at     TIMESTAMPTZ,
    CONSTRAINT chk_user_assignments_date_order CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX idx_user_assignments_tenant_id ON user_assignments (tenant_id);
CREATE INDEX idx_user_assignments_user_id ON user_assignments (user_id);
CREATE INDEX idx_user_assignments_department_id ON user_assignments (department_id);
CREATE INDEX idx_user_assignments_position_id ON user_assignments (position_id);
-- At most one open stint per user.
CREATE UNIQUE INDEX uq_user_assignments_open ON user_assignments (tenant_id, user_id) WHERE effective_to IS NULL AND deleted_at IS NULL;
