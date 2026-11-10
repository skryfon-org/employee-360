-- Daily status log entries submitted by employees for a calendar date.
-- Entries are either attached to an active project allocation or to an activity category.
CREATE TABLE status_log_entries (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    log_date          DATE NOT NULL,
    project_id        UUID REFERENCES projects (id) ON DELETE CASCADE,
    activity_category TEXT,
    hours             NUMERIC(4,2) NOT NULL,
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        UUID,
    updated_by        UUID,
    deleted_by        UUID,
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT chk_status_log_entries_activity_category CHECK (
        activity_category IS NULL OR activity_category IN ('meeting', 'training', 'internal', 'support', 'other')
    ),
    CONSTRAINT chk_status_log_entries_hours CHECK (hours > 0 AND hours <= 24),
    CONSTRAINT chk_status_log_entries_project_or_category CHECK (
        (project_id IS NOT NULL AND activity_category IS NULL) OR
        (project_id IS NULL AND activity_category IS NOT NULL)
    ),
    CONSTRAINT fk_status_log_entries_tenant_user FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_status_log_entries_tenant_project FOREIGN KEY (tenant_id, project_id) REFERENCES projects (tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_status_log_entries_tenant_id ON status_log_entries (tenant_id);
CREATE INDEX idx_status_log_entries_user_id ON status_log_entries (user_id);
CREATE INDEX idx_status_log_entries_project_id ON status_log_entries (project_id);
CREATE INDEX idx_status_log_entries_tenant_log_date ON status_log_entries (tenant_id, log_date);
CREATE INDEX idx_status_log_entries_tenant_user_log_date ON status_log_entries (tenant_id, user_id, log_date);
CREATE INDEX idx_status_log_entries_deleted_at ON status_log_entries (deleted_at);
