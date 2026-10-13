#!/usr/bin/env bash
# PostToolUse on Edit|Write. Records source edits for refactor-verdict.sh.
set -uo pipefail

payload="$(cat)"

read -r session path <<<"$(printf '%s' "$payload" | python3 -I -c '
import json, sys
d = json.load(sys.stdin)
ti = d.get("tool_input") or {}
print(d.get("session_id", "unknown"), ti.get("file_path", ""))
' 2>/dev/null)" || exit 0

case "$path" in
  *.go|*.rego) ;;
  *) exit 0 ;;
esac

case "$path" in
  *_test.go) exit 0 ;;
esac

dir="${CLAUDE_PROJECT_DIR:-.}/.claude/state"
mkdir -p "$dir"
printf '%s\n' "$path" >> "$dir/edited-$session"
exit 0
