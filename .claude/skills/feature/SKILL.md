---
name: feature
description: Implement scenarios from features/*.feature that the user names, annotating the entry points and step code with spec annotations, writing annotated unit tests for every scenario, and running the tests before finishing. Use when the user invokes /feature with scenario references.
argument-hint: <features/x.feature#scenario-id ...>
disable-model-invocation: true
---

# feature

Build the scenarios the user names, end to end, so every scenario and every step can be found
from the code and from its tests with a text search.

Arguments: $ARGUMENTS

Before writing code, read `AGENTS.md` and `docs/spec-annotations.md`. They hold the tech stack,
the dependency rules, and the exact annotation format. This skill doesn't repeat them.

## 1. Resolve the scenarios

The user passes the scenarios to build. Accept any of these forms:

- `features/job-board.feature#move-card-forward`: one scenario
- `move-card-forward`: a bare ID. Search `features/` for `@id:move-card-forward`.
- `features/job-board.feature`: every `@status:not-implemented` scenario in that file

If no arguments were given, list the `@status:not-implemented` scenarios in `features/`, grouped
by file, and ask which to build. Don't pick for the user.

For each scenario, read it in the `.feature` file and check its status:

| Status            | What to do                                                                 |
|-------------------|----------------------------------------------------------------------------|
| `not-implemented` | Build it.                                                                  |
| `implemented`     | Already built. Ask whether the user wants a change; if so, that's a spec change for the `bdd-feature` skill first. |
| `deprecated`      | Don't build on it. Tell the user and skip it.                              |
| `withdrawn`       | Don't build it. Tell the user and skip it.                                 |

If a reference doesn't match any scenario, say so and ask. Never guess an ID.

If a scenario is too vague to build without inventing behavior, ask the user, or suggest
rewording it with `/bdd-feature`, before writing code. Don't edit `.feature` files yourself in
this skill, apart from the status change in step 7.

## 2. Gather context

- **Decisions.** Grep `docs/adr/` for each scenario ID and its feature file, and read the ADRs
  that come up, plus the `universal` ones relevant to the layers you'll touch (storage, queries,
  frontend data, drag and drop, styling, testing).
- **Existing code.** Grep for the feature file path to find code already linked to it, and for
  each step's text, since a step shared with an implemented scenario may already be annotated
  and done.
- **Shared steps.** Note steps that appear in more than one of the requested scenarios. Implement
  and annotate them once, unless different code carries them out in each scenario, in which case
  scope each annotation with the scenario ID (see `docs/spec-annotations.md`).

## 3. Plan

Before editing, give the user a short plan: for each scenario, the entry point that will carry
its `@scenario` annotation, and the layers it touches (migration, `.sql` queries, store, Wails
binding in `app.go`, frontend hook, component). Then proceed unless the plan needs one of the
following, which must be raised and settled first:

- **A new dependency** (Go module, npm package, or tool). Never install one without asking, and
  justify it as `AGENTS.md` requires.
- **A decision worth recording**, like a new library, a data model choice, or a pattern other
  features will copy. Suggest the user run `/adr`. Don't write ADRs yourself.

## 4. Implement

Follow the stack in `AGENTS.md`. Work from the storage layer up, so each layer can be tested
before the next one depends on it:

1. **Schema.** Add a goose migration in the embedded migrations directory. Don't annotate
   migrations.
2. **Queries.** Write the `.sql` queries, annotate the clause that carries out each step with
   `--` comments, and run `sqlc generate`. Never edit generated Go by hand.
3. **Backend.** Store and service code, then the Wails-bound method in `app.go`.
4. **Frontend.** A custom hook that wraps the binding (only hooks call bindings), then the
   components, each with its own `*.module.css` using the token variables.

Annotate as you write, not at the end:

- `@scenario [features/x.feature#<id>] <title>` on the **entry point** where the behavior starts:
  the Wails-bound method, the event handler, or the component that renders the outcome. A
  scenario that spans the backend and frontend gets a `@scenario` at each entry point.
- `@given` / `@when` / `@then` / `@and` / `@but` directly above the **code that does the step**,
  with the step text copied exactly from the `.feature` file, keyword removed.
- Use `//` in Go and TypeScript, `{/* */}` inside JSX, and `--` in SQL. Each annotation goes on
  its own comment line.

Every step in every requested scenario must end up with an annotation on the code that carries it
out. A `Given` that's only setup state, like "I have not added any jobs", is carried out by the
code that reads that state, like the query that returns an empty list. Annotate that.

## 5. Test

Every scenario gets unit tests, and the tests carry the same annotations. Follow
`docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md`:

- **Go, outside storage:** hand-written fakes, no database.
- **Storage:** a real temporary SQLite database and directory.
- **Frontend:** Vitest and React Testing Library, with the Wails bindings mocked.

In each test:

- `@scenario [features/x.feature#<id>] <title>` on the test function, or on the subtest or `it`
  block when one function covers several scenarios.
- `@given` on the setup, `@when` on the action, `@then` / `@and` / `@but` on the assertion, with
  the exact step text.

Test each scenario at the layer that owns its outcome. A storage scenario needs a storage test, a
UI scenario needs a component test, and a scenario that crosses both needs a test in each layer.
Also add plain unit tests, without annotations, for edge cases and helpers the scenarios don't
cover.

## 6. Verify

Run the full check script from the repo root, tests included, and fix every failure before
finishing:

```sh
bash scripts/check.sh
```

It runs gofmt, `go vet`, speclint, `tsc`, and the tests (see
`docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md`).

`speclint` only reports *missing* links for `@status:implemented` scenarios, so while the
scenarios are still `not-implemented` it won't notice a step you forgot. Check coverage yourself
by tracing the requested scenarios. List what's missing across a whole feature file, then trace a
scenario to read its breakdown:

```sh
go run ./cmd/spectrace -missing features/<file>.feature
go run ./cmd/spectrace <scenario-id>
```

It lists the scenario's entry points and each step's code and tests, layer by layer, with the
same matching rules as `speclint`, and exits 1 with a `MISSING:` line for anything without a link
in code or in tests. With `-missing` it prints only those lines, under each scenario's ID, and a
count of the scenarios fully linked. Fix every `MISSING:` line. Also read the breakdown: each step's links should
sit in the layers that actually carry it out, and a UI scenario should reach the frontend.

## 7. Report

Finish with:

- The scenarios built, each with its entry point and test as `file:line`.
- Test results: the commands run and whether they passed. If something still fails, say so with
  the output rather than finishing quietly.
- Anything skipped or left open: deprecated or withdrawn scenarios, questions, a suggested `/adr`.
- A request to the user to confirm the scenarios are built. Don't change a status to
  `@status:implemented` yourself. When the user confirms, flip the tags, then run
  `go run ./cmd/speclint` again. With the scenarios now `implemented`, it checks that every step
  is linked in both code and tests.
