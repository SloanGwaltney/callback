---
name: adr
description: Write an Architecture Decision Record (ADR) capturing the decisions made, why they were made, the alternatives considered, and the features or scenarios they relate to. Pulls the content from the current conversation and asks the user for anything missing. Use when the user wants to record, document, or write up a technical or architectural decision, asks for an ADR, or invokes /adr.
disable-model-invocation: true
---

# adr

Record a decision as a Markdown file in `docs/adr/`. An ADR says what was decided, why, and what
else was on the table, so a later reader (human or agent) doesn't have to reconstruct it.

Every ADR has a **Related** section that ties it to the specs in `features/`. It uses the same
`[features/<file>.feature#<id>]` references as code annotations (see `docs/spec-annotations.md`),
so grepping a spec file or scenario ID finds the ADRs behind it alongside its code and tests.

## File layout

- Path: `docs/adr/NNNN-<kebab-case-title>.md`, e.g. `docs/adr/0003-store-calls-in-sqlite.md`.
- `NNNN` is the next unused number, zero-padded to four digits. Check `docs/adr/` (create it if
  missing) and take the highest existing number plus one. Numbers are never reused.
- The number and filename never change, even if the title is reworded later.

## Template

```markdown
# NNNN. <Title: the decision, stated as a short imperative>

- **Status:** <Proposed | Accepted | Superseded by [docs/adr/NNNN-x.md] | Deprecated>
- **Date:** <YYYY-MM-DD>

## Context

<The problem or force that required a decision. What constraints applied, what we knew at the
time. Facts, not the decision.>

## Decisions

<Each decision made, as a numbered list. One line each, stated so it can be checked against the
code: "Store calls in a local SQLite database via modernc.org/sqlite.">

## Reasons

<Why each decision was made. Tie each reason to a decision by number when there are several.
Include the trade-offs accepted, not just the upsides.>

## Alternatives considered

<One subsection or bullet per alternative: what it was and why it was rejected. If the
conversation shows none were considered, say so explicitly rather than inventing any.>

## Related

<See "The Related section" below.>

## Notes

<Optional. Leave this section out until there's a note to add. See "Notes" below.>
```

## The Related section

This section is what keeps ADRs grep-complete. It's a bullet list in one of two forms.

**Specific.** The decision is about particular features or scenarios. List each one, using the
exact reference format from `docs/spec-annotations.md`, followed by its title copied from the
`.feature` file:

```markdown
## Related

- [features/call-reminders.feature#cancelled-no-reminder] No reminder for a cancelled call
- [features/call-reminders.feature#default-lead-time] Reminder fires at the default lead time
- [features/call-scheduling.feature] Call scheduling
- [docs/adr/0002-poll-scheduler-every-minute.md] Poll the scheduler every minute
```

- Reference a **scenario** as `[features/<file>.feature#<id>] <scenario title>` when the decision
  affects specific scenarios.
- Reference a **whole feature** as `[features/<file>.feature] <Feature title>` only when the
  decision affects the feature as a whole, not a subset of its scenarios.
- Reference **other ADRs** this one builds on, supersedes, or is constrained by as
  `[docs/adr/NNNN-x.md] <title>`.
- Read the `.feature` files to get the paths, IDs, and titles. Copy them exactly. Never guess an
  ID; if the scenario doesn't exist yet, see the workflow below.
- Don't list `@status:withdrawn` scenarios. List `@status:deprecated` ones only when the decision
  is about deprecating or replacing them.

**Universal.** The decision applies across the whole app and isn't tied to any feature, e.g. the
database, the language, the frontend framework, the logging approach:

```markdown
## Related

- universal
```

`universal` stands alone. Don't mix it with feature references. If a broad decision was
prompted by one feature, that's still context, so mention the feature in the Context section, but
the decision itself is universal. Related ADRs may still be listed under `universal`:

```markdown
## Related

- universal
- [docs/adr/0001-use-wails.md] Use Wails for the desktop shell
```

To tell them apart, ask: if this feature were removed, would the decision still stand? If yes,
it's universal.

## Status and change rules

