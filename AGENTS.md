# AGENTS.md

## Grep-complete

This codebase aims to be grep-complete: starting from any point of entry, you should be able to
find everything related to it with a text search. Keep it that way by making everything you add
searchable.

How we do that:

- **Spec annotations.** Code and tests that implement a scenario in `features/*.feature` carry
  comments like `// @scenario [features/x.feature#<id>] <title>` and `// @then [features/x.feature]
  <step text>`, so grepping a spec file, scenario ID, or step text finds its implementation and
  tests. Start new features from their `@status:not-implemented` scenarios. Read
  [docs/spec-annotations.md](docs/spec-annotations.md) before implementing or testing a scenario.
- **Decision records.** Architectural decisions live in `docs/adr/` and are written with the `adr`
  skill. Each ADR's Related section lists the scenarios it affects as
  `[features/x.feature#<id>]`, the same form as code annotations, or `universal` for app-wide
  decisions, so grepping a scenario ID finds the decisions behind it. Only the user starts an
  ADR, by running `/adr`. Don't create or edit files in `docs/adr/` on your own; if a decision
  looks worth recording, suggest that the user run `/adr`.
