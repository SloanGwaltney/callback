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
// @then [features/board-search.feature#search-blank-shows-all] I see the card for "Software Engineer" at "Acme"
// @and [features/board-search.feature#search-blank-shows-all] I see the card for "Product Designer" at "Globex"
// @then [features/board-search.feature#clear-search] I see the card for "Software Engineer" at "Acme"
// @and [features/board-search.feature#clear-search] I see the card for "Product Designer" at "Globex"
// @and [features/board-search.feature#search-not-kept] I see the card for "Product Designer" at "Globex"
// @and [features/board-search.feature#add-job-clears-search] I see the card for "Software Engineer" at "Acme"
// @and [features/board-search.feature#add-job-clears-search] I see the card for the job at "Globex"
export function searching(query: string): boolean {
	return query.trim() !== "";
}

// matches reports whether a card's role or company contains query, ignoring letter case.
// Only the role and the company are searched, not the job's other details.
export function matches(card: Card, query: string): boolean {
	const text = query.trim().toLocaleLowerCase();
	// @then [features/board-search.feature#search-by-role] I see the card for "Software Engineer" at "Acme"
	// @then [features/board-search.feature#search-partial-any-case] I see the card for "Software Engineer" at "Acme"
	// @and [features/board-search.feature] I don't see the card for "Product Designer" at "Globex"
	// @then [features/board-search.feature#search-by-company] I see the card for "Product Designer" at "Globex"
	// @and [features/board-search.feature] I don't see the card for "Software Engineer" at "Acme"
	// @then [features/board-search.feature] I don't see the card for "Software Engineer" at "Globex"
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
		// @and [features/board-search.feature] I still see every status column
		columns: board.columns.map((column) => {
			// @then [features/board-search.feature] the card for "Software Engineer" at "Acme" is in the "Saved" column
			// @and [features/board-search.feature] the card for "Data Engineer" at "Globex" is in the "Interviewing" column
			const cards = column.cards.filter((card) => matches(card, query));
			// @then [features/board-search.feature] the "Saved" column shows a count of 2
			// @and [features/board-search.feature] the "Applied" column shows a count of 0
			// @and [features/board-pagination.feature] the "Saved" column shows a count of 15
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
			// @then [features/board-pagination.feature] the "Saved" column is on page 2 of 2
			const page = Math.min(Math.max(pages[column.status] ?? 1, 1), count);
			const offset = (page - 1) * PAGE_SIZE;
			// @then [features/board-pagination.feature] the "Saved" column shows its first 10 cards in order
			// @then [features/board-pagination.feature] the "Applied" column shows all 10 cards
			// @then [features/board-pagination.feature] the "Saved" column shows cards 11 to 20
			// @then [features/board-pagination.feature] the "Saved" column shows only the cards on its current page
			// @and [features/board-pagination.feature] cards on other pages of the "Saved" column can't be seen or moved until I page to them
			// @then [features/board-pagination.feature] the "Saved" column still shows 10 cards on page 1
			// @and [features/board-pagination.feature] the card that was first on page 2 is now last on page 1
			const cards = column.cards.slice(offset, offset + PAGE_SIZE);
			return { ...column, cards, page, pageCount: count, offset };
		}),
	};
}

// placeInColumn turns a move made on a column's page into a move in the whole column: a drop at
// index 1 on page 2 lands at index 11. The index still counts with the card already in the column,
// as jobs.Move expects.
// @then [features/board-pagination.feature] the "Applied" column shows "Globex", "Acme", and "Initech" in that order on page 2
export function placeInColumn(view: BoardView, move: Move): Move {
	const column = view.columns.find((c) => c.status === move.status);
	return { ...move, index: move.index + (column?.offset ?? 0) };
}
