// Pure helpers that turn the whole board, as GetBoard returns it, into what the board shows: the
// cards matching the search, a page at a time in each column. Search and pagination happen here in
// the frontend (docs/adr/0020-search-and-paginate-the-board-in-the-frontend.md); card order stays
// in the jobs service.
import type { Board, Card, Column, Move } from "../hooks/types";

// PAGE_SIZE is how many cards a column shows per page. It's a system setting, not a user one.
export const PAGE_SIZE = 10;

// Pages is each column's current page, from 1, keyed by status. A column that isn't listed is on
// page 1.
export type Pages = Record<string, number>;

// ColumnView is a column as shown: the cards on its current page, with where that page is.
export type ColumnView = Column & {
	// page is the page shown, from 1, kept within the column's pages.
	page: number;
	pageCount: number;
	// offset is how many cards come before the page in the (filtered) column.
	offset: number;
};

export type BoardView = { columns: ColumnView[] };

// searching reports whether query filters the board. A query of only spaces doesn't, so every job
// shows.
export function searching(query: string): boolean {
	return query.trim() !== "";
}

// matches reports whether a card's role or company contains query, ignoring letter case.
// Only the role and the company are searched, not the job's other details.
export function matches(card: Card, query: string): boolean {
	const text = query.trim().toLocaleLowerCase();
	return (
		card.role.toLocaleLowerCase().includes(text) || card.company.toLocaleLowerCase().includes(text)
	);
}

// filterBoard keeps the cards matching query in each column, in order. While searching, each
// column's count is how many of its cards match. Every column stays, even with no matches.
export function filterBoard(board: Board, query: string): Board {
	if (!searching(query)) {
		return board;
	}
	return {
		columns: board.columns.map((column) => {
			const cards = column.cards.filter((card) => matches(card, query));
			return { ...column, cards, count: cards.length };
		}),
	};
}

// pageCount is how many pages total cards take, at least 1 so an empty column is on page 1 of 1.
export function pageCount(total: number): number {
	return Math.max(1, Math.ceil(total / PAGE_SIZE));
}

// pageBoard shows each column's current page. A page past the column's last page shows the last
// page instead, as happens when the only card on the last page moves away.
export function pageBoard(board: Board, pages: Pages): BoardView {
	return {
		columns: board.columns.map((column) => {
			const count = pageCount(column.cards.length);
			const page = Math.min(Math.max(pages[column.status] ?? 1, 1), count);
			const offset = (page - 1) * PAGE_SIZE;
			const cards = column.cards.slice(offset, offset + PAGE_SIZE);
			return { ...column, cards, page, pageCount: count, offset };
		}),
	};
}

// placeInColumn turns a move made on a column's page into a move in the whole column: a drop at
// index 1 on page 2 lands at index 11. The index still counts with the card already in the column,
// as jobs.Move expects.
export function placeInColumn(view: BoardView, move: Move): Move {
	const column = view.columns.find((c) => c.status === move.status);
	return { ...move, index: move.index + (column?.offset ?? 0) };
}
