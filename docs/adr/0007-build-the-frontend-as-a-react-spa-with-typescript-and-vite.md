# 0007. Build the frontend as a React SPA with TypeScript and Vite

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

Wails renders the frontend in the system webview and exposes the Go backend to it through
generated bindings (see [docs/adr/0002-use-wails-for-the-desktop-shell.md]). That already gives
the frontend the shape of a single page app: it runs in the browser and calls the backend for
data. The only difference from a typical SPA is that the calls are RPC-style bindings instead of
HTTP. Whether to build an SPA was therefore settled by the shell; the open question was which
framework to build it with.

Most of the code is written by agents, and the codebase is designed to be grep-complete (see
[docs/adr/0001-design-the-codebase-to-be-grep-complete.md]). Nothing in the app is expected to be
performance heavy: no massive lists, and nothing that should run into React's known edge cases.

The project was scaffolded from the Wails React + TypeScript template, which uses Vite, so the
frontend was already React in practice. This ADR records that choice and why it stands.

## Decisions

1. Build the frontend as a single page app that talks to the Go backend through the Wails
   bindings, without a meta-framework (no Next.js, SvelteKit, Nuxt, or similar).
2. Use React as the frontend framework.
3. Write the frontend in TypeScript, not JavaScript.
4. Build and serve the frontend with Vite.

## Reasons

- **1.** The Wails bindings already give the frontend the shape of an SPA, so an SPA fits the
  shell with no extra machinery. A meta-framework would add a lot of complexity (routing
  conventions, server rendering, its own build layer) that a desktop app talking to a local
  backend doesn't need.
- **2.** React is the default for web development, and for agent-written code there needs to be
  a reason not to use it. This project has none. The training data for React is immense for any
  AI agent, which is the main reason for choosing it over Vue, its closest rival.
- **2.** Choosing React means choosing its ecosystem, not just its rendering. Many tools that
  seem framework-neutral are in practice built for React, such as shadcn/ui for styled
  components. With another framework we'd have to find less-used alternatives or community
  ports of them.
- **2.** The trade-off accepted is that React has known edge cases. This app isn't expected to
  hit them, since nothing it does is performance heavy and it doesn't render massive lists.
- **2.** The other trade-off accepted is React's re-render model. Frameworks with fine-grained
  reactivity (Solid, Svelte, Vue) avoid a class of mistakes React invites: `useEffect`
  dependency arrays that are wrong, effects that shouldn't exist, stale closures, and needless
  re-renders. At this app's scale, those are pain points of writing React, not of how it
  performs or what it can do, and agents rather than people write most of the code. The
  training data on React is rich enough, mistakes and fixes alike, that those pain points matter
  little here. React isn't the choice we'd pick on taste, but it's the right one for this
  project.
- **3.** TypeScript is better for agents and for code in general. JavaScript should only be used
  where TypeScript isn't an option. It also makes use of the TypeScript definitions Wails
  generates for the Go bindings (see [docs/adr/0002-use-wails-for-the-desktop-shell.md]).
- **4.** Vite is the modern way to set up any SPA. The alternative, webpack, is something we have
  no interest in touching again if it can be avoided.

## Alternatives considered

- **Svelte.** It can run as an SPA without SvelteKit, but the documentation and community lean
  heavily towards SvelteKit, and documentation for Svelte on its own is not abundant. Since we
  don't want a meta-framework, that made Svelte less appealing.
- **Angular.** Historically cumbersome, very verbose, and full of indirection, which is not ideal
  for agent-focused development. A quick check at the time of this decision showed it has
  modernized: standalone components are the default (v19), signals are stable (v20), and new
  projects no longer need Zone.js (v21), which removes much of the boilerplate. It's still built
  around dependency injection, decorators, and services, though, so it carries more indirection
  than React. It also has no big advantage over React for this app and less training data behind
  it. Rejected for both reasons.
- **Vue.** Neither React nor Vue has a big selling point over the other for this kind of
  application. React won on the size of its training data.
- **Solid.** Rejected because it's far less widely used than React, Vue, or Angular, so agents
  have less training data for it.
- **Lit (web components) and lightweight libraries such as Alpine.** More native JavaScript
  options were possible, but they don't make much sense here: agents write most of the code,
  nothing is performance heavy, and they give up React's training-data advantage without buying
  anything this app needs.
- **HTMX.** Doesn't fit. It expects the backend to return HTML templates, and returning templates
  over a Wails binding would be weird.
- **JavaScript instead of TypeScript.** Rejected; see reason 3.
- **webpack instead of Vite.** Rejected; see reason 4.

## Related

- universal
- [docs/adr/0001-design-the-codebase-to-be-grep-complete.md] Design the codebase to be grep-complete
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
