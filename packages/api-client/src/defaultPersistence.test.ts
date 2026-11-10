// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MockAdapter from 'axios-mock-adapter';

const KEY = 'employee360.session';
const stored = { accessToken: 'a1', refreshToken: 'r1', tenantId: 't1' };

async function load() {
  vi.resetModules();
  const session = await import('./session.ts');
  const client = await import('./client.ts');
  return { ...session, ...client };
}

describe('default-on localStorage persistence', () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => vi.restoreAllMocks());

  it('hydrates from localStorage at import, before any request', async () => {
    localStorage.setItem(KEY, JSON.stringify(stored));
    const m = await load();
    expect(m.getSession()).toMatchObject(stored);
    const mock = new MockAdapter(m.apiClient);
    mock.onGet('/probe').reply(200, {});
    await m.apiClient.get('/probe');
    expect(mock.history.get[0].headers?.Authorization).toBe('Bearer a1');
    expect(mock.history.get[0].headers?.['X-Tenant-ID']).toBe('t1');
  });

  it('persists writes and syncs across tabs without any setup', async () => {
    const m = await load();
    m.setSession({ ...stored, accessToken: 'a9' });
    expect(JSON.parse(localStorage.getItem(KEY) ?? '{}').accessToken).toBe('a9');
    window.dispatchEvent(new StorageEvent('storage', { key: KEY, newValue: null }));
    expect(m.getSession()).toBeNull();
  });

  it('ignores a corrupt stored session', async () => {
    localStorage.setItem(KEY, '{bad');
    expect((await load()).getSession()).toBeNull();
  });

  it('setSessionStorage(null) opts out of persistence', async () => {
    const m = await load();
    m.setSessionStorage(null);
    m.setSession(stored);
    expect(localStorage.getItem(KEY)).toBeNull();
  });
});

describe('refresh resilience', () => {
  it('keeps the session on a transient refresh failure and only logs out on auth rejection', async () => {
    localStorage.clear();
    const m = await load();
    const onAuthFailure = vi.fn();
    m.configureAuth({ onAuthFailure });
    const mock = new MockAdapter(m.apiClient);
    mock.onGet('/probe').reply(401);

    m.setSession(stored);
    mock.onPost('/api/v1/auth/refresh').networkErrorOnce();
    await expect(m.apiClient.get('/probe')).rejects.toBeDefined();
    expect(m.getSession()).not.toBeNull();
    mock.onPost('/api/v1/auth/refresh').replyOnce(503);
    await expect(m.apiClient.get('/probe')).rejects.toBeDefined();
    expect(m.getSession()).not.toBeNull();
    expect(onAuthFailure).not.toHaveBeenCalled();

    mock.onPost('/api/v1/auth/refresh').replyOnce(401);
    await expect(m.apiClient.get('/probe')).rejects.toBeDefined();
    expect(m.getSession()).toBeNull();
    expect(onAuthFailure).toHaveBeenCalledTimes(1);
  });

  it('replays a stale-token 401 with the newest token without refreshing', async () => {
    localStorage.clear();
    const m = await load();
    const mock = new MockAdapter(m.apiClient);
    m.setSession(stored);
    mock.onGet('/probe').reply((c) => {
      if (c.headers?.Authorization === 'Bearer a1') {
        // another tab rotated while this request was in flight
        m.applyExternalSession({ ...stored, accessToken: 'a2', refreshToken: 'r2' });
        return [401];
      }
      return [200, { ok: true }];
    });
    const res = await m.apiClient.get('/probe');
    expect(res.status).toBe(200);
    expect(mock.history.post).toHaveLength(0);
    expect(mock.history.get[1].headers?.Authorization).toBe('Bearer a2');
  });

  it('does not resurrect the session if logged out while the refresh is in flight', async () => {
    localStorage.clear();
    const m = await load();
    const mock = new MockAdapter(m.apiClient);
    m.setSession(stored);
    mock.onPost('/api/v1/auth/refresh').reply(() => {
      m.applyExternalSession(null); // other tab logged out
      localStorage.removeItem(KEY);
      return [200, { data: { access_token: 'a2', refresh_token: 'r2' } }];
    });
    await expect(m.refreshSession()).rejects.toThrow();
    expect(m.getSession()).toBeNull();
    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it('puts a timeout on the refresh request', async () => {
    localStorage.clear();
    const m = await load();
    const mock = new MockAdapter(m.apiClient);
    m.setSession(stored);
    mock.onPost('/api/v1/auth/refresh').reply(200, { data: { access_token: 'a', refresh_token: 'b' } });
    await m.refreshSession();
    expect(mock.history.post[0].timeout).toBe(15000);
  });
});
