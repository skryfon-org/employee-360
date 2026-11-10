# Frontend Architecture

This document outlines the frontend architecture and folder structure for **Employee360**.

---

## 1. Monorepo Structure Overview

The frontend is organized as a monorepo containing multiple client applications and shared packages managed via `pnpm` workspaces.

```
employee-360/
├── clients/
│   ├── admin/                    # Admin portal (Organization, Users, Roles, Holidays, Audit)
│   └── employee/                 # Employee portal (Holiday Calendar view, Profile, Email login)
├── packages/
│   ├── api-client/               # Shared typed API client + React Query hooks
│   └── ui/                       # (Optional) Shared UI design tokens & primitive components
├── package.json                  # Root workspace configuration
└── pnpm-workspace.yaml           # pnpm workspace definition
```

---

## 2. Directory Tree Breakdown

### 2.1 Admin Application (`clients/admin/`)

```
clients/admin/
├── public/                       # Static public assets (favicons, icons)
├── src/
│   ├── components/               # Shared cross-feature UI components
│   │   ├── layout/               # AppShell, Sidebar, Header, PageContainer
│   │   ├── ui/                   # Buttons, Modals, Forms, Tables, Badges, Tooltips
│   │   └── feedback/             # Toasts, AlertBanners, SkeletonLoaders
│   ├── features/                 # Domain / Feature modules
│   │   ├── auth/                 # Admin authentication (email/password)
│   │   │   ├── components/       # LoginForm, ProtectedRoute
│   │   │   ├── pages/            # LoginPage
│   │   │   ├── queries/          # Auth API hooks (useLoginMutation, useLogoutMutation)
│   │   │   ├── schemas/          # Zod validation schemas (loginSchema)
│   │   │   └── routes.tsx        # Auth route definitions
│   │   ├── users/                # User directory & employee onboarding
│   │   │   ├── components/       # UserTable, UserFormModal, UserRoleSelect, UserStatusBadge
│   │   │   ├── pages/            # UserListPage, UserDetailPage
│   │   │   ├── queries/          # useUsersQuery, useCreateUserMutation, useUpdateUserMutation
│   │   │   ├── schemas/          # userFormSchema
│   │   │   └── routes.tsx
│   │   ├── departments/          # Department hierarchy management
│   │   │   ├── components/       # DepartmentTree, DepartmentFormModal, ManagerSelect
│   │   │   ├── pages/            # DepartmentListPage
│   │   │   ├── queries/          # useDepartmentsQuery, useCreateDepartmentMutation
│   │   │   ├── schemas/          # departmentFormSchema
│   │   │   └── routes.tsx
│   │   ├── positions/            # Job titles & designations
│   │   │   ├── components/       # PositionTable, PositionFormModal
│   │   │   ├── pages/            # PositionListPage
│   │   │   ├── queries/          # usePositionsQuery, useCreatePositionMutation
│   │   │   ├── schemas/          # positionFormSchema
│   │   │   └── routes.tsx
│   │   ├── roles/                # RBAC & Role management
│   │   │   ├── components/       # RoleList, RolePermissionMatrix, UserRoleAssignmentModal
│   │   │   ├── pages/            # RoleListPage
│   │   │   ├── queries/          # useRolesQuery, useAssignRoleMutation
│   │   │   ├── schemas/          # roleFormSchema
│   │   │   └── routes.tsx
│   │   ├── holidays/             # Holiday calendar management
│   │   │   ├── components/       # HolidayFormModal, HolidayTable, HolidayCard, YearFilter
│   │   │   ├── pages/            # HolidayListPage, HolidayDetailPage
│   │   │   ├── queries/          # useHolidaysQuery, useCreateHolidayMutation, etc.
│   │   │   ├── schemas/          # holidayFormSchema
│   │   │   └── routes.tsx        # Holiday routes
│   │   ├── categories/           # Holiday categories
│   │   │   ├── components/       # CategoryList, CategoryModal
│   │   │   ├── pages/            # CategoryManagementPage
│   │   │   ├── queries/          # useCategoriesQuery, useCreateCategoryMutation
│   │   │   ├── schemas/          # categoryFormSchema
│   │   │   └── routes.tsx
│   │   ├── audit/                # Audit trail
│   │   │   ├── components/       # AuditLogTable, DiffViewerModal
│   │   │   ├── pages/            # AuditLogPage
│   │   │   └── queries/          # useAuditLogsQuery
│   │   └── dashboard/            # Admin dashboard summary & quick stats
│   │       ├── components/       # SummaryCards, UpcomingHolidaysWidget, OrgOverviewWidget
│   │       ├── pages/            # DashboardPage
│   │       └── routes.tsx
│   ├── hooks/                    # Reusable global hooks (e.g., useDebounce, useTenantContext, useHasRole)
│   ├── lib/                      # Client utilities (formatting, date-fns helpers)
│   ├── stores/                   # Zustand stores for ephemeral UI/session state
│   │   ├── authStore.ts          # Auth state (tokens, active user, user roles)
│   │   └── uiStore.ts            # UI state (sidebar collapse, theme, active modal)
│   ├── App.tsx                   # Main App root with router & React Query provider
│   ├── main.tsx                  # React entry point
│   └── index.css                 # Tailwind CSS styles & design tokens
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

---

### 2.2 Employee Application (`clients/employee/`)

```
clients/employee/
├── public/                       # Static public assets
├── src/
│   ├── components/               # Shared cross-feature UI components
│   │   ├── layout/               # EmployeeLayout, Navbar, Footer
│   │   ├── ui/                   # View Toggle (Calendar / List), CategoryFilter
│   │   └── feedback/             # Loading spinner, Error states
│   ├── features/
│   │   ├── auth/                 # Employee passwordless login
│   │   │   ├── components/       # EmailLoginForm, OtpVerifyForm
│   │   │   ├── pages/            # LoginPage, VerifyPage
│   │   │   ├── queries/          # useSendOtpMutation, useVerifyOtpMutation
│   │   │   ├── schemas/          # emailLoginSchema, otpSchema
│   │   │   └── routes.tsx
│   │   └── calendar/             # Holiday calendar & list view
│   │       ├── components/       # CalendarGrid, MonthView, HolidayListView, HolidayBadge
│   │       ├── pages/            # CalendarPage, UpcomingHolidaysPage
│   │       ├── queries/          # useEmployeeHolidaysQuery
│   │       └── routes.tsx
│   ├── hooks/                    # Custom hooks (e.g., useCalendarNavigation)
│   ├── lib/                      # Date formatting and helper utilities
│   ├── stores/                   # Zustand stores
│   │   ├── authStore.ts          # Employee session & tenant info
│   │   └── calendarStore.ts      # Active year, active view (grid vs list), filters
│   ├── App.tsx
│   ├── main.tsx
│   └── index.css
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

