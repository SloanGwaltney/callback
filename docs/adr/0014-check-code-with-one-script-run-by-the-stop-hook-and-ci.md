# 0014. Check code with one script, run by the Stop hook and CI

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

Before application development starts, callback needs linting, formatting, and a way to make
sure checks actually run. Agents write most of the code. The only automated check so far is
`cmd/speclint`, which a Claude Code Stop hook (`.claude/hooks/speclint-stop.sh`) runs at the end
of every agent turn (see [docs/adr/0001-design-the-codebase-to-be-grep-complete.md]).

The Stop hook runs on every turn, including turns that only touch docs. Anything it runs adds to
the time every turn takes, and a test suite gets slower as it grows. Agents already largely run
the tests after their work without being forced to.

The repository has no git remote, so there's nowhere for CI to run yet.

Every dependency is a supply chain risk and has to be justified (see `AGENTS.md`). The options
looked at:

- **Go.** `gofmt` and `go vet` ship with Go and add no dependency. staticcheck is a single widely
  used tool that catches bugs `vet` misses. golangci-lint bundles dozens of linters and has the
  largest tool and configuration surface.
- **Frontend.** TypeScript type checking already runs as part of `npm run build`. For linting and
  formatting, ESLint 10.12.0 has 30 direct dependencies before adding typescript-eslint, a React
  hooks plugin, and Prettier. Biome 2.5.15 is a single package that installs one prebuilt binary
  for the current OS and has no other dependencies. It both lints and formats, and includes the
  React hooks rules `useExhaustiveDependencies` and `useHookAtTopLevel`, which matter because the
  frontend wraps its data access in custom hooks (see
  [docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md]).

## Decisions

1. Format and lint Go with `gofmt` and `go vet` only.
2. Lint and format the frontend with Biome, and type-check it with `tsc`.
3. Keep one check script that runs every check: Go formatting, `go vet`, `tsc`, Biome,
   `cmd/speclint`, and the Go and frontend tests. The script can be told to skip the tests.
4. The Claude Code Stop hook runs the check script without tests at the end of every agent turn.
5. Tests are enforced by CI, which runs the full check script once the repository has a remote.
6. `AGENTS.md` tells agents to run the tests after their work.

## Reasons

- **1.** The built-in tools add no dependency. staticcheck can be added later if `vet` turns out
  to miss things.
- **2.** Biome covers linting and formatting with one package and no transitive dependencies,
  against dozens for ESLint with its plugins and Prettier. Linting and formatting aren't
  reasonable to write ourselves, so one dependency here is justified. It also includes the hooks
  rules the hook-based data layer needs.
- **3.** One script is the single definition of "the checks pass", so the hook, a developer, and
  CI all run the same thing.
- **4.** Formatting, vet, type checking, linting, and speclint are fast, so running them on every
  turn costs little and an agent can't finish a turn with those failures.
- **4, 5.** Tests are kept out of the Stop hook because they'd slow every turn, including turns
  with no code changes, and more so as the suite grows. CI is where they're enforced.
- **6.** Agents already largely run the tests after their work, so stating it in `AGENTS.md`
  makes that the expected practice without slowing every turn. Accepted trade-off: until the
  repository has a remote and CI, nothing enforces the tests. An agent that skips them can
  finish a turn with failing tests.

## Alternatives considered

- **staticcheck for Go.** Rejected for now; `vet` comes first, and staticcheck can be added if
  it falls short.
- **golangci-lint.** Rejected as the largest tool and configuration surface for a codebase this
  size.
- **ESLint and Prettier.** The most common setup, with the most training data. Rejected because of
  the number of dependencies compared with Biome.
- **`tsc` only, no linter or formatter.** Rejected because it misses hooks mistakes and leaves
  formatting unenforced.
- **Everything, tests included, in the Stop hook.** Nothing would slip through. Rejected because
  it slows every turn, including ones with no code changes, and gets slower as tests grow.
- **Script only, with no hook changes.** Rejected because nothing would make agents run the fast
  checks.
- **GitHub Actions now.** Rejected because there's no remote to run it on yet.

## Related

- universal
- [docs/adr/0001-design-the-codebase-to-be-grep-complete.md] Design the codebase to be grep-complete
- [docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md] Require unit tests and test storage against real SQLite
- [docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md] Wrap backend calls in custom React hooks
