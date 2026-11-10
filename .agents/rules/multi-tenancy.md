# Multi-Tenancy Invariant

## Strict Tenant Isolation Rules

1. **Row-Level Tenancy**:
   - Every database table except `tenants` itself **must** carry a `tenant_id uuid NOT NULL` column referencing `tenants(id)`.
   - Every tenant-owned table must include composite unique indexes / indexes on `tenant_id` (e.g., `(tenant_id, email)`, `(tenant_id, code)`, `(tenant_id, name)`).
2. **Context-Derived Tenant ID**:
   - Handlers and repositories **must never trust client-supplied tenant IDs** from request payloads, route params, or query params.
   - `tenant_id` is resolved exclusively in the authentication / tenant resolution middleware and stored into Go `context.Context`.
3. **Repository Scoping**:
   - All repository implementations in `internal/infrastructure/persistence/` **must extract `tenant_id` via `ctx.TenantIDFromContext(ctx)`** and filter all `SELECT`, `UPDATE`, `DELETE`, and `INSERT` queries by `tenant_id`.
4. **Platform Super Admin Exception**:
   - Platform governance (`super_admin`) is the only role that manages tenants across the platform. It operates through dedicated platform use cases and seeders (`cmd/bootstrap`).
