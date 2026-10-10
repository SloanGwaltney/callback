// Pure helpers behind dragging cards on the board (docs/adr/0013-use-dnd-kit-for-moving-and-reordering-board-cards.md).
// While a card is dragged, the board shows a preview: the card IDs in each column, keyed by
// status. When it's dropped, the preview becomes a jobs.Move for the MoveJob binding.
import {
	type CollisionDetection,
	closestCorners,
	pointerWithin,
	type UniqueIdentifier,
} from "@dnd-kit/core";
import type { Board, Move } from "../hooks/types";

// Items is the card IDs in each column, top to bottom, keyed by status.
export type Items = Record<string, number[]>;

export function itemsFromBoard(board: Board): Items {
	const items: Items = {};
	for (const column of board.columns) {
		items[column.status] = column.cards.map((card) => card.id);
	}
	return items;
}

// collision picks what a dragged card is over. With the mouse, it's whatever is under the pointer:
// the card there, or the column when the pointer is between cards. closestCorners alone compares
// the dragged card's corners with each column's, and a full-height column's bottom corners are far
// from a card dragged near the top, so a card had to go well past a column's edge to land in it.
// A keyboard drag has no pointer, so it, and a pointer off every column, use closestCorners.
export const collision: CollisionDetection = (args) => {
	const under = pointerWithin(args);
	return under.length > 0 ? under : closestCorners(args);
};

// containerOf returns the status of the column id names, or of the column holding the card id
// names. Columns are identified by their status and cards by their numeric job ID.
export function containerOf(items: Items, id: UniqueIdentifier): string | undefined {
	if (typeof id === "string" && id in items) {
		return id;
	}
	return Object.keys(items).find((status) => items[status].includes(Number(id)));
}

// moveAcross previews a card dragged over another column: it leaves its own column and takes
// the place of the card it's over, or goes to the bottom when it's over the column itself.
export function moveAcross(
	items: Items,
	activeId: UniqueIdentifier,
	overId: UniqueIdentifier,
): Items {
	const from = containerOf(items, activeId);
	const to = containerOf(items, overId);
	if (!from || !to || from === to) {
		return items;
	}
	const card = Number(activeId);
	const target = items[to];
	const overIndex = target.indexOf(Number(overId));
	const index = overIndex === -1 ? target.length : overIndex;
	return {
		...items,
		[from]: items[from].filter((id) => id !== card),
		[to]: [...target.slice(0, index), card, ...target.slice(index)],
	};
}

// finishDrop places a dropped card. Over another card in its column, it takes that card's place;
// moveAcross has already put it in the right column.
export function finishDrop(
	items: Items,
	activeId: UniqueIdentifier,
	overId: UniqueIdentifier,
): Items {
	const moved = moveAcross(items, activeId, overId);
	const status = containerOf(moved, activeId);
	if (!status) {
		return moved;
	}
	const column = moved[status];
	const from = column.indexOf(Number(activeId));
	const to = column.indexOf(Number(overId));
	if (to === -1 || from === to) {
		return moved;
	}
	const reordered = column.filter((id) => id !== Number(activeId));
	reordered.splice(to, 0, Number(activeId));
	return { ...moved, [status]: reordered };
}

// statusOf returns the status of the column holding the card on the board.
export function statusOf(board: Board, jobId: number): string | undefined {
	return board.columns.find((column) => column.cards.some((card) => card.id === jobId))?.status;
}

// moveFor turns where a dropped card ended up into a move, or null if it's back where it started.
// The index counts from the top of the column with the card already in it, as jobs.Move expects.
export function moveFor(board: Board, items: Items, jobId: number): Move | null {
	const status = containerOf(items, jobId);
	if (!status) {
		return null;
	}
	const index = items[status].indexOf(jobId);
	const before = itemsFromBoard(board);
	if (statusOf(board, jobId) === status && before[status].indexOf(jobId) === index) {
		return null;
	}
	// Only the card's place is sent, so its details and documents stay as they are.
	// @and [features/job-board.feature] the job description, link, and attached resume are unchanged
	return { jobId, status, index, closingReason: "", closingReasonOther: "" };
}

// applyMove returns the board with a move made, so the card shows in its new place before the
// backend answers. The board is reloaded afterwards for anything the move changed on the job,
// like its date applied or closing reason.
export function applyMove(board: Board, move: Move): Board {
	const card = board.columns.flatMap((column) => column.cards).find((c) => c.id === move.jobId);
	if (!card) {
		return board;
	}
	const columns = board.columns.map((column) => {
		// @and [features/job-board.feature] the card no longer appears in the "Saved" column
		const cards = column.cards.filter((c) => c.id !== move.jobId);
		if (column.status === move.status) {
			const index = Math.min(Math.max(move.index, 0), cards.length);
			cards.splice(index, 0, card);
		}
		return { ...column, cards, count: cards.length };
	});
	return { columns };
}
