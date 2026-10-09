# 0005. Generate database queries with sqlc

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

callback stores its data in SQLite (see [docs/adr/0003-store-user-data-in-sqlite.md]) through
`mattn/go-sqlite3` and `database/sql` (see
[docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md]). That left open how the app's Go
code writes and runs its queries. The options fall into four groups: raw `database/sql`, thin
mapping layers such as sqlx, query builders, and ORMs.

Much of the code will be written by agents, and the codebase is meant to be grep-complete (see
[docs/adr/0001-design-the-codebase-to-be-grep-complete.md]).

The app's querying needs are modest: a job board with fixed queries, such as loading the board,
moving a card, and attaching a document, and not much dynamic querying.

The final choice was between GORM and sqlc. Strictly right-sized for an app this small, GORM
would almost certainly be the answer, and sqlc is somewhat more than the app needs.

sqlc reads the database schema from the migration files. Which migration tool to use is a
separate decision and isn't covered here.

## Decisions

1. Write queries as SQL in `.sql` files, and use sqlc (`github.com/sqlc-dev/sqlc`) to generate
   typed Go code from them.
2. Run `sqlc generate` whenever queries or the schema change, and instruct agents to run it.

## Reasons

- **1.** sqlc checks queries against the schema when code is generated, and the generated code
  fails to compile if it's out of date. That compile-time safety was the main draw.
- **1.** sqlc is explicit. The SQL that runs is the SQL in the `.sql` file, with no hidden
  behavior. Less hidden behavior means less to trip up on, for agents and humans alike.
- **1.** For agent-written code, GORM's much larger body of training data and sqlc's explicitness
  were judged roughly even. With them even, sqlc's lack of hidden behavior decided it.
- **1.** It removes the hand-written `Scan` and mapping code that raw `database/sql` needs.
- **1.** It keeps the codebase grep-complete. Comments under a query's `-- name:` line, including
  spec annotations, are copied onto the generated Go method as doc comments. The query name also
  appears in the `.sql` file, the Go method, and the SQL string that runs.
- **2.** The generate step is a small cost. Agents can be told to run it. If a human forgets,
  their code won't compile.
- **Trade-off accepted:** sqlc is more than an app this size strictly needs, and GORM would be
  the right-sized choice. If that turns out to matter, switching later is acceptable, since agents
  make refactors much more bearable.

## Alternatives considered

- **Raw `database/sql`.** Rejected because it means writing a lot of boilerplate that many
  existing solutions already abstract away.
- **sqlx.** A layer above `database/sql` that scans rows into structs. Rejected because the app
  doesn't need complex querying, so there's little benefit in working at a level that low.
- **Query builders** (such as squirrel or goqu). Rejected because there's no reason to build the
  mapping layer by hand when sqlc generates it.
- **GORM.** The strongest alternative. It's right-sized for the app and has by far the most
  training data. Rejected because, with the training-data advantage roughly even against sqlc's
  explicitness, GORM's hidden behavior was the deciding factor against it. Examples of that
  hidden behavior:
  - **Implicit names.** By default, GORM derives table and column names from Go names: struct
    `JobCard` becomes table `job_cards`, and field `BoardColumnID` becomes column
    `board_column_id`. It infers foreign keys from field names too. Grepping a column name from a
    migration or a SQL error finds nothing in Go unless every field is explicitly tagged and every
    model defines `TableName()`.
  - **Zero values skipped on update.** `Updates` with a struct silently skips fields holding zero
    values (`false`, `0`, `""`), so setting a field back to its zero value does nothing unless
    you use a map or `Select`.
  - **Associations load lazily.** Related records aren't loaded unless asked for with `Preload`.
    Forgetting it leaves them empty with no error, and each `Preload` runs a separate query.
  - **Query strings are unchecked.** Conditions such as `Where("status = ?", s)` are strings, so a
    wrong column name shows up only at runtime.
  - **AutoMigrate is limited.** It creates tables and adds columns from the structs, but it
    doesn't drop or rename columns, so the schema can drift from the Go models.

  These mistakes tend to fail silently at runtime. The matching mistakes in sqlc fail when code is
  generated or compiled.
- **ent.** Rejected because it's geared towards graph-shaped data, which this app doesn't have.
- **Bun.** Rejected because it has a smaller community and less training data than the
  alternatives, and it offers nothing this app needs that sqlc doesn't.

## Related

- universal
- [docs/adr/0001-design-the-codebase-to-be-grep-complete.md] Design the codebase to be grep-complete
- [docs/adr/0003-store-user-data-in-sqlite.md] Store user data in SQLite
- [docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md] Use mattn/go-sqlite3 as the SQLite driver

## Notes

- **2026-10-09:** Before this decision was accepted, a spike verified that sqlc works as
  expected with this stack. It used sqlc v1.31.1 against a small jobs-and-documents schema
  modelled on the board specs, and the generated code was run against in-memory SQLite through
  `mattn/go-sqlite3`. Creating jobs, moving a card in a transaction (`q.WithTx`), a join with a
  document count, counts per status, and an optional search filter with an `IN` list all
  returned correct results. Rough edges found, and how to resolve each:
  - By default, nullable columns generate as `sql.NullString` and similar types, which serialize
    poorly to JSON and to the frontend. Fix: set `emit_pointers_for_null_types: true`, which
    works with the SQLite engine and generates `*string`.
  - Some expressions infer as `interface{}`, such as `MAX(position)` and an optional
    `sqlc.narg(...)` filter. Fix: wrap the expression in a `CAST`, e.g.
    `CAST(sqlc.narg(company) AS TEXT)`, which generates `*string` (or `int64` for an `INTEGER`
    cast). Nothing fails to build when this happens, so agents need to be told to watch for it.
  - Queries whose columns don't match a table exactly get their own `XxxRow` struct. Fix, where
    wanted: `sqlc.embed(table)` nests the table's model inside the row struct, e.g.
    `{Job Job; DocumentCount int64}`. Queries that return whole table rows reuse the table's
    model.
  - Comments in the schema (`CREATE TABLE` in the migrations) are not copied into the generated
    models, so model structs can't carry spec annotations from the schema. Query comments do
    carry over.
  - A column named `url` generates `Url`, not `URL`. Fix: sqlc's `rename` config option.
  - Spec annotations in query comments appear twice, in the `.sql` file and in the generated
    `.go` file, so `cmd/speclint` sees both.
