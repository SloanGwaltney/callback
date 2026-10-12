import {
	type ClientRect,
	type CollisionDetection,
	closestCorners,
	type DroppableContainer,
	type UniqueIdentifier,
} from "@dnd-kit/core";
import { describe, expect, it } from "vitest";
import { board, card } from "../test/fixtures";
import {
	applyMove,
	collision,
	containerOf,
	finishDrop,
	itemsFromBoard,
	moveAcross,
	moveFor,
	statusOf,
	toTop,
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

describe("toTop", () => {
	it("moves a card to the top of the column holding it", () => {
		expect(toTop({ saved: [], applied: [2, 3, 1] }, 1)).toEqual({ saved: [], applied: [1, 2, 3] });
	});

	it("leaves the items as they are for a card that isn't there", () => {
		const items = { saved: [2] };
		expect(toTop(items, 9)).toBe(items);
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

describe("what a dragged card is over", () => {
	const rect = (left: number, top: number, width: number, height: number): ClientRect => ({
		left,
		top,
		width,
		height,
		right: left + width,
		bottom: top + height,
	});
	// Two full-height columns side by side: Saved holds Acme at its top, Applied is empty.
	const rects = new Map<UniqueIdentifier, ClientRect>([
		["saved", rect(0, 0, 300, 1000)],
		[acme.id, rect(10, 10, 280, 80)],
		["applied", rect(300, 0, 300, 1000)],
	]);
	const args = (pointer: { x: number; y: number } | null): Parameters<CollisionDetection>[0] => ({
		active: { id: globex.id } as Parameters<CollisionDetection>[0]["active"],
		// Acme's size, dragged so its left edge is still well inside Saved.
		collisionRect: rect(220, 10, 280, 80),
		droppableRects: rects,
		droppableContainers: [...rects.keys()].map((id) => ({ id }) as DroppableContainer),
		pointerCoordinates: pointer,
	});
	const first = (pointer: { x: number; y: number } | null) => collision(args(pointer))[0]?.id;

	it("is the column under the pointer, even just past its edge", () => {
		// Corners alone still pick the card in the column the pointer has left.
		expect(closestCorners(args({ x: 310, y: 50 }))[0]?.id).toBe(acme.id);
		expect(first({ x: 310, y: 50 })).toBe("applied");
	});

	it("is the card under the pointer before the column holding it", () => {
		expect(first({ x: 100, y: 50 })).toBe(acme.id);
	});

	it("goes by the dragged card's corners when there's no pointer, as in a keyboard drag", () => {
		expect(first(null)).toBe(acme.id);
	});
});
