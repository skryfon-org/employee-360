# Self-hosting Employee360

Employee360 runs as four containers: `api`, `worker` (River email outbox), `admin`
SPA and `employee` SPA. They need PostgreSQL 16+ and, optionally, Redis. There are
two supported ways to run it:

- **Docker Compose**: the quickest way to try it on one machine.
- **Rootless Podman + systemd Quadlets** behind host nginx: the production layout
  used by this repo's own deploy workflows.

All configuration is environment variables. See [configuration.md](configuration.md)
for the full reference.

## Images

Release images are published to GHCR for each `v*` tag:

```
ghcr.io/skryfon-org/employee360-api:<version>        # api, worker, migrate, bootstrap binaries + /migrations
ghcr.io/skryfon-org/employee360-admin:<version>      # admin SPA on nginx-alpine
ghcr.io/skryfon-org/employee360-employee:<version>   # employee SPA on nginx-alpine
```

The SPA images have `VITE_API_BASE_URL` (and `VITE_EMPLOYEE_APP_URL` for admin)
**baked in at build time**, so the published SPA images only work with the URLs they
were built for. Build your own for your domain:

```bash
make images IMAGE_TAG=v0.1.0 \
  VITE_API_BASE_URL=https://api.example.com \
  VITE_EMPLOYEE_APP_URL=https://app.example.com
```

(`make images` uses Podman. With Docker, run the same `build` commands from the
`Makefile` with `docker` instead.)

## Option A: Docker Compose (evaluation)

