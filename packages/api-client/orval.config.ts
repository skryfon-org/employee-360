import { defineConfig } from 'orval';

// Regenerates from the backend's Swagger/OpenAPI spec, which `make swagger`
// (or `make generate`) keeps up to date at `backend/docs/swagger.json`.
//
// Output under `src/generated/` is checked into version control (like
// `backend/docs/swagger.json` itself) rather than gitignored, so consumers
// don't need to run `pnpm generate:api` just to install/build. Re-run
// `pnpm --filter @employee360/api-client generate` (or `pnpm generate:api`
// from the repo root) after the backend's OpenAPI spec changes, and commit
// the diff.
export default defineConfig({
  api: {
    input: {
      target: '../../backend/docs/swagger.json',
    },
    output: {
      mode: 'tags-split',
      target: 'src/generated/hooks',
      schemas: 'src/generated/models',
      client: 'react-query',
      httpClient: 'axios',
      mock: false,
      clean: true,
      override: {
        mutator: {
          path: './src/client.ts',
          name: 'apiRequest',
        },
      },
    },
  },
});
