---
name: feature
description: Implement scenarios from features/*.feature that the user names, annotating the entry points with spec annotations, writing unit tests for every scenario annotated with each step, and running the tests before finishing. Use when the user invokes /feature with scenario references.
argument-hint: <features/x.feature#scenario-id ...>
disable-model-invocation: true
---

# feature

Build the scenarios the user names, end to end, so every scenario can be found from the code
where it starts, and every step from the test that proves it, with a text search.

Arguments: $ARGUMENTS

Before writing code, read `AGENTS.md`, `tools/speclint/docs/spec-annotations.md`, and
`docs/spec-annotations.md`. They hold the tech stack, the dependency rules, the exact annotation
format, and how callback applies it. This skill doesn't repeat them.

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
- **Existing code.** Grep for the feature file path to find code and tests already linked to it,
  and for each step's text, since a step shared with an implemented scenario may already be
  tested and done.
- **Shared steps.** Note steps that appear in more than one of the requested scenarios. Implement
  and test them once, unless different code carries them out in each scenario, in which case
  scope each test's annotation with the scenario ID (see `docs/spec-annotations.md`).

## 3. Plan

Before editing, give the user a short plan: for each scenario, the entry point that will carry
its `@scenario` annotation, and the layers it touches (migration, `.sql` queries, store, Wails
binding in `app.go`, frontend hook, component). Then proceed unless the plan needs one of the
following, which must be raised and settled first:

- **A new dependency** (Go module, npm package, or tool). Never install one without asking, and
  justify it as `AGENTS.md` requires.
- **A decision worth recording**: a real choice between alternatives that later work will build
  on, like picking a library or a data model trade-off that isn't settled by an existing ADR.
  Suggest the user run `/adr`. Don't write ADRs yourself. Building something the usual way, like
  a new table, a read-only view, or a page that follows an existing pattern, isn't a decision, so
  don't suggest one for it, and don't add ADR notes to PR descriptions.

## 4. Implement

Follow the stack in `AGENTS.md`. Work from the storage layer up, so each layer can be tested
before the next one depends on it:

1. **Schema.** Add a goose migration in the embedded migrations directory. Don't annotate
   migrations.
2. **Queries.** Write the `.sql` queries, give a query that's a scenario's entry point its
   `@scenario` annotation in a `--` comment, and run `sqlc generate`. Never edit generated Go by hand.
3. **Backend.** Store and service code, then the Wails-bound method in `app.go`.
4. **Frontend.** A custom hook that wraps the binding (only hooks call bindings), then the
   components, each with its own `*.module.css` using the token variables.

Annotate as you write, not at the end:

- `@scenario [#<id>]` on the **entry point** where the behavior starts: the Wails-bound method,
  the event handler, or the component that renders the outcome. A scenario that spans the backend
  and frontend gets a `@scenario` at each entry point. If the entry point already has a
  `@scenario` line, add the new ID to the end of it rather than adding a second line.
- Use `//` in Go and TypeScript, `{/* */}` inside JSX, and `--` in SQL. Each annotation goes on
  its own comment line.

Don't add step annotations (`@given`, `@when`, and so on) to code. Steps are linked from the tests
that prove them (step 5), as
`docs/adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md` decided.

## 5. Test

Every scenario gets unit tests, and the tests link the scenario and every one of its steps. Follow
`docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md`:

- **Go, outside storage:** hand-written fakes, no database.
- **Storage:** a real temporary SQLite database and directory.
- **Frontend:** Vitest and React Testing Library, with the Wails bindings mocked.

In each test:

- `@scenario [#<id>]` on the test function, or on the subtest or `it` block when one function
  covers several scenarios.
- `@given` on the setup, `@when` on the action, `@then` / `@and` / `@but` on the assertion, with
  the exact step text. Every step in every requested scenario must end up annotated in a test.

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
go run -C tools/speclint ./cmd/spectrace -root ../.. -missing features/<file>.feature
go run -C tools/speclint ./cmd/spectrace -root ../.. <scenario-id>
```

It lists the scenario's entry points and each step's tests, layer by layer, with the same matching
rules as `speclint`, and exits 1 with a `MISSING:` line for a `@scenario` without a link in code or
in tests, or a step without a link in tests. With `-missing` it prints only those lines, under each scenario's ID, and a
count of the scenarios fully linked. Fix every `MISSING:` line. Also read the breakdown: each step's tests should
sit in the layers that actually carry it out, and a UI scenario should have frontend tests.

## 7. Report

Finish with:

- The scenarios built, each with its entry point and test as `file:line`.
- Test results: the commands run and whether they passed. If something still fails, say so with
  the output rather than finishing quietly.
- Anything skipped or left open: deprecated or withdrawn scenarios, questions, and an `/adr`
  suggestion only if step 3 raised a real decision.
- A request to the user to confirm the scenarios are built. Don't change a status to
  `@status:implemented` yourself. When the user confirms, flip the tags, then run
  `go run -C tools/speclint ./cmd/speclint -root ../..` again. With the scenarios now
  `implemented`, it checks that every scenario is linked in code and tests, and every step in
  tests.
