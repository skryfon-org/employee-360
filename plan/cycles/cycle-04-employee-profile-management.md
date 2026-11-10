# Cycle 4 — Employee Profile & Employee Management

| | |
|---|---|
| **Status** | Planned |
| **Module** | Employee Profile + Employee Management (admin) — full stack |
| **Depends on** | `plan/cycles/cycle-02-auth-onboarding.md` (tenants, users, roles, DB-verified auth middleware, audit columns from migration 000015, invitation UI/shell in `clients/admin`) |
| **Source** | Request for employee personal/employment/address/emergency-contact data and an admin page to manage employees; patterns from migrations 000005, 000011, 000015 |

This is the scope/status doc for Cycle 4. Read this before starting or resuming work.
Cross-cutting principles live in `CLAUDE.md`.

**Scope note:** migrations first, then backend API (`backend-agent`), then admin frontend
(`frontend-agent`). The employee self-service portal (`clients/employee`) is **not** in
this cycle. Migration numbers are the next free ones (`000019` and `000020` at time of
writing — `000018` is now `add_is_active_to_departments`; re-check with
`ls backend/migrations` before creating them); independent of Cycle 3 (Holiday Calendar).

---

## Objective

Give tenant admins and super admins one place to see and manage the people in their
tenant: list users/employees, view and update their profile details, and activate /
deactivate them. Requires a tenant-scoped schema for employee details, which `users`
(identity/auth fields only) doesn't have today.

---

## Sub-Features

### 1. Migrations `000019_create_employee_profiles` and `000020_create_user_assignments` (up + down, via `create-migration` skill)

**`employee_profiles`** — 1:1 with `users`
- [ ] `tenant_id` FK (`ON DELETE CASCADE`) + index; `user_id` FK (`ON DELETE CASCADE`) + index
- [ ] Personal: `employee_code`, `gender`, `date_of_birth`, `blood_group`, `marital_status`,
      `nationality`, `phone`, `personal_email` (CHECK constraints on enumerated values)
- [ ] Employment: `joining_date`, `employment_type`, `employment_status` (default `active`),
      `probation_end_date`, `confirmation_date`, `exit_date`, `work_location`,
      `manager_id` → `users` (`ON DELETE SET NULL`, indexed)
- [ ] Address: `address_line1`, `address_line2`, `city`, `state`, `country`, `pincode` (TEXT)
- [ ] Audit columns inline: `created_at`, `updated_at`, `created_by`, `updated_by`,
      `deleted_by`, `deleted_at` (actor columns plain UUID, no FK — per 000015)
- [ ] Partial unique indexes (`WHERE deleted_at IS NULL`): `(tenant_id, user_id)`;
      `(tenant_id, employee_code)` where code is not null

**`emergency_contacts`** — many per employee
- [ ] `tenant_id` FK + index; `user_id` FK + index
- [ ] `name`, `relationship`, `phone` (NOT NULL); `alternate_phone`, `email`
- [ ] `is_primary` + partial unique index: one primary per `(tenant_id, user_id)` among
      non-deleted rows
- [ ] Audit columns inline

**`user_assignments`** — department/position history ("internal experience"), many per user.
`users.department_id` / `users.position_id` stay as the denormalised *current* value
(used by invitations, list filters, dashboard); this table records every stint.
- [ ] `tenant_id` FK (`ON DELETE CASCADE`) + index; `user_id` FK (`ON DELETE CASCADE`) + index
- [ ] `department_id` FK (`ON DELETE SET NULL`) + index; `position_id` FK
      (`ON DELETE SET NULL`) + index (both nullable)
- [ ] `effective_from` DATE NOT NULL; `effective_to` DATE NULL (NULL = current stint);
      `change_reason` TEXT NULL (joined / promotion / transfer / ...)
- [ ] `CHECK (effective_to IS NULL OR effective_to >= effective_from)`
- [ ] Partial unique index `(tenant_id, user_id) WHERE effective_to IS NULL AND
      deleted_at IS NULL` — at most one open stint per user
- [ ] Audit columns inline (as above)
- Experience is **derived**, not stored: tenure = (`exit_date` or today) − `joining_date`;
  time per department/position = sum of stint lengths. Prior *external* jobs are out of scope.

**Down:** drop `user_assignments` (000020) and, separately, `emergency_contacts` then
`employee_profiles` (000019) (indexes first).

### 2. Backend API (`backend-agent`, `new-backend-feature` skill) — `/api/v1`, roles `admin` + `super_admin`

All endpoints derive `tenant_id` from auth context (never from payload); usecases take
explicit `tenantID`/actor params; handlers never touch GORM.

- [ ] `GET /employees` — paginated list of tenant users with profile summary; search
      (name/email/employee code), filters (status active/inactive, department, position,
      employment status); excludes soft-deleted
