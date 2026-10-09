# AGENTS.md

## Tech stack

Each entry links to the ADR that chose it.

- **Desktop shell:** Wails v2, with a Go backend and a web frontend
  ([docs/adr/0002-use-wails-for-the-desktop-shell.md](docs/adr/0002-use-wails-for-the-desktop-shell.md))
- **Database:** SQLite, embedded in the app
  ([docs/adr/0003-store-user-data-in-sqlite.md](docs/adr/0003-store-user-data-in-sqlite.md))
- **SQLite driver:** `mattn/go-sqlite3` through `database/sql`. Needs CGO.
  ([docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md](docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md))
- **Queries:** sqlc generates Go from `.sql` files. Run `sqlc generate` after changing queries or
  the schema.
  ([docs/adr/0005-generate-database-queries-with-sqlc.md](docs/adr/0005-generate-database-queries-with-sqlc.md))
- **Migrations:** goose, with migration files embedded in the app and run when it starts
  ([docs/adr/0006-run-database-migrations-with-goose.md](docs/adr/0006-run-database-migrations-with-goose.md))
- **Frontend:** a React single page app in TypeScript, built with Vite. No meta-framework.
  ([docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md](docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md))
- **Frontend data:** custom React hooks in `frontend/src/hooks/` wrap the Wails bindings, and
  only those hooks (and tests) import bindings. Biome's `noRestrictedImports` enforces it. No
  TanStack Query. For shared state, use Context or a hand-written store first, then
  Zustand only if justified.
  ([docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md](docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md))
- **Drag and drop:** `@dnd-kit/core` and `@dnd-kit/sortable` for moving and reordering board
  cards, with keyboard support.
  ([docs/adr/0013-use-dnd-kit-for-moving-and-reordering-board-cards.md](docs/adr/0013-use-dnd-kit-for-moving-and-reordering-board-cards.md))
- **Lint and format:** `gofmt` and `go vet` for Go. Biome and `tsc` for the frontend. One check
  script runs every check. The Stop hook runs it without tests, and CI runs it with tests.
  ([docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md](docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md))
- **Styling:** CSS Modules, one `*.module.css` per component, using CSS variables from a global
  tokens file. No Tailwind or shadcn/ui. Write our own components, using native HTML elements
  like `<dialog>` for accessible behavior.
  ([docs/adr/0011-style-the-frontend-with-css-modules-and-css-variables.md](docs/adr/0011-style-the-frontend-with-css-modules-and-css-variables.md))
- **Testing:** every change ships with unit tests that use hand-written fakes instead of storage.
  Only the storage layer is tested against a real temporary SQLite database and directory. On the
  frontend, Vitest and React Testing Library, with the Wails bindings mocked. No end-to-end tests
  for now.
  ([docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md](docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md))

## Code conventions

One line each. Each entry links to the ADR that set it.

- **Business rules:** in service packages under `internal/`, over a `store.Querier` + `InTx` interface; `app.go` bindings only call services ([docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md](docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md))
- **Opening files and links:** only through the hand-written opener in `internal/desktop`, never Wails' `BrowserOpenURL`, and only `http`/`https` links ([docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md](docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md))

## Dependencies

Every package is a supply chain risk: its code, its maintainers, and everything it pulls in ship
to the user's machine. Some of that risk has to be accepted, because writing everything ourselves
isn't feasible, but only when a package replaces work we can't reasonably do ourselves.

- **Never install a package without asking.** This covers Go modules, npm packages, and tools.
- **Justify every recommendation against that risk.** Say what the package saves us and why that's
  worth the risk compared with writing the code ourselves. If the code is small enough to write by
  hand, write it instead. See
  [docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md](docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md)
  for an example.

## Tests

Run `bash scripts/check.sh` after your work, and fix any failures before you finish. It runs every
check, including the tests. The Stop hook runs it with `--no-tests`, so nothing else will catch
a failing test until CI. See
[docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md](docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md).

## Grep-complete

This codebase aims to be grep-complete: starting from any point of entry, you should be able to
find everything related to it with a text search. Keep it that way by making everything you add
searchable. For why, see
[docs/adr/0001-design-the-codebase-to-be-grep-complete.md](docs/adr/0001-design-the-codebase-to-be-grep-complete.md).

How we do that:

- **Spec annotations.** Code and tests that implement a scenario in `features/*.feature` carry
  comments like `// @scenario [features/x.feature#<id>] <title>` and `// @then [features/x.feature]
  <step text>`, so grepping a spec file, scenario ID, or step text finds its implementation and
  tests. Start new features from their `@status:not-implemented` scenarios. Read
  [docs/spec-annotations.md](docs/spec-annotations.md) before implementing or testing a scenario.
- **Decision records.** Architectural decisions live in `docs/adr/` and are written with the `adr`
  skill. Each ADR's Related section lists the scenarios it affects as
  `[features/x.feature#<id>]`, the same form as code annotations, or `universal` for app-wide
  decisions, so grepping a scenario ID finds the decisions behind it. Only the user starts an
  ADR, by running `/adr`. Don't create or edit files in `docs/adr/` on your own; if a decision
  looks worth recording, suggest that the user run `/adr`.
- **Linted links.** A Stop hook (`.claude/hooks/check-stop.sh`) runs `scripts/check.sh
  --no-tests`, which includes `cmd/speclint`, whenever you finish a turn. If it reports problems,
  you'll be sent back to fix them.
