export type InvitationApp = 'admin' | 'employee';

export type InvitationRedirect =
  | { kind: 'stay' }
  | { kind: 'redirect'; url: string }
  | { kind: 'unconfigured'; targetApp: InvitationApp };

const ADMIN_APP_ROLES = ['admin', 'super_admin'];
const EMPLOYEE_APP_ROLES = ['employee'];

/** Which app owns an invited role; `null` for roles outside the known sets. */
export function invitationAppForRole(role: string | undefined): InvitationApp | null {
  if (!role) return null;
  if (ADMIN_APP_ROLES.includes(role)) return 'admin';
  if (EMPLOYEE_APP_ROLES.includes(role)) return 'employee';
  return null;
}

/**
 * Decide whether the invitee must be sent to the other app. Unknown roles stay
 * on the current app. A wrong-host role with no configured target URL yields
 * `unconfigured` so the UI can explain instead of redirecting to a broken URL.
 * The token is preserved in the target `/accept-invitation` URL.
 */
export function resolveInvitationRedirect(args: {
  role: string | undefined;
  currentApp: InvitationApp;
  token: string;
  adminAppUrl?: string;
  employeeAppUrl?: string;
}): InvitationRedirect {
  const target = invitationAppForRole(args.role);
  if (!target || target === args.currentApp) return { kind: 'stay' };
  const base = (target === 'admin' ? args.adminAppUrl : args.employeeAppUrl)?.trim();
  if (!base) return { kind: 'unconfigured', targetApp: target };
  const url = `${base.replace(/\/+$/, '')}/accept-invitation?token=${encodeURIComponent(args.token)}`;
  return { kind: 'redirect', url };
}
