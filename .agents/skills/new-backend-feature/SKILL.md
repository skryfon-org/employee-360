---
name: new-backend-feature
description: >-
  Scaffold a new backend feature end-to-end through every Clean Architecture layer
  (entity, repository port, usecase, GORM adapter, handler, route) for the Employee360
  Go backend. Use when adding a new backend resource, endpoint, or domain concept.
---

# New Backend Feature

Employee360's backend follows strict Clean Architecture layering (Ports & Adapters). A new feature touches every layer, in this order, so each layer only depends on interfaces defined by the layer inside it.

## Step 1 — Domain Layer (`backend/internal/domain/`)

1. `entity/<feature>.go`: Plain Go struct without framework dependencies.
2. `repository/<feature>_repository.go`: Interface defining methods (`Create`, `GetByID`, `Update`, `Delete`, `List...`). Every method taking a tenant-owned entity accepts `ctx context.Context` as its first argument.
3. `errors/errors.go`: Add domain-specific sentinel errors if new failure cases exist.

## Step 2 — Usecase Layer (`backend/internal/usecase/<feature>/`)

One file per operation (e.g. `create_<feature>.go`, `get_<feature>.go`, `list_<feature>.go`):
- Depends only on domain repository/service **interfaces**, injected via constructor.
- Contains business rules, cross-repository orchestration, and authorization logic.
- Admin mutations must call the `audit` usecase to log changes.
- Transport-agnostic (no Gin context or HTTP status codes).

## Step 3 — Infrastructure Persistence Adapter (`backend/internal/infrastructure/persistence/<feature>_repo.go`)

- Implements the domain repository interface using GORM.
- **Extracts `tenant_id` from context** via `ctx.TenantIDFromContext(ctx)` — never accepts a tenant ID as a function parameter.
- Register the repo in `infrastructure/container/container.go` so it's wired into the DI container.

## Step 4 — Delivery Layer (`backend/internal/delivery/http/`)

1. `handlers/<feature>_handler.go`:
   - Binds and validates request DTOs (`go-playground/validator` tags).
   - Calls the usecase.
   - Returns responses using `response.Success`, `response.Created`, `response.Paginated`, or `response.Error`.
   - Never accesses GORM or repositories directly.
2. Register routes in `routes.go` under `/api/v1/...`.
   - Admin routes require role-checking middleware in addition to tenant scoping middleware.

## Step 5 — Migrations

If the feature requires database changes, run the `create-migration` skill first.

## Step 6 — Automated Unit Tests

- Usecase tests: Testify unit tests with mocked repository interfaces.
- Persistence tests: Tests verifying strict multi-tenant isolation.
