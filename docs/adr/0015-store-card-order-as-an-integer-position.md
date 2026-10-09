# 0015. Store card order as an integer position

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

Cards on the job board have an order within their column. A card can be reordered within its
column, placed at a chosen position when it moves into another column, and a newly added job goes
to the top of its column. The order has to survive closing and reopening the app. Cards are moved
by drag and drop (see [docs/adr/0013-use-dnd-kit-for-moving-and-reordering-board-cards.md]), and
the order is stored in SQLite (see [docs/adr/0003-store-user-data-in-sqlite.md]).

Three ways to store the order were considered:

- **An integer position** (0 to n) per card within its column. A move renumbers the cards in the
  affected column or columns.
- **A fractional position** (a `REAL`). A moved card gets the midpoint between its new neighbors,
  so only that card's row changes. Repeated inserts at the same spot use up floating point
  precision, so positions eventually need rebalancing.
- **String keys** (fractional indexing). Keys that sort between their neighbors, so only one row
  changes per move and precision never runs out. The most complex option, needing either
  hand-written key generation or a dependency.

Columns can realistically grow past 50 cards.

## Decisions

1. Store each card's order within its column as an integer position, starting at 0.
2. When a card is added, moved, or reordered, renumber the positions of the affected column or
   columns in a single transaction.

## Reasons

- **1, 2.** The goal right now is to get something working. Integer positions are the simplest
  option to write, read, and test.
- **1, 2.** Renumbering a column should scale fine for columns of about 50 cards or fewer.
- **1, 2.** Changing the approach later isn't a large hurdle: a migration can convert the integer
  positions to whatever format replaces them, since the existing order carries over directly.
- **2.** A single transaction keeps a column's positions consistent even if a move fails partway.
- Accepted trade-off: every move rewrites the positions of the cards it shifts, not just the moved
  card. Columns past about 50 cards are a real possibility, and when that happens, the storage
  format should be revisited.

## Alternatives considered

- **Fractional positions.** Only one row changes per move. Rejected for now because positions run
  out of precision and need rebalancing, which is more complexity than the app needs yet.
- **String keys (fractional indexing).** Only one row changes per move, with no precision limit.
  Rejected for now as the most complex option. Either it or fractional positions is the likely
  replacement if columns outgrow integer positions.

## Related

- [features/job-board.feature#reorder-card] Reorder a card within its column
- [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
- [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
- [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
- [docs/adr/0003-store-user-data-in-sqlite.md] Store user data in SQLite
- [docs/adr/0013-use-dnd-kit-for-moving-and-reordering-board-cards.md] Use dnd-kit for moving and reordering board cards
