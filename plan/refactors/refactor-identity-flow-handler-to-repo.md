# Refactor: Identity (tenant_id / user_id) flows Handler → Usecase → Repository

Status: implemented · Area: `backend/` · Owner agent: `backend-agent`

## Goal

Make identity propagation explicit and one-directional:

```
Middleware (routes only) ──► ctx package ──► Handler ──► Usecase ──► Repository
   sets values              (internal/ctx)   reads once   param        param
```

- **Middleware** is used **only in route registration** (`routes.go`). It validates the
  token / resolves the tenant and stores `tenant_id`, `user_id`, `roles` in the request
  `context.Context` via `internal/ctx` (`ctx.WithTenantID`, `ctx.WithUserID`, `ctx.WithRoles`).
- **Handlers** are the *only* place that reads identity. They read it through the
  `internal/ctx` package (`ctx.go`) from `c.Request.Context()`, parse/validate it, and pass
  it down as explicit arguments.
- **Usecases** receive `tenantID` / `userID` as explicit parameters. They never read from
  `context.Context` and never import `middleware`.
- **Repositories** receive `tenantID` as an explicit parameter. They never read it from
  `context.Context`.

This keeps Invariant 1 (tenant resolved only by auth middleware, never from payloads) and
Invariant 2 (clean layering) intact, and makes the dependency visible in every signature
instead of hidden in `ctx`.

## Rules

1. `middleware.*` (including `middleware.GetTenantID`, `middleware.GetUserID`,
   `middleware.GetRoles`, `middleware.GetClaims`) must **not** be called from handlers,
   usecases, or repositories. Middleware is only referenced from `routes.go`.
2. Handlers read identity via `ctx.TenantIDFromContext`, `ctx.UserIDFromContext`,
   `ctx.RolesFromContext` (and `ClientIP` / `UserAgent` helpers) on `c.Request.Context()`.
3. Missing/invalid identity in a handler → `401` via `response.Unauthorized`; the usecase is not called.
4. Usecases and repositories must not call any `ctx.*FromContext` accessor for
   tenant/user/roles. `context.Context` is kept only for cancellation, deadlines and the
   transaction handle (`database.DBFromContext`).
5. Client-supplied tenant IDs in payloads/params remain forbidden (Invariant 1).
6. Pre-auth flows (login, forgot/reset password, invitation validate/accept) have no
   middleware-resolved tenant; they keep resolving tenant via domain lookup / token
   and pass it down the same explicit way.

## Current violations (found by grep)

| # | Location | Problem |
|---|---|---|
| 1 | `internal/delivery/http/handlers/auth_handler.go:146,151` | Handler calls `middleware.GetTenantID(c)` / `middleware.GetUserID(c)` — handler depends on middleware. |
| 2 | `internal/delivery/http/handlers/role_handler.go:35` | Passes only ctx; usecase pulls tenant from context. |
| 3 | `internal/delivery/http/handlers/invitation_handler.go` (`Invite`, `Resend`, `Revoke`, `List`) | No identity read in handler; usecases pull tenant/actor from context. |
| 4 | `internal/usecase/implementation/invitation/common.go:44,62` (`actorFromContext`, `tenantFromContext`) | Usecase reads user/tenant from ctx. |
| 5 | `internal/usecase/implementation/invitation/invite_user.go:69` | Reads inviter via `ctx.UserIDFromContext`. |
| 6 | `internal/usecase/implementation/role/list_assignable_roles.go:41` | Reads tenant via `ctx.TenantIDFromContext`. |
| 7 | `internal/usecase/implementation/invitation/validate_invitation.go:45` | Builds `ctx.WithTenantID(...)` just so the repo can read it. |
| 8 | `internal/infrastructure/persistence/role_repository.go:33-41` (`tenantString`, scope helper) | Repo reads tenant from ctx. **Prohibited.** |
| 9 | `internal/infrastructure/persistence/audit_repository.go:53-57` | Repo reads tenant from ctx via `tenantString`. **Prohibited.** |
| 10 | `internal/delivery/http/middleware/{auth,tenant}.go` `Get*` helpers | Fallbacks to `ctx.*` are fine for middleware; the exported `Get*` helpers should be removed/privatised once handlers stop using them. |

Already compliant (use as the pattern): `UserRepository`, `OrgReferenceRepository`,
`UserInvitationRepository` — `tenantID` is an explicit parameter.

## Target design

### `internal/ctx/ctx.go`
No new concepts; keep `With*` (called by middleware) and `*FromContext` (called by handlers
only). Update the doc comments ("future auth middleware" is stale) to state: *read only in
the delivery layer*. Optionally add a handler-side helper in `handlers/` (not in `ctx`):

