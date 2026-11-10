# Employee Dashboard: Next Feature Ideas

Backlog of follow-ups after the employee dashboard (EMPLOYEE36-65/66/69). Not scheduled in any cycle yet.
Company policies, company directory and announcements are already planned elsewhere and are intentionally not listed here.

Suggested order: cleanup, then 1, then 2.

## Cleanup first
- Browser-check the new dashboard and header breadcrumbs on desktop, mobile and dark mode.
- Investigate flaky tests when the full admin and employee suites run (files pass in isolation; admin `employees` and `projects`, employee `holidays`, `statusLog`, `profile`, `auth`). Confirm whether they are load-related.
- Commit pending UI tweaks: removed hover underline on dashboard card links; header breadcrumbs in `EmployeeShell`.

## 1. Leave balance and quota (high value)
**Problem:** the dashboard cannot show "X days left"; there is no quota table. `LeaveRequest.Days()` also counts calendar days and ignores weekends and holidays.
**Scope:**
- Migration: per-tenant, per-leave-type, per-year entitlement (optionally per employee override). Use the `create-migration` skill.
- Balance = entitlement minus approved days (and optionally minus pending). Count working days only (respect `weekly_off_days` and holidays).
- Admin UI to set entitlements on the leave types page.
- Backend: add `balances[]` to `GET /api/v1/dashboard/employee`; add `GET /me/leave-balances`.
- Frontend: balance tile on the dashboard and a balance strip on the Leave page; block or warn on apply when over balance.

## 2. Birthdays and work anniversaries (small, no migration)
**Data:** `date_of_birth` and `joining_date` already exist on `employee_profiles`.
**Scope:**
- Dashboard endpoint adds `celebrations[]` (name, kind, date, years) for this month, scoped by tenant.
- Dashboard card with empty state. Decide privacy: show birthday day and month only, and allow opt-out.

## 3. Notification centre
**Scope:**
- Reuse the River outbox/events from the email service to also write in-app notifications.
- Events: leave approved or rejected, new allocation, allocation ending soon, probation ending.
- Bell icon in the header with unread count, a list page, and mark-as-read.
- New table `notifications` (tenant_id, user_id, kind, payload, read_at).

## 4. Team leave calendar
**Scope:**
- Show who is out this week or month for the employee's team or manager group.
- For approvers: show "clashes with N teammates" on a leave request in the approval queue.
- Needs a team-scoped leave query (same manager or same department); decide the visibility rule first.

## 5. Dashboard reminders
**Scope:**
- Turn the unlogged-days warning into a one-click "Log now" that opens the status log for the first missing date.
- Reminder items: probation end date, allocation ending within 14 days, pending leave awaiting approval for more than N days.
- Return a `reminders[]` list from the dashboard endpoint.

## 6. Leave apply pre-check
**Scope:**
- While choosing dates, show working days counted, holidays and weekly offs inside the range, and overlap with existing requests.
- Reuse the holidays query and org weekly-off settings; no backend change is needed if done client-side, but the working-day calculation should match the backend (see item 1).

## Notes
- Follow Clean Architecture and the tenant isolation invariants in `shared-context.md`.
- Backend work goes to `backend-agent`, frontend to `frontend-agent`; cross-layer features do backend first.
- Create a Plane ticket (parent plus backend and frontend sub-tickets) per item before starting.
