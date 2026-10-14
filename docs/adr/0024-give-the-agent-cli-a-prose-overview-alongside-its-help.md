# 0024. Give the agent CLI a prose overview alongside its help

- **Status:** Accepted
- **Date:** 2026-10-13

## Context

[docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md] made agent
features a CLI. Its decision 4 said the CLI's `--help` is how an agent learns to use it, and that
no separate agent instructions are shipped.

While specifying the CLI, it became clear that an agent needs more than a list of commands and
options. It also needs to know what Callback keeps track of, how the commands fit together, how
output, IDs, dates, long text, and errors work, how to work safely by making a backup before
changing many jobs and restoring it if the changes go wrong, what it can't do, and that it can
read the stored copies of documents. That's prose, the kind of thing an `AGENTS.md` holds, but
about the CLI.

`--help` is the standard way for a command-line program to list its commands and options.

## Decisions

1. The CLI has a separate flag that prints a prose overview of Callback and the CLI for an agent,
   like an `AGENTS.md` that describes the CLI. Its name is chosen when it's built.
2. `--help` keeps the standard convention: it lists the commands, and each command's help lists
   its options and the values they take. It points to the overview flag.
3. The overview tells the agent it can read the stored copies of documents, and that it changes a
   document by attaching a new file, not by editing the stored copy.
4. Carried over from ADR 0023 unchanged: its decisions 1, 2, 3, and 5, and its note of 2026-10-13
   settling decision 6. This replaces only its decision 4.

## Reasons

- **1.** The overview gives an agent the context that a list of commands can't: what the data is,
  how to work with it, and how to work safely.
- **1, 2.** `--help` listing every command is the standard convention, and it should be respected,
  so the overview goes in a flag of its own rather than changing what `--help` does.
- **3.** The agent should read stored documents when they're relevant to its work; that's why it's
  given their paths. Attaching a new file to change a document is the same flow the user would
  have to take in the app.

## Alternatives considered

- **Keep `--help` as the only way an agent learns the CLI (ADR 0023, decision 4).** Rejected: it
  doesn't carry the prose overview the agent needs.
- **Make the overview a help topic** (for example, `help overview`). Rejected: `--help` should
  stick to the standard convention of listing the commands.

## Related

- universal
- [docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md] Let users drive agent tasks from their own agent through a CLI
