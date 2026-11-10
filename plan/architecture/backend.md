# Backend Architecture

This document outlines the backend architecture and folder structure for **Employee360**, following Clean Architecture principles (Ports & Adapters / Hexagonal Architecture) in Go.

---

## 1. Architecture Overview

The backend is structured into clear concentric layers where the inner domain layer has no dependencies on external frameworks or databases. Dependencies always point inwards.

```
       +-------------------------------------------------------+
       |                  Infrastructure Layer                 |
       |  (PostgreSQL, GORM, JWT Provider, Viper, Mailer,       |
       |   River job queue / event dispatcher, CLI)             |
       |   +-----------------------------------------------+   |
       |   |                 Delivery Layer                |   |
       |   |      (HTTP Handlers, Gin Router, Middleware)  |   |
       |   |   +---------------------------------------+   |   |
       |   |   |             Usecase Layer             |   |   |
       |   |   |      (Application Business Logic)     |   |   |
       |   |   |   +-------------------------------+   |   |   |
       |   |   |   |         Domain Layer          |   |   |   |
       |   |   |   | (Entities, Interfaces, Errors)|   |   |   |
       |   |   |   +-------------------------------+   |   |   |
       |   |   +---------------------------------------+   |   |
       |   +-----------------------------------------------+   |
       +-------------------------------------------------------+
```

---

## 2. Directory Tree Breakdown

