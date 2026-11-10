# Architecture

Employee360 is a headless Go API (`/api/v1`, JWT auth), plus two React SPAs (the
admin portal and the employee portal) that consume it through a typed client
generated from the API's OpenAPI spec.

```
clients/admin ─┐                         ┌─ PostgreSQL (all tenant data, River job queue)
clients/employee ─┼─ packages/api-client ─► backend/cmd/api ─┤
               │   (Axios + TanStack Query) │                   └─ Redis (optional: counters/cache)
packages/ui ───┘                         backend/cmd/worker ── SMTP (transactional email outbox)
```

## Where to read more

| Doc | What's in it |
|---|---|
| [`plan/architecture/overview.md`](../plan/architecture/overview.md) | Start here: Mermaid diagrams of the system context and the request lifecycle |
| [`plan/architecture/backend.md`](../plan/architecture/backend.md) | Backend Clean Architecture: layers, directory tree, DI, migrations |
| [`plan/architecture/frontend.md`](../plan/architecture/frontend.md) | pnpm monorepo, feature folders, state management, API client |
| [`plan/architecture/diagrams/`](../plan/architecture/diagrams/) | Detailed HTML/SVG diagrams (open them in a browser) |
| [`shared-context.md`](../shared-context.md) | The invariants every change must keep: tenant isolation, layer boundaries, roles |
| [`plan/cycles/`](../plan/cycles/) | Scope and status for each delivery cycle |

When the docs and the code disagree, trust the code. Then open a PR to fix the docs.
