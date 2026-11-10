// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, refreshSession } from './client.ts';
import { clearSession, getSession, setSession, setSessionStorage } from './session.ts';
import { createLocalStorageSessionAdapter } from './sessionStorage.ts';

const KEY = 'lock.session';
const base = { tenantId: 't1' };
const tokens = { success: true, data: { access_token: 'a2', refresh_token: 'r2' } };

type LockFn = (name: string, cb: () => Promise<unknown>) => Promise<unknown>;

function stubLocks(request: LockFn): void {
  Object.defineProperty(navigator, 'locks', { value: { request }, configurable: true });
}

describe('refreshSession cross-tab lock', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    localStorage.clear();
    mock = new MockAdapter(apiClient);
    setSessionStorage(createLocalStorageSessionAdapter(KEY));
  });
  afterEach(() => {
    mock.restore();
    setSessionStorage(null);
    clearSession();
    Reflect.deleteProperty(navigator, 'locks');
  });

  it('runs under navigator.locks.request with the refresh lock name', async () => {
    const request = vi.fn<LockFn>((_n, cb) => cb());
    stubLocks(request);
    setSession({ accessToken: 'a1', refreshToken: 'r1', ...base });
    mock.onPost('/api/v1/auth/refresh').reply(200, tokens);
    await expect(refreshSession()).resolves.toBe('a2');
    expect(request).toHaveBeenCalledTimes(1);
    expect(request.mock.calls[0][0]).toBe('employee360-auth-refresh');
    expect(getSession()?.refreshToken).toBe('r2');
  });

  it('falls back to running directly when Web Locks are unavailable', async () => {
    setSession({ accessToken: 'a1', refreshToken: 'r1', ...base });
    mock.onPost('/api/v1/auth/refresh').reply(200, tokens);
    await expect(refreshSession()).resolves.toBe('a2');
  });

  it('reuses the access token another tab persisted while waiting, without calling the backend', async () => {
    // The "other tab" rotates while we are queued on the lock.
    stubLocks(async (_n, cb) => {
      localStorage.setItem(
        KEY,
        JSON.stringify({ accessToken: 'a9', refreshToken: 'r9', ...base }),
      );
      return cb();
    });
    setSession({ accessToken: 'a1', refreshToken: 'r1', ...base });
    await expect(refreshSession()).resolves.toBe('a9');
    expect(mock.history.post).toHaveLength(0);
    expect(getSession()).toMatchObject({ accessToken: 'a9', refreshToken: 'r9' });
  });

  it('rejects when another tab logged out while waiting', async () => {
    stubLocks(async (_n, cb) => {
      localStorage.removeItem(KEY);
      return cb();
    });
    setSession({ accessToken: 'a1', refreshToken: 'r1', ...base });
    await expect(refreshSession()).rejects.toThrow(/No refresh token/);
    expect(mock.history.post).toHaveLength(0);
  });

  it('still dedupes concurrent callers within a tab to one lock + one request', async () => {
    const request = vi.fn<LockFn>((_n, cb) => cb());
    stubLocks(request);
    setSession({ accessToken: 'a1', refreshToken: 'r1', ...base });
    mock.onPost('/api/v1/auth/refresh').reply(200, tokens);
    await Promise.all([refreshSession(), refreshSession(), refreshSession()]);
    expect(request).toHaveBeenCalledTimes(1);
    expect(mock.history.post).toHaveLength(1);
  });
});
