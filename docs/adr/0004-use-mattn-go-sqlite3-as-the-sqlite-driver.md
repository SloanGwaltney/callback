# 0004. Use mattn/go-sqlite3 as the SQLite driver

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

callback stores the user's data in an embedded SQLite database (see
[docs/adr/0003-store-user-data-in-sqlite.md]), which left open which Go SQLite driver to use. Go
has several, which fall into two groups: drivers that use CGO to compile SQLite's C source, and
pure-Go drivers that run SQLite without a C compiler.

Whether CGO is already required depends on the platform. Wails v2 (see
[docs/adr/0002-use-wails-for-the-desktop-shell.md]) needs CGO on macOS and Linux, but not on
Windows: as of this decision, the app builds on Windows with `CGO_ENABLED=0` and fails to build
that way for macOS and Linux. So on Windows, a CGO driver would be the first thing in the app to
require a C compiler.

The app is currently developed mostly on Windows, but development isn't meant to be
Windows-only, and the CI/CD pipeline is meant to build for Linux and macOS as well.

For an app of this size, a job board with hundreds to a few thousand rows, any of the drivers is
fast enough that the user wouldn't notice the difference.

## Decisions

1. Use `github.com/mattn/go-sqlite3` as the SQLite driver.
2. Access SQLite through Go's `database/sql` package.
3. Require a C compiler to build the app on every platform, Windows included.

## Reasons

- **1, 3.** Wails already requires a C compiler on macOS and Linux, and decisions shouldn't assume
  development happens only on Windows. Once Windows is the only platform where a pure-Go driver
  avoids a C compiler, that advantage is small.
- **1.** With the C compiler requirement accepted, mattn/go-sqlite3 has no real downside. It runs
  SQLite's own C code, so it's the fastest option, and it's the most widely used and
  battle-tested Go SQLite driver.
- **3.** The trade-off accepted is that Windows developers need a C compiler (such as MinGW-w64
  gcc) that Wails alone wouldn't require. Most developers coming from Go or C will already have
  one, the requirement is well documented, and it can be listed in the README.
- **2.** mattn/go-sqlite3 is a `database/sql` driver, so the code depends on the standard
  interface rather than the driver.

## Alternatives considered

- **modernc.org/sqlite**, SQLite's C source converted to Go. It's pure Go, so Windows builds
  wouldn't need a C compiler, but it's slower than the C original. Rejected because avoiding a C
  compiler only on Windows doesn't justify giving up speed when macOS and Linux need one anyway.
- **ncruces/go-sqlite3**, SQLite compiled to WebAssembly and run in Go through the wazero
  runtime. Rejected because taking on a WebAssembly runtime only to run SQLite wasn't worth it,
  and it offers no advantage that matters here over a CGO driver or a pure-Go one.
- **crawshaw.io/sqlite** and **zombiezen.com/go/sqlite**, which have their own APIs instead of
  `database/sql`. crawshaw.io/sqlite is effectively unmaintained, and neither works with
  `database/sql`, so neither was pursued.

## Related

- universal
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
- [docs/adr/0003-store-user-data-in-sqlite.md] Store user data in SQLite
