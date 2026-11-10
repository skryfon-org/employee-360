#!/usr/bin/env bash
# Block edits to already-committed migration files.
# Hook input arrives on stdin as JSON; extract TargetFile or file_path from it.
FILE=$(python3 -c 'import json,sys
try:
    data = json.load(sys.stdin)
    # Check Antigravity toolCall arguments or Claude tool_input
    args = data.get("toolCall", {}).get("args", {})
    target = args.get("TargetFile") or args.get("file_path") or data.get("tool_input", {}).get("file_path", "")
    print(target or "")
except Exception:
    print("")')

# Only care about backend migration SQL files
echo "$FILE" | grep -qE 'backend/migrations/[0-9]{6}_.+\.(up|down)\.sql$' || exit 0

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

# If the file is tracked by git, block the edit
if git -C "$REPO_ROOT" ls-files --error-unmatch "$FILE" >/dev/null 2>&1; then
  printf '{"decision":"deny","reason":"Migration already committed. Create a new migration pair instead of editing this one (see the create-migration skill)."}\n'
  exit 0
fi

printf '{"decision":"allow"}\n'
