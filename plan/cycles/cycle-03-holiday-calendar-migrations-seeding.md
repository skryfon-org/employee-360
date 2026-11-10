# Cycle 3 — Holiday Calendar (Migrations, Seeding, API & Frontend)

| | |
|---|---|
| **Status** | Planned — blocked on Cycle 2 |
| **Module** | Holiday Calendar Management (full stack: schema + seed, backend API, admin & employee UI) |
| **Depends on** | `plan/cycles/cycle-01-project-setup.md` (backend must run, DB must be reachable, `cmd/migrate` runner must work); `plan/cycles/cycle-02-auth-onboarding.md` (tenants/users/roles schema, real auth/tenant middleware, the `cmd/bootstrap` seeder this cycle extends, and the admin/employee login flows are created there) |
| **Source** | `plan/architecture/backend.md` (entity list, migration file list), `plan/architecture/frontend.md`, original requirements (now summarised in `README.md`) |

This is the scope/status doc for Cycle 3. Read this before starting or resuming work.
Cross-cutting principles (open source, multi-tenant, self-hostable, headless) live in
`CLAUDE.md`, not here.

**Renumbered:** this cycle was originally filed as Cycle 2. Cycle 2 was repurposed for
Auth, Email Service & Onboarding Invitations (`plan/cycles/cycle-02-auth-onboarding.md`)
since the auth/tenant middleware and users/roles schema needed to exist before anything
else could be protected. Holiday Calendar moved here, now correctly sequenced after auth.

**Scope change:** this cycle was originally *migrations + seeding only*, with the API
and frontend deferred to later cycles. It now covers the whole Holiday Calendar module
end to end. File filename kept (`...-migrations-seeding.md`) so existing links keep
working.

Build order (per `CLAUDE.md` dispatch): migrations/seed → backend API
(`backend-agent`) → frontend (`frontend-agent`).

---

## Objective

Deliver the Holiday Calendar module: tenant admins configure and maintain a
year-wise, categorised holiday calendar; employees view it as a calendar or a list.
Tenants, departments, positions, users, roles, auth and audit logging already exist
from Cycle 2.

### Capabilities

1. **Holiday calendar configuration** — per-tenant calendar settings (e.g. which years
   exist, default view); admin-only.
2. **Add holidays** — admin creates a holiday (name, date, category, optional
   description).
3. **Edit holidays** — admin updates any field of an existing holiday.
4. **Remove holidays** — admin soft-deletes a holiday.
5. **Employee viewing** — employees get read-only access to their own tenant's calendar.
6. **Calendar / list display** — month-grid calendar view and a chronological list view,
   with a toggle.
7. **Categorisation** — holidays belong to a category (e.g. Public, Optional); admin can
   manage categories; UI filters/colour-codes by category (see `design-system` skill for
   category colours).
8. **Year-wise calendars** — holidays are queried and displayed per calendar year; a year
   selector drives both views.

---

## Sub-Features

### Migrations (`backend/migrations/`, via the `create-migration` skill)

Migration numbers were renumbered from `000012`–`000014` because Cycle 2 consumed
`000012`–`000020`; the next free numbers are `000021`–`000023`. Column-level spec lives in
Plane ticket EMPLOYEE36-36.

- [x] `000021_create_holiday_categories` — `tenant_id` FK + index; unique
      `(tenant_id, LOWER(name))` among non-deleted rows
- [x] `000022_create_holidays` — `tenant_id` FK + index; FK to `holiday_categories` +
      index; `date` column plus index on `(tenant_id, date)` for year-range queries;
      soft delete (`deleted_at`)
- [x] `000023_create_holiday_calendar_settings` — one row per tenant (`tenant_id` FK,
      unique); `default_view`, `default_year`, `week_start_day`.
      **Decision:** a new dedicated table rather than reusing a Cycle 2 tenant settings
      table, because Cycle 2 (migrations `000001`–`000020`) created no tenant settings
      table to reuse, and a typed table keeps the settings validated by CHECK constraints.

Each: up + down pair, `created_at`/`updated_at` on every entity table, an index on every
FK column. See the `create-migration` skill for the full invariant checklist.

### Seeding (`backend/internal/infrastructure/database/seeder/`, run via `cmd/bootstrap`)

