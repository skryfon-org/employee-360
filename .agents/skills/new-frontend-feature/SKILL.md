---
name: new-frontend-feature
description: >-
  Scaffold a new frontend feature module (components, pages, queries, schemas, routes.tsx)
  in clients/admin or clients/employee for Employee360.
---

# New Frontend Feature

Employee360 uses a feature-driven architecture within the Vite + React frontend applications (`clients/admin/` or `clients/employee/`).

## Directory Structure to Scaffold

```
clients/<app>/src/features/<feature>/
├── components/       # Feature-specific private UI components
├── pages/            # Page components mounted to router
├── queries/          # TanStack Query custom hooks wrapping API client calls
├── schemas/          # Zod validation schemas for forms
└── routes.tsx        # Route definitions for this feature
```

## Step-by-Step Scaffolding

### Step 1 — Define Zod Schema (`schemas/<feature>Schema.ts`)
Define validation schemas matching the backend request DTOs:
```ts
import { z } from 'zod';

export const createFeatureSchema = z.object({
  name: z.string().min(1, 'Name is required'),
});

export type CreateFeatureInput = z.infer<typeof createFeatureSchema>;
```

### Step 2 — React Query Hooks (`queries/use<Feature>.ts`)
Wrap API client calls using TanStack Query:
```ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/apiClient';

export const useFeatureList = () => {
  return useQuery({
    queryKey: ['<feature>'],
    queryFn: async () => {
      const response = await apiClient.get('/api/v1/<feature>');
      return response.data;
    },
  });
};
```

### Step 3 — Components & Pages
1. Build reusable sub-components in `components/`.
2. Build the top-level route pages in `pages/` using React Hook Form + Zod resolvers.

### Step 4 — Register Routes (`routes.tsx`)
Export the feature's Route objects and mount them into `src/App.tsx`.