```go
// identity.go (package handlers)
func tenantAndUser(c *gin.Context) (tenantID, userID uuid.UUID, ok bool) {
    rc := c.Request.Context()
    t, tok := ctx.TenantIDFromContext(rc)
    u, uok := ctx.UserIDFromContext(rc)
    tid, terr := uuid.Parse(t)
    uid, uerr := uuid.Parse(u)
    if !tok || !uok || terr != nil || uerr != nil || tid == uuid.Nil || uid == uuid.Nil {
        response.Unauthorized(c, "unauthorized")
        return uuid.Nil, uuid.Nil, false
    }
    return tid, uid, true
}
```
(plus a `tenantOnly` variant for endpoints that don't need the user).

### Domain repository interfaces (`internal/domain/repository/`)
Add `tenantID uuid.UUID` as the explicit second parameter to every tenant-scoped method of:

- `RoleRepository` — `GetByID`, `GetByName`, `List`, `Update` (and `Create` via `role.TenantID`
  set by usecase from the passed param). Keep system-level lookups (platform roles) as
  separate, clearly named methods if they are intentionally cross-tenant.
- `AuditLogRepository` — `Create` takes the entity whose `TenantID` is set by the usecase;
  any read methods take `tenantID`.

### Usecase interfaces + implementations
Change `Execute` signatures to take identity explicitly, e.g.:

```go
Invite.Execute(ctx, tenantID, actorID uuid.UUID, req InviteUserRequest)
Resend.Execute(ctx, tenantID, actorID, id uuid.UUID)
Revoke.Execute(ctx, tenantID, actorID, id uuid.UUID)
List.Execute(ctx, tenantID uuid.UUID, limit, offset int)
ListAssignableRoles.Execute(ctx, tenantID uuid.UUID)
```
Delete `tenantFromContext` / `actorFromContext`. Keep `requireAdmin` only if it can work from
roles passed in by the handler (`roles []string`), otherwise rely on the route-level
`RequireRole` middleware (primary check) and drop the in-usecase duplicate.
`validate_invitation.go`: stop wrapping the ctx; pass `inv.TenantID` to the repo directly.

### Repository implementations (`internal/infrastructure/persistence/`)
- `role_repository.go`, `audit_repository.go`: remove `tenantString` and the ctx-based scope
  helper; scope with `Where("tenant_id = ?", tenantID)` using the parameter.
- Guard against `uuid.Nil` at the usecase/handler boundary (not by silently matching no rows).
- Update doc comments that claim "tenant is read from ctx".

### Handlers
- Read identity once with the `ctx` package (see helper above), `401` on failure, pass to usecase.
- Remove `middleware` import from `auth_handler.go`; handlers import `internal/ctx`, not middleware.
- `client.go` (IP / user-agent) already reads from request; align with `ctx.ClientIPFromContext`
  / `ctx.UserAgentFromContext` if it currently reaches into middleware.

### Routes / middleware
- `routes.go` remains the only consumer of `middleware.Auth`, `middleware.Tenant`,
  `RequireRole`, etc.
- Unexport or remove `GetTenantID`, `GetUserID`, `GetRoles`, `GetClaims` from the
  `middleware` package once nothing outside it uses them.

## Implementation steps

1. **Repos + domain interfaces**: add `tenantID` params to `RoleRepository` / `AuditLogRepository`;
   remove ctx reads in `persistence/`; update repo tests (`role_repository_test.go`, etc.).
2. **Usecases**: change interfaces in `usecase/interface/{role,invitation,auth}` and
   implementations; delete ctx helpers; update `mocks_test.go` and usecase tests.
3. **Handlers**: add identity helper, swap `middleware.Get*` for `ctx.*`, pass IDs down;
   update `*_handler_test.go` (set identity with `ctx.With*` on the request context).
4. **Middleware cleanup**: remove exported `Get*` helpers; keep `ctx.With*` injection.
5. **Integration tests**: `integration/invitation_rollback_test.go` and other flows that
   inject identity through ctx must call usecases with explicit params.
6. **Docs**: update the comments in `domain/repository/user_repository.go`,
   `persistence/role_repository.go`, `ctx/ctx.go`; add the rule to
   `plan/architecture/backend.md` and Invariant 1/2 notes in `shared-context.md`.

## Verification

- `grep -rnE "middleware\." internal/delivery/http/handlers internal/usecase internal/infrastructure` → no results (excluding tests of middleware itself).
- `grep -rnE "ctx\.(Tenant|User)IDFromContext|ctx\.RolesFromContext" internal/usecase internal/infrastructure` → no results.
- `grep -rn "ctx.With" internal/usecase internal/infrastructure` → no results.
- `make test` green (unit + integration).
- Tenant isolation tests: a request authenticated for tenant A cannot read/modify tenant B rows
  for roles, invitations, audit logs.
- Unauthenticated / missing-tenant requests to protected routes still return `401`.

## Out of scope

- Changing the JWT claims.
- New endpoints.
- Frontend changes.

> Scope note (as implemented): two items originally listed here were included
> in this change: (1) the auth middleware now verifies identity against the DB
> (tenant + user + current roles via a single-query `IdentityReader`, uniform
> 401, 503 fail-closed), and (2) migration `000015_add_audit_columns` adds
> audit and soft-delete columns (`created_by/updated_by/deleted_by/deleted_at`),
> with soft-delete read filtering in the affected repositories.
