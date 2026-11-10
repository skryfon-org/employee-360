# Cycle 2 — Auth, Email Service & Onboarding Invitations

| | |
|---|---|
| **Status** | Active |
| **Module** | Authentication (email+password for all roles), transactional email, and onboarding invitations |
| **Depends on** | `plan/cycles/cycle-01-project-setup.md` (backend must run, DB must be reachable, `cmd/migrate`/`cmd/bootstrap` runners must work) |
| **Source** | `plan/architecture/backend.md` (entity/migration list, layering), `plan/initial-planning.md` (auth flow decisions), original requirements (now summarised in `README.md`) |

This is the scope/status doc for Cycle 2. Read this before starting or resuming work.
Cross-cutting principles (open source, multi-tenant, self-hostable, headless) live in
`CLAUDE.md`, not here.

**Renumbering note:** this cycle was originally filed as Holiday Calendar migrations +
seeding. That content moved unchanged to
`plan/cycles/cycle-03-holiday-calendar-migrations-seeding.md`. Cycle 2 was repurposed
for auth/onboarding because `plan/cycles/cycle-01-project-setup.md` already deferred
"auth/tenant middleware" to Cycle 2 ("once there's something to protect"), and every
later module needs real users, roles, and a working login before its own API is worth
building.

**Scope note:** unlike Cycle 1's scaffolding-only and Cycle 3's migrations-only slices,
this cycle runs the full stack for this one feature set — migrations through handlers,
**and now the frontend that consumes them** (admin login UI, employee OTP UI, forgot/reset
password UI, admin invitation-management UI, invitation-accept flow) — because auth
without a callable endpoint isn't testable, and the API isn't validated end-to-end until a
real client calls it. Dispatch backend work to `backend-agent` and frontend work to
`frontend-agent` per `CLAUDE.md`; frontend work should start once the corresponding backend
endpoint exists (see dependency notes in the Frontend sub-feature below).

---

## Objective

Stand up real authentication (replacing Cycle 1's middleware stubs), a reusable
transactional email service, and an admin-driven onboarding-invitation flow, so that:

1. Tenant Admins and the platform Super Admin can log in with email + password, via a real
   admin login page.
2. Employees can log in with email + password, via a real employee login page.
3. All users (admin and employee) can recover access via "forgot password", via real
   forgot/reset password pages.
4. Tenant Admins can invite new employees by email (only a platform Super Admin may invite the `admin` role); invitees land as
   pending users until they accept, via a real admin invitation-management UI and an
   invitation-accept page.

---

## Decisions made for this cycle

`plan/initial-planning.md` left the employee login mechanism as an open question
("magic link vs OTP vs company SSO"). This cycle decides: **email + password, same
mechanism as admin/super_admin** — not OTP, not magic links. This is a free choice, not
a constraint: the original requirements only required "Employee authentication using company
email" and never specifies a mechanism —
`plan/initial-planning.md:55`'s "magic link" mention was an early narrative aside, not a
locked-in requirement, and line 80 explicitly flagged the mechanism as open. Reasoning
for password-only: one login mechanism for every role means one `LoginUseCase`, one
login form component, one password-reset flow, and no separate OTP-delivery email
template/rate-limiting/expiry logic to build and maintain. Employees set their password
during invitation-accept, same as admins. If product wants passwordless (OTP/magic
link) or SSO later, that's a follow-up cycle, not a blocker to this one.

Email delivery is **asynchronous, via an event-publish + transactional-outbox
pattern backed by River** (a Postgres-backed Go job-queue library —
`github.com/riverqueue/river`), not inline synchronous SMTP. Usecases that trigger an
email (`InviteUserUseCase`, `ResendInvitationUseCase`, `ForgotPasswordUseCase`) never call
`EmailService.Send` directly; instead, inside the same DB transaction as their business
write, they call a new `EventPublisher.Publish(ctx, events...)` domain port. Its
infrastructure implementation enqueues a River job via `InsertTx` using that same
transaction's `*sql.Tx` — so the business row and the queued job commit together, or
neither does. River's own `river_job` table (on the **same** Postgres instance as the
rest of the app — no second DB, no Redis, no separate outbox table or relay/poller) *is*
the outbox; a separate `cmd/worker` process drains it and calls `EmailService.Send`.
Retries/backoff/dead-lettering are River's built-in job-state machine, not hand-rolled.
This reverses the earlier "defer async delivery" decision before any of Cycle 2 was
built, specifically so `InviteUserUseCase` et al. are written against the async shape
from the start rather than retrofitted later. See `plan/architecture/backend.md` for the
resulting `eventing/`/`job/`/`cmd/worker/` layout.

