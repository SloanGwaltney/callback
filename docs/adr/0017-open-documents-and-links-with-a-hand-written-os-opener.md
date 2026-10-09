# 0017. Open documents and links with a hand-written OS opener

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

Two features need the operating system to open something outside the app:

- An attached document opens in the computer's default app for its file type. The app opens its
  own stored copy, not the file the user attached (see
  [docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md]).
- A job's posting link opens in the user's web browser.

Go's standard library has no "open with the default handler" function. The options looked at:

- **Wails' `runtime.BrowserOpenURL`** (see [docs/adr/0002-use-wails-for-the-desktop-shell.md]).
  In Wails v2.12, its URL validator rejects the `file:` scheme, so it can't open documents. It
  returns nothing and only logs when opening fails, so the backend can't report the failure. It
  also rejects URLs containing characters such as `( ) ! ~ * [ ] { } $ ;`, which real posting URLs
  can contain, and those links silently don't open.
- **`github.com/pkg/browser`**, already in the module graph as an indirect dependency through
  Wails. It's about 20 lines: `ShellExecute` on Windows, `open` on macOS, `xdg-open` on Linux. Its
  latest version is from January 2024.
- **Hand-written code** for the same per-OS mechanisms.

On Windows, three mechanisms were compared:

- **`ShellExecuteW` called through the standard `syscall` package.** The documented Windows API for
  opening a file or URL with its default handler, and what Explorer uses. No child process, no
  command-line quoting, and it returns an error code (such as "no app is associated with this file
  type"). Calling it means writing `unsafe` pointer conversions, about 15 lines. Microsoft's docs
  say to initialize COM first because some handlers are shell extensions, though `pkg/browser`
  and Wails call it without doing so.
- **`rundll32 url.dll,FileProtocolHandler <path>`** through `os/exec`. Standard library only, with
  no `unsafe` code. But `FileProtocolHandler` is an undocumented legacy entry point. `rundll32` exits
  0 whether or not the file opened, so errors are lost. How it handles quoted paths with spaces is
  uncertain. Malware commonly abuses `rundll32`, so security tools may flag it.
- **`golang.org/x/sys/windows.ShellExecute`.** The same API, with a typed wrapper the Go team
  maintains. Already compiled into the app through Wails, so using it directly ships no new code.

A posting link is text the user typed. Passing arbitrary text to the OS default handler could
launch a local program rather than a web page.

## Decisions

1. Open attached documents and job posting links in the OS default handler with one hand-written
   opener in `internal/desktop`. The opener returns an error when opening fails.
2. On macOS, the opener runs `open <target>`, and on Linux `xdg-open <target>`, through `os/exec`
   without a shell.
3. On Windows, the opener calls `ShellExecuteW` from `shell32.dll` through the standard `syscall`
   package.
4. Don't use Wails' `runtime.BrowserOpenURL` or `github.com/pkg/browser` to open links or
   documents.
5. Only `http` and `https` links are opened, checked before the opener runs. Documents are opened
   only from the app's own stored copies.

## Reasons

- **1, 4.** `BrowserOpenURL` can't open documents at all. For links it fails silently and rejects
  some valid URLs. One opener that reports errors covers both features.
- **1, 2, 4.** `pkg/browser` is about as much code as writing it ourselves. As with
  [docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md], a dependency is
  worth its supply chain risk only when it replaces work we can't reasonably do ourselves.
- **3.** `x/sys/windows` adds no new supply chain risk, since it already ships with the app
  through Wails. Its only real cost is version coupling: as a direct dependency, its version could
  collide with whatever Wails bumps it to later. Writing the call through `syscall` instead means
  some hand-written `unsafe` code, the first in the codebase, but it's 10 to 15 lines and not a
  serious cost.
- **3.** `ShellExecuteW` is the documented API for this and reports failures, which `rundll32`
  doesn't. Losing errors would repeat the problem that rules out `BrowserOpenURL`.
- **5.** A link must be a web address (`invalid-job-link`). Checking the scheme again at open time
  means a stored link can never make the OS launch a local program.
- Accepted trade-offs: the OS calls themselves can't be unit tested. Code around the opener is
  tested with a fake, the command-building code is tested directly, and the real calls are
  checked by running the app on each OS. If a Windows file type ever fails to open, COM
  initialization is the first thing to check.

## Alternatives considered

- **Wails' `runtime.BrowserOpenURL` for both.** Rejected: it refuses `file:` URLs.
- **`BrowserOpenURL` for links, an opener for documents.** No new code for links. Rejected
  because links would fail silently and some valid URLs wouldn't open.
- **`github.com/pkg/browser` as a direct dependency.** Rejected: about 20 lines of code, not worth
  vetting and owning a dependency for, and it opens documents through `file://` URLs.
- **`rundll32 url.dll,FileProtocolHandler` on Windows.** Rejected: an undocumented entry point that
  loses errors, has uncertain quoting, and may trip security tools.
- **`golang.org/x/sys/windows.ShellExecute` on Windows.** No new supply chain risk and no `unsafe`
  code of our own. Rejected because it would make us a direct dependent, exposed to version
  collisions with Wails, to save 10 to 15 lines.

## Related

- [features/job-documents.feature#open-document] Open an attached document
- [features/job-documents.feature#document-kept-if-original-moved] An attached document still opens after the original file is moved
- [features/job-cards.feature#open-job-link] Open a job's posting link
- [features/job-cards.feature#invalid-job-link] A job link must be a web address
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
- [docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md] Store attached documents as files in the app data directory
- [docs/adr/0009-keep-user-data-in-the-os-conventional-app-data-directory.md] Keep user data in the OS-conventional app data directory
