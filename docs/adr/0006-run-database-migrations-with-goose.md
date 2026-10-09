# 0006. Run database migrations with goose

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

callback stores its data in SQLite (see [docs/adr/0003-store-user-data-in-sqlite.md]) through
`mattn/go-sqlite3` and `database/sql` (see
[docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md]), and generates its queries with
sqlc (see [docs/adr/0005-generate-database-queries-with-sqlc.md]). sqlc reads the schema from the
migration files, which left open which migration tool writes and runs them.

callback is a desktop app, so migrations run on the user's machine when the app starts, not
from a command line by a developer. That puts three requirements on the tool:

- It has to run as a Go library inside the app, with the migration files embedded in the binary.
- A failed migration can't leave the database in a state that needs a manual fix, because no
  user will run a repair command.
- Down migrations matter little. The app only ever moves forward on a user's machine, and sqlc
  ignores down migrations.

sqlc officially parses the migration files of six tools: atlas, dbmate, golang-migrate, goose,
sql-migrate and tern. Rolling our own migration runner on SQLite's `PRAGMA user_version` was also
an option.

## Decisions

1. Use goose (`github.com/pressly/goose/v3`) for database migrations, written as SQL files in
   goose's format (`-- +goose Up` / `-- +goose Down`).
2. Embed the migration files in the app binary and run any pending migrations through goose's
   library when the app starts.
3. Point sqlc's `schema` setting at the goose migrations directory.

## Reasons

- **1, 2.** goose is built for this use case: it runs as a library inside a Go program, reads
  embedded migration files, and runs the pending ones as the app starts.
- **1, 2.** It wraps each migration in a transaction by default, so a failed migration rolls back
  instead of leaving the database half-changed and blocking the app from starting.
- **1.** It works on any `database/sql` connection, so it runs on the existing `mattn/go-sqlite3`
  connection without a second driver.
- **1.** It's preferred over hand-rolling because goose's open-source community has already
  surfaced and handled the edge cases around failures and rollbacks. A hand-rolled runner would
  probably handle them worse.
- **3.** goose is one of the migration formats sqlc officially supports, so no spike was run to
  check the integration, unlike sqlc itself in ADR 0005.
- **Trade-off accepted:** goose is a dependency where a few dozen lines of Go would do the basic
  job, and it comes with format rules to follow. For example, statements that contain
  semicolons, such as triggers, need `-- +goose StatementBegin` and `StatementEnd` markers, and
  numbered filenames need zero-padding because sqlc reads files in alphabetical order.

## Alternatives considered

- **Hand-rolled runner on `PRAGMA user_version`.** The other real choice. It has no dependency
  and nothing hidden. Rejected because goose already does exactly this job, and its community has
  shaken out the edge cases around recovery and rollback that a hand-rolled runner would have to
  rediscover.
- **golang-migrate.** The most popular Go migration tool, with a `mattn/go-sqlite3` driver.
  Rejected because a failed migration marks the database "dirty" and blocks all migrations until
  someone runs `migrate force`. In a desktop app, that means an app that won't start, with no one
  to fix it.
- **Atlas.** A declarative schema tool that plans migrations from a desired schema and lints
  them. Rejected as vastly more than the app needs. It also isn't built to run migrations inside
  an app, so it would need goose or golang-migrate as its runner anyway.
- **dbmate.** An SQL-only tool, mainly used as a standalone CLI. Its Go library is a secondary
  use, and its default schema dump needs the `sqlite3` CLI. It was presented but not weighed
  individually. It was rejected along with the rest as not serving the app's purpose as well as
  goose.
- **sql-migrate and tern.** Ruled out in the initial research: tern supports only Postgres, and
  sql-migrate is little used.

## Related

- universal
- [docs/adr/0003-store-user-data-in-sqlite.md] Store user data in SQLite
- [docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md] Use mattn/go-sqlite3 as the SQLite driver
- [docs/adr/0005-generate-database-queries-with-sqlc.md] Generate database queries with sqlc
