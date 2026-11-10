// @vitest-environment jsdom
//
// Whole-file override: the package default (vitest.config.ts) is 'node'
// since this package has no DOM code, but the useHealth() suite below uses
// @testing-library/react's renderHook, which needs a DOM. jsdom is a strict
// superset for the fetchHealth() suite above it, so sharing one file is safe.
import { createElement, type PropsWithChildren } from 'react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient } from './client.ts';
import { fetchHealth, useHealth, type HealthResponse } from './health.ts';

const HEALTH_URL = '/api/v1/health';

function successEnvelope(overrides: Partial<HealthResponse> = {}) {
  return {
    success: true,
    data: {
      status: 'ok',
      app: 'employee360',
      database: 'ok',
      ...overrides,
    },
  };
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: PropsWithChildren) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe('fetchHealth (AC-5)', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
  });

  it('resolves to the unwrapped HealthResponse on a well-formed envelope', async () => {
    mock.onGet(HEALTH_URL).reply(200, successEnvelope());

    await expect(fetchHealth()).resolves.toEqual({
      status: 'ok',
      app: 'employee360',
      database: 'ok',
    });
  });

  it('rejects when the envelope is missing the data field', async () => {
    mock.onGet(HEALTH_URL).reply(200, { success: true });

    await expect(fetchHealth()).rejects.toThrow(
      'Health endpoint responded with an unexpected payload shape',
    );
  });

  it('rejects when a data field has the wrong type', async () => {
    mock.onGet(HEALTH_URL).reply(
      200,
      successEnvelope({ database: 123 as unknown as string }),
    );

    await expect(fetchHealth()).rejects.toThrow(
      'Health endpoint responded with an unexpected payload shape',
    );
  });
});

describe('useHealth (AC-5)', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
  });

  it('settles to the unwrapped HealthResponse', async () => {
    mock.onGet(HEALTH_URL).reply(200, successEnvelope());

    const { result } = renderHook(() => useHealth(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual({
      status: 'ok',
      app: 'employee360',
      database: 'ok',
    });
  });

  it('surfaces an error when the envelope is malformed', async () => {
    mock.onGet(HEALTH_URL).reply(200, { success: true });

    const { result } = renderHook(() => useHealth(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
