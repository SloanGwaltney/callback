# AGENTS.md

## Tech stack

Each entry links to the ADR that chose it.

- **Desktop shell:** Wails v2, with a Go backend and a web frontend. Run `wails generate module`
  after changing a bound method in `app.go` or a struct it sends or receives, to regenerate the
  bindings in `frontend/wailsjs/`. Never edit those files by hand.
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
- **AI agents:** agent features are a CLI, its own binary under `cmd/` shipped beside the app,
  that users drive from their own agent, outside the app. `--help` lists its commands, and a
  separate flag prints a prose overview for the agent. No MCP server, and no agent, model, or API
  key inside the app.
  ([docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md](docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md),
  [docs/adr/0024-give-the-agent-cli-a-prose-overview-alongside-its-help.md](docs/adr/0024-give-the-agent-cli-a-prose-overview-alongside-its-help.md))
- **Lint and format:** `gofmt` and `go vet` for Go. Biome and `tsc` for the frontend. One check
  script runs every check. The Stop hook runs it without tests, and CI runs it with tests.
  ([docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md](docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md))
- **CI:** GitHub Actions runs `scripts/check.sh` on a Windows runner for every pull request and
  push to `main`, and a ruleset on `main` blocks merging until it passes. Pin actions to commit SHAs.
  ([docs/adr/0018-enforce-the-check-script-in-github-actions-before-merging.md](docs/adr/0018-enforce-the-check-script-in-github-actions-before-merging.md))
- **Releases:** SemVer. Pushing an annotated `vX.Y.Z` tag on `main` builds a draft GitHub Release
  with unsigned Windows and Linux binaries and checksums attached. No macOS for now. Notes are written by hand in the draft before it's
  published. No `CHANGELOG.md` and no third-party release actions. Never move a pushed tag.
  ([docs/adr/0019-release-by-pushing-a-tag-that-builds-a-draft-github-release.md](docs/adr/0019-release-by-pushing-a-tag-that-builds-a-draft-github-release.md))
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
- **Board search and pagination:** done in the frontend over the whole board from `GetBoard`; card order stays in the jobs service ([docs/adr/0020-search-and-paginate-the-board-in-the-frontend.md](docs/adr/0020-search-and-paginate-the-board-in-the-frontend.md))
- **Spec links:** `@scenario` on entry points in code and on tests; step annotations in tests only, where `And`/`But` take their own keyword or the one they continue ([docs/adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md](docs/adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md))
- **Restoring and data versions:** restore with SQLite's online backup API, migrating an older backup's copy first; check the data's version before migrating, and stop on newer data ([docs/adr/0025-restore-backups-with-sqlites-online-backup-api-and-check-data-versions-first.md](docs/adr/0025-restore-backups-with-sqlites-online-backup-api-and-check-data-versions-first.md))
- **Opening files and links:** only through the hand-written opener in `internal/desktop`, never Wails' `BrowserOpenURL`, and only `http`/`https` links ([docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md](docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md))

## Dependencies

Every package is a supply chain risk: its code, its maintainers, and everything it pulls in ship
to the user's machine. Some of that risk has to be accepted, because writing everything ourselves
isn't feasible, but only when a package replaces work we can't reasonably do ourselves.

- **Never install a package yourself.** This covers Go modules, npm packages, and tools.
  `.claude/settings.json` denies `npm install`/`i`/`add`/`update`, `npx`, `go get`, `go install`,
  and `go mod tidy`. Recommend the package, and once the user agrees, give them the exact command
  to run. Run frontend tools through the npm scripts instead of `npx`: `npm --prefix frontend
  test`, `run lint`, `run typecheck`, or `bash scripts/check.sh`.
- **Justify every recommendation against that risk.** Say what the package saves us and why that's
  worth the risk compared with writing the code ourselves. If the code is small enough to write by
  hand, write it instead. See
  [docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md](docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md)
  for an example.

## Editing files on Windows