- [ ] `GET /employees/:id` — user + profile + emergency contacts
- [ ] `PUT/PATCH /employees/:id` — update user fields (name, department, position) and
      profile fields; upsert profile on first update; validate department/position/manager
      belong to the tenant
- [ ] Assignment history: whenever department/position changes (employee create/update),
      in ONE transaction close the open `user_assignments` row, insert a new open row,
      update `users.department_id/position_id`, and write an audit entry; employee
      creation inserts the first row (`effective_from = joining_date`). Validate
      department (must be active; reuse `OrgReferenceRepository.LockDepartmentShared`) and
      position belong to the tenant
- [ ] `GET /employees/:id/assignments` — history, newest first (admin/super_admin only;
      Auth → Tenant → `RequireRole`); cross-tenant → 404
- [ ] Tests for assignments: close/open on change, no row when nothing changed, single open
      row, date-order check, tenant isolation, cross-tenant 404
- [ ] `PUT /employees/:id/emergency-contacts` (or sub-resource CRUD) — manage contacts,
      enforce single primary
- [ ] `POST /employees/:id/deactivate` and `/activate` — flips `users.is_active`; revoke
      refresh tokens on deactivate; cannot deactivate yourself or the last active admin;
      `admin` cannot act on `super_admin`
- [ ] Role assignment: change an employee's role between `employee` / `admin`
      (super_admin only for granting `admin`) — confirm exact rules before building
- [ ] Entities, repository ports + GORM adapters, handler/usecase/repo tests (incl.
      cross-tenant isolation and soft-delete read tests), swagger regeneration

### 3. Admin frontend (`frontend-agent`, `new-frontend-feature` + `design-system` skills) — `clients/admin/src/features/employees/`

- [ ] Sidebar nav item "Employees" (admin + super_admin only) and routes
- [ ] Employees list page: table, search, filters, pagination, status badge, row actions
- [ ] Employee detail/edit page: personal, employment, address sections; emergency
      contacts editor; RHF + Zod schemas
- [ ] Activate / deactivate with confirm dialog; role change control
- [ ] TanStack Query hooks in `queries/`; Axios client additions in `packages/api-client`;
      component + page tests

### 4. Employee creation + self-completed profile (Plane EMPLOYEE36-31, sub-ticket of EMPLOYEE36-30)

Two-step setup: admin creates the user with **employment details only**; the employee
fills in the rest themselves.

- [ ] Admin: create employee with employment details (employee_code, joining_date,
      employment_type/status, department, position, manager, work_location, probation /
      confirmation dates) → user + `employee_profiles` row with personal fields NULL
- [ ] Employee: `GET/PATCH /me/profile` + emergency contacts CRUD — basic details
      (phone, personal email, DOB, marital status, nationality), gender, blood group,
      address, emergency contact(s); user ID from auth context only
- [ ] Employment fields read-only for the employee, editable by admin only
- [ ] `clients/employee` profile page (RHF + Zod) with "profile incomplete" prompt
- [ ] First login (backend support in EMPLOYEE36-47): the employee sees a welcome message
      from the organization (organization name) and then proceeds to profile completion,
      which is **mandatory** before using the rest of the app
- [ ] Backend: expose organization name and `first_login` / `profile_completed` status on
      `/me`; track completion server-side (e.g. `employee_profiles.profile_completed_at`,
      set when the required self-filled fields are saved via `PATCH /me/profile`)
- [ ] Gating: until the profile is complete, employee endpoints other than `/me`,
      `/me/profile`, emergency contacts and logout return 403 `profile_incomplete`;
      admin/super_admin are exempt
- [ ] `clients/employee`: welcome screen, then redirect to profile completion whenever the
      profile is incomplete
- [ ] Tests: gating on/off, status flips after completing mandatory fields, admins never gated
- [x] Resolved: mandatory self-filled fields are **phone, date_of_birth, gender,
      address_line1, city, country** plus **at least one emergency contact**. Completion
      is tracked in `employee_profiles.profile_completed_at`; EMPLOYEE36-31's frontend
      validation must mirror this list (source: `entity/employee_profile.go`).
- Welcome screen: there is no "acknowledge" write. `/me` returns `first_login =
  profile_required && !profile_completed`, so the welcome screen shows on every login
  until onboarding is complete; admins/super_admins (`profile_required = false`) always
  get `first_login = false`. A missing profile row means incomplete; `GET /me` never creates a row. The
  profile row is created only by `PATCH /me/profile` or an admin update.

> **Scope note:** EMPLOYEE36-47 already delivers the `/me` backend (`GET/PATCH
> /me/profile`, `/me/emergency-contacts` CRUD, `/me` status, gating), so EMPLOYEE36-31 is
> frontend-only for the self-service profile (plus admin employee creation if not covered).
>
> **Scope note:** EMPLOYEE36-48 extended `/me/profile`: the response now includes
> `first_name`/`last_name` and read-only department/position (id + name, resolved via
> `OrgNameRepository`); `first_name`/`last_name` are editable via `SelfProfileFields`.