| Status                                  | Meaning                                                  |
|-----------------------------------------|----------------------------------------------------------|
| `Proposed`                              | Under discussion. Can be freely edited.                  |
| `Accepted`                              | In effect. Locked except as noted below.                 |
| `Superseded by [docs/adr/NNNN-x.md]`    | Replaced by a newer ADR. Kept as a record.               |
| `Deprecated`                            | No longer in effect, with no replacement. Kept as a record. |

1. **New ADRs are `Accepted`** when the conversation shows the decision has been made, and
   `Proposed` when it's still open. Ask if it's unclear.
2. **Accepted ADRs aren't rewritten.** Don't change the Context, Decisions, Reasons, or
   Alternatives. You may fix typos, add to Related when a new scenario is covered by an
   existing decision, and append Notes (see below).
3. **Changing a decision means a new ADR.** Write the new one, list the old one in its Related
   section, and set the old one's status to `Superseded by [docs/adr/NNNN-new.md]`. A note isn't
   a substitute for this.
4. **Don't delete ADRs.** A `Proposed` ADR that's dropped can be deleted; anything that was ever
   `Accepted` stays as the record.

If the user asks for something that breaks a rule, say which rule and propose the compliant
alternative (usually a new, superseding ADR).

## Notes

Notes are an escape hatch for retrospective remarks on an ADR that's already Accepted (or
Superseded, or Deprecated): things learned in hindsight that are worth recording on the ADR but
don't change what was decided. Examples:

- "In hindsight, this should have been `universal` rather than tagging individual scenarios."
- "The 50 ms latency budget in Reasons turned out to be about 200 ms on Windows; the decision
  still holds."
- "Alternative B was revisited in 2027-03 and rejected again for the same reason."

Rules:

- **Append only.** Add new notes at the end of the `## Notes` section, creating it if needed.
  Never edit or remove an existing note.
- **Date every note**, as a bullet: `- **YYYY-MM-DD:** <note>`.
- **A note doesn't change the decision.** If the note would amount to "we now do X instead,"
  it's a new ADR that supersedes this one, not a note. When in doubt, ask the user which one
  they want.
- **Related corrections go through a note.** If a note says Related was wrong, like the
  `universal` example above, you may update Related to match, and the note records what it was
  before and why it changed. This is the only way to remove entries from an Accepted ADR's
  Related list.

```markdown
## Notes

- **2026-11-02:** In hindsight this decision applies to every feature that stores data, not
  just reminders. Related changed from the two `call-reminders.feature` scenarios to
  `universal`.
```

## Workflow

1. **Gather from the conversation.** Pull out the decisions, the reasons, the alternatives that
   were discussed and why they lost, and which features or scenarios are affected. Use the
   user's own reasoning; don't add reasons they didn't give.
2. **Ask for what's missing.** If the decision, the reasons, or the alternatives aren't in the
   conversation and can't be read from the code, ask a short list of questions in one go. Don't
   fill gaps with plausible-sounding rationale. A missing reason is worse when it's invented.
   It's fine to propose a draft answer in the question for the user to confirm.
3. **Determine Related.** Search `features/` for the scenarios the decision affects, or decide
   it's universal using the test above. Search `docs/adr/` for related or superseded ADRs. If
   the decision is about a feature that has no scenarios yet, say so and offer to write them
   with the `bdd-feature` skill first, then reference their IDs. Don't leave Related empty.
4. **Write the file** using the template, with today's date and the next number. If it
   supersedes an ADR, update that ADR's status in the same change.
5. **Report.** Give the path, the status, the Related entries, and anything you assumed or left
   for the user to confirm.

When the user wants to add something to an existing ADR instead, read the ADR first, decide
whether it's a note or a superseding ADR (see "Notes"), and follow the matching rules.

## Finding ADRs

ADRs for a scenario or feature (search `.` instead of `docs/adr/` to get the code and tests too):

```sh
grep -rn "#cancelled-no-reminder\]" docs/adr/
grep -rn "\[features/call-reminders.feature" docs/adr/
```

App-wide decisions:

```sh
grep -rln "^- universal$" docs/adr/
```
