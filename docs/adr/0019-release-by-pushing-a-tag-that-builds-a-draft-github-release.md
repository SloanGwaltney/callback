# 0019. Release by pushing a tag that builds a draft GitHub Release

- **Status:** Accepted
- **Date:** 2026-10-10

## Context

The app has no release process yet. A release ships a desktop executable that users download and
run on their own machines.

The project has a single developer. Commits follow Conventional Commits (`feat:`, `ci:`,
`docs:`), but they don't map cleanly to user-facing changes: much of the history is CI, tooling,
and docs work, and one user-facing change often spans many commits.

GitHub doesn't trigger workflows when a draft release is created or edited, so a release started
from the "Draft a new release" page can't have its binaries built and attached until it's
published. A tag pushed by a workflow's own `GITHUB_TOKEN` doesn't trigger other workflows.

The app is built with Wails v2 and needs CGO, so each OS's build has to run on that OS. The Go
code already handles Windows, Linux, and macOS (`internal/appdata`, `internal/desktop`). As
[docs/adr/0018-enforce-the-check-script-in-github-actions-before-merging.md] notes, building on
Linux needs the WebKitGTK and GTK headers and the `webkit2_41` build tag.

No code-signing certificates are available for any platform. Unsigned, each OS warns users
differently:

- **Windows:** SmartScreen shows "Windows protected your PC", and the user clicks **More info →
  Run anyway** once. Most users have done this before.
- **Linux:** no warning. Users need WebKitGTK 4.1 installed, which most desktop distributions
  already include.
- **macOS:** Apple Silicon requires at least an ad-hoc signature, which is free, but without an
  Apple Developer ID and notarization Gatekeeper blocks the first launch. Since macOS 15, users
  have to try to open the app, then go to **System Settings → Privacy & Security → Open
  Anyway** and enter their password.

Note added 2026-10-15: on Windows, every way to sign a binary we distribute ourselves still
depends on SmartScreen reputation, so signing wouldn't remove the warning for an app this size:

- **EV certificates no longer skip SmartScreen.** Since 2024, a new EV-signed build starts with
  low reputation, the same as an OV-signed one.
- **Free and cheap signing builds reputation the same way.** SignPath Foundation signs
  open-source projects for free, under its own name as the publisher. Azure Trusted Signing and
  Certum's open-source certificate are paid. With any of them, a build stops warning only after
  enough users have run files from that certificate without problems.
- **This app isn't expected to reach the scale where it builds that reputation.** Signing would
  add cost or setup to every release and users would most likely still see the warning.
- **The only foreseeable way around reputation is the Microsoft Store.** Individual developer
  accounts are free, and Microsoft re-signs MSIX packages submitted to the Store with its own
  certificate. That means users install from the Store instead of GitHub, and few users get
  desktop apps that way, so it isn't planned.

## Decisions

1. Version releases with Semantic Versioning, as `vX.Y.Z`.
2. Start a release by pushing an annotated `vX.Y.Z` tag on `main` by hand.
3. A GitHub Actions workflow triggered by the tag push runs `scripts/check.sh`, builds the app on
   Windows and Linux runners, and creates a draft GitHub Release with the binaries and their
   SHA-256 checksums attached. Linux ships as a `.tar.gz` of the binary.
4. Don't release for macOS for now. Release binaries aren't code-signed on any platform.
5. The release workflow uses only `git` and the `gh` CLI for tagging and releasing, with no
   third-party release actions, and its token has only `contents: write`.
6. Write the release notes by hand in the draft, then publish it. There is no `CHANGELOG.md`.
7. Never move a pushed release tag. A broken release is fixed with a new patch version.

## Reasons

- **Little ceremony (2, 6).** Release-PR bots, changeset files, and per-PR changelog entries
  solve coordination between contributors. With a single developer there's no one to
  coordinate with, so they'd only add steps. A release is one tag push plus writing the notes,
  with no version files to bump and no changelog to keep up.
