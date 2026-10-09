import type { DndContextProps, DragEndEvent, DragOverEvent, DragStartEvent } from "@dnd-kit/core";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { applyMove } from "../board/boardDnd";
import type { Board as BoardData, Move } from "../hooks/types";
import { board, card } from "../test/fixtures";
import { Board } from "./Board";

// A real drag needs element positions, which happy-dom doesn't lay out, so the tests capture the
// board's DndContext handlers and call them as dnd-kit would during a drag.
const dnd = vi.hoisted(() => ({ props: null as DndContextProps | null }));
vi.mock("@dnd-kit/core", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@dnd-kit/core")>();
	return {
		...actual,
		DndContext: (props: DndContextProps) => {
			dnd.props = props;
			return <actual.DndContext {...props} />;
		},
	};
});

// drag picks up a card, passes it over each of overs, and drops it on the last.
function drag(cardId: number, ...overs: (number | string)[]) {
	const active = { id: cardId } as DragStartEvent["active"];
	act(() => dnd.props?.onDragStart?.({ active } as DragStartEvent));
	for (const id of overs) {
		act(() => dnd.props?.onDragOver?.({ active, over: { id } } as DragOverEvent));
	}
	act(() => dnd.props?.onDragEnd?.({ active, over: { id: overs.at(-1) } } as DragEndEvent));
}

// renderBoard shows the board with a parent that applies moves like useBoard: the card moves at
// once, then the board is replaced by what the backend returns, when a reload is given.
function renderBoard(start: BoardData, reload?: (move: Move) => BoardData) {
	const onMove = vi.fn<(move: Move) => Promise<void>>();
	const onOpenCard = vi.fn();
	const onAddJob = vi.fn();
	function Parent() {
		const [current, setCurrent] = useState(start);
		onMove.mockImplementation(async (move) => {
			setCurrent((b) => applyMove(b, move));
			if (reload) {
				setCurrent(reload(move));
			}
		});
		return <Board board={current} onMove={onMove} onOpenCard={onOpenCard} onAddJob={onAddJob} />;
	}
	render(<Parent />);
	return { onMove, onOpenCard, onAddJob };
}

function column(label: string) {
	return screen.getByRole("region", { name: label });
}

// cardsIn names the cards in a column, top to bottom, as "<role> at <company>".
function cardsIn(label: string) {
	return within(column(label))
		.queryAllByRole("article")
		.map((a) => a.getAttribute("aria-label"));
}

const acme = card(1, "Acme");
const acmeIn = (label: string) => cardsIn(label).includes("Software Engineer at Acme");

