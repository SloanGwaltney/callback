# 0026. Link scenarios by ID, on one line

- **Status:** Accepted
- **Date:** 2026-10-14

## Context

Code and tests link to scenarios with `@scenario` annotations
([docs/spec-annotations.md](../spec-annotations.md)), written as
`@scenario [features/x.feature#<id>] <title>`, one scenario per line. Code that implements several
scenarios stacks one line per scenario.

[docs/adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md] moved step annotations out
of feature code and into tests, which leaves `@scenario` as the only annotation in feature code.
There are 530 `@scenario` lines in code and 549 in tests. Step annotations rarely stack: in the
tests, about 1,740 stand alone, 80 places have two in a row, and none have more than three. `@scenario` lines stack heavily at entry
points that serve many scenarios: 52 in a row in `Board.tsx`, 24 in `app.go`, 20 in
`DocumentLibrary.tsx`, and 17 in `internal/cli/job.go`. Scrolling past them is the cost, not the
bytes they take up.

The annotations are read by AI agents, not people. A person looking for a scenario's code can use
the tools (`spectrace`, `speclint`).

All 339 scenario IDs are unique across every feature file, but `speclint` only checks that an ID
is unique within its own file.

Scenario IDs are kebab-case, and grep treats a hyphen as a word boundary, so a search for
`#agent-backup` with `\b` or `grep -w` also matches `#agent-backup-needs-name`.

## Decisions

1. A `@scenario` annotation names scenarios by ID only, each written as `[#<id>]`, with no feature
   file path and no title: `// @scenario [#move-card-forward] [#move-card-back]`.
2. One annotation can name any number of scenarios.
3. Each annotated function, test, or block, in code or in tests, has at most one `@scenario`
   line.
4. Scenario IDs are unique across all feature files, and `speclint` checks it.
5. `speclint` and `spectrace` resolve each ID to its feature file and scenario, and check the
   annotations as before: an ID that doesn't exist or belongs to a withdrawn scenario is reported.
6. Step annotations don't change. They keep the feature file path, the step text, and an optional
   scenario ID.
7. ADR Related sections don't change. They keep `[features/x.feature#<id>] <title>`.

## Reasons

- **1, 2, 3.** Code that implements many scenarios gets one comment line instead of dozens, which
  cuts the scrolling and comment noise at entry points. The IDs already identify the scenario; the
  path and title only repeated it. Losing them costs little, since agents are the readers and the
  tools look up the rest.
- **1.** The closing bracket after each ID makes a fixed-string search match exactly one ID:
  `grep -rnF '#agent-backup]'` doesn't match `#agent-backup-needs-name`. The same search also
  matches ADR Related entries, which end in `#<id>]`, so one search still finds the code, the
  tests, and the decisions for a scenario.
- **3.** Applying the one-line rule to tests too keeps one rule for `speclint` to check and for
  agents to follow. Tests rarely name more than one scenario, so it costs them nothing.
- **4.** Without the path, an ID has to identify its scenario on its own. The IDs are already
  unique, so this check only guards against a future clash.
- **6.** Step annotations sit only in tests and rarely stack, so they aren't where the noise is.
  Their text documents what the test code under them checks, and grepping a step's text finds its
  test.
- **7.** Related entries are read by people and are never stacked by the dozen, so they keep their
  title.

## Alternatives considered

- **Short Jira-style IDs with step numbers**, such as `BAC-1` for a scenario and `BAC-1.3` for its
  third step, with several on one line. Rejected:
  - Numbers given by position change when a scenario or step is inserted, and old links then
    point at a different step with nothing to catch it. Today, rewording a step breaks its link
    and `speclint` reports it.
  - `BAC-1` is a prefix of `BAC-12` and `BAC-1.3`, so searches need care.
  - Step numbers are per scenario, so a step shared by several scenarios needs a reference for
    each one, where its text covers them all with one annotation.
  - `@then BAC-1.3` says nothing about what the test checks.
  - The existing kebab IDs are already short, unique, and stable.
- **Keep the title or the path and only allow several IDs per line.** Rejected: the path and the
  title repeat what the ID says and make a line naming many scenarios very long.
- **IDs separated by spaces, without brackets** (`#move-card-forward #move-card-back`). Shorter,
  but finding one exact ID needs a regex like `'#agent-backup( |$)'`, and that search doesn't find
  ADR Related entries.
- **One line in feature code only, letting tests stack.** Rejected: two rules where one will do.
- **Shorten step annotations too.** Rejected: see reason 6.

## Related

- universal
- [docs/adr/0001-design-the-codebase-to-be-grep-complete.md] Design the codebase to be grep-complete
- [docs/adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md] Link spec steps from tests, and scenarios from code
