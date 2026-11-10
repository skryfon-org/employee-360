# Shared Project Context

Facts and rules that apply regardless of which AI tool is reading them (Claude Code
via `CLAUDE.md`, other tools via `AGENTS.md`). Tool-specific dispatch mechanics
(which agent/skill runner to invoke, and how) live in the importing file, not here.

Employee360: open-source, self-hostable, multi-tenant employee platform (holidays,
policies, benefits, career, comp/tax, performance).

## Status

Open source (AGPL-3.0), pre-1.0 (`v0.x`). Cycle 1 (Project Setup & Scaffolding) is
complete. Cycle 2 (Auth, Email Service & Onboarding Invitations) is the active cycle
doc, but the code has moved past the cycle files: holiday calendar, employee
profiles, leave, status logs, projects/allocations, dashboards and audit logs exist in
`backend/` and `clients/`. Check the code, not just the cycle checkboxes, and see the
feature table in `README.md`. Only `main` exists as a long-lived branch; pushes to it
deploy staging and `v*` tags deploy production.

---

## Execution Model: Cycles (read first)

Work ships as a sequence of **cycles**, not "phases" — each cycle is one
narrowly-scoped slice of a module (e.g. migrations, then backend API, then
frontend), never the whole module at once. Each cycle has its own scope doc:
`plan/cycles/cycle-NN-<name>.md` — **read the active cycle's file before doing any
work; it is the authoritative current scope**, not the tech-stack description below.

- **Cycle 1 (complete)** — Project Setup & Scaffolding: folder structure, DB
  connectivity, config/env, a health route. No feature work. →
  `plan/cycles/cycle-01-project-setup.md`
