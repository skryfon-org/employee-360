-- Employee leave requests. approver_id is the manager or tenant admin the
-- applicant chose; status moves pending -> approved | rejected | cancelled
-- (approved -> cancelled while not yet started). Rows are never hard-deleted.
CREATE TABLE leave_requests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id          UUID NOT NULL,
    leave_type_id    UUID NOT NULL,
    start_date       DATE NOT NULL,
    end_date         DATE NOT NULL,
    day_part         TEXT NOT NULL DEFAULT 'full',
    reason           TEXT NOT NULL DEFAULT '',
    approver_id      UUID NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending',
    decided_by       UUID,
    decided_at       TIMESTAMPTZ,
    decision_comment TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       UUID,
    updated_by       UUID,
    deleted_by       UUID,
    deleted_at       TIMESTAMPTZ,
    CONSTRAINT chk_leave_requests_date_order CHECK (end_date >= start_date),
    CONSTRAINT chk_leave_requests_day_part CHECK (day_part IN ('full', 'first_half', 'second_half')),
    CONSTRAINT chk_leave_requests_half_single_day CHECK (day_part = 'full' OR start_date = end_date),
    CONSTRAINT chk_leave_requests_status CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    CONSTRAINT fk_leave_requests_tenant_user FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_leave_requests_tenant_approver FOREIGN KEY (tenant_id, approver_id) REFERENCES users (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_leave_requests_tenant_leave_type FOREIGN KEY (tenant_id, leave_type_id) REFERENCES leave_types (tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_leave_requests_tenant_id ON leave_requests (tenant_id);
CREATE INDEX idx_leave_requests_tenant_user ON leave_requests (tenant_id, user_id);
CREATE INDEX idx_leave_requests_tenant_approver_status ON leave_requests (tenant_id, approver_id, status);
CREATE INDEX idx_leave_requests_leave_type_id ON leave_requests (leave_type_id);
CREATE INDEX idx_leave_requests_tenant_dates ON leave_requests (tenant_id, start_date, end_date);
