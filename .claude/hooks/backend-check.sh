#!/usr/bin/env bash
# Post-edit checks for the Go backend. No-ops safely until backend/go.mod exists.
FILE=$(python3 -c 'import json,sys
try:
    print(json.load(sys.stdin).get("tool_input", {}).get("file_path", ""))
except Exception:
    print("")')

# $FILE arrives as an absolute path (Write/Edit always send one) — strip the repo
# root so matching works regardless of whether it's absolute or already relative.
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
REL_FILE="${FILE#"$REPO_ROOT"/}"

echo "$REL_FILE" | grep -q '^backend/.*\.go$' || exit 0

BACKEND_DIR="$REPO_ROOT/backend"

[ -f "$BACKEND_DIR/go.mod" ] || exit 0

cd "$BACKEND_DIR" || exit 0

gofmt -l -w "$FILE" 2>&1
go vet ./... 2>&1
