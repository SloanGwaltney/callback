#!/usr/bin/env bash
# Stop hook: runs scripts/check.sh --no-tests when Claude finishes a turn
# (docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md). If a check fails,
# Claude is sent back once to fix it. If it still fails on the next stop, the output is shown to
# the user instead, so Claude can't get stuck in a loop on something only the user can resolve.

input=$(cat)
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

out=$(bash scripts/check.sh --no-tests 2>&1)
[ $? -eq 0 ] && exit 0

if printf '%s' "$input" | grep -q '"stop_hook_active": *true'; then
  printf 'scripts/check.sh still reports problems:\n%s\n' "$out" >&2
  exit 1
fi

printf 'scripts/check.sh found problems (for spec links, see docs/spec-annotations.md). Fix them before finishing, or tell the user why you cannot:\n%s\n' "$out" >&2
exit 2
