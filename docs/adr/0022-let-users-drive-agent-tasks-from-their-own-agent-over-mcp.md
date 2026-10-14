# 0022. Let users drive agent tasks from their own agent over MCP

- **Status:** Superseded by [docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md]
- **Date:** 2026-10-12

## Context

Future features could let a user's AI agent work with their Callback data: drafting a resume for
a job, or suggesting which jobs to put effort into. The goal is for the user to use the agent
they already pay for through a subscription, such as Claude Code on a Pro plan, rather than an
API key with usage-based pricing.

The vendors' terms, as of this date:

- **Anthropic** doesn't permit third-party developers to offer Claude.ai login in their own
  applications, to route requests through Free, Pro, or Max plan credentials on behalf of their
  users, or to collect, store, or intermediate Claude.ai credentials or session tokens. It does
  allow an end user to sign in to the unmodified Claude Code binary with their own subscription.
  It notes that Pro and Max usage limits assume "ordinary, individual usage," and reserves the
  right to enforce without notice
  ([legal and compliance](https://code.claude.com/docs/en/legal-and-compliance)). The policy
  changed several times during 2026.
- **Google's** Gemini CLI FAQ calls third-party software piggybacking on Gemini CLI's OAuth "a
  direct violation" of its terms and points third parties to API keys
  ([FAQ](https://geminicli.com/docs/resources/faq/)). Users report suspended accounts.
- **OpenAI's** Codex app-server documents ChatGPT sign-in for embedding Codex, but its auth docs
  say to use API keys for programmatic Codex CLI workflows
  ([app-server](https://developers.openai.com/codex/app-server),
  [auth](https://learn.chatgpt.com/docs/auth)).

The Model Context Protocol (MCP) lets an app offer tools that agents call. Claude Code, Claude
Desktop, Codex, Gemini CLI, and other agents support it. With an MCP server, the user works in
their own agent, and the agent reads and changes Callback's data.

## Decisions

1. Agent features are built as an MCP server that Callback provides. Users drive them from their
   own agent, outside the application.
2. Callback doesn't run, embed, or sign in to an AI agent or model inside the application, and
   doesn't ask users for an API key.

## Reasons

- **1, 2.** Agent features would be much friendlier inside the application. But every way of
  doing that without API keys runs into a different vendor's terms, which keep changing and are
  enforced by suspending accounts. That makes in-app agents a legal landmine. With MCP, the user
  uses their own agent the normal way and Callback never touches their login, so it stays within
  every vendor's terms.
- **2.** API keys aren't practical for the average user.
- **1.** Users already working with AI flows probably already have a subscription agent. Job
  seekers who use AI tools will want to use that subscription rather than pay API pricing.
- **1.** More and more people are expected to work in the terminal over time, so the audience
  for driving Callback from an agent should grow.
- Accepted trade-off: this is a net negative for the average user. Agent features only reach
  users who already use an AI agent, and they can't be used from inside Callback. The focus now
  is on users already using AI agents. Users who aren't are left to where the future takes them.

## Alternatives considered

- **Run the user's installed agent CLI from Callback** (`claude -p`, `codex app-server`), with
  results shown in the app. The AI would work inside Callback, and Anthropic's terms allow the
  user's own unmodified Claude Code binary. Rejected: Google's terms make Gemini risky, OpenAI's
  docs contradict each other, the "ordinary, individual usage" caveat applies, and each vendor's
  terms have changed and can change again. It also means one integration per agent.
- **Speak the Agent Client Protocol (ACP)** to many agents through one protocol. Rejected:
  Claude and Codex need adapters the user installs, and the Claude adapter is built on the Agent
  SDK, whose docs say third parties may not offer claude.ai login for products built on it.
- **Ask users for an API key.** The clearest way to be within the terms, and it allows in-app
  features. Rejected: not practical for the average user, and users with a subscription don't
  want to pay API pricing on top of it.
- **Read the agent's stored login and call the model directly.** Rejected: Anthropic explicitly
  forbids it, and Google suspends accounts for the equivalent.

## Related

- universal
