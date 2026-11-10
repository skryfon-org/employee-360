#!/usr/bin/env bash
# Post-edit checks for the Go backend. No-ops safely until backend/go.mod exists.
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

echo "$REL_FILE" | grep -q '^backend/.*\.go$' || { printf '{}\n'; exit 0; }

BACKEND_DIR="$REPO_ROOT/backend"
[ -f "$BACKEND_DIR/go.mod" ] || { printf '{}\n'; exit 0; }

cd "$BACKEND_DIR" || { printf '{}\n'; exit 0; }

gofmt -l -w "$FILE" >/dev/null 2>&1
go vet ./... >/dev/null 2>&1

printf '{}\n'
