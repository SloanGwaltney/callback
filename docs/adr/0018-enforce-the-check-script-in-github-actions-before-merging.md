# 0018. Enforce the check script in GitHub Actions before merging

- **Status:** Accepted
- **Date:** 2026-10-10

## Context

[docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md] made `scripts/check.sh`
the single definition of "the checks pass" and said CI would run it, tests included, once the
repository had a remote. Until then the checks only ran locally: the Stop hook runs them without
tests, and nothing enforced the tests at all. Code could reach `main` without meeting the
formatting, lint, type, spec-link, or test standards.

The repository now has a remote on GitHub, and it's public.

The app is built with Wails v2 and uses `mattn/go-sqlite3`, which needs CGO. `main.go` embeds
`frontend/dist`, which is gitignored, so a fresh checkout can't compile the main package until the
frontend is built. On Linux, Wails v2 compiles against the WebKitGTK and GTK system headers,
which GitHub's Ubuntu runners don't ship. Current Ubuntu also only packages WebKitGTK 4.1, which
Wails v2 only uses when built with the `webkit2_41` tag, and `scripts/check.sh` passes no build
tags. On Windows, Wails needs no system headers, and GitHub's Windows runner image already ships
the gcc that CGO needs.

## Decisions

1. Run `scripts/check.sh`, tests included, in CI on every pull request and every push to `main`.
2. Use GitHub Actions for CI (`.github/workflows/check.yml`).
3. Run the job on a GitHub-hosted Windows runner, building the frontend before the check script.
4. Pin every action to a commit SHA, with the release tag in a comment.
5. Trigger on `pull_request`, never `pull_request_target`, with a read-only token and
   `persist-credentials: false`.
6. Cap the job at 10 minutes with `timeout-minutes`.
7. A repository ruleset on `main` requires a pull request and a passing `check` job, with the
   branch up to date with `main`, and has an empty bypass list.

## Reasons

- **1.** The standards in the check script are only worth having if they hold for everything that
  merges, not just for code from a machine where the hook ran. Checking at the repository level,
  before merge, keeps the code up to standard no matter who or what wrote it.
- **2.** The code is hosted on GitHub, so GitHub Actions is the reasonable default: it's built in,
  free for a public repository, and its results feed straight into the merge rules.
- **3.** Windows is the path of least resistance: the check script runs there as it is, with no
  system packages to install and no build tags to add, because Wails needs no WebKitGTK headers
  and the runner image ships the gcc that `mattn/go-sqlite3` needs. The frontend is built first because `main.go` embeds the gitignored `frontend/dist`, and
  `go vet` and `go test` can't compile the main package without it. Trade-off: Windows runners
  are slower than Linux ones, but a run takes about 2 minutes.
- **4.** Every action is a supply chain risk (see `AGENTS.md`). The three used, `actions/checkout`,
  `actions/setup-go`, and `actions/setup-node`, are GitHub's own, and setting up Go and Node by
  hand isn't worth writing. A tag can be moved to different code; a commit SHA can't.
- **5.** A pull request from a fork runs under `pull_request` with a read-only token and no
  secrets, so a fork that edits the workflow or the check script can't write to the repository.
  `pull_request_target` runs with the base repository's write token and secrets, and running fork
  code under it is the known "pwn request" hole.
- **6.** The default timeout is 6 hours. A run takes about 2 minutes, so 10 minutes caps a hung
  run, including one caused on purpose by a pull request, well before it ties up a runner for
  hours.
- **7.** Without the ruleset, a failing check is only a warning on the pull request. An empty
  bypass list means nobody, owner included, can merge a failing pull request or push to `main`
  directly. Accepted trade-off: every change, even a one-line fix, goes through a pull request.
  The owner can still disable the ruleset in the repository settings; GitHub has no way to stop
  that, but it takes a deliberate step and is recorded.

## Alternatives considered

- **Other CI providers.** Not evaluated in depth; GitHub Actions is the default for a repository
  hosted on GitHub.
- **A Linux runner.** Faster and the more common choice. Rejected because Wails v2 needs the
  WebKitGTK and GTK headers installed to compile there, and on current Ubuntu the check script
  would also have to pass the `webkit2_41` build tag. CI isn't meant to match any developer's
  machine, so this can be revisited if Linux setup becomes simpler than Windows.
- **Triggering only on pull requests, not on pushes to `main`.** Considered after the fact. Kept
  the push trigger because a run takes about 2 minutes, so rechecking `main` costs little.
- **Letting the owner bypass the ruleset.** Rejected so that a failing check is a blocker with no
  workaround short of turning the ruleset off.
- **Requiring approving reviews.** Not required, so the owner can merge their own pull requests
  once the check passes.

## Related

- universal
- [docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md] Check code with one script, run by the Stop hook and CI
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
- [docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md] Use mattn/go-sqlite3 as the SQLite driver
