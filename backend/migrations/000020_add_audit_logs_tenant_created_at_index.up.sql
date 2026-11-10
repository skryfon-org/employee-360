-- Supports the admin audit-log viewer: tenant-scoped, newest-first listing
-- (ORDER BY created_at DESC, id DESC) with optional time-range filters.
CREATE INDEX idx_audit_logs_tenant_created_at ON audit_logs (tenant_id, created_at DESC, id DESC);