**Default provider: Resend, via its SMTP interface** (`smtp.resend.com`), not its
REST API/SDK. Calling a provider through its proprietary SDK would hard-code
`mail_service.go` to one vendor. Going through Resend's SMTP credentials instead
(username `resend`, password = the Resend API key) keeps `mail_service.go` a plain,
provider-agnostic SMTP client — same code path works against Resend, Amazon SES,
Postmark, or a self-hosted relay, just by changing
`SMTP_HOST`/`SMTP_PORT`/`SMTP_USERNAME`/`SMTP_PASSWORD` in config. This satisfies the
**Self-Hostable** principle (`CLAUDE.md`: no cloud-provider lock-in) while still
defaulting to a widely-used, cost-effective provider (Resend: 3,000 emails/month free,
then usage-based) instead of requiring self-hosted SMTP infrastructure this project
doesn't have yet. Local dev points `SMTP_HOST` at Mailpit instead so email is
verifiable without hitting Resend at all.

Roles stay the fixed three-role model from `CLAUDE.md`'s Platform Roles & Governance
Model (`super_admin`, `admin`, `employee`) — no granular permissions table. A
resource/action permission system solves a different (multi-role, per-resource ACL)
problem than this project's fixed role set, so it's not part of this cycle.

`plan/initial-planning.md` also left tenant resolution as an open question ("email
domain, subdomain, or explicit tenant selection at login"). This cycle decides:
**email domain**, resolved via a separate `tenant_domains` table (one tenant to many
domains) rather than a single column on `tenants` — a company invited on `acme.com` may
also send mail from `acme-hr.com` or acquire a second brand's domain later, and a join
table avoids a schema change when that happens. Reasoning for email domain over the
alternatives: subdomain routing needs wildcard DNS configured by whoever is hosting the
instance, which conflicts with the **Self-Hostable** principle (`CLAUDE.md`:
`docker-compose`, no extra infra assumptions) — a self-hoster on a bare IP or a single
domain can't cleanly offer `*.example.com`. Email domain needs no extra infra and
matches the parenthetical already in `plan/initial-planning.md:55`.

DNS-based domain-ownership verification is still out of scope for this cycle: tenant
(and tenant-domain) provisioning is a **Platform Super Admin** action (`CLAUDE.md`'s
governance model — no self-service tenant signup yet), so domains are set by a trusted
operator, not claimed by an untrusted party, and don't need a verification workflow to
prevent spoofing at this stage. Revisit if self-service signup becomes a real
requirement.

---

## Sub-Features

### Migrations (`backend/migrations/`, via the `create-migration` skill)

In dependency order, per `plan/architecture/backend.md` (`000002` and `000008`–`000012`
are new, added by this cycle; `000003`–`000007` keep their original names but shift by
one position to make room for `000002_create_tenant_domains`). Adding `000012` here
shifts Cycle 3's Holiday Calendar migrations one more position, from `000012`/`000013`
to `000013`/`000014` — update `plan/architecture/backend.md`'s target tree and
`plan/cycles/cycle-03-holiday-calendar-migrations-seeding.md` accordingly when that
cycle is next up:

- [ ] `000001_create_tenants` — no `tenant_id` column (this is the one table that doesn't get one). Columns: `id` (uuid, PK), `name` (text), `is_active` (boolean, default true), `created_at`, `updated_at`
- [ ] `000002_create_tenant_domains` — `tenant_id` FK + index; `domain` (text, unique across all tenants — this is what login resolution matches against); `created_at`, `updated_at`
- [ ] `000003_create_departments` — `tenant_id` FK + index
- [ ] `000004_create_positions` — `tenant_id` FK + index
- [ ] `000005_create_users` — `tenant_id` FK + index; FKs to department/position (both nullable — a user can exist before being assigned either); `password_hash` nullable (null until invitation-accept sets it, for every role); `email_verified_at`, `last_login_at`, `is_active`
- [ ] `000006_create_roles` — `tenant_id` FK + index
- [ ] `000007_create_user_roles` — join table, FKs to users + roles
- [ ] `000008_create_audit_logs` — `tenant_id` FK + index; nullable `actor_user_id` FK (system-initiated actions have no actor)
- [ ] `000009_create_password_reset_tokens` — `tenant_id` FK, `user_id` FK, `token_hash` (never store the raw token), `expires_at`, `used_at`
- [ ] `000010_create_refresh_tokens` — `tenant_id` FK, `user_id` FK, `token_hash`, `family` (uuid, for rotation/revocation), `revoked_at`, `expires_at`, `ip_address`, `user_agent` — access tokens stay fully stateless per `CLAUDE.md`; refresh tokens are tracked so logout/revocation is possible
- [ ] `000011_create_user_invitations` — `tenant_id` FK, `email`, `role_id` FK, nullable `department_id`/`position_id` FK, `invited_by` FK (users), `token_hash`, `expires_at`, `accepted_at`, `revoked_at`
- [ ] `000012_create_river_schema` — River's own tables (`river_job`, `river_leader`, `river_queue`, `river_client`, `river_client_queue`, `river_migration`). No `tenant_id` column (infra table, not tenant data — `tenant_id` travels inside each job's `args jsonb` instead). Generated via River's own `river migrate-get --up`/`--down` CLI and checked in as a normal golang-migrate pair — don't hand-write this SQL

Each: up + down pair, `created_at`/`updated_at` on every entity table, an index on every
FK column. See the `create-migration` skill for the full invariant checklist.

### Domain Layer (`backend/internal/domain/`)

- [ ] Entities: `tenant_domain.go`, `user.go`, `role.go`, `user_role.go`, `password_reset_token.go`, `refresh_token.go`, `user_invitation.go`, `audit_log.go`
- [ ] Repository interfaces: `tenant_domain_repository.go` (includes a `FindTenantByDomain` lookup — how login resolves `tenant_id` from an email's domain), `user_repository.go`, `role_repository.go`, `user_role_repository.go`, `password_reset_repository.go`, `refresh_token_repository.go`, `user_invitation_repository.go`, `audit_repository.go`
- [ ] `service/token_service.go` — JWT issue/verify (access + refresh claims carrying `tenant_id`, `user_id`, roles)
- [ ] `service/hash_service.go` — password hashing (bcrypt) + generic secret-token hashing (sha256, for reset/invitation/refresh token storage — never store raw tokens)
- [ ] `service/email_service.go` — `EmailService` interface (`Send(ctx, EmailMessage) error`) + `EmailMessage`/`EmailTemplateName` types, infrastructure-agnostic (depends only on domain constructs, not SMTP/Resend specifics). Template set for this cycle: `PasswordReset`, `UserInvitation`, `Welcome` (sent when an invited user accepts their invitation; body uses the tenant's optional `organization_settings.welcome_message`, built-in default otherwise). Called only from the worker (`infrastructure/job/email_worker.go`), never from a usecase.
- [ ] `event/events.go` — replaces the empty `event/doc.go` stub: `Event{ID, TenantID, EventType, AggregateType, AggregateID, Payload, OccurredAt}` + `EventType` constants for this cycle (`UserInvited`, `InvitationResent`, `PasswordResetRequested`, `UserActivated`) + one payload struct per event carrying whatever the email template needs (recipient, plaintext token/link — only the hash is ever persisted to `user_invitations`/`password_reset_tokens`)
- [ ] `service/event_publisher.go` — `EventPublisher` interface (`Publish(ctx, events ...event.Event) error`), infrastructure-agnostic (no GORM/River types in the signature) — this is the seam usecases call instead of `EmailService.Send`

### Email Service (`backend/internal/infrastructure/service/`)

- [ ] `mail_service.go` — plain SMTP implementation of `EmailService` (Viper config: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`; TLS as needed) — no Resend SDK, no vendor-specific code
- [ ] `mail/templates/` — subject + text + HTML template per `EmailTemplateName` (`PasswordReset`, `UserInvitation`), loaded via Go's `html/template`/`text/template`
- [ ] `.env.example` — default production values pointed at Resend's SMTP endpoint: `SMTP_HOST=smtp.resend.com`, `SMTP_PORT=465` (or `587`), `SMTP_USERNAME=resend`, `SMTP_PASSWORD=<RESEND_API_KEY>`
- [ ] Local dev: `.env` points `SMTP_HOST` at a local catcher (e.g. Mailpit) instead, so email is verifiable without hitting Resend

### Eventing & Async Email Delivery (`backend/internal/infrastructure/eventing/`, `.../job/`, `backend/cmd/worker/`)

- [ ] `eventing/dispatcher.go` — maps each `EventType` to River job args (a single shared `SendEmailArgs{TemplateName, To, Subject, TemplateData}` job kind covers all three email-triggering events for this cycle)
- [ ] `eventing/river_publisher.go` — implements `EventPublisher`: pulls the in-flight `*sql.Tx` from `ctx` (set by the `Transactor` below), runs each event through the dispatcher, calls `riverClient.InsertTx` — same transaction as the usecase's business write
- [ ] `usecase/implementation/ucshared/transactor.go` — implements the `Transactor` helper stubbed in `plan/architecture/backend.md` (`WithinTransaction(ctx, fn) error`), wrapping a GORM transaction and propagating it via `ctx` so repositories and `EventPublisher` share it without leaking `*gorm.DB`/`*sql.Tx` into the usecase layer
- [ ] `job/email_worker.go` — `river.Worker[SendEmailArgs]` implementation; calls `EmailService.Send`; `river.JobCancel` on permanent validation errors, otherwise River's default retry/backoff
- [ ] `backend/cmd/worker/main.go` — boots a River client (registers `EmailWorker`) via the DI container, runs `riverClient.Start(ctx)` with graceful shutdown, mirroring `cmd/api/main.go`'s lifecycle pattern; runs as its own process (add a `worker` service to `docker-compose.yml` and a `make worker` target alongside `make dev`)

### Redis Foundation (backend only; merged from the former Cycle 5, EPIC-F / EMPLOYEE36-29)

Redis is optional infrastructure only — **foundation, no consumers** (rate limiting, OTP
counters, JWT revocation/identity caching are follow-up tickets). This reverses this cycle's
earlier "no Redis" stance for the River outbox: River still uses Postgres; Redis is separate.

- [x] `redis` service in `docker-compose.yml` and `backend/docker-compose.yml` (pinned
      `redis:8.10.2-alpine`, `--requirepass`, AOF, healthcheck, resource limits, named
      volume `employee360_redis_data`, `${REDIS_PORT:-6379}` mapping)
- [x] `RedisConfig` (`REDIS_*`, optional `REDIS_URL`), defaults, validation (port range,
      pool size, password required in production/staging when enabled), `.env.example`,
      `config.yaml.example`
- [x] `service.Cache` port (`Get/Set/Delete/Incr/Ping`; ttl required; `ErrCacheMiss`,
      `ErrCacheUnavailable`) and `service.CachePinger`; `service.CacheKey` / `GlobalCacheKey`
      tenant-namespaced key helpers
- [x] go-redis adapter + `NoopCache` in `internal/infrastructure/service/`
- [x] Health: `redis` field (`ok` / `unreachable` / `disabled`); never changes HTTP status
- [x] DI (`container.New` takes an optional cache), startup connect + shutdown close in `cmd/api`
- [x] Archtest: go-redis importable only under `internal/infrastructure/`
- [x] CI: `redis` service in the integration job; integration tests (`-tags=integration`)
- [x] Docs: README, `shared-context.md`, `plan/architecture/backend.md`

Failure-mode policy: **disabled** (`REDIS_ENABLED=false`, default) → `NoopCache`, health says
`disabled`; **enabled but down at startup** → fail fast; **down at runtime** → cache calls
error, health says `unreachable` (HTTP 200); each consumer decides (rate limiting falls back
to the in-memory limiter in `middleware/ratelimit.go`; identity caching fails closed).
Tenant-owned keys must use `CacheKey(tenantID, ...)` with a tenantID from the authenticated
context (never client input); a nil tenant is rejected. The domain `Cache` port exposes no
Redis types. Non-goals: any consumer, Redis Cluster/Sentinel, pub/sub, frontend changes.

### Auth Usecases (`backend/internal/usecase/{interface,implementation}/auth/`)

- [ ] `LoginUseCase` — email + password → access + refresh token pair, for any role (`super_admin`/`admin`/`employee`); same endpoint and usecase for all three, role comes from the resolved user's `user_roles`
- [ ] `TokenRefreshUseCase` — rotate refresh token (per `plan/architecture/backend.md`)
- [ ] `LogoutUseCase` — revoke the presented refresh token
- [ ] `ForgotPasswordUseCase` — any role; enumeration-safe (always returns success regardless of whether the email exists), generates a reset token (random bytes, hashed before storage, short expiry) and publishes a `PasswordResetRequested` event (same transaction as the token write) instead of emailing directly
- [ ] `ResetPasswordUseCase` — consumes a reset token, sets new password, invalidates the token and all existing refresh tokens for that user

### Onboarding Invitation Usecases (`backend/internal/usecase/{interface,implementation}/invitation/`)

- [ ] `InviteUserUseCase` — admin invites by email + role (+ optional department/position); creates a pending `users` row (`is_active = false`, no password) and an invitation row; publishes a `UserInvited` event (same transaction) instead of emailing directly. Tenant-scoped: an admin can only invite into their own tenant. The transaction first takes the tenant row lock (`TenantRepository.LockByID`) and only then validates the email domain (`DomainBelongsToTenant`), so an invite serialises with tenant domain removal/update and cannot create a user on a just-removed domain.
- [ ] `AcceptInvitationUseCase` — consumes the invitation token; invitee (admin or employee) sets a password and the user is activated
- [ ] `ResendInvitationUseCase` — reissues token + publishes an `InvitationResent` event, only while pending
- [ ] `RevokeInvitationUseCase` — admin cancels a pending invitation
- [ ] `ListInvitationsUseCase` — tenant-scoped list for the admin UI (later cycle)
- [ ] `AdminDashboardUseCase` / `SuperAdminDashboardUseCase`
  (`usecase/{interface,implementation}/dashboard/`) — read-only, tenant-scoped aggregates
  for the admin landing page: user counts (total/active/pending-invited), invitation counts
  by status, department and position counts, 5 most recent invitations; the super-admin
  variant adds tenant info and a per-role user breakdown (still the caller's tenant only —
  no cross-tenant aggregates). Added during the UI-fix work to give the admin shell a real
  landing page.
- [ ] Current-tenant settings usecases (`usecase/{interface,implementation}/tenant/`) — super_admin
  only. The product is self-hosted (one deployment per organisation), so there is **no tenant
  creation, listing, or cross-tenant management**: a super admin views and renames **their
  own (current) tenant** and has full CRUD (list/add/update/remove) over **that tenant's
  domains**. The tenant ID always comes from the auth context via the handler, never from the
  URL or body (Invariant 1). Guards: domains are normalised to lowercase and globally unique
  (`ErrDomainAlreadyExists`); the tenant's last domain cannot be removed; and a domain cannot be removed or changed while any non-soft-deleted user of the tenant still has an email on it (`ErrDomainInUse`, 409 `DOMAIN_IN_USE`), since login resolves the tenant from the email domain. Every mutation
  writes an audit log entry.


- [ ] `auth.go` — real JWT validation + role extraction, replacing Cycle 1's stub
- [ ] `tenant.go` — real tenant resolution from JWT claims into `context.Context`, replacing Cycle 1's stub

### Delivery / Routes (`backend/internal/delivery/http/`)

- [ ] `auth_handler.go` — `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`, `POST /api/v1/auth/forgot-password`, `POST /api/v1/auth/reset-password` — all unauthenticated except logout; `login` serves every role (`super_admin`/`admin`/`employee`)
- [ ] `invitation_handler.go` (or fold into `user_handler.go`) — `POST /api/v1/users/invitations` (admin-only), `POST /api/v1/users/invitations/:id/resend`, `DELETE /api/v1/users/invitations/:id`, `GET /api/v1/users/invitations`, and an unauthenticated `POST /api/v1/invitations/accept`
  - Added during EMPLOYEE36-23: unauthenticated `GET /api/v1/invitations/validate?token=` (same rate limiter as accept). Success returns `{email, role}` where `role` is the invitee's role name (`admin` | `employee` | ...); tenant and role are derived from the invitation row, never from input.
  - Distinct failure states (the endpoint is unauthenticated and the ticket waives enumeration-safety), returned by **both** validate and accept when the token hash matches a row: `INVITATION_EXPIRED` (410), `INVITATION_REVOKED` (403), `INVITATION_ACCEPTED` (409; also when the user is already active). An unknown/empty token stays `INVALID_TOKEN` (400). Clients must branch on the envelope `error.code`, not on message text.
  - Added during the UI-fix review: `InviteUserUseCase` requires the invitee's email domain to be a registered `tenant_domains` entry of the caller's tenant (case-insensitive; soft-deleted domains and inactive tenants don't count), otherwise `400 EMAIL_DOMAIN_NOT_ALLOWED`; emails with more than one `@` are rejected as `ErrInvalidEmail`. Consequence: a tenant must have `tenant_domains` rows for every domain it invites from (login already depends on the same table).
- [ ] `role_handler.go` — `GET /api/v1/roles` (admin/super_admin only; Auth → Tenant → `RequireRole`): lists the caller-tenant roles an admin may assign, excluding `super_admin`, returning `{id, name}` only. Added during EMPLOYEE36-21 because the invite form needs a `role_id` and no endpoint exposed role IDs. (Position CRUD is added below in this cycle; the invite form moves from raw UUIDs to selects once it lands).
- [x] `department_handler.go` — `POST/GET /api/v1/departments`, `GET/PUT/DELETE /api/v1/departments/:id` (admin/super_admin only; Auth → Tenant → `RequireRole`): full tenant-scoped department CRUD implemented in EMPLOYEE36-26. Departments carry an `is_active` flag (migration `000018_add_is_active_to_departments`, default true): create accepts optional `is_active` (default true), update accepts optional `is_active` (omitted keeps the current value; toggles are audited as `department.activate`/`department.deactivate`), and `GET /departments` takes an optional `is_active=true|false` filter (400 on other values). Deactivating a referenced department is allowed (only delete is blocked); inviting into an inactive department is rejected inside the invite transaction under the department shared lock with 400 `DEPARTMENT_INACTIVE`.
- [x] `audit_log_handler.go` — read-only audit-log viewer (new Cycle 2 sub-feature; the audit entries themselves are written by the mutation usecases): `GET /api/v1/audit-logs` (paginated `page`/`page_size`, newest first; optional filters `action` exact or prefix when it ends in `.`, `actor_user_id`, `entity_type`, inclusive RFC3339 `from`/`to` with from<=to; 400 on invalid) and `GET /api/v1/audit-logs/:id` (cross-tenant -> 404). Auth -> Tenant -> `RequireRole(admin, super_admin)`; tenant from the auth context only. Response carries the resolved (tenant-scoped, nullable) actor, `entity_type`/`entity_id` and the stored `metadata` JSON with sensitive keys (password/token/hash/secret/otp/...) stripped by the usecase. Index migration `000020_add_audit_logs_tenant_created_at_index`. Frontend viewer UI done: admin `/audit-logs` page (`clients/admin/src/features/audit-logs/`) with sidebar entry for admin/super_admin, entity-type/action-group/date-range filters kept in URL params (Zod+RHF validation, from<=to), pagination, expandable metadata rendered as escaped text, and loading/empty/error-with-retry states.
- [ ] `dashboard_handler.go` — `GET /api/v1/dashboard/admin` (`RequireRole(admin)`) and
  `GET /api/v1/dashboard/super-admin` (`RequireRole(super_admin)`); Auth → Tenant → role
  guard; tenant from the auth context only
- [ ] `tenant_handler.go` — all `RequireRole(super_admin)` (Auth → Tenant → role guard), acting
  on the caller's own tenant: `GET /api/v1/tenant`, `PATCH /api/v1/tenant` (rename),
  `GET /api/v1/tenant/domains`, `POST /api/v1/tenant/domains`,
  `PATCH /api/v1/tenant/domains/:domainId`, `DELETE /api/v1/tenant/domains/:domainId`
- [ ] Wire `auth.go`/`tenant.go` middleware onto every route above except login/refresh/forgot-password/reset-password/invitation-accept

### Employee Dashboard (employee portal home; backend + frontend)

New scope added to Cycle 2 (the employee home previously showed only a System Status health card). Read-only; no migration.

- Backend: `GET /api/v1/dashboard/employee` (Auth -> Tenant -> `RequireRole(employee, admin, super_admin)` -> `RequireCompleteProfile`). Handler extends `dashboard_handler.go` (identity via `tenantAndUser(c)`); usecase `usecase/{interface,implementation}/dashboard` takes explicit `tenantID, userID` and reuses the allocation, status-log, leave-request, holiday, holiday-category, organization-settings (`welcome_message`, `weekly_off_days`) and user repositories. Response types in `internal/types/dashboard/employee.go`: `greeting` (first name, welcome message, today's kind `working_day|holiday|weekly_off`), `stats` (pending leave, upcoming approved leave days, allocation %, hours over the last 7 days, `pending_approvals` nullable and only set for approvers/admins), `leave` (next approved + 3 recent), `holidays` (next 5), `projects` (active allocations) and `week` (7 days with hours / `is_working_day`, plus `missing_days` that respects weekly offs, holidays and approved leave).
- Frontend (`clients/employee`): replaces the System Status card with greeting + quick actions, KPI tiles, leave overview, upcoming holidays, my projects and this-week hours cards, each with skeleton/error/empty states; leave and status-log mutations invalidate the dashboard query.

### Seeding (`backend/internal/infrastructure/database/seeder/`, run via `cmd/bootstrap`)

- [ ] Seed one system tenant
- [ ] Seed default roles (`super_admin`, `admin`, `employee`)
- [ ] Seed the platform Super Admin user (password from env config, never hardcoded)
- [ ] `seeder_test.go` — verify bootstrap is idempotent (running it twice doesn't duplicate the tenant/roles/admin)

### Frontend (`clients/admin/`, `clients/employee/`, `packages/api-client/`, via the
`new-frontend-feature` skill)

Depends on the corresponding backend endpoint from the sections above; don't start a
frontend item before its backend endpoint is callable.

- [ ] `packages/api-client` — auth + invitation API methods (login, refresh, logout,
  forgot/reset password, invite/accept/resend/revoke/list invitations); Axios
  interceptors for attaching the access token and `X-Tenant-ID`, and for silent
  refresh-on-401 using `TokenRefreshUseCase`'s endpoint
- [ ] `clients/admin/src/features/auth/` — login page (email + password), forgot-password
  page, reset-password page; Zustand store for the access token / auth state, TanStack
  Query mutations via the API client
- [ ] `clients/admin/src/features/invitations/` — invite-user form (email + role +
  optional department/position), invitations list (pending/accepted/revoked), resend and
  revoke actions; the list is the landing page at `/invitations`, the form lives on
  `/invitations/new`; the role select is fed by `GET /api/v1/roles`
- [x] `clients/admin/src/features/departments/` — departments list/create/edit/delete UI at `/departments` (admin/super_admin sidebar entry), backed by the EMPLOYEE36-26 department CRUD endpoints; create/edit in a modal, delete behind a confirm modal, 409 `CONFLICT` mapped to duplicate-name (create/edit) or in-use (delete) messages.
- [x] `clients/admin/src/features/positions/` — positions list/create/edit/delete UI at `/positions` (admin/super_admin sidebar entry), mirroring `features/departments/`; 409 `CONFLICT` mapped to duplicate-name or in-use messages. The invitation form's position field is now an active-positions select (`position_id`), with 400 `POSITION_INACTIVE`/`POSITION_NOT_FOUND` shown inline and options refetched.
- [ ] `clients/admin/src/features/tenants/` — super_admin-only "Organization" settings nav item
  and page (`/settings/organization`): view and rename the current tenant, and a domains
  manager (list/add/edit/remove). No tenant list or create screens (self-hosted). Depends on
  the tenant endpoints above.
- [ ] `clients/admin/` app shell — collapsible sidebar + top header (nav links from
  `components/layout/navItems.ts`, UI state in a Zustand `uiStore`); added during
  EMPLOYEE36-21 so later admin features have somewhere to register navigation
- [ ] `clients/employee/src/features/auth/` — login page (email + password), forgot-password
  page, reset-password page; Zustand store for auth state — same shape as the admin auth
  feature, since login is one mechanism for every role
  - `admin`/`super_admin` accounts are intentionally allowed into the employee portal
    (only roles outside `employee`/`admin`/`super_admin` are rejected)
- [ ] Invitation-accept page — a token-in-URL route (unauthenticated) that calls
  `POST /api/v1/invitations/accept`; invitee (admin or employee) sets a password to
  activate. Lives wherever `plan/architecture/frontend.md` places shared/public routes —
  confirm before building if that's not yet decided.
  - **Decision (EMPLOYEE36-23, host switching):** the admin and employee apps are separate hosts. The invite email link targets the *invited role's* app (`admin`/`super_admin` -> `APP_ADMIN_URL`, `employee` -> `APP_EMPLOYEE_URL`, each falling back to `APP_FRONTEND_URL` when unset). The accept page exists in both apps; on load it calls validate, and if the returned `role` belongs to the other app it redirects (carrying the token) to the correct app's accept page. Forgot/reset-password links still use `APP_FRONTEND_URL`.

**Done when:** `make migrate` applies all migrations cleanly, `make migrate-down`
reverses them cleanly, `cmd/bootstrap` seeds a working system tenant + super admin, and
end-to-end **through the UI, with both `cmd/api` and `cmd/worker` running**: an admin can
log in on the admin login page, request a password reset and complete it via the
forgot/reset password pages, invite a new user from the admin invitation-management UI
(invitation email shows up in a local SMTP catcher shortly after — asynchronously, not in
the same request/response cycle), and that invitee can complete the invitation-accept
page (setting a password) to become an active user who can then log in — via the admin
login page if invited as admin, via the employee login page if invited as employee. A
targeted test also confirms the outbox guarantee: rolling back an `InviteUserUseCase`
transaction leaves neither the `user_invitations` row nor the River job behind.

---

## Out of Scope for This Cycle

- SSO / OAuth login for employees — not decided yet, see "Decisions made for this
  cycle".
- Passwordless login (OTP/magic link) for employees — considered and dropped in favor
  of one email+password mechanism for every role; revisit as a follow-up cycle if
  product wants it later.
- Admin-facing email delivery-status UI (e.g. surfacing failed/discarded River jobs per
  tenant) — the data exists in `river_job` but nothing queries/exposes it yet.
- Event replay/reprocessing tooling.
- Splitting the worker onto a separate Postgres instance from the app — revisit only if
  this Postgres becomes a scaling bottleneck; the current design deliberately uses
  River's job table as the outbox because there's a single DB.
- Domain event types beyond the three email-triggering ones (`UserInvited`,
  `InvitationResent`, `PasswordResetRequested`) — the event catalog/dispatcher is built
  generically so later cycles can add more, but this cycle only wires up email.
- Granular permissions (resource/action ACLs) — out of scope; this project uses the
  fixed three-role model.
- Holiday Calendar, or any other planned module (Work Status, Leave Management,
  Courses/Certifications, Benefits, Career Growth, Salary/Taxation, Appraisal, Company
  Policies) — see `CLAUDE.md`'s Execution Model section.

---

## Reference

- Entity list, migration file list, layering: `plan/architecture/backend.md`
- Auth flow decisions and open questions: `plan/initial-planning.md`
- Skill to use for each migration pair: `create-migration` (`.claude/skills/`)
- Skill to use for the usecase/handler/route scaffolding: `new-backend-feature`
  (`.claude/skills/`)
- Agent to use: `backend-agent` (`.claude/agents/`)
- Previous cycle: `plan/cycles/cycle-01-project-setup.md`
- Redis foundation: merged into this cycle (formerly `cycle-05-redis-foundation.md`)
- Next cycle: `plan/cycles/cycle-03-holiday-calendar-migrations-seeding.md`
