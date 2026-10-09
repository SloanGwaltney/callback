# 0012. Wrap backend calls in custom React hooks

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

The React frontend (see [docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md])
gets its data by calling Go methods through the TypeScript bindings Wails generates in
`frontend/wailsjs/` (see [docs/adr/0002-use-wails-for-the-desktop-shell.md]). It needs a
consistent way to load that data, track loading and errors, and refresh after changes.

Everything is local. The Go side and its SQLite database are the source of truth, calls don't go
over a network, and there's little need for caching. Libraries like TanStack Query exist mainly
to cache and synchronize server state over a network.

Some state may eventually need to be shared across the app rather than owned by one component.
React provides Context and `useSyncExternalStore` for this. Zustand is a small library for
global state.

Frontend unit tests mock the Wails bindings (see
[docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md]).

## Decisions

1. Load and change backend data through custom React hooks (e.g. `useJobs`) that wrap the Wails
   bindings and expose the data along with its loading and error state. Don't use a data
   fetching library such as TanStack Query.
2. Only these hooks call the Wails bindings. Components never call bindings directly.
3. If state needs to be shared across the app, use React Context or a small hand-written store
   built on `useSyncExternalStore` first. Adopt Zustand only if those fall short, justified under
   the dependency policy in `AGENTS.md` at that time.

## Reasons

- **1.** TanStack Query is overkill for this app. Its value is caching and synchronizing remote
  data, and with everything local there's little to cache. Plain hooks fit the shape of the app
  and add no dependency.
- **2.** Each binding gets one call site, so grepping a Go method name leads to the single hook
  that uses it. It also simplifies testing: only the hooks' own tests need to mock the bindings,
  and components can be tested by mocking the hooks.
- **3.** Global state may never be needed. If it is, Context or a small store of our own is
  small enough to hand-write, so the supply chain policy calls for trying that first. Zustand is
  the right shape if they fall short.

## Alternatives considered

- **TanStack Query.** Rejected as overkill: its caching and synchronization solve problems a
  local app doesn't have.
- **Zustand from the start, or pre-approved as the global state tool.** Rejected because global
  state may never be needed, and when it is, a hand-written option comes first under the
  dependency policy.
- **Letting components call bindings directly.** Rejected because it scatters call sites, which
  makes them harder to find and to mock in tests.

## Related

- universal
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
- [docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md] Build the frontend as a React SPA with TypeScript and Vite
- [docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md] Require unit tests and test storage against real SQLite
