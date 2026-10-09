# 0013. Use dnd-kit for moving and reordering board cards

- **Status:** Accepted
- **Date:** 2026-10-09

## Context

On the job board, cards move between status columns, and they also have positions within a
column: a card can be reordered within its column, or placed at a chosen position when it moves
into another column. The scenarios say "move the card" and never say "drag", so dragging isn't
required by the spec, but it's what users expect from a Kanban board.

A "Move to…" control such as a native `<select>` covers moving between columns, but not choosing
a position. Without dragging, positions would need extra controls such as move up and move down
buttons.

The browser's built-in HTML5 drag-and-drop API needs no dependency, but it only works with a
mouse, it leaves computing drop positions to us, its drop preview is hard to style, and it
behaves differently across webviews. Keyboard users would need a separate set of controls that
we'd also build.

dnd-kit is a React drag-and-drop library. `@dnd-kit/core` provides drag handling with keyboard
support and screen reader announcements, and `@dnd-kit/sortable` adds sortable lists that items
can move between. As of this decision, `@dnd-kit/core` 6.3.1 depends on `@dnd-kit/accessibility`,
`@dnd-kit/utilities`, and `tslib`, and `@dnd-kit/sortable` 10.0.0 depends on
`@dnd-kit/utilities` and `tslib`, so it brings in five packages, four of them from the same
maintainers.

## Decisions

1. Move and reorder board cards by drag and drop, using `@dnd-kit/core` and
   `@dnd-kit/sortable`, with each status column as a sortable list that cards can move between.
2. Use dnd-kit's keyboard support so cards can be moved and reordered without a mouse.
3. How card order is stored is a separate decision, made with the first database migration.

## Reasons

- **1.** The board's position scenarios need a card to be placed at a specific point in a
  column. Drag and drop expresses that directly, and it's what users expect from a Kanban board.
- **1.** Under the dependency policy, dnd-kit is worth the risk. A keyboard-operable, accessible
  sortable list that works across columns isn't something we can reasonably write ourselves: on
  top of drop position calculation, it needs keyboard interaction, focus management, and screen
  reader announcements. Accepted trade-off: five packages, four from one set of maintainers.
- **2.** Drag-only interaction would shut out keyboard users. dnd-kit provides keyboard movement
  as part of the same mechanism, so there's no second set of controls to build.
- **3.** Order storage (an integer position versus a fractional or string key) is a schema
  question, best decided when the schema is written.

## Alternatives considered

- **The browser's HTML5 drag-and-drop API plus our own keyboard controls.** No dependency.
  Rejected because we'd compute drop positions ourselves, build a separate keyboard path, and
  work around differences between webviews, which is the hard part dnd-kit already solves.
- **No dragging: a "Move to…" menu plus move up and down buttons.** No dependency and little
  code. Rejected because placing a card at a position is clumsy this way, and it doesn't feel
  like a Kanban board.

## Related

- [features/job-board.feature#move-card-forward] Move a card to the next status
- [features/job-board.feature#move-card-any-status] Move a card to any other status
- [features/job-board.feature#reorder-card] Reorder a card within its column
- [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
- [docs/adr/0007-build-the-frontend-as-a-react-spa-with-typescript-and-vite.md] Build the frontend as a React SPA with TypeScript and Vite
