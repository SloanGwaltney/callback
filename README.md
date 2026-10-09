# README

## About

This is the official Wails React-TS template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Prerequisites

Building callback needs a C compiler with CGO enabled on every platform, because the SQLite driver
(`github.com/mattn/go-sqlite3`) compiles SQLite's C source. See
[docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md](docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md).

- **Windows:** install MinGW-w64 gcc, for example through [MSYS2](https://www.msys2.org/)
  (`pacman -S mingw-w64-ucrt-x86_64-gcc`), and add its `bin` directory (e.g.
  `C:\msys64\ucrt64\bin`) to your `PATH`.
- **macOS:** install the Xcode Command Line Tools (`xcode-select --install`). Wails needs them too.
- **Linux:** install gcc (e.g. `build-essential` on Debian/Ubuntu). Wails needs it too, along with
  its GTK and WebKitGTK packages; run `wails doctor` to check.

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.
