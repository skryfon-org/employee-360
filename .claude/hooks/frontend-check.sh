#!/usr/bin/env bash
# Post-edit checks for the pnpm frontend workspace. No-ops safely until it's scaffolded.
FILE=$(python3 -c 'import json,sys
try:
    print(json.load(sys.stdin).get("tool_input", {}).get("file_path", ""))
except Exception:
    print("")')

# $FILE arrives as an absolute path (Write/Edit always send one) — strip the repo
# root so matching works regardless of whether it's absolute or already relative.
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
REL_FILE="${FILE#"$REPO_ROOT"/}"

case "$REL_FILE" in
  clients/admin/*)    FILTER="admin" ;;
  clients/employee/*) FILTER="employee" ;;
  packages/api-client/*) FILTER="api-client" ;;
  packages/ui/*)       FILTER="ui" ;;
  *) exit 0 ;;
esac

echo "$REL_FILE" | grep -qE '\.(ts|tsx)$' || exit 0

[ -f "$REPO_ROOT/pnpm-workspace.yaml" ] || exit 0
command -v pnpm >/dev/null 2>&1 || exit 0

cd "$REPO_ROOT" || exit 0

pnpm --filter "$FILTER" format 2>&1
pnpm --filter "$FILTER" typecheck 2>&1
pnpm --filter "$FILTER" lint 2>&1
