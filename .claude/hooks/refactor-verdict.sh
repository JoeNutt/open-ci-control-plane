#!/usr/bin/env bash
# Stop / SubagentStop. Blocks a turn that edited source without a refactor verdict.
set -uo pipefail

payload="$(cat)"
dir="${CLAUDE_PROJECT_DIR:-.}/.claude/state"

session="$(printf '%s' "$payload" | python3 -I -c '
import json, sys
print(json.load(sys.stdin).get("session_id", "unknown"))
' 2>/dev/null)" || exit 0

marker="$dir/edited-$session"
[ -s "$marker" ] || exit 0

if printf '%s' "$payload" | python3 -I -c '
import json, sys, re
msg = json.load(sys.stdin).get("last_assistant_message") or ""
sys.exit(0 if re.search(r"(Refactored:|No refactor needed:)", msg) else 1)
' 2>/dev/null; then
    rm -f "$marker"
    exit 0
fi

files="$(sort -u "$marker" | head -10 | sed 's/^/  - /')"

cat <<JSON
{
  "decision": "block",
  "reason": "Refactor step not completed. Source was edited this turn and no refactor verdict was stated.\n\nEdited:\n${files//$'\n'/\\n}\n\nThe cycle is red, green, REFACTOR. Code at green is unrefactored by construction. Work the five questions now, reading the entry rather than recalling it:\n\n1. Can a design pattern express this better?  knowledge-base/design-patterns/README.md\n2. Are the architectural boundaries right?    knowledge-base/architecture/README.md\n3. Is this clean code?                        knowledge-base/engineering/README.md\n4. Is hexagonal design applied or invoked?    knowledge-base/architecture/ports-and-adapters.md\n5. Does this defend the OWASP Top 10?         knowledge-base/security/owasp-top-10/README.md\n\nThen end with exactly one of:\n  Refactored: <what changed and why>\n  No refactor needed: checked <which questions>, none fired."
}
JSON
exit 0
