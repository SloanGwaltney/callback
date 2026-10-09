#!/usr/bin/env bash
# Stop hook: runs speclint (cmd/speclint) when Claude finishes a turn. If it finds problems, Claude
# is sent back once to fix them. If they're still there on the next stop, they're shown to the
# user instead, so Claude can't get stuck in a loop on something only the user can resolve.

input=$(cat)
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

out=$(go run ./cmd/speclint 2>&1)
[ $? -eq 0 ] && exit 0

# The linter itself failed to build or run: report it without blocking.
if ! printf '%s' "$out" | grep -q '^speclint: [0-9]* problem(s)$'; then
  printf 'speclint could not run:\n%s\n' "$out" >&2
  exit 1
fi

if printf '%s' "$input" | grep -q '"stop_hook_active": *true'; then
  printf 'speclint still reports problems:\n%s\n' "$out" >&2
  exit 1
fi

printf 'speclint found spec-linkage problems (see docs/spec-annotations.md). Fix them before finishing, or tell the user why you cannot:\n%s\n' "$out" >&2
exit 2
