import { defineConfig } from 'vitest/config';

// This package has no DOM code of its own — the default environment is
// 'node'. `src/health.test.ts` opts a single file into 'jsdom' (via a
// `@vitest-environment` docblock) purely to exercise `useHealth()` with
// `@testing-library/react`'s `renderHook`, since that's the hook both apps
// actually consume.
export default defineConfig({
  test: {
    environment: 'node',
  },
});
