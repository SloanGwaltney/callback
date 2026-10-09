# callback

A desktop app for tracking a job search, built with AI coding agents in a codebase designed so
agents can find, build, and check their own work. Each job you're applying to is a card on a
Kanban board, laid out by status, with its details, closing reason, and the documents you sent
attached.

## Built for AI agents

Most of callback's code is written by Claude Code. The interesting part of this repo isn't the
app; it's the structure around it, which makes agent-written code traceable and checkable:
every behavior is specified first, every line that implements or tests it links back to that
spec, and tools verify those links on every turn the agent takes.

### Every behavior traces from spec to SQL to UI

Each feature starts as plain-language scenarios in [features/](features/), every one with a
stable ID and a status:

```gherkin
@status:implemented @id:card-shows-document-count
Scenario: The card shows that documents are attached
  Given I have a job for "Software Engineer" at "Acme" with 2 attached documents
  When I look at the card on the board
  Then the card shows that it has 2 documents
```

The code that carries out each step, and the tests that check it, carry a comment naming that
step, at every layer from the SQL query up to the React component:

```ts
{/* @then [features/job-documents.feature] the card shows that it has 2 documents */}
{card.documentCount > 0 && <p className={styles.documents}>{documents}</p>}
```

So a text search for a scenario ID or a step finds everything about it: its code, its tests,
and the decisions behind it. The codebase is **grep-complete**
([ADR 0001](docs/adr/0001-design-the-codebase-to-be-grep-complete.md)), which is how an agent
starting cold finds its way around without reading everything. `spectrace` lays it out layer by
layer:

```
$ go run ./cmd/spectrace card-shows-document-count

Then the card shows that it has 2 documents  (features/job-documents.feature:32)
  SQL query      internal/db/queries/jobs.sql:6               @then
  storage test   internal/db/store_test.go:237                @then
  service        internal/jobs/board.go:90                    @then
  service test   internal/jobs/board_test.go:88               @then
  frontend       frontend/src/components/JobCard.tsx:42       @then
  frontend test  frontend/src/components/JobCard.test.tsx:36  @then

Fully linked: @scenario and all 3 steps in code and tests.
```

### The links are enforced, not hoped for

- **`speclint`** ([internal/speclint/](internal/speclint/)) fails on any annotation that doesn't
  match the spec word for word, uses the wrong keyword, or points at a missing or withdrawn
  scenario, and on any implemented scenario or step with no link in code or in tests.
- **One script** ([scripts/check.sh](scripts/check.sh)) runs speclint alongside gofmt, `go vet`,
  `tsc`, Biome, and every test, so the agent, a person, and a pipeline all check the same thing.
  The agent runs it on every turn; see the Stop hook below.

### A repeatable workflow, as agent skills

The development loop is captured as project skills in [.claude/skills/](.claude/skills/), so
each step runs the same way every time:

1. **`/bdd-feature`** turns a plain-language description into scenarios with IDs and statuses.
2. **`/adr`** records a decision (a library, a data model, a pattern) with the reasons and the
   alternatives rejected, and links it to the scenarios it affects. 17 ADRs so far, in
   [docs/adr/](docs/adr/).
3. **`/feature`** builds named scenarios end to end: it plans, builds from the schema up to the
   UI, annotates and tests every step, runs every check, and traces each scenario with
   `spectrace` to prove nothing is missing.
4. A person confirms the feature works in the app. Only then is the scenario marked
   `implemented`, and from then on speclint holds every one of its links in place.

### Guardrails for the agent

- **The agent can't end a turn with failing checks.** A Claude Code Stop hook
  ([.claude/hooks/check-stop.sh](.claude/hooks/check-stop.sh)) runs `scripts/check.sh`, minus the
  slower tests, every time the agent finishes a turn. If anything fails (a broken spec link, a type error, lint,
  formatting), the agent is sent back to fix it before it can hand control back. If it's still
  failing on the next try, the hook shows the problems to the person instead, so the agent can't
  get stuck in a loop over something only a person can resolve.
- **Architecture rules are enforced by lint, not by instructions alone.** Only the custom React
  hooks may call the backend bindings; Biome's `noRestrictedImports` fails any other file that
  imports them ([ADR 0012](docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md)).
- **The agent can't install packages at all.** Claude Code permission rules in
  [.claude/settings.json](.claude/settings.json) deny `npm install`, `npm add`, `npm update`,
  `npx`, `go get`, `go install`, and `go mod tidy`, so the agent can recommend a dependency but
  only a person can add one. [AGENTS.md](AGENTS.md) sets the policy behind it: each package must
  be justified by weighing what it saves against the supply chain risk of writing the code by
  hand. That's why the frontend has no component library
  or data fetching library, and why the one drag and drop library, dnd-kit, has
  [an ADR](docs/adr/0013-use-dnd-kit-for-moving-and-reordering-board-cards.md) explaining why it
  was worth it.