- [x] Seed default holiday categories for the system tenant seeded in Cycle 2 (e.g.
      "Public Holiday", "Optional Holiday" — confirm the exact set against
      the original requirements before hardcoding)
- [x] `seeder_test.go` addition — verify the category seed step is idempotent
      (Note: a default category an admin soft-deleted is recreated on the next bootstrap run.)

### Backend API (`backend/internal/{domain,usecase,delivery,infrastructure}/`, via `new-backend-feature`)

All routes under `/api/v1`, tenant resolved from auth context only (Invariant 1).

- [ ] Domain entities + repository ports: `Holiday`, `HolidayCategory`, calendar settings
- [ ] Usecases: create / update / delete / get holiday; list holidays by `year`
      (optional `category_id`, `month` filters); category CRUD; get/update settings
- [ ] Validation: non-empty name, valid date, category must belong to the same tenant,
      no duplicate (tenant, date, name)
- [ ] GORM repositories scoped by explicit `tenantID`
- [ ] Handlers + routes:
  - `GET /holidays?year=YYYY` — admin + employee
  - `GET /holidays/:id` — admin + employee
  - `POST /holidays`, `PUT /holidays/:id`, `DELETE /holidays/:id` — admin only
  - `GET /holiday-categories` — admin + employee
  - `POST|PUT|DELETE /holiday-categories[/:id]` — admin only (deleting a category in
    use is rejected)
  - `GET|PUT /holiday-calendar/settings` — read: admin + employee; write: admin only
- [ ] Usecase unit tests and repository/handler tests, including cross-tenant isolation
      and role enforcement (employee gets 403 on writes)

**Cross-cycle note — uniform list pagination (introduced in EMPLOYEE36-42, commit
`00b074c`):** every paginated list endpoint now follows one convention
(`backend/internal/types/pagination`: `page`/`page_size`, defaults 1/20, cap 100,
non-integer -> 400). This changed the list response shape of the Cycle 2 endpoints
(departments, positions, audit-logs, invitations) as well as the new holiday-category
list; the admin client and generated api-client were updated in the same change.
Any external consumer of those Cycle 2 list endpoints must adopt the new shape.

### Frontend (`clients/admin/`, `clients/employee/`, `packages/api-client/`, via `new-frontend-feature` + `design-system`)

- [ ] `packages/api-client`: typed holiday + category + settings endpoints
- [ ] Admin `features/holidays/`: year selector, calendar/list toggle, add/edit holiday
      form (React Hook Form + Zod), delete confirmation, category management, calendar
      settings screen
- [ ] Employee `features/holidays/`: read-only year selector and calendar/list views,
      category filter
- [ ] TanStack Query for all server state (query keys include year); Zustand only for
      view mode / filters
- [ ] Loading, empty-year and error states

**Done when:**
- `make migrate` / `make migrate-down` apply and reverse cleanly on top of Cycle 2's
  schema; `cmd/bootstrap` seeds default categories idempotently.
- An admin can add, edit and remove holidays and categories and see them in both calendar
  and list views, per year.
- An employee in the same tenant sees the same calendar read-only and cannot mutate it;
  a user in another tenant sees none of it.
- `make test` and `pnpm typecheck` pass.

---

## Out of Scope for This Cycle

- Leave management, work status, or anything that consumes holidays (e.g. leave-day
  calculation) — later modules.
- Bulk import/export of holidays, recurring-rule generation, ICS/calendar-app sync,
  notifications/reminders — not listed; file separately if wanted.
- Any other planned module (Work Status, Leave Management, Courses/Certifications,
  Benefits, Career Growth, Salary/Taxation, Appraisal, Company Policies) — see
  `CLAUDE.md`'s Execution Model section.

---

## Reference

- Entity list, migration file list, layering: `plan/architecture/backend.md`
- Frontend layout: `plan/architecture/frontend.md`
- Skills: `create-migration`, `new-backend-feature`, `new-frontend-feature`,
  `design-system` (`.claude/skills/`)
- Agents: `backend-agent` then `frontend-agent` (`.claude/agents/`)
- Previous cycle: `plan/cycles/cycle-02-auth-onboarding.md`
