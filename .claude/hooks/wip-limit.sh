#!/usr/bin/env bash
# PreToolUse on Bash. Enforces one In Progress card per agent on the delivery board.
set -uo pipefail

root="${CLAUDE_PROJECT_DIR:-$(pwd)}"
board="$root/.claude/board.json"
hooks="$root/.claude/hooks"
[ -r "$board" ] || exit 0

payload="$(cat)"

cmd="$(printf '%s' "$payload" | python3 -I -c '
import json, sys
print(((json.load(sys.stdin).get("tool_input") or {}).get("command") or ""))
' 2>/dev/null)" || exit 0

case "$cmd" in
  *"project item-edit"*|*updateProjectV2ItemFieldValue*) ;;
  *) exit 0 ;;
esac

project_id="$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["projectId"])' "$board")" || exit 0

snapshot="$(mktemp)"
trap 'rm -f "$snapshot"' EXIT

gh api graphql -f query='
query($id: ID!) {
  node(id: $id) {
    ... on ProjectV2 {
      items(first: 100) {
        nodes {
          id
          content { ... on Issue { number title } }
          fieldValues(first: 20) {
            nodes {
              ... on ProjectV2ItemFieldSingleSelectValue {
                optionId
                field { ... on ProjectV2SingleSelectField { id } }
              }
            }
          }
        }
      }
    }
  }
}' -F id="$project_id" > "$snapshot" 2>/dev/null || exit 0

python3 -I "$hooks/wip_limit.py" "$board" "$snapshot" "$cmd"
