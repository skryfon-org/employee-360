# Employee360

[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Contributing](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Security policy](https://img.shields.io/badge/security-policy-red.svg)](SECURITY.md)

**An open-source, self-hostable, multi-tenant employee platform.** Holidays, leave,
work status, projects, employee profiles and onboarding are in one place, and
policies, benefits, career growth, compensation and appraisals are on the roadmap.

- **Self-hostable.** Your data stays on your infrastructure: Docker Compose or rootless
  Podman, with no cloud lock-in.
- **Multi-tenant.** One deployment serves many organisations. Every row is scoped by
  `tenant_id` on the server, and the tenant is never taken from client input.
- **Headless.** A versioned `/api/v1` REST API with an OpenAPI spec and JWT auth. The
  admin and employee portals are just two clients of it.

> **Status: early (v0.x).** Usable, but the APIs and the schema may still change
> between minor versions. See the [CHANGELOG](CHANGELOG.md).

## Features

| Module | Status | What's there |
|---|---|---|
| Auth & onboarding | ✅ Shipped | Admin email+password login, employee OTP login, forgot/reset password, admin invitations (with resend), async email outbox |
| Organisation | ✅ Shipped | Tenants and domains, departments, positions, roles, organisation settings (employee code format) |
| Holiday calendar | ✅ Shipped | Holiday categories, year-wise calendars, calendar and list views |
| Employee profiles | 🚧 In progress | Admin-created employees, self-completed profiles, emergency contacts |
| Leave management | 🚧 In progress | Leave types, requests, approver routing, email notifications |
| Work status | 🚧 In progress | Daily status logs with leave, holidays and weekly offs |
| Projects | 🚧 In progress | Projects, assignments and allocations |
| Dashboards & audit log | 🚧 In progress | Employee home dashboard, admin audit log |
| Courses, benefits, career growth, salary & tax, appraisal, company policies | 📋 Planned | |

## Quick start

You need Go 1.26+, Node 24 with pnpm 11 (`corepack enable`), and Docker.

```bash
git clone https://github.com/skryfon-org/employee-360.git && cd employee-360
cp .env.example .env            # set BOOTSTRAP_SUPER_ADMIN_PASSWORD to a real password
cp clients/admin/.env.example clients/admin/.env
cp clients/employee/.env.example clients/employee/.env

docker compose up -d postgres mailpit
make bootstrap                  # install pnpm + Go dependencies
make migrate                    # create the schema
make bootstrap-admin            # system tenant, roles and your super admin
make dev                        # API, email worker and both portals
```

| URL | What |
|---|---|
| http://localhost:5173 | Admin portal. Log in with `BOOTSTRAP_SUPER_ADMIN_EMAIL` / `_PASSWORD`. |
| http://localhost:5174 | Employee portal |
| http://localhost:8080/swagger/index.html | API docs (development only) |
| http://localhost:8025 | Mailpit: catches invitation, OTP and reset emails |

Next, read [CONTRIBUTING.md](CONTRIBUTING.md) for the full development workflow.

## Self-hosting

See **[docs/self-hosting.md](docs/self-hosting.md)**. It covers the container images,
Docker Compose versus rootless Podman with systemd Quadlets behind nginx, TLS, the
first super-admin bootstrap, backups, and running the GitHub Actions deploy in your
own fork. Every setting is in **[docs/configuration.md](docs/configuration.md)**.

## Tech stack

| | |
|---|---|
| **Backend** | Go · Gin · GORM + PostgreSQL · golang-migrate · River (job queue) · JWT · Viper · Zerolog · Redis (optional) · Clean Architecture |
| **Frontend** | React + TypeScript + Vite · Tailwind CSS · TanStack Query · Zustand · React Hook Form + Zod · Orval-generated API client · pnpm workspaces |

Architecture: **[docs/architecture.md](docs/architecture.md)**.

## Repository layout

```
backend/          Go API, worker, migrate and bootstrap commands; migrations/
clients/admin     Admin portal (React)
clients/employee  Employee portal (React)
packages/         Shared api-client (generated from the OpenAPI spec) and ui
infra/            Containerfiles, Podman quadlets, nginx config, host bootstrap
plan/             Architecture docs and per-cycle scope (plan/cycles/)
docs/             Self-hosting, configuration and architecture guides
```

## Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) for setup, project
rules, the DCO sign-off and the review procedure. Report vulnerabilities privately as
described in [SECURITY.md](SECURITY.md). This project follows the
[Contributor Covenant](CODE_OF_CONDUCT.md).

## License

[GNU Affero General Public License v3.0](LICENSE). If you run a modified version as a
network service, you must offer its source code to that service's users.

---

Built by [Skryfon](https://skryfon.com).
