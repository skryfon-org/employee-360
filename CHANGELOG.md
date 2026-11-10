# Changelog

Notable changes to this project are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/). Before 1.0, a minor version may include
breaking changes.

## [Unreleased]

## [0.1.0]

The first public release.

### Added

- Multi-tenant Go API (`/api/v1`) with JWT auth. Every request is checked against the
  database: tenant and user must be active, and roles are reloaded each time.
- Admin email+password login, employee OTP login, forgot/reset password, and
  per-account and per-IP auth throttling.
- Admin onboarding invitations (create, resend, accept), with role-aware links to the
  admin or employee portal.
- An asynchronous transactional email outbox, backed by River and processed by
  `cmd/worker`.
- Tenants and domains, departments, positions, roles and organisation settings.
- Holiday calendar: categories, year-wise holidays, and calendar and list views.
- Early versions of employee profiles, leave management, daily status logs,
  projects and allocations, the employee dashboard and the admin audit log.
- Admin and employee React portals, and an Orval-generated typed API client.
- Container images, Podman Quadlet units, nginx config and GitHub Actions
  release/deploy/rollback workflows.
- Open-source project files: AGPL-3.0 license, contributing guide, security policy,
  code of conduct, and issue and PR templates.

[Unreleased]: https://github.com/skryfon-org/employee-360/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/skryfon-org/employee-360/releases/tag/v0.1.0
