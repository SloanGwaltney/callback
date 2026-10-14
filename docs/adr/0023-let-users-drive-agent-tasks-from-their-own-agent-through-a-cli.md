# 0023. Let users drive agent tasks from their own agent through a CLI

- **Status:** Superseded by [docs/adr/0024-give-the-agent-cli-a-prose-overview-alongside-its-help.md]
- **Date:** 2026-10-13

## Context

[docs/adr/0022-let-users-drive-agent-tasks-from-their-own-agent-over-mcp.md] decided that agent
features are an MCP server that users drive from their own agent, outside the application, and
that Callback doesn't run, embed, or sign in to an agent or model, or ask for an API key. Its
account of the vendors' terms still holds.

Callback is a local app. A local MCP server for it would run over stdio as a separate process
that the agent starts, opening the same SQLite database as the app. A command-line program would
do the same. Either way, the agent works outside the application, not with the user inside it.
Terminal agents such as Claude Code, Codex, and Gemini CLI can already run commands.

What MCP adds over a CLI:

- Chat-only agents such as Claude Desktop, which can't run commands, reach local tools only
  through MCP. Adding a local MCP server to Claude Desktop still means editing a JSON config, often
  with an absolute path to the binary, since it runs with a minimal PATH. One-click packaged MCP
  extensions would avoid that, but they are separate packaging and distribution work.
- An MCP connection is a session. The rules in `features/agent-backups.feature` are written
  around "this session", and a CLI call is its own process with no session.

The Wails app is built as a Windows GUI-subsystem executable. Run from a terminal, such an
executable returns the prompt at once, and its output is lost unless it attaches to the console.

## Decisions

1. Agent features are a command-line program that users drive from their own agent, outside the
   application. This replaces decision 1 of ADR 0022, an MCP server.
2. The CLI is its own Go binary, in its own folder under `cmd/`, built over the same services as
   the app.
3. The release build produces the CLI alongside the app, for every platform the app ships on, and
   puts both binaries in the same folder.
4. The CLI's `--help` is how an agent learns to use it. No separate agent instructions are shipped
   for that.
5. Carried over from ADR 0022 unchanged: Callback doesn't run, embed, or sign in to an AI agent or
   model inside the application, and doesn't ask users for an API key.
6. How the session rules in `features/agent-backups.feature` work over a CLI is left to a later
   decision.

## Reasons

- **1.** An MCP server wouldn't bring the agent any closer to the app than a CLI does. Both are a
  separate process over the same database, and the agent works outside the app either way. MCP
  would add a protocol on top of the same work.
- **1.** It keeps ADR 0022's legal reasoning. The user's agent runs a program the normal way, and
  Callback never touches their login.
- **1.** It serves the two groups Callback is aimed at, and not the middle. Users who don't use
  agents work in the UI and never see agent features. Users who already work in Claude Code or
  another terminal agent get a tool that fits how they work. The middle group, users of chat-only
  agents like Claude Desktop, is a decent portion of users, but MCP serves them only halfway: they
  still have to edit a config file, and anyone who can do that can run a terminal agent.
- **1.** The same subscription covers both. A Claude Desktop user on a Pro plan can use Claude
  Code at no extra cost, so what stands in their way is comfort with the terminal, not money, and
  ADR 0022 already expects terminal use to grow.
- **1.** It's the cheaper way to be wrong. A CLI needs no dependency: Go's `flag` package and a
  hand-written subcommand switch are enough. An MCP server would need the MCP Go SDK or a
  hand-written JSON-RPC layer. If chat-only agents turn out to matter, an MCP adapter can be added
  later over the CLI's commands, mostly as a translation layer. Starting with MCP takes on its
  dependency before it's known to be needed.
- **1.** A CLI loads its help only when the agent asks for it, where an MCP server's tool schemas
  are loaded up front. People and scripts can use a CLI too.
- **2.** In Go, another binary is just another folder under `cmd/`. A console binary of its own
  avoids the GUI-subsystem executable's lost output. Building it over the services from
  [docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md] keeps
  the business rules in one place.
- **3.** Both binaries open the database through `db.Open`, which runs the migrations. Shipping
  them from the same release keeps them on the same schema. A CLI from a newer release run against
  an older app would migrate the database out from under it.
- **4.** The regular help already lists every command and flag, which is what an agent needs.
- Accepted trade-offs:
  - Agent features don't reach users of chat-only agents.
  - Putting both binaries in one folder doesn't put the CLI on the user's PATH. Without an
    installer, the user adds the folder to their PATH or gives the agent the CLI's full path.
  - The user has to tell their agent the CLI exists, where an MCP server's tools appear in the
    agent once it's registered.
  - The CLI needs CGO, like the app, because of the SQLite driver.

## Alternatives considered

- **Keep the MCP server from ADR 0022.** Rejected for the reasons above. Its one advantage that
  holds is reaching chat-only agents, which Callback isn't aimed at.
- **An MCP adapter alongside the CLI, for chat-only agents.** Rejected until there are enough
  users in that middle group to be worth the extra work. It isn't clear that an MCP server would
  give them a complete experience, and building a middle-ground solution that may not serve them
  fully isn't worth it. If the middle group grows enough, the adapter can be built over the CLI's
  commands.
- **Run the CLI from the app's own binary as subcommands.** Rejected: the app is a Windows
  GUI-subsystem executable, which returns the prompt at once and loses its output when run from a
  terminal unless it attaches to the console. A binary of its own avoids that.
- **An HTTP API served by the running app.** Rejected: it works only while the app is open, and
  it needs a port and some form of auth on localhost.
- **An agent inside the application.** Rejected again, for ADR 0022's reasons, which haven't
  changed.

## Related

- universal
- [docs/adr/0022-let-users-drive-agent-tasks-from-their-own-agent-over-mcp.md] Let users drive agent tasks from their own agent over MCP
- [docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md] Put business rules in services over a transactional store interface
- [docs/adr/0019-release-by-pushing-a-tag-that-builds-a-draft-github-release.md] Release by pushing a tag that builds a draft GitHub Release

## Notes

- **2026-10-13:** Decision 6 is settled in `features/agent-backups.feature`: agent backups have no
  sessions. The decision to drop the session rules was made because agents can already reach the
  backup files on disk, so session rules could only ever be a guardrail on the CLI, not a real
  block. An agent lists and restores any backup an agent made, never a daily backup, and can't
  delete backups. Each agent backup is kept for 1 to 90 days, chosen when it's made, with its
  expiry date in its file name rather than in the database, so that restoring a backup doesn't
  change which backups are kept. Expired backups are removed when the app starts and on every CLI
  run. The backup a restore makes first is kept for 7 days.