When a script edits files (Python, `sed` with inserted lines, and so on), it must keep the file's
LF line endings. Python on Windows writes CRLF in text mode, which `gofmt` and Biome then reject,
so open files with `newline=''`, or use the Edit tool instead. Keep each shell call to one step: a
long chain of heredocs fails on quoting and applies none of its edits.

## Adding something to a job

A new kind of record on a job, like its contacts or interviews, touches every layer. Copy the
closest one that exists (`internal/jobs/interviews.go` and `InterviewList` are a good start) and
go through each step:

1. **Storage.** A goose migration whose table has `job_id ... REFERENCES jobs (id) ON DELETE
   CASCADE`, a query file in `internal/db/queries/`, then `sqlc generate`. Storage tests in
   `internal/db/` for the schema's checks, the queries, the list order, and the cascade. If the
   migration also fills in rows for data already there, test it with `openBefore`
   (`internal/db/store_test.go`), which opens a database as the release before it left it.
2. **Service.** The rules in `internal/jobs/<thing>.go`, an `Err<Thing>NotFound` in `jobs.go`, the
   list on `Job` and loaded in `Service.Job` (`job.go`), and, if the board shows it, a field on
   `Card` (`board.go`).
3. **Fakes.** The new `Querier` methods on `fakeStore`, the table in `tables` and its `clone`
   (`internal/jobs/fakes_test.go`), and a `seed<Thing>` harness helper. Tests that compare a
   whole `Job` (`job_test.go`) need the new empty list.
4. **Bindings.** Methods in `app.go`, then `wails generate module`.
5. **Frontend.** The types in `frontend/src/hooks/types.ts` (and the field in `Job`'s `Omit`
   list), the default in `frontend/src/test/fixtures.ts`, labels in `frontend/src/jobs/constants.ts`,
   a hook in `frontend/src/hooks/`, and a list component in `JobDetails`. Mock the new hook in
   `JobDetails.test.tsx`.
6. **Deleting a job.** Name the new record in `JobDetails`' delete confirmation and in the comments
   on `DeleteJob` in `app.go`, `job.go`, and `internal/db/queries/jobs.sql`.
7. **Seeing it.** Let `page.seed` in `scripts/drive/cdp.mjs` add the new record to a job.

## Adding a field to a job

A new field stored on the job itself, like its interest rating or next action, is a column on
`jobs`, not a table. Copy the closest one (`interest` for a field edited with the job's details,
`internal/jobs/next_action.go` for one with its own bindings) and go through each step:

1. **Storage.** A goose migration that runs `ALTER TABLE jobs ADD COLUMN` with its checks, then
   either the column in `CreateJob` and `UpdateJobDetails` or its own query in
   `internal/db/queries/jobs.sql`, then `sqlc generate`. Storage tests for the checks and the
   queries. Check that `UpdateJobDetails` and `UpdateJobPlacement` leave the column alone if
   editing or moving a job shouldn't change it. If the migration fills in the column for jobs
   already there, test it with `openBefore` (`internal/db/store_test.go`).
2. **Service.** The field on `Job` and set in `Service.Job` (`job.go`). If it's one of `Details`,
   copy it in `Job.Details()` too, or the agent CLI's `job update` clears it. If the board shows it, it
   also goes on `Card` and in `cardFromRow` (`board.go`). Anything that depends on today comes
   from `s.today()`.
3. **Fakes.** The fake copies each field by hand, so add the column to the `ListJobsRow` built in
   `fakeStore.ListJobs`, and to `CreateJob` or `UpdateJobDetails` if they set it, in
   `internal/jobs/fakes_test.go`. Any new `Querier` methods go on `fakeStore` too. If you miss one,
   the board tests see an empty field and fail with nothing to explain why.
4. **Bindings.** If it has its own methods, add them to `app.go`. Then run `wails generate module`.
5. **Frontend.** Go sends a missing value as `null`, so in `frontend/src/hooks/types.ts` add the
   field to the `Omit` lists for `Card` and `Job` and re-add it as `| null`. If the field's own
   bindings go in `useJob`, add them to the `fakeUseJob` defaults in `frontend/src/test/fixtures.ts`.
