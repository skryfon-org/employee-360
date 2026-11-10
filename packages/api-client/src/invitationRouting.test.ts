import { describe, expect, it } from 'vitest';
import { invitationAppForRole, resolveInvitationRedirect } from './invitationRouting.ts';

describe('invitationAppForRole', () => {
  it.each([
    ['admin', 'admin'],
    ['super_admin', 'admin'],
    ['employee', 'employee'],
    ['manager', null],
    [undefined, null],
  ])('%s -> %s', (role, app) => {
    expect(invitationAppForRole(role)).toBe(app);
  });
});

describe('resolveInvitationRedirect', () => {
  const base = { token: 'a b&c', adminAppUrl: 'http://admin.test/', employeeAppUrl: 'http://emp.test' };
  it('stays when role belongs to the current app', () => {
    expect(resolveInvitationRedirect({ ...base, role: 'employee', currentApp: 'employee' })).toEqual({ kind: 'stay' });
    expect(resolveInvitationRedirect({ ...base, role: 'super_admin', currentApp: 'admin' })).toEqual({ kind: 'stay' });
  });
  it('stays for unknown roles', () => {
    expect(resolveInvitationRedirect({ ...base, role: 'manager', currentApp: 'admin' })).toEqual({ kind: 'stay' });
  });
  it('redirects to the other app preserving the encoded token', () => {
    expect(resolveInvitationRedirect({ ...base, role: 'employee', currentApp: 'admin' })).toEqual({
      kind: 'redirect',
      url: 'http://emp.test/accept-invitation?token=a%20b%26c',
    });
    expect(resolveInvitationRedirect({ ...base, role: 'admin', currentApp: 'employee' })).toEqual({
      kind: 'redirect',
      url: 'http://admin.test/accept-invitation?token=a%20b%26c',
    });
  });
  it('reports unconfigured when the target URL is missing or blank', () => {
    expect(
      resolveInvitationRedirect({ role: 'employee', currentApp: 'admin', token: 't', employeeAppUrl: ' ' }),
    ).toEqual({ kind: 'unconfigured', targetApp: 'employee' });
  });
});
