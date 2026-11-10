---
name: new-backend-feature
description: >
  Scaffold a new backend feature end-to-end through every Clean Architecture layer
  (entity, repository port, usecase, GORM adapter, handler, route) for the Employee360
  Go backend. Use when the user asks to add a new backend resource, endpoint, or domain
  concept (e.g. "add a holiday attachments feature", "add an endpoint for X").
---

# New Backend Feature

Employee360's backend follows strict Clean Architecture layering (see the `backend-agent`
agent and `plan/architecture/backend.md`). A new feature touches every layer, in this
order, so each layer only depends on interfaces defined by the layer inside it.

## Step 1 — Domain layer (`backend/internal/domain/`)

1. `entity/<feature>.go` — plain Go struct, no GORM tags beyond what's needed for the
   infra adapter to map it (prefer keeping GORM tags in the persistence layer's own
   model if the project separates domain entities from persistence models — check
   existing entities for the pattern in use before deciding).
2. `repository/<feature>_repository.go` — an interface (port): `Create`, `GetByID`,
   `Update`, `Delete`, `List...` as needed. Every method that touches a tenant-owned
   table takes `ctx context.Context` first — the implementation pulls `tenant_id` out
   of it, callers never pass a tenant ID explicitly.
3. If the feature introduces new error cases, add sentinel errors to `domain/errors/errors.go`.

## Step 2 — Usecase layer (`backend/internal/usecase/<feature>/`)

One file per operation (matches existing packages like `usecase/holiday/`):
`create_<feature>.go`, `update_<feature>.go`, `delete_<feature>.go`, `get_<feature>.go`,
`list_<feature>.go`. Each usecase:
- Depends only on the domain repository/service **interfaces**, injected via constructor.
- Contains the business rules (validation beyond struct tags, authorization decisions
  that need domain knowledge, orchestration across repos).
- If the operation is an admin mutation, call the `audit` usecase to record the change
  (who, what, before/after) — check `usecase/audit/log_action.go` for the pattern.
- Is transport-agnostic: no Gin types, no JSON tags here.

## Step 3 — Infrastructure adapter (`backend/internal/infrastructure/persistence/<feature>_repo.go`)

- Implements the domain repository interface using GORM.
- **Every query and mutation must scope by `tenant_id` pulled from context** via the
  `ctx` package helper (e.g. `ctx.TenantIDFromContext(ctx)`) — never accept a tenant ID
  as a method parameter from the caller.
- Register the new repo in `infrastructure/container/container.go` so it's wired into
  the corresponding usecase.

## Step 4 — Delivery layer (`backend/internal/delivery/http/`)

1. `handlers/<feature>_handler.go` — thin: bind/validate the request DTO
   (`go-playground/validator` tags), call the usecase, translate the result/error into
   a response via `response.Success/Created/Paginated/Error` — never `c.JSON(...)`
   directly and never call a repository from a handler.
2. Register routes in `routes.go` under `/api/v1/...`. If the endpoint is admin-only,
   add the role-check middleware in addition to the standard tenant-scoping middleware
   — tenant scoping is not an authorization check by itself.

## Step 5 — Migration

If the feature needs a new table/column, use the `create-migration` skill first (before
Step 1, in practice) so the entity/repo code matches the schema.

## Step 6 — Tests

- Usecase: testify unit test with a mocked repository — assert business rules, not
  persistence details.
- Repository: a test that proves tenant isolation — writing with tenant A's context and
  reading with tenant B's context must never leak rows.

## Step 7 — Frontend follow-up

If this backend feature needs to be consumed by a client app, tell the user to run
`pnpm generate:api` (once `packages/api-client` exists) to regenerate the Orval client
from the updated OpenAPI/swagger spec, then use the `new-frontend-feature` skill to wire
up the corresponding feature module.
