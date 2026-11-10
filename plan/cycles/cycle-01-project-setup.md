# Cycle 1 — Project Setup & Scaffolding

| | |
|---|---|
| **Status** | Complete |
| **Module** | Initial backend + frontend scaffolding (no feature work) |
| **Source** | `plan/architecture/backend.md`, `plan/architecture/frontend.md`, `plan/initial-planning.md` |

This is the scope/status doc for Cycle 1. Read this before starting or resuming work.
There is no application feature in this cycle — the goal is a running skeleton: both
backends and frontends boot, talk to each other and to Postgres, and are configurable
via env. Cross-cutting principles (open source, multi-tenant, self-hostable, headless)
live in `CLAUDE.md`, not here.

---

## Objective

Stand up the full-stack skeleton so every later cycle starts from a working, runnable
project instead of an empty repo: correct folder structure, a server that boots, a
database it can reach, config/env plumbing, and a health check proving the whole chain
works.

---

## Sub-Features

### Backend (`backend/`)

- [x] `go.mod` + directory tree per `plan/architecture/backend.md` (`cmd/api`, `cmd/migrate`, `cmd/bootstrap`, `internal/{delivery,domain,usecase,infrastructure,ctx}`, `pkg/`, `shared/`, `migrations/`, `config/`)
- [x] Viper config loading (`config/config.go` + `config.yaml.example`) — reads from env vars / `.env`
- [x] `docker-compose.yml` — Postgres service for local dev
- [x] DB connection (`internal/infrastructure/database/postgres.go`) — GORM + Postgres, reads config, fails fast with a clear error if unreachable
- [x] `golang-migrate` wired via `cmd/migrate` (even with zero migrations yet — the runner must work)
- [x] Base middleware chain stubbed in order: `request_id → logger → cors → recovery` (auth/tenant middleware comes in Cycle 2 once there's something to protect)
- [x] Health route: `GET /api/v1/health` (or `/healthz`) — returns 200 + DB ping status, no auth required
- [x] `cmd/api/main.go` — server boot, dependency wiring stub, graceful shutdown
- [x] `Makefile` targets: `make dev`, `make migrate`, `make test` (even if `make test` has nothing to test yet)

**Done when:** `go run ./cmd/api` starts, `docker-compose up -d` brings up Postgres and the
app connects to it, and `curl localhost:PORT/api/v1/health` returns 200.

### Frontend (root + `clients/`, `packages/`)

- [x] Root `pnpm-workspace.yaml` + root `package.json`
- [x] `clients/admin/` — Vite + React + TS scaffold, Tailwind configured, renders a placeholder page
- [x] `clients/employee/` — same scaffold
- [x] `packages/api-client/` — Axios instance + interceptor stubs (JWT, `X-Tenant-ID` — wired but with nothing to attach yet), one typed call to the backend health route as the first smoke test
- [x] `packages/ui/` — optional, empty/minimal placeholder package is fine for this cycle
- [x] `.env.example` per app (API base URL, etc.)
- [x] Root workspace scripts: `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm typecheck`

**Done when:** `pnpm --filter admin dev` and `pnpm --filter employee dev` both boot, and
each app successfully calls the backend health route through `packages/api-client` and
renders the result (proves the client → API → DB chain is wired end to end).

### Not in this cycle

No auth, no tenant resolution, no entities/migrations beyond the runner itself, no real
routes besides health. That's Cycle 2.

---

## Reference

- Backend directory tree & layering: `plan/architecture/backend.md`
- Frontend monorepo layout: `plan/architecture/frontend.md`
- Agents to use while executing this cycle: `backend-agent`, `frontend-agent` (`.claude/agents/`)
- Next cycle: `plan/cycles/cycle-02-auth-onboarding.md`
