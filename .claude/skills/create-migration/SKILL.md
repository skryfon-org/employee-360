---
name: create-migration
description: >
  Scaffold a new golang-migrate SQL migration pair (up + down) for the Employee360
  backend. Use when the user asks to create, add, or write a new database migration.
---

# Create Migration

## Step 0 — check the backend exists

```bash
ls backend/migrations 2>/dev/null
```

If `backend/migrations/` doesn't exist yet, this is the first migration — create the directory and start numbering at `000001`.

## Step 1 — determine the next sequence number

```bash
ls backend/migrations/*.up.sql 2>/dev/null | grep -oE '[0-9]{6}' | sort -n | tail -1
```

Increment by 1. Zero-pad to **6 digits** (e.g. `9` → `000010`) — matches the convention in `plan/architecture/backend.md` (`000001_create_tenants.up.sql`, etc.).

## Step 2 — derive the file names

Ask the user for a short snake_case name if not already provided (e.g. `holiday_attachments`).

Files to create:
- `backend/migrations/NNNNNN_<name>.up.sql`
- `backend/migrations/NNNNNN_<name>.down.sql`

## Step 3 — write the up migration

Follow these invariants (from `plan/architecture/backend.md`'s multi-tenancy strategy — violations break tenant isolation):

| Rule | Detail |
|------|--------|
| **Tenancy** | Every table *except* `tenants` itself must include a `tenant_id` column referencing `tenants(id)`, plus an index on it |
| **Timestamps** | Include `created_at` and `updated_at` (default now) on every entity table |
| **FK indexes** | Every foreign-key column needs a matching `CREATE INDEX` |
| **Audit** | If the table is admin-mutable (holidays, categories, departments, positions, roles), consider whether mutations to it should also write an `audit_logs` row via the usecase layer — not the migration, just keep it in mind for the corresponding usecase |
| **Naming** | snake_case table and column names, matching the entity list: `tenants`, `departments`, `positions`, `users`, `roles`, `user_roles`, `holiday_categories`, `holidays`, `audit_logs` |

## Step 4 — write the down migration

The down migration must exactly reverse the up migration:
- `CREATE TABLE` → `DROP TABLE IF EXISTS`
- `ALTER TABLE … ADD COLUMN` → `ALTER TABLE … DROP COLUMN IF EXISTS`
- `CREATE INDEX` → `DROP INDEX IF EXISTS`
- `CREATE TYPE` → `DROP TYPE IF EXISTS`

If the up migration is destructive (e.g. `DROP COLUMN`), the down should be a no-op comment explaining why reversal is impossible.

## Step 5 — confirm and run

Tell the user the two file paths created and remind them to run (once `cmd/migrate` exists):

```bash
make migrate
```

to apply, or check migration status first if such a target exists.

**Never edit a migration file that has already been committed to git** — always add a new migration pair instead. This is enforced by the `check-migration.sh` PreToolUse hook.
