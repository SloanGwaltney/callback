# 0011. Style the frontend with CSS Modules and CSS variables

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

The React frontend (see [docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md])
needs a styling approach and a source for its UI components. Agents write most of the code.

Vite supports CSS Modules natively for any file named `*.module.css`, and the frontend's
`src/vite-env.d.ts` already references `vite/client`, which types those imports. Using them
requires no new packages or configuration.

Tailwind is the main alternative. It's heavily represented in training data, and it's the
foundation for ready-made component collections, most notably shadcn/ui. shadcn/ui is built on
Radix, a headless library that supplies accessible behavior: focus trapping, closing with Escape,
keyboard navigation, and ARIA attributes. Not using Tailwind means writing our own components,
including that behavior. The `delete-job` scenario's confirmation step is the first place a
dialog is needed.

Native HTML covers some of that behavior. `<dialog>` opened with `showModal()` traps focus,
closes on Escape, and renders a backdrop, and both WebView2 and WKWebView support it.

## Decisions

1. Style components with CSS Modules: one `*.module.css` file per component, imported into it.
2. Define design tokens (colors, spacing, type, and similar values) as CSS variables in one
   global tokens file, e.g. `src/styles/tokens.css` on `:root`, imported once at startup. Modules
   use `var(--...)` instead of hard-coding those values.
3. Don't use Tailwind or a Tailwind-based component collection such as shadcn/ui. Write our own
   components.
4. Get accessible interactive behavior from native HTML elements first (such as `<dialog>` and
   `<select>`). Reach for a headless component library only for a widget native HTML can't
   provide, and justify it under the dependency policy in `AGENTS.md`.

## Reasons

- **1.** CSS Modules are essentially a free upgrade over plain CSS: Vite already supports them,
  so they add no dependency, and they scope class names to a component so styles don't leak
  between components. They also keep styles grep-complete, since `styles.card` in a component
  leads to `.card` in the module file beside it.
- **2.** A single tokens file gives agents one place to find and reuse the app's values, which
  keeps the UI consistent across components written separately.
- **3.** Tailwind's utility classes make more sense for a human than for an agent. With an agent
  writing most of the code, long utility class strings are mostly extra token spend, and it's
  not clear they help a machine. Agents write good CSS directly.
- **3.** Accepted trade-off: we give up existing Tailwind-based components, mainly shadcn/ui,
  and build our own instead. For this project that's acceptable.
- **4.** Native elements bring accessible behavior with no dependency, which fits the supply
  chain policy. Accepted trade-off: widgets native HTML doesn't cover, such as a combobox, will
  need either our own implementation or a justified dependency.

## Alternatives considered

- **Plain global CSS.** What the Wails template starts with. Rejected because CSS Modules cost
  nothing extra and add scoping.
- **Tailwind, with shadcn/ui for components.** Has a lot of training data and ready-made
  accessible components. Rejected for the token-spend reason in 3; losing shadcn/ui was judged
  acceptable.
- **Adopt a headless library (Radix or React Aria) from the start,** styled with CSS Modules.
  Would give consistent accessible behavior across all widgets. Rejected in favor of native HTML
  first, to avoid a dependency until a widget actually needs one.

## Related

- universal
- [docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md] Build the frontend as a React SPA with TypeScript and Vite
