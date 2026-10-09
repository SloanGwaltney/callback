# 0003. Store user data in SQLite

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

callback is a local desktop application that keeps the user's job search data on their machine
(see [docs/adr/0002-use-wails-for-the-desktop-shell.md]). It needs a storage mechanism for that
data. The main contender groups were the file system, a NoSQL database, and a SQL database.

Because the app is local, storage shouldn't be excessively heavy or require an external
connection. That constraint alone doesn't rule out any of the three groups.

The data is relational: jobs, their places on the board, and the documents attached to them relate
to one another, and none of it is naturally expressed in a non-relational form.

This ADR doesn't cover which Go SQLite driver to use, or where the content of attached document
files is stored. Each of those is its own decision.

## Decisions

1. Store the user's data in a SQL database.
2. Use SQLite, embedded in the app, as that database.

## Reasons

- **1.** The data is inherently relational. A SQL database manages relationships for us, so we
  don't have to maintain them by hand in Go code.
- **1.** Nothing in the data makes sense to express in a non-relational form, so a NoSQL database
  offers no advantage that would justify giving up relational modelling.
- **2.** SQLite is embedded and runs in-process, which meets the constraint that local storage not
  be excessively heavy or require an external connection. Among SQL databases, that constraint is
  what makes SQLite the choice.

## Alternatives considered

- **The file system** (storing data as files managed by the app). At the app's current scale,
  managing relationships by hand might be workable. Rejected because if the app grows and gains
  features, managing relationships in Go code would quickly become extremely cumbersome.
- **A NoSQL database.** NoSQL shines when the data isn't inherently relational, and when a
  database needs to be sharded. Our data is relational, and a local, single-user database has no
  need for sharding, so neither advantage applies and it was rejected.
- **A server-based SQL database** (such as PostgreSQL or MySQL). The server could run on the
  user's machine, but that means installing an entire database server there and making sure its
  services are running. That conflicts with keeping local storage lightweight, and it brings no
  benefit over an embedded database, so it was rejected.

## Related

- universal
- [docs/adr/0002-use-wails-for-the-desktop-shell.md] Use Wails for the desktop shell