- **People keep the final say.** Only a person starts an ADR, and only a person's confirmation
  marks a scenario `implemented`. The agent proposes; it doesn't sign off on its own work.
- **Conventions come with their reasons.** AGENTS.md gives the stack and conventions, each linked
  to the ADR that chose it, so the agent follows a decision knowing why it was made.

## What the app does

- **Board.** Five columns: Saved, Applied, Interviewing, Offer, and Closed. Drag cards between
  columns and reorder them, with the mouse or the keyboard. Each column shows how many jobs it
  holds.
- **Job cards.** Record the company, role, job description, posting link, salary, location,
  notes, and date applied. Moving a card to Applied fills in today's date, and moving it back to
  Saved clears it.
- **Closing a job.** Moving a card to Closed asks why: rejected, withdrew, no response, declined
  offer, position filled, or your own reason.
- **Documents.** Attach the resume, cover letter, or other files you sent for each job. The app
  keeps its own copy, so an attachment still opens if you move or delete the original.

Everything stays on your computer: the jobs in a SQLite database and the attached documents as
files, both in your OS's usual app data directory under `callback`.

## Tech stack

- **Desktop shell:** [Wails](https://wails.io) v2, with a Go backend and the OS's webview
- **Backend:** Go, with business rules in services over a transactional store interface
- **Database:** SQLite, with queries generated by sqlc and migrations run by goose
- **Frontend:** React and TypeScript, built with Vite, styled with CSS Modules over design tokens
- **Tests:** Go tests with hand-written fakes, storage tests against real SQLite, and Vitest with
  React Testing Library on the frontend
- **Lint:** gofmt, `go vet`, Biome, `tsc`, and speclint

Each choice, and why, is in [AGENTS.md](AGENTS.md) and [docs/adr/](docs/adr/).

## Prerequisites

- [Go](https://go.dev/dl/) 1.25 or later
- [Node.js](https://nodejs.org/) and npm
- The [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2:
  `go install github.com/wailsapp/wails/v2/cmd/wails@latest`. Run `wails doctor` to check your
  setup.
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html), only if you change the SQL
  queries or the schema
- A C compiler with CGO enabled, on every platform, because the SQLite driver
  (`github.com/mattn/go-sqlite3`) compiles SQLite's C source. See
  [docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md](docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md).
  - **Windows:** install MinGW-w64 gcc, for example through [MSYS2](https://www.msys2.org/)
    (`pacman -S mingw-w64-ucrt-x86_64-gcc`), and add its `bin` directory (e.g.
    `C:\msys64\ucrt64\bin`) to your `PATH`.
  - **macOS:** install the Xcode Command Line Tools (`xcode-select --install`). Wails needs them
    too.
  - **Linux:** install gcc (e.g. `build-essential` on Debian/Ubuntu). Wails needs it too, along
    with its GTK and WebKitGTK packages; `wails doctor` checks for them.

## Development

Run the app with hot reload:

```sh
wails dev
```

The frontend reloads as you edit it, and the app restarts when Go code changes. Database
migrations run when the app starts.

After changing Go methods bound to the frontend in `app.go`, regenerate the TypeScript bindings
in `frontend/wailsjs/` with `wails generate module`. After changing a query in
`internal/db/queries/` or the schema, run `sqlc generate`. Never edit generated code by hand.

To check everything, tests included:

```sh
bash scripts/check.sh             # everything
bash scripts/check.sh --no-tests  # what the Stop hook runs
```

To build the app into `build/bin/`:

```sh
wails build
```

## Project layout

| Path | What's there |
|---|---|
| `features/` | The specs: one `.feature` file per feature |
| `docs/adr/` | Architecture decision records |
| `docs/spec-annotations.md` | How code and tests link to the specs |
| `AGENTS.md` | Instructions for AI agents: stack, conventions, dependency policy |
| `.claude/` | Agent skills and the Stop hook |
| `main.go`, `app.go` | App startup, and the methods bound to the frontend |
| `internal/jobs/` | The business rules for jobs, the board, and documents |
| `internal/db/` | Opening the database, migrations, SQL queries, and the transactional store |
| `internal/store/` | Go generated by sqlc from the queries |
| `internal/docfiles/` | The app's copies of attached documents |
| `internal/desktop/` | Opening documents and links with the OS |
| `internal/appdata/` | Finding the app data directory |
| `internal/speclint/` | The spec link linter and tracer, run by `cmd/speclint` and `cmd/spectrace` |
| `frontend/src/hooks/` | React hooks wrapping the Wails bindings, the only code that calls them |
| `frontend/src/components/` | React components, each with its own CSS Module |
| `frontend/src/styles/tokens.css` | Design tokens: colors, spacing, and type |
| `scripts/check.sh` | Runs every check |