---

### 2.3 Shared API Client Package (`packages/api-client/`)

```
packages/api-client/
├── src/
│   ├── client.ts                 # Axios instance with interceptors for JWT & X-Tenant-ID
│   ├── customInstance.ts         # Custom fetch / request wrapper
│   ├── secureStorage.ts          # Token storage abstraction (memory / secure cookies)
│   ├── unwrap.ts                 # Standard response unwrapping helper
│   ├── generated/                # Auto-generated API endpoints, DTOs & TanStack Query hooks
│   │   ├── models/               # Auto-generated TypeScript types & models
│   │   └── hooks/                # Auto-generated React Query hooks
│   └── index.ts                  # Public package exports
├── orval.config.ts               # Code generation config (OpenAPI -> TypeScript / React Query)
├── package.json
└── tsconfig.json
```

---

## 3. Layer Responsibilities & Conventions

### 3.1 Feature-Driven Structure (`features/<feature>/`)
Each feature module is self-contained:
- **`components/`**: Private UI components specific to the feature.
- **`pages/`**: Full page components mounted by the router.
- **`queries/`**: React Query hooks wrapping API calls for data fetching and mutations.
- **`schemas/`**: Zod validation schemas for forms and inputs.
- **`routes.tsx`**: Route configurations for the feature.

### 3.2 State Management Separation
- **Server State**: Managed exclusively by **TanStack Query (React Query)** via `@employee360/api-client`. Handles caching, background refetching, and optimistic updates.
- **Client/UI State**: Managed via **Zustand** stores. Strictly holds client-only state (e.g., active filters, sidebar collapsed state, modal open/close).

### 3.3 Multi-Tenant Context Handling
- Tenant resolution happens upon login based on email domain or authenticated token claims.
- The API client interceptor in `packages/api-client` automatically attaches the active `X-Tenant-ID` header and `Authorization` Bearer token to all outgoing requests.
