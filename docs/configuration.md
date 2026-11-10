# Configuration reference

The backend (`api`, `worker`, `migrate`, `bootstrap`) reads configuration in this
order, and later sources win: built-in defaults → `.env` (current directory or a
parent) → `config.yaml` (see `backend/config/config.yaml.example`) → environment
variables. The source of truth is `backend/config/config.go`. Copy `.env.example`
to `.env` for local development.

## App

| Variable | Default | Notes |
|---|---|---|
| `APP_ENV` | *(unset)* | **Required.** `development` and `test` relax the secret checks. Anything else, including unset, is treated as production: the default `JWT_SECRET` is rejected, and Swagger (`/swagger`) is off. |
| `LOG_LEVEL` | `debug` | zerolog level |
| `APP_FRONTEND_URL` | `http://localhost:5173` | Base URL for email links (e.g. password reset). Must be set explicitly in `production`/`staging`. |
| `APP_ADMIN_URL` | falls back to `APP_FRONTEND_URL` | Invitation links for admin roles |
| `APP_EMPLOYEE_URL` | falls back to `APP_FRONTEND_URL` | Invitation links for employees |

## Server

| Variable | Default | Notes |
|---|---|---|
| `SERVER_HOST` / `SERVER_PORT` | `0.0.0.0` / `8080` | |
| `SERVER_READ_TIMEOUT`, `SERVER_WRITE_TIMEOUT`, `SERVER_SHUTDOWN_TIMEOUT` | `10s` each | |
| `SERVER_TRUSTED_PROXIES` | empty | Comma-separated IPs/CIDRs of your reverse proxy. Needed behind nginx so that per-IP rate limits work; see [self-hosting.md](self-hosting.md#client-ip-and-server_trusted_proxies). |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated. Set it to your admin and employee origins in production. |

## Database

| Variable | Default | Notes |
|---|---|---|
| `DATABASE_URL` | empty | `postgres://` URL. When set it **overrides** the fields below. URL-encode the password. Without `?sslmode=` it uses `prefer`. |
| `DATABASE_HOST` / `DATABASE_PORT` | `localhost` / `5432` | |
| `DATABASE_USER` / `DATABASE_PASSWORD` | `postgres` / `postgres` | |
| `DATABASE_NAME` | `employee360` | Integration tests use `<DATABASE_NAME>_test`. |
| `DATABASE_SSLMODE` | `disable` | |
| `DATABASE_MAX_OPEN_CONNS`, `DATABASE_MAX_IDLE_CONNS` | `25`, `5` | |
| `DATABASE_CONN_MAX_LIFETIME`, `DATABASE_CONN_TIMEOUT` | `15m`, `5s` | |

## Auth

| Variable | Default | Notes |
|---|---|---|
| `JWT_SECRET` | dev placeholder | Use `openssl rand -hex 32`. At least 32 characters, and must be set outside `development`/`test`. |
| `JWT_ACCESS_EXPIRY` / `JWT_REFRESH_EXPIRY` | `15m` / `168h` | |

## Rate limiting

| Variable | Default | Notes |
|---|---|---|
| `RATE_LIMIT_ENABLED` | `true` | Global, per client IP. Health probes are exempt. |
| `RATE_LIMIT_REQUESTS_PER_SECOND`, `RATE_LIMIT_BURST` | `10`, `20` | |
| `RATE_LIMIT_CLEANUP_INTERVAL`, `RATE_LIMIT_IDLE_TTL` | `1m`, `10m` | |
| `AUTH_RATE_LIMIT_ENABLED` | `true` | Strict throttle on login, forgot/reset password and invitation accept/validate |
| `AUTH_RATE_LIMIT_PER_ACCOUNT_MAX`, `AUTH_RATE_LIMIT_PER_IP_MAX` | `5`, `30` | Attempts per window |
| `AUTH_RATE_LIMIT_WINDOW` | `1m` | Fixed window. Uses Redis counters when Redis is enabled, otherwise memory (per instance). |

## Email (SMTP)

Email is sent asynchronously by the `worker` from a River outbox table, so the API
never blocks on SMTP.

| Variable | Default | Notes |
|---|---|---|
| `SMTP_HOST` / `SMTP_PORT` | `localhost` / `1025` | The defaults point at the dev Mailpit (`docker compose up -d mailpit`, inbox at http://localhost:8025). |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | empty | |
| `SMTP_FROM` | `no-reply@employee360.local` | e.g. `Employee360 <no-reply@example.com>` |
| `SMTP_USE_TLS` | `false` | `true` for implicit TLS (port 465) |

## Bootstrap (`make bootstrap-admin` / `cmd/bootstrap`)

| Variable | Default | Notes |
|---|---|---|
| `BOOTSTRAP_SYSTEM_TENANT_NAME` | `System` | |
| `BOOTSTRAP_SUPER_ADMIN_EMAIL` | none | Required |
| `BOOTSTRAP_SUPER_ADMIN_PASSWORD` | none | Required, at most 72 bytes. The `.env.example` placeholder `<CHANGE_ME>` is rejected. Remove it from `.env` after the first run. |

## Redis (optional)

Redis is an optional dependency. Today it backs the auth throttle counters; nothing
else requires it.

| Variable | Default | Notes |
|---|---|---|
| `REDIS_ENABLED` | `false` | Master switch |
| `REDIS_HOST` / `REDIS_PORT` | `localhost` / `6379` | Ignored when `REDIS_URL` is set |
| `REDIS_PASSWORD` | (compose default `redis`) | Required in `production`/`staging` when enabled |
| `REDIS_DB`, `REDIS_TLS` | `0`, `false` | |
| `REDIS_URL` | empty | `redis://` / `rediss://` URL. Overrides the fields above. |
| `REDIS_DIAL_TIMEOUT`, `REDIS_READ_TIMEOUT`, `REDIS_WRITE_TIMEOUT`, `REDIS_POOL_SIZE` | `5s`, `3s`, `3s`, `10` | |

Start it with `docker compose up -d redis`. Data persists in the
`employee360_redis_data` volume. Keys are namespaced
`e360:{tenant_id}:<feature>:<key>` (or `e360:global:...` before auth) via
`service.CacheKey`.

Failure modes:

- **Disabled**: a no-op cache is used, and `/api/v1/health` reports `redis: "disabled"`.
- **Enabled but unreachable at startup**: the API exits (fail fast).
- **Down at runtime**: cache calls return errors and health reports
  `redis: "unreachable"` (the HTTP status stays 200). Rate limiting falls back to the
  in-memory limiter, and identity caching fails closed.

## Frontend

The SPAs are configured at **build time** (Vite inlines `VITE_*` variables). See
`clients/admin/.env.example` and `clients/employee/.env.example`.

| Variable | App | Notes |
|---|---|---|
| `VITE_API_BASE_URL` | both | API origin, including `http(s)://` |
| `VITE_EMPLOYEE_APP_URL` | admin | Employee portal origin |
| `VITE_ADMIN_APP_URL` | employee | Admin portal origin |
| `ADMIN_PORT` / `EMPLOYEE_PORT` | dev server | `5173` / `5174`. Read from the shell environment, not from `.env`. |