```
backend/
├── cmd/
│   ├── api/                      # Application entrypoint
│   │   └── main.go               # Server initialization, dependency wiring, startup
│   ├── migrate/                  # Database migration CLI runner
│   │   └── main.go               # Runs up/down SQL migrations
│   ├── bootstrap/                # Database seeder / Initial setup CLI
│   │   └── main.go               # Seeds system tenant, core roles, and platform Super Admin
│   └── worker/                   # Async job-processing entrypoint
│       └── main.go               # Boots the River client (registers EmailWorker), Start(ctx) with graceful shutdown
├── internal/
│   ├── delivery/                 # Presentation / Delivery Layer
│   │   └── http/
│   │       ├── handlers/         # Gin request handlers (input parsing & response writing)
│   │       │   ├── auth_handler.go
│   │       │   ├── invitation_handler.go
│   │       │   ├── user_handler.go
│   │       │   ├── role_handler.go
│   │       │   ├── department_handler.go
│   │       │   ├── position_handler.go
│   │       │   ├── holiday_handler.go
│   │       │   ├── category_handler.go
│   │       │   ├── audit_handler.go
│   │       │   └── tenant_handler.go
│   │       ├── middleware/       # HTTP middlewares
│   │       │   ├── auth.go       # JWT validation & role checking
│   │       │   ├── tenant.go     # Multi-tenant resolution & context injection
│   │       │   ├── cors.go       # CORS configuration
│   │       │   ├── logger.go     # Structured request/response logging
│   │       │   ├── recovery.go   # Panic recovery
│   │       │   └── request_id.go # Unique request tracing ID
│   │       ├── response/         # Standardized JSON response envelope & error helpers
│   │       │   └── response.go
│   │       ├── routes.go         # API router setup & route group definitions
│   │       └── routes_test.go    # HTTP integration tests
│   ├── domain/                   # Enterprise Core (No external dependencies)
│   │   ├── entity/               # Core domain models
│   │   │   ├── tenant.go
│   │   │   ├── tenant_domain.go
│   │   │   ├── user.go
│   │   │   ├── role.go
│   │   │   ├── user_role.go
│   │   │   ├── department.go
│   │   │   ├── position.go
│   │   │   ├── holiday.go
│   │   │   ├── holiday_category.go
│   │   │   ├── audit_log.go
│   │   │   ├── password_reset_token.go
│   │   │   ├── refresh_token.go
│   │   │   └── user_invitation.go
│   │   ├── repository/           # Repository interfaces (Ports)
│   │   │   ├── tenant_repository.go
│   │   │   ├── tenant_domain_repository.go
│   │   │   ├── user_repository.go
│   │   │   ├── role_repository.go
│   │   │   ├── user_role_repository.go
│   │   │   ├── department_repository.go
│   │   │   ├── position_repository.go
│   │   │   ├── holiday_repository.go
│   │   │   ├── category_repository.go
│   │   │   ├── audit_repository.go
│   │   │   ├── password_reset_repository.go
│   │   │   ├── refresh_token_repository.go
│   │   │   └── user_invitation_repository.go
│   │   ├── service/              # Domain service interfaces (Token service, Hasher, Email, Event publisher, Cache + CachePinger, tenant-namespaced CacheKey helper)
│   │   │   ├── token_service.go
│   │   │   ├── hash_service.go
│   │   │   ├── email_service.go  # EmailService interface, EmailMessage/EmailTemplateName
│   │   │   └── event_publisher.go # EventPublisher interface: Publish(ctx, events ...event.Event) error — no GORM/River types
│   │   ├── errors/               # Domain-specific sentinel errors
│   │   │   └── errors.go
│   │   └── event/                # Domain events
│   │       └── events.go         # Event{ID, TenantID, EventType, AggregateType, AggregateID, Payload, OccurredAt} + EventType constants + per-event payload structs
│   ├── usecase/                  # Application Business Rules
│   │   ├── interface/             # Usecase ports — one file per feature, one interface per operation
│   │   │   ├── auth/
│   │   │   │   └── auth.go        # LoginUseCase, RequestOTPUseCase, VerifyOTPUseCase, TokenRefreshUseCase, LogoutUseCase, ForgotPasswordUseCase, ResetPasswordUseCase
│   │   │   ├── invitation/
│   │   │   │   └── invitation.go  # InviteUserUseCase, AcceptInvitationUseCase, ResendInvitationUseCase, RevokeInvitationUseCase, ListInvitationsUseCase
│   │   │   ├── user/
│   │   │   │   └── user.go        # CreateUserUseCase, UpdateUserUseCase, GetUserUseCase, ListUsersUseCase
│   │   │   ├── rbac/
│   │   │   │   └── rbac.go        # AssignRoleUseCase, RemoveRoleUseCase, ListRolesUseCase
│   │   │   ├── department/
│   │   │   │   └── department.go  # CreateDepartmentUseCase, UpdateDepartmentUseCase, ListDepartmentsUseCase
│   │   │   ├── position/
│   │   │   │   └── position.go    # CreatePositionUseCase, UpdatePositionUseCase, ListPositionsUseCase
│   │   │   ├── holiday/
│   │   │   │   └── holiday.go     # CreateHolidayUseCase, UpdateHolidayUseCase, DeleteHolidayUseCase, GetHolidaysUseCase, ListByYearUseCase
│   │   │   ├── category/
│   │   │   │   └── category.go    # CreateCategoryUseCase, ListCategoriesUseCase
│   │   │   ├── tenant/
│   │   │   │   └── tenant.go      # GetTenantUseCase
│   │   │   └── audit/
│   │   │       └── audit.go       # LogActionUseCase, GetAuditLogsUseCase
│   │   └── implementation/        # Usecase adapters — one file per operation, implements the matching interface
│   │       ├── auth/              # Admin login, Employee passwordless verification
│   │       │   ├── login.go
│   │       │   ├── request_otp.go
│   │       │   ├── verify_otp.go
│   │       │   ├── token_refresh.go
│   │       │   ├── logout.go
│   │       │   ├── forgot_password.go
│   │       │   └── reset_password.go
│   │       ├── invitation/        # Admin-driven onboarding invitations
│   │       │   ├── invite_user.go
│   │       │   ├── accept_invitation.go
│   │       │   ├── resend_invitation.go
│   │       │   ├── revoke_invitation.go
│   │       │   └── list_invitations.go
│   │       ├── user/              # User management operations
│   │       │   ├── create_user.go
│   │       │   ├── update_user.go
│   │       │   ├── get_user.go
│   │       │   └── list_users.go
│   │       ├── rbac/              # Role & Permission operations
│   │       │   ├── assign_role.go
│   │       │   ├── remove_role.go
│   │       │   └── list_roles.go
│   │       ├── department/        # Department hierarchy operations
│   │       │   ├── create_department.go
│   │       │   ├── update_department.go
│   │       │   └── list_departments.go
│   │       ├── position/          # Position / Job title operations
│   │       │   ├── create_position.go
│   │       │   ├── update_position.go
│   │       │   └── list_positions.go
│   │       ├── holiday/           # Holiday business operations
│   │       │   ├── create_holiday.go
│   │       │   ├── update_holiday.go
│   │       │   ├── delete_holiday.go
│   │       │   ├── get_holidays.go
│   │       │   └── list_by_year.go
│   │       ├── category/          # Category management operations
│   │       │   ├── create_category.go
│   │       │   └── list_categories.go
│   │       ├── tenant/            # Tenant lookup & resolution operations
│   │       │   └── get_tenant.go
│   │       ├── audit/             # Audit trail logging operations
│   │       │   ├── log_action.go
│   │       │   └── get_audit_logs.go
│   │       └── ucshared/          # Shared usecase-layer helpers
│   │           └── transactor.go  # Transactor.WithinTransaction(ctx, fn) error — wraps a GORM tx, propagates it via ctx so repositories and EventPublisher share it without leaking *gorm.DB/*sql.Tx into the usecase layer
│   ├── infrastructure/           # Frameworks, Drivers & Adapters
│   │   ├── database/             # PostgreSQL connection pool, GORM instance & seeder
│   │   │   ├── postgres.go
│   │   │   └── seeder/           # Database seeding logic
│   │   │       ├── seeder.go     # Super Admin & default roles seeder
│   │   │       └── seeder_test.go
│   │   ├── persistence/          # GORM repository implementations (Adapters)
│   │   │   ├── tenant_repo.go
│   │   │   ├── tenant_domain_repo.go
│   │   │   ├── user_repo.go
│   │   │   ├── role_repo.go
│   │   │   ├── user_role_repo.go
│   │   │   ├── department_repo.go
│   │   │   ├── position_repo.go
│   │   │   ├── holiday_repo.go
│   │   │   ├── category_repo.go
│   │   │   ├── audit_repo.go
│   │   │   ├── password_reset_repo.go
│   │   │   ├── refresh_token_repo.go
│   │   │   └── user_invitation_repo.go
│   │   ├── service/              # External service implementations
│   │   │   ├── jwt_service.go    # JWT generation & validation
│   │   │   ├── hash_service.go   # Password hashing & comparison
│   │   │   ├── mail_service.go   # SMTP EmailService implementation — called only from infrastructure/job/, never from a usecase
│   │   │   ├── mail/
│   │   │   │   └── templates/    # Subject + text + HTML template per EmailTemplateName
│   │   │   ├── redis.go          # NewRedisClient (URL/fields, TLS, timeouts, ping) + RedisCache (Cache + CachePinger); only place go-redis is imported (archtest-enforced)
│   │   │   └── noop_cache.go     # NoopCache used when REDIS_ENABLED=false (returns ErrCacheUnavailable)
│   │   ├── eventing/              # EventPublisher implementation + event->job mapping
│   │   │   ├── dispatcher.go      # Maps EventType -> River job args (SendEmailArgs)
│   │   │   └── river_publisher.go # EventPublisher impl: extracts *sql.Tx from ctx, dispatches, riverClient.InsertTx (same transaction as the caller's business write)
│   │   ├── job/                   # River workers
│   │   │   └── email_worker.go    # river.Worker[SendEmailArgs] — calls EmailService.Send; river.JobCancel on permanent validation errors
│   │   ├── container/            # Dependency Injection container
│   │   │   └── container.go      # Initializes and wires all layers
│   │   └── server/               # HTTP server lifecycle & graceful shutdown
│   │       └── server.go
│   └── ctx/                      # Context keys & helper extractors
│       └── ctx.go                # TenantIDFromContext, UserIDFromContext, RolesFromContext, etc.
├── pkg/                          # Reusable cross-cutting utility packages
│   ├── logger/                   # Zerolog wrapper
│   │   └── logger.go
│   └── validator/                # Payload validation rules (go-playground/validator)
│       └── validator.go
├── shared/                       # Shared constants and utility types
│   └── constants.go
├── migrations/                   # Versioned SQL migrations (golang-migrate)
│   ├── 000001_create_tenants.up.sql
│   ├── 000001_create_tenants.down.sql
│   ├── 000002_create_tenant_domains.up.sql
│   ├── 000002_create_tenant_domains.down.sql
│   ├── 000003_create_departments.up.sql
│   ├── 000003_create_departments.down.sql
│   ├── 000004_create_positions.up.sql
│   ├── 000004_create_positions.down.sql
│   ├── 000005_create_users.up.sql
│   ├── 000005_create_users.down.sql
│   ├── 000006_create_roles.up.sql
│   ├── 000006_create_roles.down.sql
│   ├── 000007_create_user_roles.up.sql
│   ├── 000007_create_user_roles.down.sql
│   ├── 000008_create_audit_logs.up.sql
│   ├── 000008_create_audit_logs.down.sql
│   ├── 000009_create_password_reset_tokens.up.sql
│   ├── 000009_create_password_reset_tokens.down.sql
│   ├── 000010_create_refresh_tokens.up.sql
│   ├── 000010_create_refresh_tokens.down.sql
│   ├── 000011_create_user_invitations.up.sql
│   ├── 000011_create_user_invitations.down.sql
│   ├── 000012_create_river_schema.up.sql       # River's own tables (river_job, river_leader, river_queue, river_client*, river_migration) — generated via `river migrate-get`, not hand-written
│   ├── 000012_create_river_schema.down.sql
│   ├── 000013_create_holiday_categories.up.sql
│   ├── 000013_create_holiday_categories.down.sql
│   ├── 000014_create_holidays.up.sql
│   └── 000014_create_holidays.down.sql
├── config/                       # Configuration definition & loading (Viper)
│   ├── config.go
│   └── config.yaml.example
├── docker-compose.yml            # Local development infrastructure (Postgres, API)
├── Makefile                      # Build, run, test, seed, and migration automation commands
├── go.mod
└── go.sum
```

