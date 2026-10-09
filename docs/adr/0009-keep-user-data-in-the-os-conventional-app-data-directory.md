# 0009. Keep user data in the OS-conventional app data directory

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

callback keeps the user's data on their machine: a SQLite database (see
[docs/adr/0003-store-user-data-in-sqlite.md]) and a folder of copied attached documents, each up
to 10 MB (see [docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md]).
Both need a directory to live in, and ADR 0008 left where that directory is to this decision.

Each OS has a convention for where an app keeps its data:

- **Windows:** the known folders `%APPDATA%` (Roaming) and `%LOCALAPPDATA%` (Local). Roaming is
  synced between machines on domain-joined networks and is meant for small settings.
- **macOS:** `~/Library/Application Support/<app>`.
- **Linux:** the XDG Base Directory spec's `$XDG_DATA_HOME`, which defaults to `~/.local/share`
  when unset.

Go's standard library has no function for the data directory. `os.UserConfigDir()` is the
closest, but it returns Roaming on Windows and `$XDG_CONFIG_HOME` (`~/.config`) on Linux, which
are config locations, not data locations. Wails v2 doesn't provide one either.

Without an override, `wails dev` and any test that runs the real startup path would read and
write the same directory as the user's installed app.

## Decisions

1. Keep all user data in a directory named `callback` under the OS-conventional data location:
   `%LOCALAPPDATA%\callback` on Windows, `~/Library/Application Support/callback` on macOS, and
   `$XDG_DATA_HOME/callback` on Linux, falling back to `~/.local/share/callback` when
   `XDG_DATA_HOME` is unset.
2. Resolve that location with a small hand-written Go function that switches on
   `runtime.GOOS`, not with `os.UserConfigDir()` or a third-party library.
3. When the `CALLBACK_DATA_DIR` environment variable is set, use it as the data directory instead
   of the OS location.

## Reasons

- **1.** Following each OS's convention puts the data where users, backup tools, and uninstallers
  expect an app's data to be.
- **1.** On Windows, Local rather than Roaming, because the directory holds a live SQLite file
  and documents up to 10 MB each. Roaming is meant for small settings and gets synced on domain
  networks, which suits neither a large folder nor an open database.
- **1.** The folder is named `callback` to match the app name in `wails.json`. Accepted
  trade-off: a generic name could in principle clash with another app's folder, which a
  reverse-DNS ID would avoid.
- **2.** Some dependencies are necessary. Writing everything ourselves isn't feasible, and the
  app already relies on Wails, sqlc, goose, and a SQLite driver for that reason. But every
  dependency is an inherent supply chain risk: its code, its maintainers, and everything it
  pulls in become part of what ships to the user's machine. That risk is worth taking only when a
  dependency replaces work we can't reasonably do ourselves. Resolving a data directory is about
  twenty lines of code, so the risk isn't worth it here.
- **2.** `os.UserConfigDir()` would avoid both a dependency and hand-written code, but it returns
  the wrong locations on Windows and Linux.
- **3.** An override lets development runs and tests use a throwaway directory, so they never
  touch the user's real data.

## Alternatives considered

- **`%APPDATA%` (Roaming) on Windows.** Rejected because Roaming is for small settings and is
  synced on domain networks. That doesn't suit a live database or a folder of documents.
- **`os.UserConfigDir()`.** Standard library, no code to write, but it uses Roaming on Windows
  and `~/.config` on Linux, which are config locations, not data locations. Rejected.
- **The `adrg/xdg` library.** Implements XDG plus Windows and macOS equivalents, and its Windows
  data home is `%LOCALAPPDATA%`. Rejected because every dependency is a supply chain risk, and
  that risk isn't worth taking to avoid writing about twenty lines of code.
- **A reverse-DNS folder name** (e.g. `com.sloangwaltney.callback`). Avoids name clashes and is
  common on macOS, but is uglier on Windows and Linux. Rejected in favor of `callback`.
- **No override.** Always use the OS location and have tests pass a directory into the code
  directly. Rejected because `wails dev` would then share data with the installed app.

## Related

- universal
- [docs/adr/0003-store-user-data-in-sqlite.md] Store user data in SQLite
- [docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md] Store attached documents as files in the app data directory