describe("Board", () => {
	// @scenario [features/job-board.feature#board-status-columns] Board shows the status columns in order
	it("shows the status columns in order", () => {
		// @given [features/job-board.feature] I open the job search board
		renderBoard(board());

		// @then [features/job-board.feature] I see the columns "Saved", "Applied", "Interviewing", "Offer", and "Closed" in that order
		expect(screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent)).toEqual([
			"Saved",
			"Applied",
			"Interviewing",
			"Offer",
			"Closed",
		]);
	});

	// @scenario [features/job-board.feature#empty-board] Empty board when no jobs have been added
	it("shows empty columns and a prompt when there are no jobs", async () => {
		// @given [features/job-board.feature] I have not added any jobs
		// @when [features/job-board.feature] I open the job search board
		const { onAddJob } = renderBoard(board());

		// @then [features/job-board.feature] every status column is empty
		for (const label of ["Saved", "Applied", "Interviewing", "Offer", "Closed"]) {
			expect(cardsIn(label)).toEqual([]);
			expect(within(column(label)).getByText("No jobs")).toBeInTheDocument();
		}
		// @and [features/job-board.feature] I am prompted to add my first job
		await userEvent.click(screen.getByRole("button", { name: "Add your first job" }));
		expect(onAddJob).toHaveBeenCalledWith("");
	});

	it("doesn't prompt for a first job once there is one", () => {
		renderBoard(board({ offer: [acme] }));
		expect(screen.queryByRole("button", { name: "Add your first job" })).not.toBeInTheDocument();
	});

	// @scenario [features/job-board.feature#column-card-count] Each column shows how many cards it holds
	it("shows how many cards each column holds", () => {
		// @given [features/job-board.feature] I have 3 jobs in "Saved" and 1 job in "Applied"
		// @when [features/job-board.feature] I open the job search board
		renderBoard(
			board({
				saved: [card(1, "Acme"), card(2, "Globex"), card(3, "Initech")],
				applied: [card(4, "Umbrella")],
			}),
		);

		// @then [features/job-board.feature] the "Saved" column shows a count of 3
		expect(within(column("Saved")).getByText("3 jobs")).toBeInTheDocument();
		// @and [features/job-board.feature] the "Applied" column shows a count of 1
		expect(within(column("Applied")).getByText("1 job")).toBeInTheDocument();
		// @and [features/job-board.feature] the "Interviewing" column shows a count of 0
		expect(within(column("Interviewing")).getByText("0 jobs")).toBeInTheDocument();
	});

	it("offers to add a job to every column but Closed", async () => {
		const { onAddJob } = renderBoard(board());
		await userEvent.click(screen.getByRole("button", { name: "Add a job to Applied" }));
		expect(onAddJob).toHaveBeenCalledWith("applied");
		expect(screen.queryByRole("button", { name: "Add a job to Closed" })).not.toBeInTheDocument();
	});

	it("opens a card when it's clicked", async () => {
		const { onOpenCard } = renderBoard(board({ saved: [acme] }));
		await userEvent.click(screen.getByRole("button", { name: /^Software Engineer/ }));
		expect(onOpenCard).toHaveBeenCalledWith(acme.id);
	});

	// @scenario [features/job-board.feature#move-card-forward] Move a card to the next status
	it("moves a card to the next status", () => {
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
		const { onMove } = renderBoard(board({ saved: [acme] }));

		// @when [features/job-board.feature] I move the card to "Applied"
		drag(acme.id, "applied");

		expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ jobId: 1, status: "applied" }));
		// @then [features/job-board.feature] the card appears in the "Applied" column
		expect(acmeIn("Applied")).toBe(true);
		// @and [features/job-board.feature] the card no longer appears in the "Saved" column
		expect(acmeIn("Saved")).toBe(false);
	});

	// @scenario [features/job-board.feature#move-card-any-status] Move a card to any other status
	it.each([
		["saved", "Saved", "interviewing", "Interviewing"],
		["applied", "Applied", "interviewing", "Interviewing"],
		["interviewing", "Interviewing", "offer", "Offer"],
		["offer", "Offer", "interviewing", "Interviewing"],
		["closed", "Closed", "applied", "Applied"],
	])("moves a card from %s to %s", (from, fromLabel, to, toLabel) => {
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "<from>"
		const { onMove } = renderBoard(board({ [from]: [acme] }));

		// @when [features/job-board.feature] I move the card to "<to>"
		drag(acme.id, to);

		expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ jobId: 1, status: to }));
		// @then [features/job-board.feature] the card appears in the "<to>" column
		expect(acmeIn(toLabel)).toBe(true);
		expect(acmeIn(fromLabel)).toBe(false);
	});

	// @scenario [features/job-board.feature#move-keeps-details] Moving a card keeps its details and documents
	it("keeps a moved card's details", () => {
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Applied"
		// @and [features/job-board.feature] the job has a job description, a link, and an attached resume
		const detailed = card(1, "Acme", { location: "Remote (US)", documentCount: 1 });
		const { onMove } = renderBoard(board({ applied: [detailed] }));

		// @when [features/job-board.feature] I move the card to "Interviewing"
		drag(detailed.id, "interviewing");

		// @then [features/job-board.feature] the card still shows the company "Acme" and the role "Software Engineer"
		const moved = within(column("Interviewing")).getByRole("article");
		expect(moved).toHaveTextContent("Acme");
		expect(moved).toHaveTextContent("Software Engineer");
		// @and [features/job-board.feature] the job description, link, and attached resume are unchanged
		expect(onMove).toHaveBeenCalledWith({
			jobId: 1,
			status: "interviewing",
			index: 0,
			closingReason: "",
			closingReasonOther: "",
		});
		expect(moved).toHaveTextContent("1 document");
	});

	// The backend decides the date applied; the board shows what it returns after the move.
	describe("date applied after a move", () => {
		// @scenario [features/job-board.feature#move-to-applied-sets-date] Moving a card to Applied fills in today's date when no date is set
		it("shows the date filled in when moved to Applied", () => {
			// @and [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Saved" with no date applied
			renderBoard(board({ saved: [acme] }), () =>
				board({ applied: [card(1, "Acme", { dateApplied: "2026-10-09" })] }),
			);

			// @when [features/job-board.feature] I move the card to "Applied"
			drag(acme.id, "applied");

			// @then [features/job-board.feature] the job's date applied is 2026-10-09
			expect(within(column("Applied")).getByText("Applied 2026-10-09")).toBeInTheDocument();
		});

		// @scenario [features/job-board.feature#move-to-applied-keeps-date] Moving a card to Applied keeps a date applied that is already set
		it("shows the date kept when moved to Applied", () => {
			// @and [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing" with a date applied of 2026-10-01
			const dated = card(1, "Acme", { dateApplied: "2026-10-01" });
			renderBoard(board({ interviewing: [dated] }), () => board({ applied: [dated] }));

			// @when [features/job-board.feature] I move the card to "Applied"
			drag(dated.id, "applied");

			// @then [features/job-board.feature] the job's date applied is still 2026-10-01
			expect(within(column("Applied")).getByText("Applied 2026-10-01")).toBeInTheDocument();
		});

		// @scenario [features/job-board.feature#move-to-saved-clears-date] Moving a card back to Saved clears the date applied
		it.each([["applied"], ["interviewing"], ["offer"], ["closed"]])(
			"shows no date when moved from %s back to Saved",
			(from) => {
				// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "<from>" with a date applied of 2026-10-01
				const dated = card(1, "Acme", { dateApplied: "2026-10-01" });
				renderBoard(board({ [from]: [dated] }), () => board({ saved: [acme] }));

				// @when [features/job-board.feature] I move the card to "Saved"
				drag(dated.id, "saved");

				// @then [features/job-board.feature#move-to-saved-clears-date] the job has no date applied
				expect(within(column("Saved")).getByRole("article")).not.toHaveTextContent("Applied 20");
			},
		);

		// @scenario [features/job-board.feature#reorder-saved-keeps-date] Reordering a card within Saved keeps its date applied
		it("keeps the date when reordering within Saved", () => {
			// @given [features/job-board.feature] the "Saved" column holds the jobs at "Globex" and "Acme" in that order
			// @and [features/job-board.feature] the job at "Acme" has a date applied of 2026-10-01
			const dated = card(1, "Acme", { dateApplied: "2026-10-01" });
			const globex = card(2, "Globex");
			const { onMove } = renderBoard(board({ saved: [globex, dated] }));

			// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
			drag(dated.id, globex.id);

			expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ status: "saved", index: 0 }));
			// @then [features/job-board.feature] the job's date applied is still 2026-10-01
			expect(within(column("Saved")).getByText("Applied 2026-10-01")).toBeInTheDocument();
		});

		// @scenario [features/job-board.feature#reorder-applied-no-date] Reordering a card within Applied doesn't fill in a date applied
		it("leaves the date empty when reordering within Applied", () => {
			// @given [features/job-board.feature] the "Applied" column holds the jobs at "Globex" and "Acme" in that order
			// @and [features/job-board.feature] the job at "Acme" has no date applied
			const globex = card(2, "Globex");
			const { onMove } = renderBoard(board({ applied: [globex, acme] }));

			// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
			drag(acme.id, globex.id);

			expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ status: "applied", index: 0 }));
			// @then [features/job-board.feature#reorder-applied-no-date] the job has no date applied
			expect(cardsIn("Applied")).toEqual([
				"Software Engineer at Acme",
				"Software Engineer at Globex",
			]);
			expect(within(column("Applied")).queryByText(/^Applied 20/)).not.toBeInTheDocument();
		});
	});

	describe("closing a job", () => {
		const interviewing = () => board({ interviewing: [acme] });
		const closedWith = (closingReasonText: string) => () =>
			board({ closed: [card(1, "Acme", { closingReason: "x", closingReasonText })] });

		// @scenario [features/job-board.feature#close-asks-reason] Moving a card to Closed asks for a reason
		it.each([
			["rejected", "Rejected"],
			["withdrew", "Withdrew"],
			["no_response", "No response"],
			["declined_offer", "Declined offer"],
			["position_filled", "Position filled"],
		])("asks for a reason and closes with %s", async (value, reason) => {
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			const { onMove } = renderBoard(interviewing(), closedWith(reason));

			// @when [features/job-board.feature] I move the card to "Closed"
			drag(acme.id, "closed");
			expect(onMove).not.toHaveBeenCalled();

			// @and [features/job-board.feature] I choose the closing reason "<reason>"
			await userEvent.click(screen.getByRole("radio", { name: reason }));
			await userEvent.click(screen.getByRole("button", { name: "Close job" }));

			expect(onMove).toHaveBeenCalledWith({
				jobId: 1,
				status: "closed",
				index: 0,
				closingReason: value,
				closingReasonOther: "",
			});
			// @then [features/job-board.feature] the card appears in the "Closed" column
			expect(acmeIn("Closed")).toBe(true);
			// @and [features/job-board.feature] the card shows the closing reason "<reason>"
			expect(within(column("Closed")).getByText(`Closed: ${reason}`)).toBeInTheDocument();
			expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
		});

		// @scenario [features/job-board.feature#close-other-reason] Write my own reason when closing with Other
		it("closes with a reason written for Other", async () => {
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			const { onMove } = renderBoard(interviewing(), closedWith("Company went through layoffs"));

			// @when [features/job-board.feature] I move the card to "Closed"
			drag(acme.id, "closed");
			// @and [features/job-board.feature] I choose the closing reason "Other" and write "Company went through layoffs"
			await userEvent.click(screen.getByRole("radio", { name: "Other" }));
			await userEvent.type(screen.getByLabelText("What happened?"), "Company went through layoffs");
			await userEvent.click(screen.getByRole("button", { name: "Close job" }));

			expect(onMove).toHaveBeenCalledWith(
				expect.objectContaining({
					closingReason: "other",
					closingReasonOther: "Company went through layoffs",
				}),
			);
			// @then [features/job-board.feature] the card appears in the "Closed" column
			expect(acmeIn("Closed")).toBe(true);
			// @and [features/job-board.feature] the card shows the closing reason "Company went through layoffs"
			expect(
				within(column("Closed")).getByText("Closed: Company went through layoffs"),
			).toBeInTheDocument();
		});

		// @scenario [features/job-board.feature#close-other-needs-text] Closing with Other needs a written reason
		it("doesn't close with Other and nothing written", async () => {
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			const { onMove } = renderBoard(interviewing());

			// @when [features/job-board.feature] I move the card to "Closed"
			drag(acme.id, "closed");
			// @and [features/job-board.feature] I choose the closing reason "Other" without writing a reason
			await userEvent.click(screen.getByRole("radio", { name: "Other" }));
			await userEvent.click(screen.getByRole("button", { name: "Close job" }));

			// @then [features/job-board.feature#close-other-needs-text] the card stays in the "Interviewing" column
			expect(onMove).not.toHaveBeenCalled();
			expect(acmeIn("Interviewing")).toBe(true);
			// @and [features/job-board.feature] I am told to write why the job was closed
			expect(screen.getByRole("alert")).toHaveTextContent("Write why the job was closed.");
		});

		// @scenario [features/job-board.feature#close-cancel] Cancelling the closing reason leaves the card where it was
		it("leaves the card where it was when the reason is cancelled", async () => {
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			const { onMove } = renderBoard(interviewing());

			// @when [features/job-board.feature] I move the card to "Closed"
			drag(acme.id, "closed");
			// @but [features/job-board.feature] I cancel instead of choosing a closing reason
			await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

			// @then [features/job-board.feature#close-cancel] the card stays in the "Interviewing" column
			expect(onMove).not.toHaveBeenCalled();
			expect(acmeIn("Interviewing")).toBe(true);
			expect(acmeIn("Closed")).toBe(false);
			expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
		});

		it("shows the backend's message and stays open when closing fails", async () => {
			const { onMove } = renderBoard(interviewing());
			drag(acme.id, "closed");
			onMove.mockRejectedValueOnce("write why the job was closed");
			await userEvent.click(screen.getByRole("radio", { name: "Rejected" }));
			await userEvent.click(screen.getByRole("button", { name: "Close job" }));

			expect(screen.getByRole("alert")).toHaveTextContent("write why the job was closed");
			expect(screen.getByRole("dialog")).toBeInTheDocument();
		});

		// @scenario [features/job-board.feature#reopen-clears-reason] Moving a card out of Closed clears the closing reason
		it("moves a card out of Closed without asking", () => {
			// @given [features/job-board.feature#reopen-clears-reason] I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
			const closed = card(1, "Acme", {
				closingReason: "no_response",
				closingReasonText: "No response",
			});
			const { onMove } = renderBoard(board({ closed: [closed] }), () =>
				board({ interviewing: [acme] }),
			);

			// @when [features/job-board.feature] I move the card to "Interviewing"
			drag(closed.id, "interviewing");

			expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
			expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ status: "interviewing" }));
			// @then [features/job-board.feature] the card appears in the "Interviewing" column
			expect(acmeIn("Interviewing")).toBe(true);
			// @and [features/job-board.feature] the job has no closing reason
			expect(screen.queryByText(/^Closed:/)).not.toBeInTheDocument();
		});

		// @scenario [features/job-board.feature#reorder-closed-keeps-reason] Reordering a card within Closed keeps its closing reason
		it("reorders within Closed without asking for a reason", () => {
			// @given [features/job-board.feature] the "Closed" column holds the jobs at "Globex" and "Acme" in that order
			// @and [features/job-board.feature] the job at "Acme" has the closing reason "Rejected"
			const globex = card(2, "Globex", {
				closingReason: "withdrew",
				closingReasonText: "Withdrew",
			});
			const closed = card(1, "Acme", { closingReason: "rejected", closingReasonText: "Rejected" });
			const { onMove } = renderBoard(board({ closed: [globex, closed] }));

			// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
			drag(closed.id, globex.id);

			expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
			expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ status: "closed", index: 0 }));
			// @then [features/job-board.feature] the card shows the closing reason "Rejected"
			const [first] = within(column("Closed")).getAllByRole("article");
			expect(first).toHaveTextContent("Closed: Rejected");
		});
	});

	describe("ordering cards", () => {
		// @scenario [features/job-board.feature#reorder-card] Reorder a card within its column
		it("reorders a card within its column", () => {
			// @given [features/job-board.feature] the "Saved" column holds the jobs at "Acme", "Globex", and "Initech" in that order
			const initech = card(3, "Initech");
			const { onMove } = renderBoard(board({ saved: [acme, card(2, "Globex"), initech] }));

			// @when [features/job-board.feature] I move the "Initech" card above the "Acme" card
			drag(initech.id, acme.id);

			expect(onMove).toHaveBeenCalledWith(
				expect.objectContaining({ jobId: 3, status: "saved", index: 0 }),
			);
			// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
			expect(cardsIn("Saved")).toEqual([
				"Software Engineer at Initech",
				"Software Engineer at Acme",
				"Software Engineer at Globex",
			]);
		});

		// @scenario [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
		it("places a moved card at a chosen position in another column", () => {
			// @given [features/job-board.feature] the "Applied" column holds the jobs at "Globex" and "Initech" in that order
			// @and [features/job-board.feature] I have a job at "Acme" in "Saved"
			const globex = card(2, "Globex");
			const initech = card(3, "Initech");
			const { onMove } = renderBoard(board({ saved: [acme], applied: [globex, initech] }));

			// @when [features/job-board.feature] I move the "Acme" card into "Applied" between "Globex" and "Initech"
			drag(acme.id, initech.id, acme.id);

			expect(onMove).toHaveBeenCalledWith(
				expect.objectContaining({ jobId: 1, status: "applied", index: 1 }),
			);
			// @then [features/job-board.feature] the "Applied" column shows "Globex", "Acme", and "Initech" in that order
			expect(cardsIn("Applied")).toEqual([
				"Software Engineer at Globex",
				"Software Engineer at Acme",
				"Software Engineer at Initech",
			]);
		});

		// @scenario [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
		it("shows cards in the order the backend gives them", () => {
			// @then [features/job-board.feature] the "Saved" column shows "Acme", "Globex", and "Initech" in that order
			renderBoard(board({ saved: [acme, card(2, "Globex"), card(3, "Initech")] }));
			expect(cardsIn("Saved")).toEqual([
				"Software Engineer at Acme",
				"Software Engineer at Globex",
				"Software Engineer at Initech",
			]);
		});

		// @scenario [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
		it("shows the saved order when the app reopens", () => {
			// @given [features/job-board.feature] the "Saved" column holds the jobs at "Initech", "Acme", and "Globex" in that order
			// @when [features/job-board.feature] I close and reopen the app
			renderBoard(board({ saved: [card(3, "Initech"), acme, card(2, "Globex")] }));
			// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
			expect(cardsIn("Saved")).toEqual([
				"Software Engineer at Initech",
				"Software Engineer at Acme",
				"Software Engineer at Globex",
			]);
		});

		// @scenario [features/job-board.feature#board-persists] Board is the same after reopening the app
		it("shows each card in its saved column when the app reopens", () => {
			// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			// @when [features/job-board.feature] I close and reopen the app
			renderBoard(board({ interviewing: [acme] }));
			// @then [features/job-board.feature] the card for "Software Engineer" at "Acme" is in the "Interviewing" column
			expect(acmeIn("Interviewing")).toBe(true);
		});
	});

	it("makes no move when a card is dropped where it started", () => {
		const { onMove } = renderBoard(board({ saved: [acme] }));
		drag(acme.id, acme.id);
		expect(onMove).not.toHaveBeenCalled();
	});

	it("shows the backend's message when a move fails", async () => {
		const { onMove } = renderBoard(board({ saved: [acme] }));
		onMove.mockRejectedValueOnce("database is locked");
		drag(acme.id, "offer");
		expect(await screen.findByRole("alert")).toHaveTextContent("database is locked");
	});
});
