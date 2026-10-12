# 0020. Search and paginate the board in the frontend

- **Status:** Accepted
- **Date:** 2026-10-11

## Context

The job board is getting search (by role and company, see [features/board-search.feature]) and
per-column pagination with arrows (see [features/board-pagination.feature]). Cards can still be
moved between columns while searching and paging, and some of the new rules involve card order:
a page fills back up after a card moves out of it, the last card leaving a page shows the page
before it, and a card moved while searching goes to the top of its new column.

Today `GetBoard` loads every card in every column, and `useBoard` reloads the whole board after
each add, move, and delete. `applyMove` updates the whole board on screen before the backend
answers. Card order is stored as an integer position per column
([docs/adr/0015-store-card-order-as-an-integer-position.md]) and set by the jobs service.

Callback is a desktop app with one user and a local SQLite database. It isn't a web server and
won't become one. A realistic upper bound for one person's job search is about a thousand cards.

The question was whether search and pagination should be done by the backend (paged, filtered
queries per column) or by the frontend (filter and slice the board it already loads).

## Decisions

1. Keep loading the whole board with `GetBoard`, and do search and pagination in the frontend by
   filtering and slicing the loaded board.
2. Search with a case-insensitive substring match on role and company in TypeScript.
3. Keep the search text and each column's current page as frontend view state. Card order and
   every other rule about moving cards stay in the jobs service.

## Reasons

- **1.** With one user and about a thousand cards at most, loading the whole board is cheap, and
  filtering or slicing it in memory is a non-issue. If search becomes slow, memoizing the filtered
  board is the first fix.
- **1.** It keeps moving cards simple. `applyMove` keeps working on the whole board, so a move
  shows correctly before the backend answers, including the page refilling from the next page.
  There are no out-of-order responses to discard, and a page change doesn't wait on a backend call
  before its transition.
- **1.** The backend approach would mean paged, filtered queries with counts for each column,
  more bindings, and more tests, to solve a scaling problem this app doesn't have.
- **1.** The decision is easy to reverse. If frontend search or pagination becomes a problem, it
  can move to the backend without changing the specs.
- **2.** JavaScript's case-insensitive comparison handles non-ASCII letters, while SQLite's `LIKE`
  is case-insensitive only for ASCII.
- **3.** Card order is a business rule and stays in the service, per
  [docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md]. The
  frontend only decides which cards are shown, not where they are stored.
- Accepted trade-off: this wouldn't scale beyond one user or a much larger board. Callback is a
  one-user app, so that's acceptable.
- Accepted trade-off: the frontend gains view state (search text and a page per column) and the
  logic that derives what each column shows from it, including keeping the current page valid
  after cards move.

## Alternatives considered

- **Search and pagination in the backend.** Paged, filtered queries per column, with the hooks
  fetching one page at a time. This would keep the presentation layer thinner, put the paging edge
  cases in Go service tests, and give one source of truth for matches and counts. Rejected because
  it adds a larger API, debounced calls, out-of-order responses, and harder instant moves (the
  frontend would only hold the current page and couldn't refill it until a reload) to solve a
  scaling problem a one-user app with about a thousand cards doesn't have.
- **Split: search in the backend, pagination in the frontend** (or the backend returning every
  match for the frontend to page). Rejected along with the backend option. It keeps most of the
  backend option's cost for the same unneeded scaling.

## Related

- [features/board-search.feature] Board search
- [features/board-pagination.feature] Board pagination
- [docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md] Wrap backend calls in custom React hooks
- [docs/adr/0015-store-card-order-as-an-integer-position.md] Store card order as an integer position
- [docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md] Put business rules in services over a transactional store interface
