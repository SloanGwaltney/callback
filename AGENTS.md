# AGENTS.md

## Implementing features from Gherkin-like specs

Specs live in `features/*.feature` and are written with the `bdd-feature` skill
(`.claude/skills/bdd-feature/SKILL.md`). Every scenario has a `@status:` tag and an `@id:` tag.

### 1. Start from the spec

Before writing code for a new feature, find the scenarios it covers:

- Search `features/` for scenarios tagged `@status:not-implemented` that describe the feature.
- If none exist, stop and write them first with the `bdd-feature` skill. Don't build behavior
  that has no spec.
- Don't build on `@status:deprecated` or `@status:withdrawn` scenarios.
- Don't change a scenario's status to `@status:implemented` yourself. That happens only when the
  user says the feature is built (see the skill's change rules).

### 2. Annotate the code that executes each step

Every scenario you implement, and every step in it, must be traceable from the source code.
Put a comment directly above the line or block of code that carries out that part of the
scenario, in this format:

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
- **Step annotations go on the code that does the step**, not on the function that wraps it. If
  you have to read past the comment to find the code that does the step, the comment is in the
  wrong place.
- **One piece of code can carry more than one annotation** when it fulfills several steps or
  scenarios. Stack the comments.
- **A `Then` checks an observable outcome.** Annotate the code that produces that outcome, such
  as the emitted event, the returned value, or the UI state change.
- Use the language's line comment: `//` in Go and TypeScript, `{/* */}` inside JSX.
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

Code:

```go
// @scenario [features/call-reminders.feature#cancelled-no-reminder] No reminder for a cancelled call
// @when [features/call-reminders.feature] the time reaches 2:45 PM
func (s *Scheduler) onTick(now time.Time) {
	// @given [features/call-reminders.feature] I had a call scheduled for 3:00 PM
	for _, call := range s.callsDueForReminder(now) {
		// @and [features/call-reminders.feature] I cancelled the call
		// @then [features/call-reminders.feature] I do not receive a reminder
		if call.Cancelled {
			continue
		}
		s.sendReminder(call)
	}
}
```

### 3. Annotate tests the same way

Tests use the same annotations, so a grep for a scenario finds its tests next to its
implementation.

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

### Finding a spec in the code

Every annotation for a feature file, in code and tests:

```sh
grep -rn "\[features/call-reminders.feature" --include=*.go --include=*.ts --include=*.tsx .
```

Entry points and tests for one scenario:

```sh
grep -rn "#cancelled-no-reminder\]" --include=*.go --include=*.ts --include=*.tsx .
```
