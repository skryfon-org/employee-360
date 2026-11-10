## Summary

<!-- What does this change and why? -->

Closes #

## How was this tested?

<!-- Commands you ran, new tests, manual steps. -->

## Checklist

- [ ] Commits are signed off (`git commit -s`, see [DCO](https://github.com/skryfon-org/employee-360/blob/main/CONTRIBUTING.md#developer-certificate-of-origin))
- [ ] `make check-backend` and/or `make check-clients` pass locally
- [ ] Tests added or updated for the change
- [ ] **Tenant data:** this touches tenant data, and every query and mutation is scoped by the explicit `tenantID` parameter (never a client-supplied tenant ID). Leave it unchecked if no tenant data is touched.
- [ ] **Layers:** handlers don't call GORM; domain imports no frameworks ([shared-context.md](https://github.com/skryfon-org/employee-360/blob/main/shared-context.md#core-architectural-invariants))
- [ ] **Migrations:** new up/down pair, both applied and reverted locally (`make migrate` / `make migrate-down`). Committed migrations are not edited.
- [ ] API changed → `make generate` run and the regenerated Swagger / `packages/api-client` committed
- [ ] Docs updated (README, `docs/`, active cycle file in `plan/cycles/`) where behaviour changed

## Reviewer checklist

- [ ] Scope matches the linked issue / active cycle; no unrelated changes
- [ ] Authorization: role checks on new routes; no identity read outside handlers
- [ ] Tenant isolation verified (see above), including list/count queries
- [ ] Errors don't leak internals; inputs validated at the handler boundary
- [ ] No secrets, real hostnames or personal data in code, fixtures or logs
- [ ] Auth, tenancy, migrations or `.github/` touched → second maintainer approval
