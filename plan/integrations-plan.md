# Plan: Universal Messaging Integration System (Zulip first, then Telegram/WhatsApp/Slack/…)

## Context
Employee360 is open-source and self-hostable, so each organisation will want its own chat
tool (Zulip for us; others want Telegram, WhatsApp, Slack, Teams, Discord, Matrix, generic
webhook). Instead of hard-coding Zulip, build one **provider-agnostic integration layer**
where a channel is a plug-in adapter. Adding a new messenger = one adapter file + register
call; no changes to usecases, events, or frontend core.

Fits existing Cycle 2 design (verified in code): usecases publish domain events inside a
transaction → `RiverPublisher` inserts a River job (outbox) → `cmd/worker` drains it. Today
only `send_email` exists, with one global env-configured SMTP and no per-tenant config or
encrypted secrets. This plan reuses that pipeline and adds the missing pieces.

## Core design

### 1. Domain (`backend/internal/domain/`)
- `service/messenger.go` – the single plug-in port:
  ```go
  type Messenger interface {
      Kind() string                                   // "zulip", "telegram", ...
      ConfigSchema() []ConfigField                    // drives admin UI form (name, type, secret?, required)
      Validate(cfg map[string]any) error
      Send(ctx, cfg ProviderConfig, msg OutboundMessage) (DeliveryResult, error)
      Test(ctx, cfg ProviderConfig) error             // "send test message" button
  }
  type OutboundMessage struct { Recipient Recipient; Title, Body string; Link string; EventType string }
  ```
  Errors typed as `ErrPermanent` (bad config, user blocked bot) vs transient (retry).
- `entity/integration.go`:
  - `Integration` {id, tenant_id, kind, name, config (secrets encrypted), enabled}
  - `IntegrationRoute` {tenant_id, integration_id, event_type, target (channel/stream/topic or "user DM"), template_key}
  - `UserChannelIdentity` {tenant_id, user_id, integration_id, external_id (zulip email / telegram chat_id / phone), verified_at}
  - `MessageDelivery` {event_id, integration_id, status, attempts, last_error} – audit + idempotency
- Repository ports for each, all with explicit `tenantID` params (Invariant 1).
- Reuse `domain/event/events.go` events as-is (UserInvited, PasswordResetRequested, …); new modules just add events.

### 2. Registry (extensibility seam)
- `infrastructure/messaging/registry.go`: `Register(Messenger)` / `Get(kind)`; each adapter
  self-registers in its package `init()` or in the DI container. Adding a provider = new
  package `infrastructure/messaging/<kind>/` implementing `Messenger` + a test.
- Adapters shipped: `zulip` (REST `/api/v1/messages`, bot email + API key + site URL; stream/topic and DM),
  `webhook` (generic signed HTTP POST – covers Slack/Teams/Discord/Mattermost incoming webhooks and anything unlisted),
  then `telegram` (Bot API), later `whatsapp` (Cloud API), `matrix`, `slack`.
  Email stays as-is but is later wrapped as a `Messenger` too so it is just another channel.

### 3. Delivery pipeline (reuse River outbox)
- `eventing/dispatcher.go`: for each domain event, look up enabled `IntegrationRoute`s for
  (tenant, event_type) and enqueue one `send_message` River job per route (job args:
  tenant_id, integration_id, event payload ref). Email path unchanged.
- `job/send_message_worker.go`: load integration → decrypt config → render template →
  `Messenger.Send`. Permanent error ⇒ `river.JobCancel` + mark delivery failed; transient ⇒ River backoff.
  Idempotency via `MessageDelivery(event_id, integration_id)` unique key.
- Message templates: Go templates per event type with channel-neutral fields
  (title/body/link); each adapter formats to its markup (Zulip markdown, Telegram HTML, plain for webhook).
- Avoid putting plaintext secrets/tokens in job args (completed jobs are retained 1h); pass ids, load at run time.

### 4. Secrets & multi-tenancy
- New `infrastructure/crypto/secretbox.go`: AES-256-GCM, key from env `INTEGRATION_ENCRYPTION_KEY` (validated in `config.go` like the JWT secret). Secret fields (per `ConfigSchema`) encrypted at rest, never returned by API (masked `••••`).
- Every table has `tenant_id`; tenant resolved from auth ctx only. Per-tenant integrations (unlike global SMTP).
- Outbound SSRF guard for `webhook`/self-hosted Zulip URLs: block private/link-local ranges unless `INTEGRATION_ALLOW_PRIVATE_HOSTS=true` (self-hosters often need it).

### 5. API (`/api/v1`, admin role) – via `new-backend-feature` skill
- `GET /integrations/providers` (kinds + ConfigSchema), `POST/GET/PATCH/DELETE /integrations`,
  `POST /integrations/:id/test`, CRUD `/integrations/:id/routes`, `GET /integrations/:id/deliveries`.
- Employee: `PUT /me/channel-identities` (link my Zulip/Telegram), verification step for Telegram/WhatsApp (bot deep link / code).

### 6. Frontend (`clients/admin`, `clients/employee`) – via `new-frontend-feature`
- Admin `features/integrations/`: provider picker → **form rendered from `ConfigSchema`** (so new providers need no UI code), test button, route matrix (event → channel/target), delivery log.
- Employee: "Connect my chat" / notification preferences page.
- Types/hooks in `packages/api-client`.

### 7. Docs for contributors
- `docs/integrations/adding-a-provider.md`: implement `Messenger`, register, add test; ~100 lines/provider. Provide `messaging/zulip` as the reference adapter and a `messengertest` conformance suite every adapter must pass.

## Phasing (new cycle file `plan/cycles/cycle-NN-integrations.md`; scope narrowly per repo rules)
1. Migrations (integrations, routes, identities, deliveries) + secretbox + domain ports.
2. Registry + Zulip + generic webhook adapters, `send_message` worker, dispatcher fan-out.
3. Admin API + tests. 4. Admin UI + employee linking UI. 5. Telegram, then others; contributor docs.
Dispatch: `backend-agent` first (steps 1–3), then `frontend-agent`. Cycle 2 is active, so schedule this as a follow-on cycle (needs Cycle 2 outbox merged).

## Critical files
- Reuse: `domain/event/events.go`, `infrastructure/eventing/{dispatcher,river_publisher,client}.go`, `infrastructure/job/{send_email_args,email_worker}.go`, `database/transactor.go`, `container/container.go`, `cmd/worker/main.go`, `config/config.go`.
- New: `domain/{service/messenger.go,entity/integration.go}`, `infrastructure/messaging/**`, `infrastructure/crypto/secretbox.go`, `infrastructure/job/send_message_*.go`, migrations via `create-migration`.

## Verification
- Unit: each adapter vs `httptest` server (Zulip/Telegram request shape, error classification); secretbox round-trip; registry.
- Integration: publish `UserInvited` in a tx → route exists → worker delivers to a mock Zulip server; permanent failure cancels job; duplicate event doesn't double-send; cross-tenant integration never fires.
- Manual: `docker-compose` + a local Zulip (or mock) → create integration in admin UI, hit Test, trigger an invitation, see message and delivery log.
- `make test`, `pnpm typecheck`.

## Open choices (defaults chosen; easy to change)
- Per-tenant config in DB (chosen) vs env-only; WhatsApp via official Cloud API only.
- Outbound-only first; inbound (bot commands like "/holidays") is a later phase using a per-provider webhook receiver on the same registry.