---

## 3. Layer Responsibilities & Data Flow

### 3.1 Request Flow Lifecycle

```
HTTP Request
     │
     ▼
[Middleware Chain] (RequestID -> Logger -> CORS -> JWT Auth -> Tenant Resolution)
     │
     ▼
[Delivery / Handler] (Binds JSON/Query, Validates Input, Calls Usecase)
     │
     ▼
[Usecase / Interactor] (Executes Business Rules, Manages Audit Log, Calls Repositories)
     │
     ▼
[Domain Repository Interface] (Defined in domain/repository)
     │
     ▼
[Infrastructure Persistence] (GORM implementation, scoped by explicit tenantID param)
     │
     ▼
PostgreSQL Database
```

### 3.2 Layer Rules & Boundaries

1. **Domain Layer (`internal/domain`)**:
   - Contains pure Go structs representing core business entities (`User`, `Role`, `UserRole`, `Department`, `Position`, `Holiday`, `HolidayCategory`, `AuditLog`, `Tenant`, `TenantDomain`, `PasswordResetToken`, `RefreshToken`, `UserInvitation`).
   - Defines repository and external service interfaces.
   - Defines domain errors.
   - **Zero dependencies** on Gin, GORM, database drivers, or external libraries.

2. **Usecase Layer (`internal/usecase`)**:
   - Split into `usecase/interface/<feature>/` (ports — one file per feature declaring one `XxxUseCase` interface per operation, each with a single `Execute(ctx, ...) (result, error)` method) and `usecase/implementation/<feature>/` (adapters — one file per operation, e.g. `create_user.go` defines `CreateUserUseCaseImpl` with injected repository/service dependencies, a `NewCreateUserUseCase` constructor, and a compile-time assertion `var _ userUC.CreateUserUseCase = (*CreateUserUseCaseImpl)(nil)`).
   - The Delivery layer depends only on the `usecase/interface` types, never on `usecase/implementation` directly — implementations are wired in via the DI container (`infrastructure/container`).
   - Contains application-specific business logic; coordinates domain entities, repositories, and domain services.
   - Independent of transport protocol (knows nothing about HTTP, Gin, or JSON).

