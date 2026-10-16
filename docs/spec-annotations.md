# Spec annotations

How code and tests link back to the Gherkin-like specs in `features/`.

The annotation format, and the rules `speclint` and `spectrace` check, are in
[tools/speclint/docs/spec-annotations.md](../tools/speclint/docs/spec-annotations.md). Read it
first. speclint is its own repository
([SloanGwaltney/speclint](https://github.com/SloanGwaltney/speclint)), a git submodule at
`tools/speclint/`, so the format and the checker are versioned together. If `tools/speclint/` is
empty, run `git submodule update --init`.

This page covers only what's specific to callback.

## Why the rules are what they are

- Scenarios are linked from code, and steps from tests
  ([docs/adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md](adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md)).
- A `@scenario` line names scenarios by ID, all on one line
  ([docs/adr/0026-link-scenarios-by-id-on-one-line.md](adr/0026-link-scenarios-by-id-on-one-line.md)).

## Where links go in callback

- **Entry points** are usually the Wails-bound methods in `app.go`, the commands in
  `internal/cli/`, the React components and hooks that start a behavior, and the SQL queries in
  `internal/db/queries/`.
- **sqlc copies a query's comments** into the Go it generates in `internal/store/`, so a query's
  `@scenario` annotation appears there too. That copy is expected. Change the `.sql` file and run
  `sqlc generate`; never edit the generated copy.
- **Don't annotate migrations** in `internal/db/migrations/`. Annotate the storage tests that check
  the schema's behavior instead.

## Configuration

[`.speclint.json`](../.speclint.json) at the repo root tells speclint about callback:

- `skipDirs` leaves out the generated Wails bindings in `frontend/wailsjs/`, and the submodule
  itself.
- `layers` names the layers `spectrace` groups links by, from the SQL query up to the frontend:
  SQL query, storage (`internal/db/`), service (`internal/`), Wails binding (`.go` files at the
  root), frontend hook (`frontend/src/hooks/`), and frontend.

## Running the tools

From the repo root:

```sh
go run -C tools/speclint ./cmd/speclint -root ../..
go run -C tools/speclint ./cmd/spectrace -root ../.. move-card-forward
go run -C tools/speclint ./cmd/spectrace -root ../.. features/job-board.feature
go run -C tools/speclint ./cmd/spectrace -root ../.. -missing features/job-board.feature
```

`scripts/check.sh` runs `speclint` with the other checks. A Claude Code Stop hook
(`.claude/hooks/check-stop.sh`) runs that script at the end of every turn and sends the agent back
once to fix any problems; if they remain after that, they're shown to the user. CI runs it on every
pull request.

To update speclint, pull a newer version and commit the new submodule pointer, then run
`bash scripts/check.sh` to check callback still passes:

```sh
git -C tools/speclint pull
git add tools/speclint
```