The root `docker-compose.yml` starts Postgres, Redis, Mailpit and the worker for local
development. Follow the [README quick start](../README.md#quick-start).

**Do not use it as-is in production.** It ships dev credentials (`postgres/postgres`,
Redis password `redis`) and exposes Mailpit on all interfaces. For a production compose
setup, run the images above with your own secrets and a real SMTP provider.

## Option B: Podman Quadlets behind nginx (production)

```
Internet ─► nginx (host, :80/:443)
              ├─ api.example.com    → 127.0.0.1:8080  employee360-api
              ├─ admin.example.com  → 127.0.0.1:3003  employee360-admin
              └─ app.example.com    → 127.0.0.1:3004  employee360-employee

Rootless Podman, user `deploy`, network `employee360`
  employee360-api · employee360-worker · employee360-postgres
  employee360-redis (only when REDIS_ENABLED=true)
```

All containers publish on loopback only, so nginx is the single entry point. Postgres
runs as a container with the named volume `employee360-postgres-data`. To use a managed
database instead, set `DATABASE_URL` in `api.env`; the deploy then skips the bundled
Postgres container.

The hostnames are a convention. They must match in DNS, the nginx `server_name` lines,
the certbot `-d` arguments and the URL variables below. The ports are fixed in the
quadlet files, so run one environment per host.

### Files

| Path | Purpose |
|---|---|
| `infra/Containerfile.api` | api + worker + migrate + bootstrap binaries and `/migrations` |
| `infra/Containerfile.web` | admin / employee SPA (`--build-arg APP=`) on nginx-alpine |
| `infra/server/quadlets/` | systemd units, copied to `~deploy/.config/containers/systemd/` |
| `infra/server/nginx/employee360.conf` | host reverse proxy (replace the `example.com` hostnames) |
| `infra/server/bootstrap.sh` | one-time host setup (Ubuntu 24.04+ / Amazon Linux 2023) |

### One-time host setup

1. A Linux host with at least 2 vCPU and 4 GB RAM. Open only 22 (your IP), 80 and
   443. Do not expose 5432, 6379, 8080, 3003 or 3004.
2. `sudo bash infra/server/bootstrap.sh`. It creates the `deploy` user, enables
   linger and installs Podman (4.4+ is needed for Quadlet) and nginx.
3. Install `infra/server/nginx/employee360.conf` as
   `/etc/nginx/conf.d/employee360.conf` with your hostnames, then
   `nginx -t && systemctl reload nginx`.
4. TLS: `certbot --nginx -d api.example.com -d admin.example.com -d app.example.com`,
   or terminate TLS at a load balancer / CDN in front of the host.
5. Write the env files to `/home/deploy/.config/employee360/` (mode `0600`):
   `api.env` (below), `postgres.env` (`POSTGRES_USER`, `POSTGRES_PASSWORD`,
   `POSTGRES_DB`) and, if Redis is on, `redis.env` (`REDIS_PASSWORD=...`).

### `api.env` (production)

```
APP_ENV=production
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
DATABASE_HOST=employee360-postgres
DATABASE_PORT=5432
DATABASE_USER=<same as POSTGRES_USER>
DATABASE_PASSWORD=<same as POSTGRES_PASSWORD>
DATABASE_NAME=<same as POSTGRES_DB>
DATABASE_SSLMODE=disable
JWT_SECRET=<openssl rand -hex 32>
APP_FRONTEND_URL=https://admin.example.com
APP_ADMIN_URL=https://admin.example.com
APP_EMPLOYEE_URL=https://app.example.com
CORS_ALLOWED_ORIGINS=https://admin.example.com,https://app.example.com
SMTP_HOST=<your SMTP host>
SMTP_PORT=465
SMTP_USERNAME=<user>
SMTP_PASSWORD=<password or API key>
SMTP_FROM=Employee360 <no-reply@example.com>
SMTP_USE_TLS=true
REDIS_ENABLED=false
SERVER_TRUSTED_PROXIES=<CIDR of the employee360 podman network, see below>
```

With `APP_ENV=production` the API refuses to start with the default JWT secret, or
with Redis enabled and no password.

### Deploy and upgrade

As `deploy` on the host, for a release tag `TAG` (for example `v0.1.0`):

```bash
IMG=ghcr.io/skryfon-org/employee360   # or your own registry
for n in api admin employee; do
  podman pull "$IMG-$n:$TAG"
  podman image exists localhost/employee360-$n:current && \
    podman tag localhost/employee360-$n:current localhost/employee360-$n:previous
  podman tag "$IMG-$n:$TAG" localhost/employee360-$n:current
done
systemctl --user daemon-reload
systemctl --user start employee360-network employee360-postgres

# migrations (one-shot)
podman run --rm --network employee360 --entrypoint /usr/local/bin/migrate \
  --env-file ~/.config/employee360/api.env localhost/employee360-api:current -dir /migrations up

systemctl --user restart employee360-api employee360-worker employee360-admin employee360-employee
curl -fsS http://127.0.0.1:8080/health
```

`.github/workflows/deploy.yml` automates exactly these steps over SSH, with a smoke
test. **Rollback** re-tags `:previous` as `:current` and restarts the units.
Migrations are not reversed.

### Bootstrap the platform super admin (first deploy only)

Create the system tenant, roles and the platform super admin. Pass the credentials
through a `0600` env file rather than `-e`, so the password never shows in the process
list:

```bash
umask 077
printf 'BOOTSTRAP_SUPER_ADMIN_EMAIL=%s\nBOOTSTRAP_SUPER_ADMIN_PASSWORD=%s\n' you@example.com '<strong password>' > /tmp/boot.env
podman run --rm --network employee360 \
  --env-file ~/.config/employee360/api.env --env-file /tmp/boot.env \
  --entrypoint /usr/local/bin/bootstrap localhost/employee360-api:current
rm /tmp/boot.env
```

It is idempotent. The command refuses the `.env.example` placeholder password.

### Client IP and `SERVER_TRUSTED_PROXIES`

The API rate-limits per client IP (a global limiter, plus a throttle on login and
password reset). nginx sets `X-Real-IP` from `$remote_addr`, but the API honours that
header **only when the direct TCP peer is listed in `SERVER_TRUSTED_PROXIES`**
(comma-separated IPs/CIDRs). If it is unset or doesn't match, every user shares the
proxy's address, and one client can get everyone a 429.

1. Find the address nginx appears as: run `podman network inspect employee360` (subnet
   and gateway), then check `client_ip` in `podman logs employee360-api` for a request
   you made. With rootless Podman it is usually the network gateway (e.g. `10.89.0.1`)
   or an address inside the subnet.
2. Set it to the network's CIDR in `api.env`, e.g. `SERVER_TRUSTED_PROXIES=10.89.0.0/24`,
   then restart `employee360-api`.
3. Verify: requests from two different client hosts must log different `client_ip`
   values. While it is misconfigured, the API logs a warning (at most once every 10
   minutes) that mentions `SERVER_TRUSTED_PROXIES`.

List only your own reverse proxy: any address you list can set the client IP.

### Roles

`super_admin` is a role for platform operators only. Never grant it to customers or
tenant staff. It can register tenant domains without ownership verification, and
domains route logins. If you ever offer self-service tenant creation, add DNS-TXT
ownership verification first.

### Operations

```bash
systemctl --user status employee360-api          # as deploy
journalctl --user -u employee360-worker -f
podman logs --tail 100 employee360-api
```

**Backups.** The Postgres volume lives only on this host. Run a cron job as `deploy`
and copy the dumps off the host:

```bash
podman exec employee360-postgres pg_dump -U <user> <db> | gzip > ~/backups/e360-$(date +%F).sql.gz
```

## Using the GitHub Actions deploy in your own fork

`release-images.yml`, `deploy.yml`, `rollback.yml` and `bootstrap-environment.yml` only
run in `skryfon-org/employee-360`. To use them in your fork, change the
`github.repository` guard in each, then create the `staging` and `production` GitHub
Environments with:

| Kind | Name | Value |
|---|---|---|
| secret | `SSH_HOST` | host DNS name / IP |
| secret | `SSH_PRIVATE_KEY` | key in `/home/deploy/.ssh/authorized_keys` |
| secret | `API_ENV_FILE` | contents of `api.env` |
| secret | `POSTGRES_ENV_FILE` | contents of `postgres.env` |
| secret | `REDIS_ENV_FILE` | `REDIS_PASSWORD=...` (empty if Redis is off) |
| secret | `BOOTSTRAP_SUPER_ADMIN_PASSWORD` | super-admin password |
| variable | `BOOTSTRAP_SUPER_ADMIN_EMAIL` | super-admin email (optional; deploy bootstraps when set) |
| variable | `API_URL` | `https://api.example.com` (baked into the SPAs) |
| variable | `EMPLOYEE_APP_URL` | `https://app.example.com` |

A push to `main` releases `staging-<sha>` images and deploys staging. A `v*` tag
releases `production-<sha>` plus `<tag>` images and deploys production.
