#!/usr/bin/env bash
# PreToolUse on Bash(git commit *). Gates the commit boundary.
set -uo pipefail

payload="$(cat)"
root="${CLAUDE_PROJECT_DIR:-$(pwd)}"
cd "$root" || exit 0

command_text="$(printf '%s' "$payload" | python3 -I -c '
import json, sys
print(((json.load(sys.stdin).get("tool_input") or {}).get("command") or ""))
' 2>/dev/null)" || exit 0

deny() {
    python3 -I -c '
import json, sys
print(json.dumps({"hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": sys.argv[1],
}}))' "$1"
    exit 0
}

case "$command_text" in
  *--no-verify*|*-n\ *)
      deny "git commit --no-verify bypasses the project's own gates. Fix the finding instead." ;;
esac

staged="$(git diff --cached --name-only --diff-filter=ACM 2>/dev/null)" || exit 0
[ -n "$staged" ] || exit 0

prod_go="$(printf '%s\n' "$staged" | grep -E '\.go$' | grep -vE '_test\.go$' || true)"
test_go="$(printf '%s\n' "$staged" | grep -E '_test\.go$' || true)"
if [ -n "$prod_go" ] && [ -z "$test_go" ]; then
    deny "Staged production Go changes with no staged test change:
$(printf '%s\n' "$prod_go" | sed 's/^/  - /')

Red comes first. Either stage the test that drove this change, or state why the code is untestable."
fi

# The toolchain comes from the flake, so run the Go gates inside it when go is
# not already on PATH. A missing toolchain is reported as such, never as a
# failing check.
if [ -f go.mod ]; then
    if command -v go >/dev/null 2>&1; then
        run() { bash -c "$1" 2>&1; }
    elif [ -f flake.nix ] && command -v nix >/dev/null 2>&1; then
        run() { nix develop --command bash -c "$1" 2>&1; }
    else
        deny "go.mod is present but neither the go toolchain nor nix is available, so the commit gate cannot verify this change."
    fi

    if ! unformatted="$(run 'gofmt -l .')"; then
        deny "gofmt could not run:

$unformatted"
    fi
    unformatted="$(printf '%s\n' "$unformatted" | grep -E '\.go$' || true)"
    [ -n "$unformatted" ] && deny "gofmt reports unformatted files:
$(printf '%s\n' "$unformatted" | sed 's/^/  - /')"

    if ! vet_out="$(run 'go vet ./...')"; then
        deny "go vet failed:

$(printf '%s' "$vet_out" | tail -40)"
    fi

    if ! test_out="$(run 'go test ./...')"; then
        deny "Tests are not green, so there is nothing to commit:

$(printf '%s' "$test_out" | tail -40)"
    fi
fi

exit 0
