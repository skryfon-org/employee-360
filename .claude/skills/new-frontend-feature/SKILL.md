---
name: new-frontend-feature
description: >
  Scaffold a new feature module (components/pages/queries/schemas/routes.tsx) in the
  Employee360 Admin or Employee client app. Use when the user asks to add a new
  page/screen/feature to the admin portal or employee portal.
---

# New Frontend Feature

Employee360's frontend follows a strict feature-folder convention (see the
`frontend-agent` agent and `plan/architecture/frontend.md`). Every feature is
self-contained and follows the same shape in both `clients/admin/` and
`clients/employee/`.

## Step 1 — confirm which app

Ask (if not already clear): is this feature for `clients/admin/` (email+password auth,
org/holiday/RBAC management) or `clients/employee/` (passwordless login, read-only
holiday calendar)? Don't build cross-app shared feature code — shared logic belongs in
`packages/`.

## Step 2 — create the folder structure

```
clients/<app>/src/features/<feature>/
├── components/      # private UI specific to this feature
├── pages/            # route-mounted page components
├── queries/           # TanStack Query hooks
├── schemas/           # Zod schemas
└── routes.tsx         # this feature's <Route> definitions, exported
```

## Step 3 — schemas (`schemas/`)

Define Zod schemas for any form input and/or API response shape this feature owns.
Infer TypeScript types from the schema (`z.infer<typeof xSchema>`) rather than hand
writing parallel interfaces.

## Step 4 — queries (`queries/`)

- Import the generated client/hooks from `packages/api-client` — never raw
  `fetch`/`axios`.
- Wrap them in feature-named hooks (`useHolidaysQuery`, `useCreateHolidayMutation`) so
  pages/components don't import `packages/api-client` directly.
- This is server state — it lives in TanStack Query only. Do not mirror it into a
  Zustand store.

## Step 5 — components & pages

- `components/` — presentational + feature-specific interactive pieces (tables, form
  modals, cards).
- `pages/` — compose components + queries into a full route-level page.
- Forms use React Hook Form + the Zod schema from Step 3 (`zodResolver`).
- Gate admin-only UI with the app's permission/role check pattern (check
  `clients/admin/src/hooks/` for an existing `useHasRole`-style hook before inventing a
  new one).

## Step 6 — routes.tsx

Export an array/element of route definitions from this feature. Wire it into the app's
route composition (`App.tsx` or a central `routeConfig`) — check the existing pattern in
other features before adding a new one.

## Step 7 — Zustand (only if truly needed)

Only add a store entry if the feature has genuine ephemeral client-only state (active
filter, selected year, modal open/close) that doesn't belong in TanStack Query. Don't
create a store just to cache API data.

## Step 8 — test

Add at least a smoke test (Vitest + Testing Library) for the main page component.

## Step 9 — verify

Once the workspace is scaffolded, run from repo root:

```bash
pnpm --filter <app> typecheck
pnpm --filter <app> lint
pnpm --filter <app> test -- --run
```