3. **Delivery Layer (`internal/delivery/http`)**:
   - Handles HTTP serialization, deserialization, HTTP status codes, and headers.
   - Injects tenant and user claims into the request `context.Context`.
   - Never accesses repositories or databases directly; delegates all logic to use cases.

4. **Infrastructure Layer (`internal/infrastructure`)**:
   - Implements interfaces defined in the domain layer (GORM database repositories, JWT token generation, email providers, the `EventPublisher` port).
   - Contains the seeder (`infrastructure/database/seeder/`) for bootstrapping the platform Super Admin and default system roles.
   - Responsible for technical details like connection pooling, transaction management, and server lifecycle.
   - Owns async side-effect delivery: `eventing/` turns a published domain `Event` into a River job **inside the same DB transaction** as the usecase's business write (River's own job table is the outbox — no separate outbox table needed on a single-Postgres setup); `job/` holds the River workers (running in the separate `cmd/worker` process) that actually perform the side effect (e.g. sending mail via `mail_service.go`).

### 3.3 Multi-Tenancy & Platform Governance Strategy

- **Shared Database, Shared Schema**: All tenants share the same PostgreSQL database and tables.
- **Row-Level Isolation**: Every table (except `tenants`) includes a `tenant_id` column.
- **Platform Super Admin Governance**:
  - The `super_admin` role operates at the platform level to govern tenants, manage platform settings, and monitor system health.
  - Initial platform governance is bootstrapped via `cmd/bootstrap/main.go` using environment-configured credentials.
