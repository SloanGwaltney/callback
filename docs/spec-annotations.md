# Spec annotations

How code and tests link back to the Gherkin-like specs in `features/`.

Specs live in `features/*.feature` and are written with the `bdd-feature` skill
(`.claude/skills/bdd-feature/SKILL.md`). Every scenario has a `@status:` tag and an `@id:` tag.

## 1. Start from the spec

Before writing code for a new feature, find the scenarios it covers:

- Search `features/` for scenarios tagged `@status:not-implemented` that describe the feature.
- If none exist, stop and write them first with the `bdd-feature` skill. Don't build behavior
  that has no spec.
- Don't build on `@status:deprecated` or `@status:withdrawn` scenarios.
- Don't change a scenario's status to `@status:implemented` yourself. That happens only when the
  user says the feature is built (see the skill's change rules).

## 2. Where links go

Every scenario you implement must be traceable from the code where it starts, and every step in
it from the tests that prove it
([docs/adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md](adr/0021-link-spec-steps-from-tests-and-scenarios-from-code.md)):

- **In code,** put a `@scenario` annotation on each entry point. Step annotations in code aren't
  required, so don't add them.
- **In tests,** put a `@scenario` annotation on the test and a step annotation on the test code
  for each step (see section 3).

Grepping a step's text finds its test, and the test leads to the code it calls.

Each annotation is a comment directly above the line or block it links, in this format:

```
@<keyword> [<path to .feature file>#<scenario id>] <exact text from the file>
```

| Spec line           | Annotation                                           |
|---------------------|------------------------------------------------------|
| `Scenario:`         | `@scenario [features/x.feature#<id>] <title>`        |
| `Scenario Outline:` | `@scenario [features/x.feature#<id>] <title>`        |
| `Given`             | `@given [features/x.feature] <step text>`            |
| `When`              | `@when [features/x.feature] <step text>`             |
| `Then`              | `@then [features/x.feature] <step text>`             |
| `And`               | `@and [features/x.feature] <step text>`              |
| `But`               | `@but [features/x.feature] <step text>`              |

Rules:

- **Copy the text exactly** from the `.feature` file, keyword removed. That includes outline
  placeholders like `<sound>`. The text is the search key, so don't paraphrase it.
- **`And` and `But` take their own keyword or the one they continue.** "And I clear the search"
  after a `When` is linked by `@and` or `@when`, so one `@when` annotation covers the step in a
  scenario that writes it with `When` and in one that writes it with `And`. An `And` or `But` that
  starts a scenario takes only its own keyword.
- **Use the repo-relative path** with forward slashes, e.g. `[features/call-reminders.feature]`.
- **`@scenario` always carries the scenario's ID**, taken from its `@id:` tag:
  `[features/call-reminders.feature#cancelled-no-reminder]`.
- **Step annotations leave the ID off by default.** A step is identified by its file and text,
  so a step shared by several scenarios is annotated once and covers all of them.
- **Add the ID to a step annotation when its text is ambiguous**, meaning the same step text
  appears in more than one scenario in the file but is carried out by different code in each.
  Scope each annotation to its scenario: `@then [features/x.feature#custom-lead-time] I receive
  a reminder`.
- **`@scenario` goes on the entry point**: the function, handler, or Wails-bound method where the
  behavior starts.
- **Step annotations go on the test code that does the step:** the setup, the action, or the
  assertion. If you have to read past the comment to find that code, the comment is in the wrong
  place.
- **Annotate a step only in a test whose layer carries it out.** When a step happens in another
  layer, like "today is 2026-10-09" when the backend decides today, or a list order that a SQL
  query sets, link it from that layer's test, not from a comment with no code under it. A test
  that relies on another layer can say so in a plain comment that names the test proving it.
- **One piece of code can carry more than one annotation** when it fulfills several steps or
  scenarios. Stack the comments.
- **A `Then` checks an observable outcome.** Annotate the assertion on that outcome, such as the
  emitted event, the returned value, or the UI state.
- Use the language's line comment: `//` in Go and TypeScript, `{/* */}` inside JSX, `--` in SQL.
- **SQL queries are annotated like code.** A query that's a scenario's entry point gets its
  `@scenario` annotation. sqlc copies a query's comments into the Go it generates, so the
  annotation also appears on the generated method. That copy is expected; change the `.sql`
  file and regenerate rather than editing it.
- **Don't annotate migrations.** A migration records a past change to the schema, and a later
  migration can undo what an earlier one did, leaving its annotation pointing at code that no
  longer runs. Annotate the storage tests that check the schema's behavior instead, since they run
  against the current schema.
- **Keep annotations in sync with the spec.** When a step is reworded, update its annotations in
  the same change. When a scenario is deprecated or withdrawn, remove its `@scenario` tags and
  any step tags no other active scenario still uses. Don't remove a shared step's tag while
  another active scenario depends on it.

### Example

Spec (`features/call-reminders.feature`):

```gherkin
  @status:not-implemented @id:cancelled-no-reminder
  Scenario: No reminder for a cancelled call
    Given I had a call scheduled for 3:00 PM
    And I cancelled the call
    When the time reaches 2:45 PM
    Then I do not receive a reminder
```

Code, with only the entry point linked:

```go
// @scenario [features/call-reminders.feature#cancelled-no-reminder] No reminder for a cancelled call
func (s *Scheduler) onTick(now time.Time) {
	for _, call := range s.callsDueForReminder(now) {
		if call.Cancelled {
			continue
		}
		s.sendReminder(call)
	}
}
```

## 3. Annotate the tests

Tests carry the scenario and every step, so a grep for a scenario finds its tests next to its
entry points, and a grep for a step finds the test that proves it.

- Put `@scenario` on the test function, or on the subtest/`it` block when one test function
  covers several scenarios.
- Put each step annotation directly above the test code for that step: `@given` on the setup,
  `@when` on the action, `@then` / `@and` / `@but` on the assertion.
- Same rules as production code: exact text, repo-relative path, scenario ID on `@scenario` and
  on ambiguous steps, and update the tags in the same change when the spec changes.

```go
// @scenario [features/call-reminders.feature#cancelled-no-reminder] No reminder for a cancelled call
func TestNoReminderForCancelledCall(t *testing.T) {
	s, notifier := newTestScheduler(t)

	// @given [features/call-reminders.feature] I had a call scheduled for 3:00 PM
	call := s.Schedule("Standup", at(15, 0))
	// @and [features/call-reminders.feature] I cancelled the call
	s.Cancel(call.ID)

	// @when [features/call-reminders.feature] the time reaches 2:45 PM
	s.onTick(at(14, 45))

	// @then [features/call-reminders.feature] I do not receive a reminder
	if got := notifier.Sent(); len(got) != 0 {
		t.Fatalf("expected no reminders, got %v", got)
	}
}
```

## Finding a spec in the code

Every annotation for a feature file, in code, tests, and ADRs:

```sh
grep -rn "\[features/call-reminders.feature" --include=*.go --include=*.ts --include=*.tsx --include=*.md .
```

Entry points, tests, and ADRs for one scenario:

```sh
grep -rn "#cancelled-no-reminder\]" --include=*.go --include=*.ts --include=*.tsx --include=*.md .
```

Everything linked to one scenario, layer by layer, from the SQL query up to the frontend:

```sh
go run ./cmd/spectrace move-card-forward
go run ./cmd/spectrace features/job-board.feature#move-card-forward
go run ./cmd/spectrace features/job-board.feature
go run ./cmd/spectrace -missing features/job-board.feature
```

It prints the scenario's `@scenario` entry points, the tests (and any code) for each step, and the
ADRs that list it, using the same matching rules as `speclint`: a step's annotation counts when it
has no scenario ID or this scenario's ID. It exits 1 if the scenario has no `@scenario` link in
code or in tests, or any step has no link in tests, whatever the scenario's status, so it can check a scenario before it's marked
`implemented`. Given a feature file, it traces each of its scenarios that isn't withdrawn and ends
with how many are fully linked. `-missing` prints only the missing links, under the scenario they
belong to. Its code is in `cmd/spectrace/` and `internal/speclint/trace.go`.

## Checking the links

`speclint` checks these rules. A Claude Code Stop hook (`.claude/hooks/check-stop.sh`) runs
it, through `scripts/check.sh`, at the end of every turn and sends the agent back once to fix any problems; if they remain
after that, they're shown to the user. To run it by hand from the repo root:

```sh
go run ./cmd/speclint
```

It prints `file:line: problem` for each finding and exits 1 if there are any. It reports:

- **Missing links.** Every `@status:implemented` scenario needs a `@scenario` annotation in both
  code and tests, and each of its steps a step annotation in tests. A test file is `*_test.go`,
  `*.test.*`, `*.spec.*`, or anything under `__tests__/`.
- **Broken links.** Annotations, in code or tests, that point at a missing file or ID, copy a title or step text
  inexactly, use the wrong keyword (`@given` for an `And` step), or leave the ID off `@scenario`.
- **Stale links.** Annotations on withdrawn scenarios, or on steps only withdrawn scenarios use.
- **Malformed specs.** Scenarios without exactly one valid `@status:` and one kebab-case `@id:`,
  duplicate IDs, status or ID tags on `Feature`/`Rule`/`Examples`, and `Background`.
- **ADR Related sections** that are missing, empty, mix `universal` with feature references, or
  reference a file, ID, or title that doesn't match.

Annotations must sit on their own comment line; a trailing comment after code isn't read. It reads
Go, TypeScript, JavaScript, and SQL files, and in SQL files only `--` comments. The linter skips `node_modules`, `vendor`, `dist`, `build`, `testdata`, `frontend/wailsjs`, and
dot-directories. Its code is in `internal/speclint/`.
