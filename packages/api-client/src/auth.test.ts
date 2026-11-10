import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import MockAdapter from 'axios-mock-adapter';
import { apiClient } from './client.ts';
import { clearSession, getSession, setSession, subscribeSession } from './session.ts';
import {
  acceptInvitation,
  forgotPassword,
  inviteUser,
  listInvitations,
  listRoles,
  login,
  logout,
  refresh,
  resendInvitation,
  resetPassword,
  revokeInvitation,
} from './auth.ts';

const invitation = { id: 'inv-1', email: 'a@b.co', status: 'pending' } as const;

describe('auth & invitation api methods', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    clearSession();
  });

  afterEach(() => {
    mock.restore();
    clearSession();
  });

  it('login posts credentials, unwraps data and stores the session with tenant', async () => {
    mock.onPost('/api/v1/auth/login').reply(200, {
      success: true,
      data: {
        access_token: 'a',
        refresh_token: 'r',
        expires_at: '2030-01-01T00:00:00Z',
        user: { id: 'u1', tenant_id: 't1' },
      },
    });
    const seen: Array<unknown> = [];
    const off = subscribeSession((s) => seen.push(s));

    const res = await login({ email: 'a@b.co', password: 'pw' });
    off();

    expect(JSON.parse(mock.history.post[0].data)).toEqual({ email: 'a@b.co', password: 'pw' });
    expect(res.user?.id).toBe('u1');
    expect(getSession()).toEqual({
      accessToken: 'a',
      refreshToken: 'r',
      tenantId: 't1',
      expiresAt: '2030-01-01T00:00:00Z',
    });
    expect(seen).toHaveLength(1);
  });

  it('login rejects and stores nothing when tokens are missing', async () => {
    mock.onPost('/api/v1/auth/login').reply(200, { success: true, data: {} });
    await expect(login({ email: 'a', password: 'b' })).rejects.toThrow();
    expect(getSession()).toBeNull();
  });

  it('login does not store a session on bad credentials', async () => {
    mock.onPost('/api/v1/auth/login').reply(401, { success: false });
    await expect(login({ email: 'a', password: 'b' })).rejects.toBeDefined();
    expect(getSession()).toBeNull();
    expect(mock.history.post).toHaveLength(1);
  });

  it('refresh rotates the stored tokens and keeps the tenant', async () => {
    setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' });
    mock.onPost('/api/v1/auth/refresh').reply(200, {
      success: true,
      data: { access_token: 'a2', refresh_token: 'r2' },
    });
    await expect(refresh()).resolves.toBe('a2');
    expect(getSession()).toMatchObject({ accessToken: 'a2', refreshToken: 'r2', tenantId: 't1' });
  });

  it('refresh rejects when there is no session', async () => {
    await expect(refresh()).rejects.toThrow();
  });

  it('logout sends the refresh token and clears the session', async () => {
    setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' });
    mock.onPost('/api/v1/auth/logout').reply(200, { success: true });
    await logout();
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ refresh_token: 'r' });
    expect(getSession()).toBeNull();
  });

  it('logout clears the session even when the request fails', async () => {
    setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' });
    mock.onPost('/api/v1/auth/logout').reply(500);
    await expect(logout()).rejects.toBeDefined();
    expect(getSession()).toBeNull();
  });

  it('logout refreshes once on an expired access token, then revokes the rotated token', async () => {
    setSession({ accessToken: 'old', refreshToken: 'r1', tenantId: 't1' });
    mock
      .onPost('/api/v1/auth/logout')
      .replyOnce(401)
      .onPost('/api/v1/auth/logout')
      .replyOnce(200, { success: true });
    mock.onPost('/api/v1/auth/refresh').reply(200, {
      success: true,
      data: { access_token: 'new', refresh_token: 'r2' },
    });
    await logout();
    const logouts = mock.history.post.filter((r) => r.url === '/api/v1/auth/logout');
    expect(logouts).toHaveLength(2);
    expect(JSON.parse(logouts[1].data)).toEqual({ refresh_token: 'r2' });
    expect(logouts[1].headers?.Authorization).toBe('Bearer new');
    expect(getSession()).toBeNull();
  });

  it('logout still clears the session when the refresh fails', async () => {
    setSession({ accessToken: 'old', refreshToken: 'r1', tenantId: 't1' });
    mock.onPost('/api/v1/auth/logout').reply(401);
    mock.onPost('/api/v1/auth/refresh').reply(401);
    await expect(logout()).resolves.toBeUndefined();
    expect(getSession()).toBeNull();
  });

  it('forgotPassword posts the email', async () => {
    mock.onPost('/api/v1/auth/forgot-password').reply(200, { success: true });
    await forgotPassword({ email: 'a@b.co' });
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ email: 'a@b.co' });
  });

  it('resetPassword posts token and new password', async () => {
    mock.onPost('/api/v1/auth/reset-password').reply(200, { success: true });
    await resetPassword({ token: 't', new_password: 'Newpass123!' });
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ token: 't', new_password: 'Newpass123!' });
  });

  it('listRoles returns the role options', async () => {
    mock.onGet('/api/v1/roles').reply(200, { success: true, data: [{ id: 'r1', name: 'admin' }] });
    expect(await listRoles()).toEqual([{ id: 'r1', name: 'admin' }]);
  });

  it('listRoles returns [] when data is null', async () => {
    mock.onGet('/api/v1/roles').reply(200, { success: true, data: null });
    expect(await listRoles()).toEqual([]);
  });

  it('inviteUser posts the body and returns the invitation', async () => {
    mock.onPost('/api/v1/users/invitations').reply(201, { success: true, data: invitation });
    const res = await inviteUser({ email: 'a@b.co', first_name: 'A', last_name: 'B', role_id: 'r1' });
    expect(res).toEqual(invitation);
    expect(JSON.parse(mock.history.post[0].data).role_id).toBe('r1');
  });

  it('listInvitations passes paging params and returns data + meta', async () => {
    mock.onGet('/api/v1/users/invitations').reply(200, {
      success: true,
      data: [invitation],
      meta: { page: 2, page_size: 10, total_items: 11, total_pages: 2 },
    });
    const res = await listInvitations({ page: 2, page_size: 10 });
    expect(mock.history.get[0].params).toEqual({ page: 2, page_size: 10 });
    expect(res.data).toEqual([invitation]);
    expect(res.meta?.total_pages).toBe(2);
  });

  it('listInvitations returns an empty array when data is null', async () => {
    mock.onGet('/api/v1/users/invitations').reply(200, { success: true, data: null });
    expect((await listInvitations()).data).toEqual([]);
  });

  it('resendInvitation POSTs to the id route', async () => {
    mock.onPost('/api/v1/users/invitations/inv-1/resend').reply(200, { success: true, data: invitation });
    await expect(resendInvitation('inv-1')).resolves.toEqual(invitation);
  });

  it('revokeInvitation DELETEs the id route', async () => {
    mock.onDelete('/api/v1/users/invitations/inv-1').reply(200, { success: true });
    await expect(revokeInvitation('inv-1')).resolves.toBeUndefined();
    expect(mock.history.delete).toHaveLength(1);
  });

  it('acceptInvitation posts token and password', async () => {
    mock.onPost('/api/v1/invitations/accept').reply(200, { success: true });
    await acceptInvitation({ token: 'tok', password: 'Pass1234!' });
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ token: 'tok', password: 'Pass1234!' });
  });

  it('an expired token on an invitation call transparently refreshes once and succeeds', async () => {
    setSession({ accessToken: 'old', refreshToken: 'r', tenantId: 't1' });
    mock.onGet('/api/v1/users/invitations').reply((config) =>
      config.headers?.Authorization === 'Bearer new'
        ? [200, { success: true, data: [invitation] }]
        : [401, { success: false }],
    );
    mock.onPost('/api/v1/auth/refresh').reply(200, {
      success: true,
      data: { access_token: 'new', refresh_token: 'r2' },
    });

    const res = await listInvitations();

    expect(res.data).toEqual([invitation]);
    expect(mock.history.post).toHaveLength(1);
  });
});
