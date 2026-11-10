#!/usr/bin/env bash
# Post-edit checks for frontend (TypeScript / Vite). No-ops safely until pnpm-workspace.yaml exists.
FILE=$(python3 -c 'import json,sys
try:
    data = json.load(sys.stdin)
    args = data.get("toolCall", {}).get("args", {})
    target = args.get("TargetFile") or args.get("file_path") or data.get("tool_input", {}).get("file_path", "")
    print(target or "")
except Exception:
    print("")')

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
REL_FILE="${FILE#"$REPO_ROOT"/}"

echo "$REL_FILE" | grep -qE '^(clients|packages)/.*\.(ts|tsx|js|jsx)$' || { printf '{}\n'; exit 0; }

[ -f "$REPO_ROOT/pnpm-workspace.yaml" ] || { printf '{}\n'; exit 0; }

cd "$REPO_ROOT" || { printf '{}\n'; exit 0; }

# Run typecheck quietly if pnpm is available
if command -v pnpm >/dev/null 2>&1; then
  pnpm typecheck >/dev/null 2>&1 || true
fi

printf '{}\n'