- **Cycle 2 (active)** — Auth, Email Service & Onboarding Invitations: real
  auth/tenant middleware (replacing Cycle 1's stubs), admin email+password login,
  employee passwordless (OTP) login, forgot/reset password, an asynchronous
  event-driven transactional email service (River-backed outbox), and admin-driven
  user onboarding invitations, plus department/position CRUD and the optional Redis
  foundation (formerly Cycle 5, no consumers). Full stack,
  migrations through handlers **and the frontend that consumes them** (admin login,
  employee OTP login, forgot/reset password, admin invitation-management UI,
  invitation-accept page). →
  `plan/cycles/cycle-02-auth-onboarding.md`
- **Cycle 3 (planned, blocked on Cycle 2)** — Holiday Calendar, full stack:
  migrations + seeding, backend API (holiday/category CRUD, year-wise queries,
  calendar settings), and admin/employee frontend (calendar + list views). →
  `plan/cycles/cycle-03-holiday-calendar-migrations-seeding.md`
- Further modules (Work Status, Leave Management, Courses/Certifications, Benefits,
  Career Growth, Salary/Taxation, Appraisal, Company Policies): no cycle file until
  work starts on them — don't foreclose them.

New cycle: create `plan/cycles/cycle-NN-<name>.md`, scope narrowly, break into
sub-features before coding.

Everything from here down describes the **target** architecture across all
cycles — it is not a green light to build all of it now.

---

## Project Overview & Tech Stack

### Backend
- **Language**: Go
- **Architecture**: Go Clean Architecture (Hexagonal / Ports & Adapters)
- **HTTP Framework**: Gin
- **ORM / Persistence**: GORM with PostgreSQL
- **Database Migrations**: `golang-migrate` (checked into `backend/migrations/`)
- **Authentication**: JWT (Stateless access & refresh tokens carrying `tenant_id`
  and role claims)
- **Configuration**: Viper (Environment variables & YAML)
- **Structured Logging**: Zerolog

### Frontend
- **Monorepo Structure**: `pnpm` workspaces (`clients/admin`, `clients/employee`,
  `packages/api-client`)
- **Framework / Bundler**: React with TypeScript & Vite
- **Styling**: Tailwind CSS
- **Server State Management**: TanStack Query (React Query)
- **UI / Ephemeral State**: Zustand
- **Forms & Validation**: React Hook Form + Zod
- **API Client**: Axios instance in `packages/api-client` (with automated JWT &
  `X-Tenant-ID` interceptors)

---

## Core Architectural Invariants

### Invariant 1: Strict Multi-Tenancy Isolation
- Every database table holding tenant data **must** carry a `tenant_id` column.
- Handlers and repositories **must never trust client-supplied tenant IDs** in
  request payloads.
- `tenant_id` is resolved exclusively via authentication middleware and injected
  into the Go `context.Context` (via `internal/ctx`).
- A validly signed JWT is not sufficient: on every authenticated request the
  auth middleware also verifies against the database (through the
  `VerifyIdentityUseCase` port) that the tenant exists, is active and not
  soft-deleted, and that the user exists **in that tenant**, is active and not
  soft-deleted. Failures are a uniform 401; a lookup error fails closed (503).
  The user's current roles are loaded from the database and replace the token's
  role claims.
- Only handlers read identity from the context; they pass `tenant_id` / user ID
  down as explicit usecase and repository parameters. Usecases and repositories
  never read identity from `context.Context`.
- All repository queries and mutations **must scope queries by the explicit
  `tenantID` parameter** (sourced from the authenticated context by the handler).

### Invariant 2: Go Clean Architecture Boundaries
- **Domain Layer (`internal/domain/`)**: Pure Go entities and repository
  interfaces. Has **zero dependencies** on external frameworks, database drivers,
  or Gin.
- **Usecase Layer (`internal/usecase/`)**: Contains pure application business
  logic. Coordinates domain entities and repositories.
- **Delivery Layer (`internal/delivery/http/`)**: Gin handlers, route
  registration, middleware, and request/response serialisation. **Handlers must
  never call GORM or database queries directly.** Middleware is referenced only
  from route registration; handlers read identity via `internal/ctx`, never via
  `middleware.*`.
- **Infrastructure Layer (`internal/infrastructure/`)**: Implements repository
  interfaces using GORM, external services (JWT, bcrypt, mailer), database
  seeder, and server lifecycle.

### Invariant 3: Frontend Separation of Concerns
- **Server State**: Managed exclusively by TanStack Query.
- **Client/UI State**: Managed via Zustand stores (sidebar toggle, active modal,
  active filters).
- Feature directories in `clients/*/src/features/<feature>/` are self-contained
  with their own `components/`, `pages/`, `queries/`, `schemas/`, and
  `routes.tsx`.

---

## Platform Roles & Governance Model

- **Platform Super Admin (`super_admin`)**: System-level governance (tenant
  provisioning, platform health, global configuration). Initialized via
  `backend/cmd/bootstrap/main.go`.
- **Tenant Admin (`admin`)**: Manages organization holidays, departments,
  positions, users, and roles for their tenant.
- **Employee (`employee`)**: Email + password login to view holiday calendar
  and company directory.

---

## Development Workflow & Commands

The common targets are below. `make help` lists all of them, and `CONTRIBUTING.md`
has the full first-run setup.

### Backend Commands
```bash
# Start the dev server
make dev

# Apply / reverse migrations
make migrate
make migrate-down

# Seed system tenant, roles, and platform Super Admin
make bootstrap-admin

# Run tests
make test
```

### Frontend Commands
```bash
# Install dependencies
pnpm install

# Run dev servers
pnpm --filter admin dev
pnpm --filter employee dev

# Build
pnpm build

# Type check
pnpm typecheck
```

---

## Principles

- **Open Source** — usable/customisable/contributable by any organisation.
- **Multi-Tenant** — `tenant_id` row-scoped server-side, never trusted from
  client input.
- **Self-Hostable** — `docker-compose`, no cloud-provider lock-in.
- **Headless** — versioned `/api/v1` REST, client-agnostic, no cookie-only auth.

---

## Documentation Map

- `plan/cycles/` — current, authoritative per-cycle scope (source of truth) —
  start here.
- `README.md` — pitch, feature status per module, quick start.
- `CONTRIBUTING.md` — dev setup, branch model (fork → PR to `main`), DCO,
  review procedure. `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md` alongside.
- `docs/self-hosting.md`, `docs/configuration.md` (full env reference),
  `docs/architecture.md` (pointer into `plan/architecture/`).
- `plan/initial-planning.md` — tech stack + early "Design Cycles" notes (unrelated
  naming to `plan/cycles/` — don't conflate).
- `plan/architecture/backend.md` — complete backend Clean Architecture directory
  tree and design.
- `plan/architecture/frontend.md` — complete frontend monorepo directory tree and
  design.
- `plan/architecture/overview.md` — quick-glance Mermaid diagrams (system
  context, request lifecycle) — start here for a visual before reading
  backend.md/frontend.md.
- `plan/architecture/diagrams/` — detailed, exportable HTML+SVG companion
  diagrams for backend.md/frontend.md/overview.md (PNG/PDF export via the
  `architecture-diagram` skill) — for deep dives, not quick reading.

Prefer real code over these docs once it exists.
