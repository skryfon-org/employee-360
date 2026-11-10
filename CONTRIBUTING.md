# Contributing to Employee360

Thanks for helping. This guide covers local setup, how changes flow into `main`, and
how pull requests are reviewed. Everyone who takes part agrees to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

- **Bugs and ideas**: open an [issue](https://github.com/skryfon-org/employee-360/issues/new/choose).
- **Security issues**: never open a public issue. See [SECURITY.md](SECURITY.md).
- **Bigger changes** (a new module, a schema change, a new dependency): open an
  issue first so we can agree on scope before you write code.

## Development setup

### Prerequisites

- **Go** 1.26+ (see `backend/go.mod`)
- **Node.js** 24 and **pnpm** 11 (`corepack enable` picks up the version in `package.json`)
- **Docker** with Compose (or Podman) for Postgres, Redis and Mailpit
- Optional: [`air`](https://github.com/air-verse/air) for API hot reload, and
  `golangci-lint`

### First run

```bash
git clone https://github.com/<you>/employee-360.git && cd employee-360
cp .env.example .env                    # then set BOOTSTRAP_SUPER_ADMIN_PASSWORD
cp clients/admin/.env.example clients/admin/.env
cp clients/employee/.env.example clients/employee/.env

docker compose up -d postgres mailpit   # add `redis` only if you set REDIS_ENABLED=true
make bootstrap                          # pnpm install + go mod download
make migrate                            # apply DB migrations
make bootstrap-admin                    # system tenant, roles, platform super admin
make dev                                # API :8080, worker, admin :5173, employee :5174
```

- Admin portal: http://localhost:5173. Log in as `BOOTSTRAP_SUPER_ADMIN_EMAIL`.
- Employee portal: http://localhost:5174
- API docs (Swagger, dev only): http://localhost:8080/swagger/index.html
- Outgoing email (invitations, OTP, password reset): Mailpit at http://localhost:8025

Run `make help` to list every target. The ones you'll use most:

| Command | What it does |
|---|---|
| `make test-backend` | Go unit tests (no DB) |
| `make test-backend-integration` | Go integration tests (needs Postgres; creates `<DATABASE_NAME>_test`) |
| `make check-backend` | build, vet, gofmt, structure, unit and integration tests (what CI runs) |
| `make check-clients` | lint, typecheck, test and build for every workspace package (what CI runs) |
| `make migrate-down` | roll back the last migration |
| `make generate` | regenerate Swagger and the typed `packages/api-client` after API changes |

## Project rules

The architecture rules are in [`shared-context.md`](shared-context.md). Read
**Core Architectural Invariants** before your first backend PR. In short:

- **Tenant isolation.** Every tenant table has `tenant_id`. The tenant comes only from
  the authenticated context, never from the request body. Handlers pass `tenantID` to
  usecases and repositories, and every query is scoped by it.
- **Clean Architecture layers.** `domain` → `usecase` → `delivery/http` →
  `infrastructure`. Handlers never touch GORM, and the domain has no framework imports.
- **Frontend.** Server state lives in TanStack Query, UI state in Zustand, and forms
  use React Hook Form + Zod. Each feature is a self-contained folder under
  `clients/*/src/features/`.
- **Migrations.** They live in `backend/migrations/` as golang-migrate up/down pairs.
  Never edit a migration that has been merged. Add a new one instead.

Work is planned in **cycles**: narrow, full-stack slices of a module, each scoped in
`plan/cycles/cycle-NN-<name>.md`. Before starting a feature, check the cycle file
that covers it. For a new module, propose a cycle file in your PR or issue. The
architecture docs are in [`plan/architecture/`](plan/architecture/) (see also
[docs/architecture.md](docs/architecture.md)).

If you use AI coding tools, `CLAUDE.md`, `AGENTS.md`, `.claude/` and `.agents/` hold
agent instructions, skills and hooks that encode these rules.

## Branches and pull requests

1. Fork the repo and create a branch from `main` (`feat/leave-carry-over`,
   `fix/otp-expiry`, ...).
2. Keep each PR to one concern. Include tests, and update docs if behaviour changes.
3. Open the PR against `main` and fill in the template.

`main` is always deployable: every merge to it deploys Skryfon's staging environment,
and `v*` tags cut releases.

### Commit style

Write short imperative subjects, prefixed with the area, as in the existing history:

```
Leave: send email notifications for requested, approved and rejected
Invitations: allow resend for expired invitations
```

Explain the *why* in the body when it isn't obvious.

### Developer Certificate of Origin

Every commit must be signed off, which certifies the
[Developer Certificate of Origin](https://developercertificate.org/): you wrote the
change, or otherwise have the right to submit it under the project's license
(AGPL-3.0).

```bash
git commit -s -m "Holidays: add ICS export"
# fix up the commits already on a branch:
git rebase --signoff main
```

The DCO GitHub App checks every PR. We don't ask for a CLA.

## Review procedure

- **Approvals.** Every PR needs **1 maintainer approval**. PRs that touch **auth,
  tenancy, migrations or `.github/`** need **2**. CODEOWNERS requests the right
  reviewers automatically.
- **CI.** All required checks must be green: Backend CI, Frontend CI, Security and DCO.
- **Checklist.** Reviewers work through the checklist in the
  [PR template](.github/pull_request_template.md). Tenant scoping and layer
  boundaries are always checked.
- **Merging.** PRs are **squash-merged**, so the PR title becomes the commit subject.
  Make it a good one.
- **First-time contributors.** A maintainer approves the CI run before it starts.
  That's a routine GitHub safeguard, nothing personal.

## License

By contributing, you agree that your contributions are licensed under the
[GNU AGPL-3.0](LICENSE).
