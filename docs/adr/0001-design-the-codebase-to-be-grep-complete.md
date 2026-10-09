# 0001. Design the codebase to be grep-complete

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

The context behind a task has usually been spread across many files and services, and much of it
has lived outside the code: specs, decisions, design docs. That works for human developers. A
person learns where things are once or twice, then knows where to look, and adding or removing a
place to look costs little.

An agent starts every session with nothing and has to rediscover all of that each time. The bigger
the codebase, the more searching it takes. Each extra hop means the agent has to merge data from
another source and work out how it connects to what it already has, often with little help. Where
the link isn't explicit, the agent guesses at connections that may not exist and draws incorrect
conclusions. Searching across loosely connected sources is a method built for humans, and it
scales badly in both tokens and accuracy.

## Decisions

1. Keep the context an agent needs for a task in the repository, linked to the code by plain-text
   references, so that a text search from any entry point finds everything related to it.
2. Link code and tests to the feature specs in `features/*.feature` with `@scenario` and step
   annotations, as defined in `docs/spec-annotations.md`.
3. Record architectural decisions as ADRs in `docs/adr/`, each with a Related section that uses
   the same `[features/x.feature#<id>]` references as code annotations, or `universal`.
4. Treat grep-completeness as evergreen: whenever a new kind of context starts to matter to the
   agent, find a way to make it greppable from an entry point.
5. Have the agent write code that carries these links, so that the work it does now keeps the
   codebase grep-complete for later sessions.

## Reasons

The goal is for the agent to get nearly all the context it needs just by looking at an entry
point, without blowing up token use as the codebase grows.

- **1.** Explicit links in the repository replace the agent's multi-hop search and the inference that
   comes with it. The connections are written down, so the agent doesn't have to guess them, and
   following one is a single deterministic search.
- **2 and 3.** Specs and decisions are the context that has usually lived outside the code. Using one
   reference format for both means a single search for a spec file or scenario ID returns its
   code, its tests, and the decisions behind it.
- **4.** The approach is meant to keep growing. Specs and ADRs are the first two kinds of linked context,
   not the only ones.
- **5.** The agent writes most of the code. If it adds the links as it goes, the system maintains
   itself instead of depending on separate documentation work that falls behind.

## Alternatives considered

- **Let the agent search from scratch every time.** Rejected. The more hops it takes, the more
  sources it has to merge and the more contextual links it has to infer, often without enough to
  go on. It ends up assuming connections that aren't there and drawing incorrect conclusions.
- **Keep context outside the repository** (wiki, issue tracker, external design docs). Rejected.
  The agent can reach it, but it isn't linked to the code. The agent still has to work out how
  each document applies to the code in front of it, which is the same merging problem.
- **A semantic index or RAG.** Rejected. The infrastructure is painful to set up, and it's very
  hard to build one that gives deterministic results. Retrieval quality depends on the embeddings,
  and it only works when related things are semantically similar, which in code they often
  aren't. Explicit links connect exactly the things that belong together.
- **One large, always-loaded context file** (`AGENTS.md` / `CLAUDE.md`). Rejected. It spends
  tokens on every task whether or not the task needs them, and past a certain size agents don't
  read the whole file. They skim and jump around it.
- **Rich docs in the repository, with pointers to them from `AGENTS.md`.** Rejected. It's a good
  idea in theory, but docs kept apart from the code they describe tend to go stale.

## Related

- universal
