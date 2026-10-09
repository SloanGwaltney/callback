# 0002. Use Wails for the desktop shell

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

callback holds a user's job search: the companies they've applied to, where each application
stands, and the documents that go with it. That information should stay on the user's machine,
not sit on a web server.

The app is essentially a web app that we want to run natively. It doesn't need much from the
desktop: no heavy use of the system tray, native menus, or similar OS integration. Most of the
code is written by agents, and the codebase is designed to be grep-complete (see
[docs/adr/0001-design-the-codebase-to-be-grep-complete.md]), so the backend language and shell
affect how well that design holds up.

## Decisions

1. Build callback as a local desktop application that keeps the user's job search data on their
   machine, with no web server holding it.
2. Write the backend in Go.
3. Use Wails (`github.com/wailsapp/wails`) as the desktop shell: a Go backend with a web frontend
   rendered in the operating system's webview.
4. Use Wails v2 (`github.com/wailsapp/wails/v2`), not v3.

## Reasons

- **1.** Job search data is personal. Running locally keeps it on the user's machine instead of a
  server we'd have to host and secure.
- **2.** Go is simple enough for this app and serves it well, without the cost of a language like
  Rust (see Tauri under Alternatives).
- **2.** Go suits the grep-complete design. Struct tags serve two purposes at once: they're read by
  reflection at runtime, and they're plain text in the source, so the same tag that drives the
  program can also be found by a text search. More broadly, Go has little of the metaprogramming
  that spreads one idea across several files and layers of abstraction. Without it, the code that
  implements a concept tends to sit where a text search for that concept will find it.
- **2.** The trade-off accepted is that Go isn't entirely free of indirection: it relies on code
  generation (`go:generate`), and Wails generates the frontend bindings. Generated code can still
  separate a concept from where it's defined. It's less of a problem than metaprogramming, though,
  because generated code is plain text written to a known location, which makes it easier to
  link. For example, `AGENTS.md` could point to where generated files land, or code comments
  could link generated code to its source. No approach is chosen yet, but there are workable ones,
  which isn't true of much runtime metaprogramming.
- **3.** Within Go, Wails is the closest thing to Electron. It has all the features we need and more
  that we probably won't use, it's well documented, and it's well supported and actively
  maintained. There was no compelling reason not to use it.
- **3.** Rendering in the system webview instead of a bundled Chromium keeps the app light. The
  trade-off accepted is that the UI runs on a different engine per platform (WebView2 on Windows,
  WebKit on macOS, WebKitGTK on Linux), so rendering can differ between them. For a simple web
  app without heavy desktop integration, that's acceptable.
- **3.** Wails generates TypeScript definitions for the Go functions it binds to the frontend.
  Webview-based options like Lorca and webview_go just bind a Go function into JavaScript, and
  TypeScript has no way to know it exists, so we'd have to maintain our own type definition files
  or TypeScript library by hand. AI agents perform significantly better with a type system, and
  having one that's generated deterministically from the bindings, with no hand upkeep, is a big
  win. It also loosely supports grep-completeness, since the generated definitions name the same
  functions as the Go code they bind.
- **4.** Wails v3 is still in pre-release, and v2 is stable. This is a personal preference for
  stability rather than a flaw in v3: v3 is used in production, and choosing it would not have
  been wrong.

## Alternatives considered

- **webview/webview_go** (Go bindings for the webview library). It would work, but it felt
  primitive and not as well documented as we'd like. It also shows little ongoing maintenance: as
  of this decision, the repository has had no commits since August 2024. Like Lorca, it binds Go
  functions into JavaScript without generating TypeScript definitions for them.
- **Lorca** (`zserge/lorca`), which launches the user's installed Chrome or Edge and drives it
  from Go. It would also work, but it depends on the user having a Chromium browser installed,
  and it also shows little ongoing maintenance: as of this decision, the repository has had no
  commits since Jan 2024. It binds Go functions into JavaScript without generating
  TypeScript definitions for them.
- **Electron.** Rejected as resource-heavy: it embeds its own Chromium. Its main advantages over a
  system webview, such as deeper desktop integration through the system tray and native menus,
  don't matter for this app, so the bloat buys little. Its TypeScript story is also poor: using
  TypeScript means setting up Vite to transpile several separate scripts, which is a lot of
  headache for little benefit.
- **Tauri.** The most reasonable alternative. It's written in Rust, extremely performant, and
  memory safe. Rejected because taking on Rust is overkill for this app. Even with agents writing
  most of the code, Rust has slow compile times, and its memory safety and speed aren't free. We'd
  reach for it if the app would benefit greatly from those features, but it's simple, and Go
  serves it well.
- **Wails v3.** A valid choice, since it's used in production. We chose v2 because v3 is still in
  pre-release and we prefer stability.

## Related

- universal
- [docs/adr/0001-design-the-codebase-to-be-grep-complete.md] Design the codebase to be grep-complete
