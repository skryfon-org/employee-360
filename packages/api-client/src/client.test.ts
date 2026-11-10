import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, configureApiClient, configureAuth, refreshSession } from './client.ts';
import { clearSession, getSession, setSession } from './session.ts';

/**
 * Minimal shape of axios's internal (untyped-in-the-public-API) interceptor
 * manager, used only to assert that handlers were actually registered.
 * Avoids `any` per project coding rules while still reaching into the one
 * axios internal this suite needs to introspect.
 */
interface InterceptorManager {
  handlers: Array<unknown | null>;
}

describe('configureApiClient (AC-2)', () => {
  const originalBaseURL = apiClient.defaults.baseURL;

  afterEach(() => {
    apiClient.defaults.baseURL = originalBaseURL;
  });

  it('updates apiClient.defaults.baseURL to the configured value', () => {
    configureApiClient({ baseURL: 'http://example.test' });

    expect(apiClient.defaults.baseURL).toBe('http://example.test');
  });

  it('leaves the existing baseURL untouched when given an empty string', () => {
    configureApiClient({ baseURL: 'http://example.test' });
    configureApiClient({ baseURL: '' });

    expect(apiClient.defaults.baseURL).toBe('http://example.test');
  });
});

describe('request interceptor (token + tenant)', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    clearSession();
  });

  afterEach(() => {
    mock.restore();
    clearSession();
  });

  it('registers request and response interceptor handlers', () => {
    const req = apiClient.interceptors.request as unknown as InterceptorManager;
    const res = apiClient.interceptors.response as unknown as InterceptorManager;
    expect(req.handlers.filter(Boolean).length).toBeGreaterThanOrEqual(1);
    expect(res.handlers.filter(Boolean).length).toBeGreaterThanOrEqual(1);
  });

  it('sends no auth headers without a session', async () => {
    mock.onGet('/probe').reply(200, {});
    await apiClient.get('/probe');
    const [request] = mock.history.get;
    expect(request.headers?.Authorization).toBeUndefined();
    expect(request.headers?.['X-Tenant-ID']).toBeUndefined();
  });

  it('attaches Authorization and X-Tenant-ID from the session', async () => {
    setSession({ accessToken: 'a1', refreshToken: 'r1', tenantId: 't1' });
    mock.onGet('/probe').reply(200, {});
    await apiClient.get('/probe', { params: { a: 1 } });
    const [request] = mock.history.get;
    expect(request.headers?.Authorization).toBe('Bearer a1');
    expect(request.headers?.['X-Tenant-ID']).toBe('t1');
    expect(request.params).toEqual({ a: 1 });
  });
});

describe('response interceptor: silent refresh on 401', () => {
  let mock: MockAdapter;
  const onAuthFailure = vi.fn();

  const refreshOk = {
    success: true,
    data: { access_token: 'a2', refresh_token: 'r2', token_type: 'Bearer' },
  };

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    onAuthFailure.mockReset();
    configureAuth({ onAuthFailure });
    setSession({ accessToken: 'a1', refreshToken: 'r1', tenantId: 't1' });
  });

  afterEach(() => {
    mock.restore();
    configureAuth({ onAuthFailure: undefined });
    clearSession();
  });

  it('refreshes once, retries the original request and succeeds', async () => {
    mock.onGet('/probe').replyOnce(401).onGet('/probe').reply(200, { ok: true });
    mock.onPost('/api/v1/auth/refresh').reply(200, refreshOk);

    const res = await apiClient.get('/probe');

    expect(res.data).toEqual({ ok: true });
    expect(mock.history.post).toHaveLength(1);
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ refresh_token: 'r1' });
    expect(mock.history.get).toHaveLength(2);
    expect(mock.history.get[1].headers?.Authorization).toBe('Bearer a2');
    expect(getSession()).toMatchObject({ accessToken: 'a2', refreshToken: 'r2', tenantId: 't1' });
    expect(onAuthFailure).not.toHaveBeenCalled();
  });

  it('shares a single in-flight refresh across concurrent 401s', async () => {
    mock.onGet(/\/p\d/).reply((config) =>
      config.headers?.Authorization === 'Bearer a2' ? [200, { ok: true }] : [401],
    );
    mock.onPost('/api/v1/auth/refresh').reply(200, refreshOk);

    const results = await Promise.all([
      apiClient.get('/p1'),
      apiClient.get('/p2'),
      apiClient.get('/p3'),
    ]);

    expect(results.map((r) => r.status)).toEqual([200, 200, 200]);
    expect(mock.history.post).toHaveLength(1);
  });

  it('does not loop: a 401 after the retry is surfaced without another refresh', async () => {
    mock.onGet('/probe').reply(401);
    mock.onPost('/api/v1/auth/refresh').reply(200, refreshOk);

    await expect(apiClient.get('/probe')).rejects.toMatchObject({ response: { status: 401 } });

    expect(mock.history.post).toHaveLength(1);
    expect(mock.history.get).toHaveLength(2);
  });

  it('forces logout when the refresh fails', async () => {
    mock.onGet('/probe').reply(401);
    mock.onPost('/api/v1/auth/refresh').reply(401);

    await expect(apiClient.get('/probe')).rejects.toMatchObject({ response: { status: 401 } });

    expect(getSession()).toBeNull();
    expect(onAuthFailure).toHaveBeenCalledTimes(1);
    expect(mock.history.post).toHaveLength(1);
  });

  it('forces logout when there is no refresh token to use', async () => {
    setSession({ accessToken: 'a1', refreshToken: '', tenantId: 't1' });
    mock.onGet('/probe').reply(401);

    await expect(apiClient.get('/probe')).rejects.toBeDefined();

    expect(mock.history.post).toHaveLength(0);
    expect(onAuthFailure).toHaveBeenCalledTimes(1);
  });

  it.each([
    '/api/v1/auth/login',
    '/api/v1/auth/logout',
    '/api/v1/auth/forgot-password',
    '/api/v1/auth/reset-password',
    '/api/v1/invitations/accept',
  ])('never tries to refresh on %s', async (url) => {
    mock.onPost(url).reply(401);

    await expect(apiClient.post(url, {})).rejects.toBeDefined();

    expect(mock.history.post.filter((r) => r.url === '/api/v1/auth/refresh')).toHaveLength(0);
    expect(onAuthFailure).not.toHaveBeenCalled();
  });

  it('does not recurse when the refresh call itself 401s', async () => {
    mock.onPost('/api/v1/auth/refresh').reply(401);
    await expect(refreshSession()).rejects.toBeDefined();
    expect(mock.history.post).toHaveLength(1);
  });

  it('leaves anonymous 401s alone', async () => {
    clearSession();
    mock.onGet('/probe').reply(401);
    await expect(apiClient.get('/probe')).rejects.toBeDefined();
    expect(mock.history.post).toHaveLength(0);
    expect(onAuthFailure).not.toHaveBeenCalled();
  });

  it('does not refresh on non-401 errors', async () => {
    mock.onGet('/probe').reply(500);
    await expect(apiClient.get('/probe')).rejects.toBeDefined();
    expect(mock.history.post).toHaveLength(0);
  });
});
