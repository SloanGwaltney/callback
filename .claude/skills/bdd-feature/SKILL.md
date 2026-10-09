---
name: bdd-feature
description: Draft Gherkin feature definitions from a user's plain-language description of a feature, with a status label (not-implemented, implemented, deprecated, withdrawn) on every scenario. Use when the user describes a feature, behavior, or requirement and wants BDD/Gherkin specs written or updated, or invokes /bdd-feature.
---

# bdd-feature

Turn a user's description of a feature into Gherkin `.feature` files. Every scenario carries a
**status** label, and the status controls what changes are allowed.

## Status labels

Status is a non-standard extension to Gherkin. It is written as a tag so the files still parse
with standard tools such as godog and Cucumber:

| Tag                         | Meaning                                                                 |
|-----------------------------|-------------------------------------------------------------------------|
| `@status:not-implemented`   | Specified but not built yet. Can be freely edited.                      |
| `@status:implemented`       | Built and shipped. Locked. See the rules below.                         |
| `@status:deprecated`        | Still in the app, but scheduled for removal. Don't build on it.         |
| `@status:withdrawn`         | No longer supported and removed from the app. Kept as a historical record. |

Status goes on `Scenario` and `Scenario Outline` only, and every one of them gets exactly one
status tag. `Feature`, `Rule`, and `Examples` never get a status tag. They're just grouping.

A scenario is a **behavior**. Its steps are the **criteria** for that behavior.

## Change rules

These rules apply whenever you edit an existing `.feature` file. Read the file first, every time.

1. **New scenarios can go anywhere.** Features and Rules have no status, so you can always add a
   new scenario or Rule to any file. New scenarios are tagged `@status:not-implemented`.
2. **Implemented scenarios can get additional criteria, not additional behavior.** You may add
   `And` / `But` steps, or extra `Given` / `When` / `Then` steps, to an implemented scenario when
   they pin down the behavior it already describes: more precise context, an extra check on the
   same outcome, a detail that was left implicit. You may not add steps that give it a new
   trigger, a new outcome, or a new capability. Test: if the step describes something the app
   would have to newly *do*, it's new behavior and belongs in a new scenario. Don't reword or
   remove an implemented scenario's existing steps.
3. **Changing behavior means replacing it.** If the request changes what an implemented scenario
   does, tag the old one `@status:deprecated` and write the replacement as a new
   `@status:not-implemented` scenario. Link them in a comment, e.g.
   `# Replaces: "Old scenario title"`.
4. **Deprecated can't be extended.** Don't add steps to a deprecated scenario. It can only move to
   `@status:withdrawn`.
5. **Withdrawn is final.** Don't edit or un-withdraw a `@status:withdrawn` scenario. If the
   behavior comes back, write it as a new scenario. Don't delete withdrawn scenarios either;
   they're the record of what was removed.
6. **Status only moves forward:** `not-implemented → implemented → deprecated → withdrawn`. A
   `not-implemented` scenario that's dropped before shipping can be deleted outright instead of
   withdrawn. Only change a status to `implemented` when the user says it's built.

If the user asks for something that breaks a rule, don't do it silently. Say which rule it hits
and propose the compliant alternative (usually a new scenario, per rule 2 or 3).

## Workflow

1. **Understand the request.** Read the user's description. If an actor, a key outcome, or an
   important edge case is unclear and you can't infer it, ask a short list of questions before
   drafting. Don't ask about things you can reasonably decide; state those as assumptions instead.
2. **Find existing specs.** Look under `features/` (create it if missing) for files covering the
   same area. If one exists, you're editing it, so apply the change rules above.
3. **Draft.** Write or update the file(s):
   - One Feature per file, named in kebab-case, e.g. `features/call-scheduling.feature`.
   - A short user-story description under the Feature title (`As a ... I want ... So that ...`).
   - Optionally group scenarios into `Rule`s, one per distinct business rule.
   - Cover the happy path, the main alternate paths, and error/edge cases. Use a
     `Scenario Outline` when scenarios differ only in data.
   - Write steps in domain language, from the user's point of view. No UI selectors, API calls,
     or implementation details. `Given` sets state, `When` is the action, `Then` checks
     observable outcomes.
   - Use `Background` only for setup shared by every scenario in the Feature (or Rule).
4. **Report.** Summarize what you wrote: the files, the scenarios added or changed and their
   status, any assumptions you made, and any requests you redirected because of the change rules.
   List open questions.

## Example

```gherkin
Feature: Call reminders
  As a user
  I want to be reminded before a scheduled call
  So that I don't miss it

  @status:implemented
  Scenario: Reminder fires at the default lead time
    Given I have a call scheduled for 3:00 PM
    When the time reaches 2:45 PM
    Then I receive a reminder for that call
    # Additional criterion on an implemented scenario (allowed: same behavior, more precise)
    And the reminder shows the call's title and start time

  @status:not-implemented
  Scenario: No reminder for a cancelled call
    Given I had a call scheduled for 3:00 PM
    And I cancelled the call
    When the time reaches 2:45 PM
    Then I do not receive a reminder

  @status:deprecated
  Scenario: Reminder plays the default sound
    Given I have a call scheduled for 3:00 PM
    When the reminder fires
    Then the default reminder sound plays

  # Replaces: "Reminder plays the default sound"
  @status:not-implemented
  Scenario: Reminder plays the sound chosen in settings
    Given I have chosen the "chime" reminder sound
    And I have a call scheduled for 3:00 PM
    When the reminder fires
    Then the "chime" sound plays

  @status:withdrawn
  Scenario: SMS reminder is sent
    Given I have a call scheduled for 3:00 PM
    And I have a verified phone number
    When the time reaches 2:45 PM
    Then I receive an SMS reminder
```

A request like "let users snooze reminders" is new behavior. Adding
`When I snooze the reminder / Then it fires again in 5 minutes` to the implemented
"Reminder fires at the default lead time" scenario would break rule 2. It goes in a new
`@status:not-implemented` scenario instead.
