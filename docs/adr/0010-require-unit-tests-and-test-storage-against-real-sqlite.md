# 0010. Require unit tests and test storage against real SQLite

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

Before application development starts, callback needs a testing approach for both sides of the
app: the Go backend (see [docs/adr/0002-use-wails-for-the-desktop-shell.md]) and the React
frontend (see [docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md]).

Tests carry the same `@scenario` and step annotations as code (see `docs/spec-annotations.md`),
so they're part of what keeps the codebase grep-complete. Agents write much of the code, and they
can write unit tests and annotate them easily.

The app stores data in SQLite through sqlc-generated queries and goose migrations (see
[docs/adr/0003-store-user-data-in-sqlite.md], [docs/adr/0005-generate-database-queries-with-sqlc.md],
[docs/adr/0006-run-database-migrations-with-goose.md]), and stores attached documents as files
(see [docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md]). Tests that
mock storage never run the real SQL, migrations, or file handling, so mistakes there (a wrong
query, a missing cascade, files and rows getting out of step) can only be caught by tests that
use real storage.

Driving the real UI end to end is possible but awkward. `wails dev` serves the frontend at
`localhost:34115` with working Go bindings, so a browser automation tool such as Playwright can
drive it in a regular browser. On Windows, Playwright can also attach to the real WebView2 over
the Chrome DevTools Protocol, but macOS's WKWebView offers no equivalent. Both routes add a large
dependency and slow tests. The app currently has three features and one developer.

The frontend has no test runner installed.

## Decisions

1. Every change ships with unit tests. There's no coverage target.
2. Unit tests don't touch storage. Code above the storage layer is tested against fakes of the
   storage interfaces, such as the `Querier` interface sqlc can generate.
3. Storage fakes in Go are written by hand. Don't use a mock generator.
4. The storage layer itself (queries, migrations, and document file handling) is tested in Go
   against a real SQLite database and a real directory, both temporary per test.
5. Frontend unit tests use Vitest and React Testing Library, with the Wails bindings mocked.
6. Don't write end-to-end tests that drive the UI for now.

## Reasons

- **1.** Requiring unit tests makes sense in itself, and agents can write them cheaply and
  annotate them with the spec annotations, so the tests also make each scenario's behavior
  findable. Full coverage isn't the goal. The requirement is that code comes with tests, not
  that every line is covered.
- **2.** Keeping storage out of unit tests keeps them fast and focused on the logic under test.
- **3.** Every dependency is a supply chain risk (see `AGENTS.md`), and fakes for our storage
  interfaces are small enough to write by hand. Accepted trade-off: more boilerplate than
  generated mocks if the interfaces grow large.
- **4.** Mocked unit tests never run the real SQL, migrations, or file handling, so this tier is
  the only automated check that they work. Limiting real storage to the storage layer's own tests
  keeps decision 2 intact for everything above it. It uses `go test` and the SQLite driver the
  app already has, so it adds no dependencies, and temporary databases and directories keep
  tests away from the user's real data.
- **5.** Vitest runs on the existing Vite config. A test runner can't reasonably be written by
  hand, so its supply chain risk is worth accepting. React Testing Library renders components and
  queries them the way a user sees them, which is what the frontend's scenarios describe, and
  rendering and querying components isn't reasonable to write ourselves either.
- **6.** At the app's current scale, the value of UI-driven tests doesn't justify the cost: a
  large dependency (Playwright), slow tests, and no way to drive the real webview on macOS.
  Accepted trade-off: wiring between the frontend and the Go bindings is only checked by running
  the app.

## Alternatives considered

- **Unit tests only, with no storage-layer tests.** Rejected because SQL, migrations, and
  document file handling would then be verified only by running the app.
- **Generated mocks (mockery or gomock).** Less boilerplate for large interfaces. Rejected
  because they add a dependency and generated code for something we can write by hand.
- **Vitest without React Testing Library.** Would test hooks and logic but not rendered
  components. Rejected because the frontend's scenarios are about what the user sees.
- **End-to-end tests with Playwright,** either against `wails dev` in a browser or attached to
  WebView2 on Windows. Rejected for now for the reasons in 6. Worth revisiting if the app grows or
  regressions in the frontend-to-Go wiring slip through.

## Related

- universal
- [docs/adr/0001-design-the-codebase-to-be-grep-complete.md] Design the codebase to be grep-complete
- [docs/adr/0003-store-user-data-in-sqlite.md] Store user data in SQLite
- [docs/adr/0005-generate-database-queries-with-sqlc.md] Generate database queries with sqlc
- [docs/adr/0006-run-database-migrations-with-goose.md] Run database migrations with goose
- [docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md] Build the frontend as a React SPA with TypeScript and Vite
- [docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md] Store attached documents as files in the app data directory
- [docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md] Keep user data in the OS-conventional app data directory
