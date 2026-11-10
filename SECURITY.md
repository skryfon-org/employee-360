# Security Policy

## Reporting a vulnerability

**Please don't report security vulnerabilities in public issues, discussions or pull
requests.**

Report them privately in one of these ways:

1. **GitHub Private Vulnerability Reporting** (preferred): go to
   [Security → Report a vulnerability](https://github.com/skryfon-org/employee-360/security/advisories/new).
2. **Email**: security@skryfon.com

Please include:

- the affected version or commit, and the component (API, admin portal, employee portal, deployment files)
- steps to reproduce, or a proof of concept
- the impact you think it has, for example a cross-tenant data read, privilege escalation or auth bypass

## What to expect

| Step | Target |
|---|---|
| We acknowledge your report | within **3 business days** |
| We triage it and tell you our assessment | within **10 business days** |
| We release a fix | within **90 days** of the report, sooner for critical issues |

We'll keep you updated while we work on it, agree a disclosure date with you, and
credit you in the advisory unless you'd rather we didn't. Please give us a reasonable
chance to ship a fix before you disclose anything publicly.

## Supported versions

Employee360 is pre-1.0. We fix security issues only in the **latest release** and on
`main`.

| Version | Supported |
|---|---|
| latest `v0.x` release | ✅ |
| older releases | ❌ (please upgrade) |

## Scope

In scope: this repository's code, its container images, and the deployment files in
`infra/`. Tenant isolation is a core guarantee. Any way for one tenant to read or
change another tenant's data is treated as **critical**.

Out of scope: vulnerabilities that need a compromised host or a `super_admin` account,
denial of service through volume alone, and findings from automated scanners with no
demonstrated impact.

## Hardening notes for operators

- `super_admin` is a role for platform operators only. It can register tenant domains
  without proof of ownership, and domains route logins. Never grant it to customers
  or tenant staff.
- Set `APP_ENV=production` (or `staging`). Anything other than `development`/`test`
  rejects the default JWT secret and disables Swagger.
- Behind a reverse proxy, set `SERVER_TRUSTED_PROXIES`. Otherwise every user shares one
  rate-limit bucket. See [docs/self-hosting.md](docs/self-hosting.md#client-ip-and-server_trusted_proxies).
- The root `docker-compose.yml` is for development only. Its default credentials
  must never be used in production.
