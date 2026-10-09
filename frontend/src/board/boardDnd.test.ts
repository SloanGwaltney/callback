import { describe, expect, it } from "vitest";
import { board, card } from "../test/fixtures";
import {
	applyMove,
	containerOf,
	finishDrop,
	itemsFromBoard,
	moveAcross,
	moveFor,
	statusOf,
} from "./boardDnd";

const acme = card(1, "Acme");
const globex = card(2, "Globex");
const initech = card(3, "Initech");

describe("dropping a card", () => {
	// @scenario [features/job-board.feature#reorder-card] Reorder a card within its column
	it("reorders a card within its column", () => {
		// @given [features/job-board.feature] the "Saved" column holds the jobs at "Acme", "Globex", and "Initech" in that order
		const before = board({ saved: [acme, globex, initech] });

		// @when [features/job-board.feature] I move the "Initech" card above the "Acme" card
		const items = finishDrop(itemsFromBoard(before), initech.id, acme.id);

		// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
		expect(items.saved).toEqual([initech.id, acme.id, globex.id]);
		expect(moveFor(before, items, initech.id)).toMatchObject({
			jobId: initech.id,
			status: "saved",
			index: 0,
		});
	});

	// @scenario [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
	it("places a card at a chosen position in another column", () => {
		// @given [features/job-board.feature] the "Applied" column holds the jobs at "Globex" and "Initech" in that order
		// @and [features/job-board.feature] I have a job at "Acme" in "Saved"
		const before = board({ saved: [acme], applied: [globex, initech] });

		// @when [features/job-board.feature] I move the "Acme" card into "Applied" between "Globex" and "Initech"
		const over = moveAcross(itemsFromBoard(before), acme.id, initech.id);
		const items = finishDrop(over, acme.id, acme.id);

		// @then [features/job-board.feature] the "Applied" column shows "Globex", "Acme", and "Initech" in that order
		expect(items.applied).toEqual([globex.id, acme.id, initech.id]);
		expect(items.saved).toEqual([]);
		expect(moveFor(before, items, acme.id)).toMatchObject({
			jobId: acme.id,
			status: "applied",
			index: 1,
		});
	});

	it("puts a card dropped on a column at the bottom of it", () => {
		const before = board({ saved: [acme], applied: [globex] });
		const items = finishDrop(itemsFromBoard(before), acme.id, "applied");
		expect(items.applied).toEqual([globex.id, acme.id]);
		expect(moveFor(before, items, acme.id)).toMatchObject({ status: "applied", index: 1 });
	});

	it("makes no move when the card is dropped where it started", () => {
		const before = board({ saved: [acme, globex] });
		const items = finishDrop(itemsFromBoard(before), acme.id, acme.id);
		expect(moveFor(before, items, acme.id)).toBeNull();
	});

	it("sends only the card's place, with no closing reason", () => {
		const before = board({ saved: [acme] });
		const items = finishDrop(itemsFromBoard(before), acme.id, "offer");
		expect(moveFor(before, items, acme.id)).toEqual({
			jobId: acme.id,
			status: "offer",
			index: 0,
			closingReason: "",
			closingReasonOther: "",
		});
	});
});

describe("containerOf and statusOf", () => {
	it("finds the column for a column ID or a card ID", () => {
		const b = board({ applied: [acme] });
		const items = itemsFromBoard(b);
		expect(containerOf(items, "offer")).toBe("offer");
		expect(containerOf(items, acme.id)).toBe("applied");
		expect(containerOf(items, 99)).toBeUndefined();
		expect(statusOf(b, acme.id)).toBe("applied");
	});
});

describe("applyMove", () => {
	it("moves the card and updates both columns' counts", () => {
		const b = applyMove(board({ saved: [acme, globex], applied: [initech] }), {
			jobId: acme.id,
			status: "applied",
			index: 1,
			closingReason: "",
			closingReasonOther: "",
		});
		const [saved, applied] = b.columns;
		expect(saved.cards.map((c) => c.id)).toEqual([globex.id]);
		expect(saved.count).toBe(1);
		expect(applied.cards.map((c) => c.id)).toEqual([initech.id, acme.id]);
		expect(applied.count).toBe(2);
	});

	it("clamps an out-of-range index and ignores an unknown card", () => {
		const start = board({ saved: [acme, globex] });
		const moved = applyMove(start, {
			jobId: acme.id,
			status: "saved",
			index: 9,
			closingReason: "",
			closingReasonOther: "",
		});
		expect(moved.columns[0].cards.map((c) => c.id)).toEqual([globex.id, acme.id]);
		const unknown = {
			jobId: 99,
			status: "saved",
			index: 0,
			closingReason: "",
			closingReasonOther: "",
		};
		expect(applyMove(start, unknown)).toBe(start);
	});
});
