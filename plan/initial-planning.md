# Initial Planning

## Tech Stack

### Frontend

| Concern                      | Choice                         | Purpose / Notes                                               |
|------------------------------|--------------------------------|---------------------------------------------------------------|
| Language                     | TypeScript                     | static typing across both client apps                         |
| UI library                   | React (Vite as build tool)     | component model + fast dev/build tooling                      |
| Styling                      | Tailwind CSS                   | utility-first styling, shared design tokens                   |
| Routing                      | React Router                   | client-side routing per app                                   |
| Server state / data fetching | TanStack Query (React Query)   | caching, refetching, mutation state for API data              |
| Client/UI state              | Zustand                        | ephemeral UI/auth state only, kept separate from server state |
| Forms & validation           | React Hook Form + Zod          | holiday create/edit forms with schema validation              |
| API client                   | Axios                          | thin wrapper with interceptors for auth token + tenant header |
| Testing                      | Vitest + React Testing Library | unit/component tests                                          |

Two client apps share a common component/UI library and API client, but ship as separate builds:

- **Admin App** – holiday CRUD, categories, audit log view.
- **Employee App** – read-only calendar/list view, login via company email.

### Backend

| Concern        | Choice                         | Purpose / Notes                                                    |
|----------------|--------------------------------|--------------------------------------------------------------------|
| Language       | Go                             | performance + simple deployment (single binary), fits self-hosting |
| HTTP framework | Gin                            | routing, middleware chain, request binding                         |
| ORM            | GORM                           | models, migrations helper, query building                          |
| Database       | PostgreSQL                     | primary datastore, `jsonb` for audit metadata                      |
| Migrations     | golang-migrate                 | versioned SQL migrations, checked into `backend/migrations/`       |
| Auth           | JWT (access + refresh tokens)  | stateless auth carrying `tenant_id` + role claims                  |
| Config         | Viper                          | env-based config, per-deployment for self-hosting                  |
| Validation     | go-playground/validator        | request payload validation via Gin binding                         |
| Logging        | zerolog                        | structured logging                                                 |
| Testing        | Go `testing` package + testify | unit/integration tests                                             |

### Cross-cutting

- **Headless**: backend exposes a versioned REST API (`/api/v1/...`) consumed independently by the Admin and Employee frontends.
- **Multi-tenant**: every table that holds tenant-owned data carries a `tenant_id`; tenant is resolved per-request (see Cycle 1 below).
- **Self-hostable**: local dev and self-hosted deployment both run via `docker-compose` (Postgres + backend + frontend builds); no dependency on a specific cloud provider's managed services.

---

## Design Cycles

### Cycle 1 — Frontend & Backend Architecture

#### Frontend architecture

- **App split**: two separate Vite/React builds — `admin-app` and `employee-app` — sharing a common `packages/ui` (components) and `packages/api-client` (typed Axios client + React Query hooks) via a monorepo (npm/pnpm workspaces).
- **Structure per app**: `pages/` (routes), `features/` (feature-scoped components + hooks, e.g. `holidays/`, `auth/`), `components/` (shared local UI), `lib/` (api client instance, auth storage).
- **Auth flow**: employee logs in with company email (magic link or SSO-style verification, tenant resolved from email domain); admin logs in with email + password. Access token stored in memory, refresh token in httpOnly cookie.
- **Tenant context**: resolved once at login and attached to every request as a header (`X-Tenant-ID`) or embedded in the JWT claims; frontend never lets a user switch tenants mid-session.
- **Data flow**: React Query owns all server state (holidays, categories, audit log) with cache invalidation on mutations; Zustand only holds ephemeral UI/auth state.

#### Backend architecture

- **Layering**: `handler` (Gin routes, request/response shaping) → `service` (business rules, authorization) → `repository` (GORM queries) → `model` (structs/entities). Handlers never call GORM directly.
- **Package layout (proposed)**:
  ```
  backend/
    cmd/api/            # main.go, wiring
    internal/
      auth/              # JWT issuing/parsing, middleware
      tenant/             # tenant resolution middleware
      holiday/            # handler + service + repository + model
      audit/              # audit log writer, queried by admin holiday changes
      config/             # Viper config loading
    migrations/           # golang-migrate SQL files
  ```
- **Middleware chain**: request ID → logging → CORS → auth (JWT parse) → tenant resolution (derive `tenant_id` from JWT claims, inject into request context) → route handler.
- **Multi-tenancy strategy**: shared database, shared schema, row-level isolation via a `tenant_id` column on every tenant-owned table; every repository query is scoped by `tenant_id` pulled from context — never trusted from client-supplied input alone.
- **API design**: versioned REST (`/api/v1/holidays`, `/api/v1/categories`, `/api/v1/admin/...`); admin-only routes protected by a role check in addition to tenant scoping.

#### Open questions to resolve before Cycle 2

- Employee login mechanism: magic link vs OTP vs company SSO (affects auth package design).
- Whether tenant is resolved from email domain, subdomain, or explicit tenant selection at login.

### Cycle 2 — Backend Architecture & Database Setup

#### Detailed backend setup

- Finalize `internal/` package boundaries from Cycle 1 and stub each package (interfaces for service/repository so handlers can be built against mocks before DB work lands).
- Add `docker-compose.yml` with a Postgres service for local development.
- Wire `golang-migrate` into `cmd/api` (or a separate `cmd/migrate`) so migrations run on startup or via a CLI command.

