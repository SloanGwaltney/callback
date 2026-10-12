import { describe, expect, it } from "vitest";
import { board, card } from "../test/fixtures";
import { filterBoard, matches, PAGE_SIZE, pageBoard, pageCount, placeInColumn } from "./boardView";

const many = (n: number) => Array.from({ length: n }, (_, i) => card(i + 1, `Company ${i + 1}`));
const ids = (cards: { id: number }[]) => cards.map((c) => c.id);
const range = (first: number, last: number) =>
	Array.from({ length: last - first + 1 }, (_, i) => first + i);

describe("matches", () => {
	const acme = card(1, "Acme", { role: "Software Engineer" });

	it("matches part of the role or the company in any letter case", () => {
		for (const text of ["acme", "ACM", "soft", "ENGINEER", "acme engineer"]) {
			expect(matches(acme, text), text).toBe(text !== "acme engineer");
		}
	});

	it("matches letters outside ASCII in any case", () => {
		expect(matches(card(1, "Société Générale"), "SOCIÉTÉ")).toBe(true);
	});

	it("ignores spaces around the search", () => {
		expect(matches(acme, "  acme  ")).toBe(true);
	});

	it("doesn't match the location", () => {
		expect(matches(card(1, "Globex", { location: "Acme Tower" }), "Acme")).toBe(false);
	});
});

describe("filterBoard", () => {
	it("returns the board as it is when the search is blank", () => {
		const b = board({ saved: [card(1, "Acme")] });
		expect(filterBoard(b, "")).toBe(b);
		expect(filterBoard(b, "   ")).toBe(b);
	});

	it("keeps every column, with the matching cards in order and counted", () => {
		const b = board({
			saved: [card(1, "Acme"), card(2, "Globex"), card(3, "Acme Robotics")],
			applied: [card(4, "Initech")],
		});
		const filtered = filterBoard(b, "acme");
		expect(filtered.columns.map((c) => c.status)).toEqual(b.columns.map((c) => c.status));
		expect(ids(filtered.columns[0].cards)).toEqual([1, 3]);
		expect(filtered.columns[0].count).toBe(2);
		expect(filtered.columns[1].count).toBe(0);
	});
});

describe("pageCount", () => {
	it("is at least 1 and rounds up", () => {
		expect(pageCount(0)).toBe(1);
		expect(pageCount(PAGE_SIZE)).toBe(1);
		expect(pageCount(PAGE_SIZE + 1)).toBe(2);
		expect(pageCount(25)).toBe(3);
	});
});

describe("pageBoard", () => {
	it("shows page 1 of a column with no page set", () => {
		const [saved] = pageBoard(board({ saved: many(25) }), {}).columns;
		expect(ids(saved.cards)).toEqual(range(1, 10));
		expect(saved).toMatchObject({ page: 1, pageCount: 3, offset: 0, count: 25 });
	});

	it("shows the column's page, with the last page holding what's left", () => {
		const [saved] = pageBoard(board({ saved: many(25) }), { saved: 3 }).columns;
		expect(ids(saved.cards)).toEqual(range(21, 25));
		expect(saved).toMatchObject({ page: 3, offset: 20 });
	});

	it("keeps a page past the end, or below 1, within the column's pages", () => {
		const b = board({ saved: many(20) });
		expect(pageBoard(b, { saved: 3 }).columns[0].page).toBe(2);
		expect(pageBoard(b, { saved: 0 }).columns[0].page).toBe(1);
		expect(pageBoard(board(), { saved: 4 }).columns[0]).toMatchObject({ page: 1, pageCount: 1 });
	});
});

describe("placeInColumn", () => {
	it("adds the cards before the page to a move's index", () => {
		const view = pageBoard(board({ applied: many(25) }), { applied: 2 });
		const move = {
			jobId: 99,
			status: "applied",
			index: 1,
			closingReason: "",
			closingReasonOther: "",
		};
		expect(placeInColumn(view, move).index).toBe(11);
	});

	it("leaves a move to a column's first page as it is", () => {
		const view = pageBoard(board({ applied: many(25) }), {});
		const move = {
			jobId: 99,
			status: "applied",
			index: 3,
			closingReason: "",
			closingReasonOther: "",
		};
		expect(placeInColumn(view, move).index).toBe(3);
	});
});
