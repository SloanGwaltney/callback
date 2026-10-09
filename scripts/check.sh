#!/usr/bin/env bash
# Runs every check on the codebase (docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md):
# gofmt, go vet, speclint, tsc, and the Go tests. Every check runs even if an earlier one fails,
# and the script exits 1 if any failed.
#
# Usage: scripts/check.sh [--no-tests]
#   --no-tests  skip the tests (the Stop hook runs it this way; CI runs it without)

run_tests=1
for arg in "$@"; do
  case "$arg" in
    --no-tests) run_tests=0 ;;
    *) printf 'usage: scripts/check.sh [--no-tests]\n' >&2; exit 2 ;;
  esac
done

cd "$(dirname "$0")/.." || exit 2

failed=()

# check <name> <command...>: runs the command and records <name> as failed if it exits non-zero.
check() {
  local name=$1; shift
  local out
  if ! out=$("$@" 2>&1); then
    printf '\n== %s failed ==\n%s\n' "$name" "$out"
    failed+=("$name")
  fi
}

gofmt_check() {
  # Tracked and untracked .go files, minus anything ignored (like node_modules).
  local files unformatted
  mapfile -t files < <(git ls-files --cached --others --exclude-standard -- '*.go')
  unformatted=$(gofmt -l "${files[@]}") || return 1
  if [ -n "$unformatted" ]; then
    printf 'not gofmt-formatted (run gofmt -w on them):\n%s\n' "$unformatted"
    return 1
  fi
}

check gofmt gofmt_check
check "go vet" go vet ./...
check speclint go run ./cmd/speclint
check tsc npx --prefix frontend --no-install tsc --noEmit -p frontend

# Biome isn't set up yet. The user will add it; uncomment this then.
# check biome npx --prefix frontend --no-install biome check frontend

if [ "$run_tests" -eq 1 ]; then
  check "go test" go test ./...
  # Frontend tests aren't set up yet. The user will add them; uncomment this then.
  # check "frontend tests" npm --prefix frontend test
fi

if [ ${#failed[@]} -gt 0 ]; then
  printf '\ncheck: %d failed: %s\n' "${#failed[@]}" "$(IFS=,; echo "${failed[*]}")"
  exit 1
fi
printf 'check: all passed\n'