### 5. Employee code format (Plane EMPLOYEE36-49 backend, EMPLOYEE36-50 admin frontend)

Per-organization format for `employee_profiles.employee_code`, configured in organization
settings. Example: prefix `SKRYFON`, padding 4, starting number 40001 → first code
`SKRYFON40001`.

- [x] Migration: tenant-scoped settings `employee_code_prefix`, `employee_code_padding`,
      `employee_code_start_number`, plus server-managed `employee_code_next_number`;
      existing tenants backfilled with defaults (prefix `EMP`, padding 4, start 1)
- [x] Settings GET/PUT endpoints (admin + super_admin); `tenant_id` from auth context only;
      counter not client-writable
- [x] Validation: prefix trimmed, uppercased, 1–10 chars, `[A-Z0-9]` only, must not end in a
      digit; padding 1–10; start number integer ≥ 1 and not at or below any number already
      issued; total code length ≤ 20
- [x] Prefix change applies to new codes only; existing codes are never rewritten
- [x] Auto-generate the code on employee creation: atomically take and increment the counter
      in the creation transaction (no duplicates under concurrency)
- [x] Manual override must match `^<PREFIX>\d+$` and be unique per tenant (non-deleted)
- [x] Tests: format function (padding, overflow past width), every validation rule, counter
      increment, concurrency (parallel creates → unique codes), cross-tenant isolation
- [x] Admin UI: organization settings section with prefix / padding / start number fields,
      live next-code preview, Zod schema mirroring backend rules, start number constrained
      once codes are issued, prefix-change warning; employee form shows the code
      auto-filled and read-only with optional validated override
- [x] Resolved: no separator allowed; no prefix lock (a prefix change applies to new codes only). Open question was: allow a separator (e.g. `SKRYFON-0001`)? (suggested: no); lock prefix after the
      first code is issued? (suggested: no, new codes only)

> **Implementation note (EMPLOYEE36-49, backend):** settings live in the new
> `organization_settings` table (migration `000026`, one row per tenant). Defaults, also
> used for a tenant with no row: prefix `EMP`, padding `4`, start number `1`, next number
> `1` (first code `EMP0001`); existing tenants are backfilled with a default row. Columns:
> `employee_code_prefix|padding|start_number` (client-editable), `employee_code_next_number`
> and `employee_code_last_issued_number` (server-managed). Endpoints (admin + super_admin):
> `GET|PUT /api/v1/organization/settings`. A code is generated by
> `EmployeeProfileRepository.Create` (single `INSERT .. ON CONFLICT DO UPDATE .. RETURNING`
> on the counter, same transaction as the profile insert) whenever a profile row is created
> without an explicit `employee_code`. Counter values already held by a manual override are skipped.
> Admin `PUT /employees/:id` with a changed `employee_code` must match `^<PREFIX>\d+$`.

> **Implementation note (EMPLOYEE36-50, admin frontend):** `clients/admin/src/features/organization-settings/`
> provides the "Employee codes" settings, mounted as the `employee-codes` tab of the Organization page
> (`/settings/organization/employee-codes`; `/settings/employee-codes` redirects there). The Organization nav
> entry/page is visible to admin + super_admin, but admins see and reach only this tab (General and Domains
> stay super_admin-only; the tenant detail fetch, super_admin-only on the backend, is skipped for it). Generated client
> regenerated for `GET|PUT /organization/settings`. The Zod schema (`createOrgSettingsSchema(minStart)`)
> mirrors backend rules; once `employee_codes_issued` the start number's minimum is the server's
> `employee_code_next_number`. Live preview is computed client-side; server 400 messages are mapped inline
> to prefix/padding/start fields by keyword. The employee employment form shows the code read-only
> ("Generated automatically") with an "Override code" button; overrides are checked against `^<PREFIX>\d+$`
> using the saved prefix (server remains authoritative). There is no admin "create employee" form (employees
> arrive via invitations), so the override lives in the edit form.

### Out of scope (deliberately)
- Government IDs (PAN/Aadhaar/SSN), bank details, salary — sensitive PII/compensation;
  belongs to the Salary/Taxation module with its own access rules.
- Employee self-service beyond the onboarding profile completion in sub-feature 4
  (no leave, documents, or other self-service).
- Bulk import/export, org chart, position CRUD (department CRUD shipped in
  EMPLOYEE36-26; positions still have no API, unique name or `is_active`).
- Prior external work experience (previous employers); only internal department/position
  history is tracked.
- Hard delete / purge of users.

**Done when:** migrations apply and revert cleanly; an admin can list, search, filter,
view, update, activate and deactivate employees in their tenant through the admin UI;
a deactivated user can no longer authenticate (DB-verified middleware → 401); no endpoint
returns or mutates another tenant's data (covered by tests);
`make test` and `pnpm --filter admin test` + `pnpm typecheck` pass.
