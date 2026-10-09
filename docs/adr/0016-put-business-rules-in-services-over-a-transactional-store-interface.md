# 0016. Put business rules in services over a transactional store interface

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

The first features to be built (the job board, job cards, and job documents) need business rules:
a job needs a company and a role, moving a card to Applied fills in today's date, closing a job
needs a reason, and moving a card renumbers the positions in its columns. Something has to decide
where those rules live and how they're tested.

Two accepted decisions constrain that:

- Unit tests don't touch storage. Code above the storage layer is tested against hand-written
  fakes of the storage interfaces, such as sqlc's generated `Querier` (see
  [docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md]).
- A move renumbers the positions of the affected columns in a single transaction (see
  [docs/adr/0015-store-card-order-as-an-integer-position.md]).

sqlc's `Querier` interface has no way to start a transaction. The only route into one is
`Queries.WithTx(*sql.Tx)`, which needs a real `*sql.DB` (see
[docs/adr/0005-generate-database-queries-with-sqlc.md]). As generated, code that needs a
transaction can't be tested against a `Querier` fake.

The app's backend entry points are the Go methods Wails binds for the frontend, in `app.go` (see
[docs/adr/0002-use-wails-for-the-desktop-shell.md]).

## Decisions

1. Business rules live in service packages under `internal/`. The Wails-bound methods in `app.go`
   call a service and don't hold rules themselves.
2. A service depends on a store interface made of sqlc's `Querier` plus one method,
   `InTx(ctx context.Context, fn func(store.Querier) error) error`, which runs `fn` in a single
   transaction.
3. The real store, which wraps `*sql.DB` and runs `fn` against `Queries.WithTx`, lives in
   `internal/db`, the storage layer.
4. Service unit tests use a hand-written in-memory fake of the store interface. The real store,
   its queries, and its transactions are tested against a real SQLite database.

## Reasons

- **1, 4.** Unit tests shouldn't run against a real database. For the app as it is now, that
  would be quick to do, but it adds complications later, such as when running tests in
  parallel, and keeping databases out of unit tests is best practice. Rules written in the
  Wails-bound methods would only be testable against the `*sql.DB` the `App` holds.
- **2.** Adding `InTx` to `Querier` gives services the transactions ADR 0015 requires while
  keeping them behind an interface that a hand-written fake can satisfy. The fake runs `fn`
  against itself.
- **3.** Opening and committing transactions is storage work, so it stays in the storage layer
  with the rest of the database code, and is tested there.
- Accepted trade-off: the fake has no real transaction, so it can't show that a failed `fn`
  rolls back. Rollback is checked only by the storage-layer tests against real SQLite.

## Alternatives considered

- **Put the rules in the Wails-bound methods and call the sqlc queries directly.** Fewest layers,
  but testing any rule would mean a real database. Rejected for the reasons under 1.
- **Give the storage layer high-level methods, like moving a job, that apply the rules inside.**
  Hides transactions in storage and keeps services thin. Rejected because the storage layer
  would mix data access with business rules.
- **Put the rules in SQL**, for example renumbering positions with
  `UPDATE ... SET position = position + 1`. Not treated as an alternative to this decision.
  Business logic will never all live in SQL, so how much goes in SQL versus Go is a separate
  question, settled case by case. Neither extreme is the answer, and this ADR doesn't decide it.

## Related

- universal
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
- [docs/adr/0005-generate-database-queries-with-sqlc.md] Generate database queries with sqlc
- [docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md] Require unit tests and test storage against real SQLite
- [docs/adr/0015-store-card-order-as-an-integer-position.md] Store card order as an integer position
