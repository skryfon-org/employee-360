# Frontend Architecture & Conventions

## Monorepo & Feature Layout

1. **Monorepo Layout**:
   - `clients/admin`: Admin portal for tenant & organization management, roles, departments, positions, holidays, and audit logs.
   - `clients/employee`: Employee self-service calendar view and passwordless login.
   - `packages/api-client`: Shared Axios client with automated JWT & `X-Tenant-ID` interceptors, generated React Query hooks, and typed models.
2. **Feature-Driven Architecture**:
   - Each feature in `clients/*/src/features/<feature>/` is self-contained:
     - `components/`: Feature-specific UI components.
     - `pages/`: Page components mounted to router.
     - `queries/`: React Query custom hooks.
     - `schemas/`: Zod validation schemas.
     - `routes.tsx`: Feature route definitions.
3. **State Management Separation**:
   - **Server State**: Managed exclusively by **TanStack Query (React Query)** via `@employee360/api-client`.
   - **UI / Ephemeral State**: Managed via **Zustand** stores (`authStore`, `uiStore`). Never duplicate server state into Zustand stores.