6. **Seeing it.** Let `page.seed` in `scripts/drive/cdp.mjs` set the field on a job.

## Tests

Run `bash scripts/check.sh` after your work, and fix any failures before you finish. It runs every
check, including the tests. The Stop hook runs it with `--no-tests`, so nothing else will catch
a failing test until CI. See
[docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md](docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md).

### Language server diagnostics

Generating code (`sqlc generate`, `wails generate module`) can leave a language server out of
date. It may then report errors, such as missing fields or undefined types, that the compiler
doesn't have. When a diagnostic doesn't match what you expect, check it with `go vet`, `go test`,
`npm --prefix frontend run typecheck`, or the check script before acting on it. You may not need
the language server much here: these checks are fast, and because the codebase is grep-complete, a
text search usually finds a definition or its uses as well as "go to definition" would.

## Seeing the app

You, the agent, can run the app and drive it in headless Chrome to see a change for yourself:
take screenshots and read them, click, type, drag cards, and switch light and dark mode. It runs
without the user: nothing opens on their screen, they don't need to watch, and their data isn't
touched. Use it to check visual and interaction changes, which the unit tests can't see, before
telling the user they work.

1. Start the app in the background with `bash scripts/drive/dev.sh`. It runs `wails dev -s -m`
   with `CALLBACK_START_HIDDEN` to keep the window hidden and `CALLBACK_DATA_DIR` pointed at
   `temp/drive/appdata`, and serves the app at `http://localhost:34115`. Then run
   `bash scripts/drive/dev.sh wait`, which returns once the app is served.
2. Write a steps file and run it with `node scripts/drive/cdp.mjs <steps.mjs>`. See
   `scripts/drive/example.mjs`. Screenshots land in `temp/drive/shots`; read them to look.
3. Stop with `bash scripts/drive/dev.sh stop`. Stopping the background shell alone leaves the
   app running.

To check something that depends on the date, like days in status or a next action coming due,
start the app with `CALLBACK_TODAY=2026-10-01 bash scripts/drive/dev.sh`. The backend takes that
date as today, so what you seed is dated then. Stop it and start it again with a later date, and
the data is still there, days older. The frontend's date pickers still use the real date.

To set up the data a check needs, call `page.clearBoard()` and `page.seed([...jobs])` from the
steps file while the app runs. A seeded job can list its `interviews`, `contacts`, and
`nextAction`, and can be in any status, including `closed` with a `closingReason`; see `seed` in
`scripts/drive/cdp.mjs`. Or delete it all with `bash scripts/drive/dev.sh reset` while it's
stopped. `page.click(selector, { wait: 0 })` returns right away, to screenshot an animation
partway through. `page.select(selector, option)` chooses a `<select>` option by value or text.

Don't click "Attach a document…" or anything else that opens a native dialog; it would appear
on the user's screen. When `form.module.css` changes, re-save the stylesheets that `composes`
from it, or Vite keeps serving their old copies of its rules.

## Grep-complete

This codebase aims to be grep-complete: starting from any point of entry, you should be able to
find everything related to it with a text search. Keep it that way by making everything you add
searchable. For why, see
[docs/adr/0001-design-the-codebase-to-be-grep-complete.md](docs/adr/0001-design-the-codebase-to-be-grep-complete.md).

How we do that:

- **Spec annotations.** A scenario in `features/*.feature` is linked by a comment like
  `// @scenario [features/x.feature#<id>] <title>` on its entry points in code and on its tests,
  and each step by a comment like `// @then [features/x.feature] <step text>` in the tests, so
  grepping a spec file, scenario ID, or step text finds its implementation and tests. A step
  annotation goes on the test code that carries the step out, in the layer that does it, never
  on a comment with nothing under it. Start new features from their `@status:not-implemented` scenarios. Read
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
