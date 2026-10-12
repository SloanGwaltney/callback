# 0021. Link spec steps from tests, and scenarios from code

- **Status:** Accepted
- **Date:** 2026-10-12

## Context

Code and tests link back to the specs in `features/` with annotations
([docs/spec-annotations.md](../spec-annotations.md)). Until now, `speclint` required every step of
an implemented scenario to be annotated in both code and tests, on the line that carries the step
out or checks it.

Building board search and pagination showed what that costs where many scenarios meet in one
component. `Board.tsx` was 587 lines with 43 `@scenario` lines and 84 step annotations, and the app's
code held 378 step annotations, not counting tests. Most of the volume came from four places:

- Code annotations repeat what the tests already prove. An annotated assertion is evidence that a
  step works; the same line above the code is only a claim, and nothing checks it.
- `Given` steps land on code that just reads data. Twenty `Given` lines sat above one
  `filterBoard(board, search)` call.
- Steps with literal data, like "holds 25 jobs" and "holds 21 jobs", each need their own annotation,
  because matching uses the exact text.
- An `And` or `But` step needed an annotation with its own keyword, so a step written as `When` in
  one scenario and `And` in another needed two lines on the same code.

The specs aren't executable BDD tests. They exist so an agent can read a scenario and follow it to
the code and tests that implement it. Linking also serves as the check that a scenario was built.

## Decisions

1. An implemented scenario needs a `@scenario` annotation in code, on each entry point, and in tests.
2. Each step of an implemented scenario needs a step annotation in tests only. Code isn't required
   to carry step annotations.
3. Step annotations that remain in code are still checked: their file, keyword, and text must match
   a live step, as before.
4. An `And` or `But` step accepts an annotation with its own keyword or with the kind of the step
   before it, so "And I clear the search" after a `When` takes `@and` or `@when`. An `And` or `But`
   that starts a scenario takes only its own keyword.
5. `spectrace` reports a missing `@scenario` link in code or tests, and a missing step link in tests.
   It still lists step annotations it finds in code.
6. The existing step annotations in code stay for now, and are removed in a separate change.

## Reasons

1, 2. Tests are the proof that a scenario's behavior works, so they're where each step belongs. The
   entry point in code is what's needed to find where a behavior starts. Grepping a step's text now
   finds the test, and the test leads to the code it calls. That one extra hop doesn't meaningfully
   break grep-completeness ([docs/adr/0001-design-the-codebase-to-be-grep-complete.md]).
   `spectrace` still serves both of its purposes, checking that a behavior was built and grounding
   an agent's context, and now answers "where does this start, and what proves each step".
   The trade-off: a step's text no longer leads straight to the exact line of code that carries it
   out.
3. Keeping the check means an annotation left in code after a step is reworded is reported instead
   of going stale.
4. This removes duplicate annotations for the same step under different keywords, and keeps the
   meaning of setup, action, and check that the keyword gives a reader. Writing the wrong keyword
   is caught by `speclint`, so the cost is a retry, not a silent error. Grep isn't affected, since
   the search key is the step text.
5. `spectrace` uses the same rules as `speclint`, so it can check a scenario before it's marked
   implemented.
6. This isn't a breaking change: annotations already in code still match. Removing them is a large
   mechanical change, kept out of the change that alters the rules.

## Alternatives considered

- **Require code links for `Then` steps as well as `@scenario`.** Outcomes are the steps most worth
  finding in code. Rejected: the tests already prove the outcomes, and the scenario's entry point
  plus its tests are enough to find the code.
- **Placeholders in step annotations**, such as `the "{column}" column holds {n} jobs`, matched
  against any step of that shape the way Cucumber step definitions work. Rejected: it breaks
  grepping for a step's exact text, and BDD-style step definitions aren't the goal; the specs are
  for agents to link back to.
- **Ignore the keyword when matching**, matching an annotation on file and text only. It removes
  all keyword duplication with no rule to learn, but the keyword would become decoration. Kept as a
  fallback if inheriting the keyword trips agents up in practice.
- **Change how specs are written instead**, with fewer `And` steps and literal data moved into
  `Examples` tables. Rejected: it pushes the cost onto spec authors and doesn't touch the
  duplication between code and tests.
- **Move annotations into a separate index file.** Rejected: it breaks finding the spec by grepping
  the code, which ADR 0001 is built on.

## Related

- universal
- [docs/adr/0001-design-the-codebase-to-be-grep-complete.md] Design the codebase to be grep-complete
- [docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md] Check code with one script, run by the Stop hook and CI
