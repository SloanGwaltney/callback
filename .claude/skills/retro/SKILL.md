---
name: retro
description: Run a retrospective on the work just done in this conversation. Look back at what was painful, slow, or took several tries, sort each cause into your own mistakes, tooling gaps, awkward code or conventions, and costs of working as designed, and propose concrete fixes for the user to choose from. Use when the user invokes /retro or asks what was painful about a task and what would make it easier.
argument-hint: [what to look back on, e.g. "the search feature"]
disable-model-invocation: true
---

# retro

Look back on the work just done and find what made it harder than it had to be, so the next task
goes more smoothly. The goal is less friction in the harness: the tools, scripts, skills, docs,
and conventions an agent works with in this repo. It isn't to grade the work or apologize for it.

If the user names something to look back on, focus on that. Otherwise, cover the most recent task
in the conversation.

## 1. Gather the evidence

Read back through the conversation and list every point of friction, with what actually happened.
Look for:

- **Retries.** Commands that failed and were run again, edits that didn't apply, files rewritten
  more than once.
- **Check failures.** What `bash scripts/check.sh`, the Stop hook, `speclint`, `spectrace`, Biome,
  `tsc`, or the tests caught, and how many rounds it took to fix.
- **Workarounds.** Scripts written in the scratchpad, shell loops, one-off steps files, calls into
  the app's internals, or anything else done by hand because nothing existing covered it.
- **Lookups.** Files read only to find a format, a name, or a convention that could have been
  written down where it's needed.
- **Repetition.** The same kind of edit or annotation made many times over.
- **Corrections.** Anything the user had to correct or explain, and why it was missed.

Stick to what's in the conversation. Don't invent friction, and don't leave any out because it
was your own fault.

## 2. Sort each cause

For each point, find the cause, and put it in one group:

| Group | Meaning | Typical fix |
|---|---|---|
| **My mistakes** | You chose a poor approach or made an error a tool wouldn't have prevented. | Name it plainly. If it's likely to recur, a memory, or a line in `AGENTS.md` or a skill. |
| **Tooling gaps** | A tool, script, or helper was missing, or lacked an option, so you worked around it. | A new tool, flag, script, or helper. |
| **Awkward code or docs** | The code, docs, or a skill made the work harder than it needed to be: hard to find, hard to test, unclear, or out of date. | A refactor, a doc fix, or a skill edit. |
| **Working as designed, but costly** | A convention or rule did what it's meant to, at a real cost. | A discussion. Changing it is a decision, so suggest `/adr`. |

Be honest about which group something belongs in. If you could have avoided it with the tools you
had, it's your mistake, not a tooling gap.

## 3. Propose fixes

For each fix, say what it is, what friction it removes, and roughly how big it is. Rank by payoff:
friction that will come up on every task beats friction that came up once. Leave out fixes whose
cost is bigger than the friction they remove, and say so if that's the honest answer for a point.

Follow the repo's rules when proposing:

- A fix that needs a new package goes through the dependency rules in `AGENTS.md`: justify it,
  and never install it.
- A fix that changes a convention, a rule, or the annotation format is a decision. Lay out the
  trade-off and suggest `/adr`. Don't edit `docs/adr/`.
- Prefer extending a tool that already exists over adding a new one.

Also note what worked well and why, briefly, so it isn't changed by accident.

## 4. Report, and wait

Report in the groups from step 2, each point with what happened and its fix. End with the fixes
you'd do first, and ask which ones the user wants. Don't start on any fix until the user picks.
