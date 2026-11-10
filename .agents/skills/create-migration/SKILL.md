---
name: create-migration
description: >-
  Scaffold a new golang-migrate SQL migration pair (up + down) for the Employee360
  backend. Use when the user asks to create, add, or write a new database migration.
---

# Create Migration

## Step 0 — Check if backend migrations directory exists

```bash
ls backend/migrations 2>/dev/null
```

If `backend/migrations/` doesn't exist yet, create the directory and start numbering at `000001`.

## Step 1 — Determine the next sequence number

```bash
ls backend/migrations/*.up.sql 2>/dev/null | grep -oE '[0-9]{6}' | sort -n | tail -1
```

Increment by 1. Zero-pad to **6 digits** (e.g. `9` → `000010`) — matches the convention in `plan/architecture/backend.md` (`000001_create_tenants.up.sql`, etc.).

## Step 2 — Derive the file names

Name files using short snake_case (e.g. `000010_holiday_attachments`):
- `backend/migrations/NNNNNN_<name>.up.sql`
- `backend/migrations/NNNNNN_<name>.down.sql`

## Step 3 — Write the Up migration

Follow these multi-tenancy invariants:

| Rule | Detail |
|---|---|
| **Tenancy** | Every table *except* `tenants` itself must include a `tenant_id uuid NOT NULL` column referencing `tenants(id) ON DELETE CASCADE` + an index on it |
| **Timestamps** | Include `created_at` and `updated_at` (`timestamptz NOT NULL DEFAULT now()`) on every entity table |
| **Foreign Keys** | Every foreign-key column needs a matching index |
| **Composite Uniqueness** | Enforce multi-tenant uniqueness via composite unique indexes: `UNIQUE(tenant_id, name)` or `UNIQUE(tenant_id, email)` |
| **Naming** | snake_case table and column names: `tenants`, `departments`, `positions`, `users`, `roles`, `user_roles`, `holiday_categories`, `holidays`, `audit_logs` |

## Step 4 — Write the Down migration

The down migration must exactly reverse the up migration:
- `CREATE TABLE` → `DROP TABLE IF EXISTS <table_name> CASCADE;`
- `ALTER TABLE … ADD COLUMN` → `ALTER TABLE … DROP COLUMN IF EXISTS <col_name>;`
- `CREATE INDEX` → `DROP INDEX IF EXISTS <index_name>;`

If the up migration is destructive (e.g. `DROP COLUMN`), the down should be a comment explaining why reversal is impossible.

## Step 5 — Apply Migration

```bash
go run backend/cmd/migrate/main.go up
```

> [!WARNING]
> **Never edit a migration file that has already been committed to git** — always add a new migration pair instead. This is enforced by the `check-migration.sh` hook.