- **Context Injection**: The tenant resolution middleware (referenced only from `routes.go`) resolves the `tenant_id` from the JWT token or verified domain and stores it in the request context via `internal/ctx`.
- **Identity Verification**: `middleware.Auth(tokenService, verifier)` validates the JWT, then calls the `VerifyIdentityUseCase` port (`usecase/interface/auth`, implemented in `usecase/implementation/auth/verify_identity.go` over `TenantReader`, `UserRepository.GetByID(tenantID, id)` and `UserRoleRepository.GetRolesByUserID`). It checks the tenant exists, `is_active` and `deleted_at IS NULL`, and the user exists in that tenant, is active and not soft-deleted, so deleted/deactivated users and tenants lose access immediately rather than at token expiry. Any unusable identity returns one uniform 401 (the response never says which check failed); a repository error fails closed with 503. The user's current roles from the database (not the token claims) populate the context used by `RequireRole`. This costs three indexed lookups per authenticated request; there is deliberately no cache, so revocation is immediate. `Tenant()` runs after `Auth` and only consumes the verified tenant.
- **Identity Flow**: Handlers are the only layer that reads identity (`ctx.TenantIDFromContext` / `UserIDFromContext` / `RolesFromContext` on `c.Request.Context()`, 401 on failure) and pass `tenantID` / `actorID` to usecases as explicit parameters. Usecases and repositories never read identity from `context.Context` and never import `middleware`.
- **Query Scoping**: Repository implementations in `internal/infrastructure/persistence` receive `tenantID` as an explicit parameter and enforce tenant filtering with it on all queries and mutations.