- **Commits don't map cleanly to release notes (6).** Notes generated from commits would need
  rewriting anyway, so they're written directly. GitHub's "Generate release notes" can still give
  a list of merged pull requests to start from.
- **A person approves every release, for security (2, 3, 5, 6).** Publishing ships executables to
  users' machines.
  - A compromised dependency, a pinned action whose behavior changed, or a bad merge can't reach
    users unless the developer publishes it.
  - The draft lets the developer check the built files and checksums before anything is public. A
    failed or suspicious build is deleted and never seen.
  - Nothing releases just because a commit landed on `main`, so tricking CI or slipping in a
    commit with a `feat:` prefix doesn't create a release.
  - Third-party release actions would need write access to the repository; `git` and `gh` are
    already on GitHub's runners.
- **Draft until complete (3).** Building on the tag push means the draft already has its binaries
  when the notes are written, so the public never sees a release without downloads or without
  notes. Trade-off: a release starts from the command line instead of a button.
- **Cover the majority path for now (3, 4).** Windows and Linux users can run an unsigned build
  with at most one familiar step. On macOS, an unsigned app puts several unfamiliar steps between
  the user and running it, and doing it properly means adding Apple signing and notarization.
  That's good for users but a lot more work for us, so macOS waits until there's a need for it.
  This isn't permanent: adding macOS, or signing on any platform, means adding to the release
  workflow, not redesigning it. Building on Linux needs the WebKitGTK packages and build tag, but
  only in the release job, not in the check that runs on every pull request.
- **Fixed tags (7).** A tag marks exactly which commits a release contains. Moving one would
  change what a version means for anyone who already downloaded it.

## Alternatives considered

- **semantic-release.** Releases automatically on every releasable push to `main`. Rejected
  because there's no person approving the release.
- **release-please.** Keeps a release pull request that bumps the version and writes
  `CHANGELOG.md` from commits. Rejected because it's a third-party action that needs write
  access, and its generated changelog doesn't fit how commits map to changes here.
- **A `CHANGELOG.md` updated in every pull request (Keep a Changelog), or Changesets files.** Too
  much ceremony for a single developer.
- **Starting from the "Draft a new release" page and building on `release: published`.** Drafts
  don't trigger workflows, so the binaries would be built only after publishing, leaving a public
  release with no downloads while it builds, or with none at all if the build fails.
- **A `workflow_dispatch` button that tags and builds.** Leaves the same draft as the tag push.
  The tag push was chosen.
- **Windows only.** Rejected because Linux costs only a second build job and its users face no
  warning at all.
- **macOS, ad-hoc signed but not notarized.** Free, but rejected for now because users would have
  to get past Gatekeeper through System Settings, and the release notes would have to walk them
  through it.
- **macOS, signed with an Apple Developer ID and notarized.** The right experience for users, but
  rejected for now because of the yearly fee and the extra signing and notarization steps and
  secrets in the release workflow. To be revisited if macOS users need the app.
- **Windows, signed (EV, OV, SignPath Foundation, or Azure Trusted Signing).** Rejected because
  each still has to build SmartScreen reputation, which this app isn't expected to reach (see
  Context).
- **Windows, through the Microsoft Store as MSIX.** Microsoft signs the package, so there's no
  certificate of our own and no reputation to build. Rejected because users would have to install
  from the Store, which few use for desktop apps, and every release would need MSIX packaging and
  Store certification.
- **Linux packages (AppImage, `.deb`, Flatpak).** Each needs extra work and tooling. A `.tar.gz`
  covers Linux until there's a need for more.

## Related

- universal
- [docs/adr/0018-enforce-the-check-script-in-github-actions-before-merging.md] Enforce the check script in GitHub Actions before merging
- [docs/adr/0014-check-code-with-one-script-run-by-the-stop-hook-and-ci.md] Check code with one script, run by the Stop hook and CI
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
