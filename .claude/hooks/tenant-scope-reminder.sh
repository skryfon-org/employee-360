#!/usr/bin/env bash
# Remind about tenant-scoping invariants when touching persistence or handler code.
# Hook input arrives on stdin as JSON; read file_path from it.
FILE=$(python3 -c 'import json,sys
try:
    print(json.load(sys.stdin).get("tool_input", {}).get("file_path", ""))
except Exception:
    print("")')

case "$FILE" in
  *internal/infrastructure/persistence/*_repo.go)
    printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"Reminder: every query/mutation in a persistence repo must scope by tenant_id pulled from context (ctx.TenantIDFromContext) — never accept or trust a tenant ID passed in as a parameter."}}\n'
    ;;
  *internal/delivery/http/handlers/*_handler.go)
    printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"Reminder: handlers must not call GORM/repositories directly — go through a usecase. Admin-only routes need an explicit role check in addition to tenant-scoping middleware; tenant scoping alone is not authorization."}}\n'
    ;;
  *)
    exit 0
    ;;
esac
