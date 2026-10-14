import type { DndContextProps, DragEndEvent, DragOverEvent, DragStartEvent } from "@dnd-kit/core";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { applyMove } from "../board/boardDnd";
import type { DueFilter, Pages } from "../board/boardView";
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
		// The real overlay only draws during a drag dnd-kit itself started, so show what the board
		// puts in it.
		DragOverlay: ({ children }: { children?: React.ReactNode }) => (
			<div data-testid="drag-overlay">{children}</div>
		),
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
// once, then the board is replaced by what the backend returns, when a reload is given. Like App,
// the parent keeps the search, the filter, and each column's page; view sets where they start.
function renderBoard(
	start: BoardData,
	reload?: (move: Move) => BoardData,
	view: { search?: string; filter?: DueFilter; pages?: Pages } = {},
) {
	const onMove = vi.fn<(move: Move) => Promise<void>>();
	const onOpenCard = vi.fn();
	const onAddJob = vi.fn();
	const controls = { setSearch: (_: string) => {}, setFilter: (_: DueFilter) => {} };
	function Parent() {
		const [current, setCurrent] = useState(start);
		const [search, setSearch] = useState(view.search ?? "");
		const [filter, setFilter] = useState<DueFilter>(view.filter ?? "all");
		const [pages, setPages] = useState<Pages>(view.pages ?? {});
		controls.setSearch = setSearch;
		controls.setFilter = setFilter;
		onMove.mockImplementation(async (move) => {
			setCurrent((b) => applyMove(b, move));
			if (reload) {
				setCurrent(reload(move));
			}
		});
		return (
			<Board
				board={current}
				onMove={onMove}
				onOpenCard={onOpenCard}
				onAddJob={onAddJob}
				search={search}
				filter={filter}
				pages={pages}
				onPageChange={(status, page) => setPages((p) => ({ ...p, [status]: page }))}
			/>
		);
	}
	render(<Parent />);
	const clearSearch = () => act(() => controls.setSearch(""));
	const showAll = () => act(() => controls.setFilter("all"));
	return { onMove, onOpenCard, onAddJob, clearSearch, showAll };
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

// many makes n cards at "Company <id>", with IDs from first.
function many(n: number, first = 1) {
	return Array.from({ length: n }, (_, i) => card(first + i, `Company ${first + i}`));
}

// names is how cardsIn names the cards with IDs from first to last.
function names(first: number, last: number) {
	return Array.from(
		{ length: last - first + 1 },
		(_, i) => `Software Engineer at Company ${first + i}`,
	);
}

const arrow = (label: string, which: "Previous" | "Next") =>
	within(column(label)).getByRole("button", { name: `${which} page of ${label}` });

const acmeIn = (label: string) => cardsIn(label).includes("Software Engineer at Acme");

describe("Board", () => {
	// @scenario [#board-status-columns]
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

	// @scenario [#empty-board]
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

	// @scenario [#column-card-count]
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

	it("draws the dragged card above the columns, hidden from screen readers, until it's dropped", () => {
		renderBoard(board({ saved: [acme] }));
		const overlay = screen.getByTestId("drag-overlay");
		const active = { id: acme.id } as DragStartEvent["active"];

		act(() => dnd.props?.onDragStart?.({ active } as DragStartEvent));
		const picture = within(overlay).getByRole("article", { hidden: true });
		expect(picture).toHaveAccessibleName("Software Engineer at Acme");
		expect(picture.closest("[aria-hidden='true']")).not.toBeNull();
		expect(cardsIn("Saved")).toEqual(["Software Engineer at Acme"]);

		act(() =>
			dnd.props?.onDragCancel?.({ active } as Parameters<
				NonNullable<DndContextProps["onDragCancel"]>
			>[0]),
		);
		expect(overlay).toBeEmptyDOMElement();

		drag(acme.id, "applied");
		expect(overlay).toBeEmptyDOMElement();
	});

	// @scenario [#move-card-forward]
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

	// @scenario [#move-card-any-status]
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

	// @scenario [#move-keeps-details]
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
		// @scenario [#move-to-applied-sets-date]
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

		// @scenario [#move-to-applied-keeps-date]
		it("shows the date kept when moved to Applied", () => {
			// @and [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing" with a date applied of 2026-10-01
			const dated = card(1, "Acme", { dateApplied: "2026-10-01" });
			renderBoard(board({ interviewing: [dated] }), () => board({ applied: [dated] }));

			// @when [features/job-board.feature] I move the card to "Applied"
			drag(dated.id, "applied");

			// @then [features/job-board.feature] the job's date applied is still 2026-10-01
			expect(within(column("Applied")).getByText("Applied 2026-10-01")).toBeInTheDocument();
		});

		// @scenario [#move-to-saved-clears-date]
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

		// @scenario [#reorder-saved-keeps-date]
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

		// @scenario [#reorder-applied-no-date]
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

		// @scenario [#close-asks-reason]
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

		// @scenario [#close-other-reason]
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

		// @scenario [#close-other-needs-text]
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

		// @scenario [#close-cancel]
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

		// @scenario [#history-ignores-cancelled-close]
		it("leaves the history alone when the close is cancelled", async () => {
			// @given [features/job-status-history.feature] I have a job for "Software Engineer" at "Acme" whose history ends with "Interviewing"
			const { onMove } = renderBoard(interviewing());

			// @when [features/job-status-history.feature#history-ignores-cancelled-close] I move the card to "Closed"
			drag(acme.id, "closed");
			// @but [features/job-status-history.feature] I cancel instead of choosing a closing reason
			await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

			// Only MoveJob adds to the history, and it's never called, so the history still ends with
			// "Interviewing" when the card is opened (TestHistoryIgnoresCancelledClose).
			expect(onMove).not.toHaveBeenCalled();
			expect(acmeIn("Interviewing")).toBe(true);
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

		// @scenario [#reopen-clears-reason]
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

		// @scenario [#reorder-closed-keeps-reason]
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
		// @scenario [#reorder-card]
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

		// @scenario [#move-card-to-position]
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

		// @scenario [#new-card-at-top]
		it("shows cards in the order the backend gives them", () => {
			// @then [features/job-board.feature] the "Saved" column shows "Acme", "Globex", and "Initech" in that order
			renderBoard(board({ saved: [acme, card(2, "Globex"), card(3, "Initech")] }));
			expect(cardsIn("Saved")).toEqual([
				"Software Engineer at Acme",
				"Software Engineer at Globex",
				"Software Engineer at Initech",
			]);
		});

		// @scenario [#card-order-persists]
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

		// @scenario [#board-persists]
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

	describe("searching", () => {
		// @scenario [#search-move-card]
		it("moves a matching card to another status while searching", () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			// @and [features/board-search.feature] I have searched the board for "Acme"
			const { onMove } = renderBoard(board({ saved: [acme, card(2, "Globex")] }), undefined, {
				search: "Acme",
			});

			// @when [features/board-search.feature] I move the card to "Applied"
			drag(acme.id, "applied");

			expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ jobId: 1, status: "applied" }));
			// @then [features/board-search.feature] the card appears in the "Applied" column
			expect(acmeIn("Applied")).toBe(true);
			// @and [features/board-search.feature] the search for "Acme" is still applied
			expect(cardsIn("Saved")).toEqual([]);
		});

		// @scenario [#search-move-to-top]
		it("puts a card moved while searching at the top of its new column", () => {
			// @given [features/board-search.feature] the "Applied" column holds the jobs at "Globex" and "Initech" in that order
			// @and [features/board-search.feature] I have a job at "Acme" in "Saved"
			// @and [features/board-search.feature] I have searched the board for "Acme"
			const { onMove, clearSearch } = renderBoard(
				board({ saved: [acme], applied: [card(2, "Globex"), card(3, "Initech")] }),
				undefined,
				{ search: "Acme" },
			);

			// @when [features/board-search.feature] I move the "Acme" card to "Applied"
			drag(acme.id, "applied");
			expect(onMove).toHaveBeenCalledWith(
				expect.objectContaining({ jobId: 1, status: "applied", index: 0 }),
			);
			// @and [features/board-search.feature] I clear the search
			clearSearch();

			// @then [features/board-search.feature] the "Applied" column shows "Acme", "Globex", and "Initech" in that order
			expect(cardsIn("Applied")).toEqual([
				"Software Engineer at Acme",
				"Software Engineer at Globex",
				"Software Engineer at Initech",
			]);
		});

		// @scenario [#search-no-reorder]
		it("doesn't reorder a card within its column while searching", () => {
			// @given [features/board-search.feature] the "Saved" column holds the jobs at "Acme", "Globex", and "Acme Robotics" in that order
			// @and [features/board-search.feature] I have searched the board for "Acme"
			const robotics = card(3, "Acme Robotics");
			const { onMove, clearSearch } = renderBoard(
				board({ saved: [acme, card(2, "Globex"), robotics] }),
				undefined,
				{ search: "Acme" },
			);

			// @when [features/board-search.feature] I try to move the "Acme Robotics" card above the "Acme" card
			drag(robotics.id, acme.id);

			// @then [features/board-search.feature] the "Acme Robotics" card stays where it was
			expect(onMove).not.toHaveBeenCalled();
			expect(cardsIn("Saved")).toEqual([
				"Software Engineer at Acme",
				"Software Engineer at Acme Robotics",
			]);
			// @and [features/board-search.feature] I am told cards can't be reordered while searching
			expect(screen.getByRole("alert")).toHaveTextContent(
				"Cards can't be reordered while searching.",
			);
			// @and [features/board-search.feature] after I clear the search the "Saved" column shows "Acme", "Globex", and "Acme Robotics" in that order
			clearSearch();
			expect(cardsIn("Saved")).toEqual([
				"Software Engineer at Acme",
				"Software Engineer at Globex",
				"Software Engineer at Acme Robotics",
			]);
		});

		it("still reorders a card once the search is only spaces", () => {
			const globex = card(2, "Globex");
			const { onMove } = renderBoard(board({ saved: [acme, globex] }), undefined, {
				search: "   ",
			});
			drag(globex.id, acme.id);
			expect(onMove).toHaveBeenCalledWith(expect.objectContaining({ jobId: 2, index: 0 }));
		});
	});

	describe("pages", () => {
		// @scenario [#column-first-page]
		it("shows the first page of a column with more than 10 cards", () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @when [features/board-pagination.feature] I open the job search board
			renderBoard(board({ saved: many(25) }));

			// @then [features/board-pagination.feature] the "Saved" column shows its first 10 cards in order
			expect(cardsIn("Saved")).toEqual(names(1, 10));
			// @and [features/board-pagination.feature] the "Saved" column shows that it is on page 1 of 3
			expect(within(column("Saved")).getByText("Page 1 of 3")).toBeInTheDocument();
		});

		// @scenario [#column-fits-one-page]
		it("has no pages to move between in a column of 10 cards", () => {
			// @given [features/board-pagination.feature] the "Applied" column holds 10 jobs
			// @when [features/board-pagination.feature] I open the job search board
			renderBoard(board({ applied: many(10) }));

			// @then [features/board-pagination.feature] the "Applied" column shows all 10 cards
			expect(cardsIn("Applied")).toEqual(names(1, 10));
			// @and [features/board-pagination.feature] the "Applied" column offers no other pages
			expect(
				within(column("Applied")).queryByRole("navigation", { name: "Pages of Applied" }),
			).not.toBeInTheDocument();
		});

		// @scenario [#count-shows-column-total]
		it("counts every card in the column on a later page", () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Saved" column is on page 2
			renderBoard(board({ saved: many(25) }), undefined, { pages: { saved: 2 } });

			// @then [features/board-pagination.feature] the "Saved" column shows a count of 25
			expect(within(column("Saved")).getByText("25 jobs")).toBeInTheDocument();
			expect(cardsIn("Saved")).toHaveLength(10);
		});

		// @scenario [#next-page]
		it("goes to a column's next page", async () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Saved" column is on page 1
			renderBoard(board({ saved: many(25) }));

			// @when [features/board-pagination.feature] I go to the next page of the "Saved" column
			await userEvent.click(arrow("Saved", "Next"));

			// @then [features/board-pagination.feature] the "Saved" column shows cards 11 to 20
			expect(cardsIn("Saved")).toEqual(names(11, 20));
			// @and [features/board-pagination.feature] the "Saved" column shows that it is on page 2 of 3
			expect(within(column("Saved")).getByText("Page 2 of 3")).toBeInTheDocument();
		});

		// @scenario [#previous-page]
		it("goes to a column's previous page", async () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Saved" column is on page 3
			renderBoard(board({ saved: many(25) }), undefined, { pages: { saved: 3 } });
			expect(cardsIn("Saved")).toEqual(names(21, 25));

			// @when [features/board-pagination.feature] I go to the previous page of the "Saved" column
			await userEvent.click(arrow("Saved", "Previous"));

			// @then [features/board-pagination.feature] the "Saved" column shows cards 11 to 20
			expect(cardsIn("Saved")).toEqual(names(11, 20));
			// @and [features/board-pagination.feature] the "Saved" column shows that it is on page 2 of 3
			expect(within(column("Saved")).getByText("Page 2 of 3")).toBeInTheDocument();
		});

		// @scenario [#page-bounds]
		it.each([
			[1, "Previous" as const],
			[3, "Next" as const],
		])("can't leave page %i with the %s arrow", async (page, which) => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Saved" column is on page <page>
			renderBoard(board({ saved: many(25) }), undefined, { pages: { saved: page } });

			// @then [features/board-pagination.feature] I can't go to the <direction> page of the "Saved" column
			expect(arrow("Saved", which)).toHaveAttribute("aria-disabled", "true");
			await userEvent.click(arrow("Saved", which));
			expect(within(column("Saved")).getByText(`Page ${page} of 3`)).toBeInTheDocument();
		});

		// @scenario [#columns-page-independently]
		it("pages one column without moving the others", async () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Applied" column holds 15 jobs
			renderBoard(board({ saved: many(25), applied: many(15, 101) }));

			// @when [features/board-pagination.feature] I go to the next page of the "Saved" column
			await userEvent.click(arrow("Saved", "Next"));

			// @then [features/board-pagination.feature] the "Saved" column is on page 2
			expect(within(column("Saved")).getByText("Page 2 of 3")).toBeInTheDocument();
			// @and [features/board-pagination.feature] the "Applied" column is still on page 1
			expect(within(column("Applied")).getByText("Page 1 of 2")).toBeInTheDocument();
			expect(cardsIn("Applied")).toEqual(
				Array.from({ length: 10 }, (_, i) => `Software Engineer at Company ${101 + i}`),
			);
		});

		// @scenario [#one-page-per-column]
		it("shows only the current page of a column", () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @when [features/board-pagination.feature] I open the job search board
			renderBoard(board({ saved: many(25) }));

			// @then [features/board-pagination.feature] the "Saved" column shows only the cards on its current page
			expect(cardsIn("Saved")).toHaveLength(10);
			// @and [features/board-pagination.feature] cards on other pages of the "Saved" column can't be seen or moved until I page to them
			expect(
				screen.queryByRole("button", { name: "Move Software Engineer at Company 11" }),
			).not.toBeInTheDocument();
		});
	});

	describe("page arrows", () => {
		// @scenario [#page-arrows]
		it("shows previous and next arrows on a column with more than one page", () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @when [features/board-pagination.feature] I open the job search board
			renderBoard(board({ saved: many(25) }));

			// @then [features/board-pagination.feature] the "Saved" column shows a previous arrow and a next arrow
			expect(arrow("Saved", "Next")).toHaveAttribute("aria-disabled", "false");
			// @and [features/board-pagination.feature] the previous arrow can't be used on page 1
			expect(arrow("Saved", "Previous")).toHaveAttribute("aria-disabled", "true");
		});

		// @scenario [#page-transition]
		it.each([
			["next" as const, "Next" as const, "left", 3, "right", names(21, 25)],
			["previous" as const, "Previous" as const, "right", 1, "left", names(1, 10)],
		])("slides the cards when the %s arrow is used", async (_, which, out, page, into, shown) => {
			vi.spyOn(window, "matchMedia").mockReturnValue({ matches: false } as MediaQueryList);
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Saved" column is on page 2
			renderBoard(board({ saved: many(25), applied: many(15, 101) }), undefined, {
				pages: { saved: 2 },
			});

			// @when [features/board-pagination.feature] I use the <arrow> arrow of the "Saved" column
			await userEvent.click(arrow("Saved", which));

			// @then [features/board-pagination.feature] the cards on page 2 slide out toward the <out> side of the column
			const leaving = column("Saved").querySelector(`[data-slide="out-${out}"]`);
			expect(leaving).not.toBeNull();
			expect(
				within(leaving as HTMLElement)
					.getAllByRole("article", { hidden: true })
					.map((a) => a.getAttribute("aria-label")),
			).toEqual(names(11, 20));
			// @and [features/board-pagination.feature] the cards on page <page> slide in from the <in> side of the column
			expect(within(column("Saved")).getByText(`Page ${page} of 3`)).toBeInTheDocument();
			expect(cardsIn("Saved")).toEqual(shown);
			expect(column("Saved").querySelector(`[data-slide="in-${into}"]`)).not.toBeNull();
			// @and [features/board-pagination.feature] the other columns don't move
			expect(column("Applied").querySelector("[data-slide]")).toBeNull();
		});

		it("removes the page that slid out once its animation ends", async () => {
			vi.spyOn(window, "matchMedia").mockReturnValue({ matches: false } as MediaQueryList);
			renderBoard(board({ saved: many(25) }));
			await userEvent.click(arrow("Saved", "Next"));
			const leaving = column("Saved").querySelector('[data-slide="out-left"]') as HTMLElement;
			const entering = column("Saved").querySelector('[data-slide="in-right"]') as HTMLElement;

			act(() => {
				leaving.dispatchEvent(new Event("animationend", { bubbles: true }));
				entering.dispatchEvent(new Event("animationend", { bubbles: true }));
			});

			expect(column("Saved").querySelector("[data-slide]")).toBeNull();
			expect(cardsIn("Saved")).toEqual(names(11, 20));
		});

		// @scenario [#page-transition-reduced-motion]
		it("changes page without sliding when the system asks for reduced motion", async () => {
			// @given [features/board-pagination.feature] my system is set to reduce motion
			vi.spyOn(window, "matchMedia").mockReturnValue({ matches: true } as MediaQueryList);
			// @and [features/board-pagination.feature] the "Saved" column holds 25 jobs
			renderBoard(board({ saved: many(25) }));

			// @when [features/board-pagination.feature] I use the next arrow of the "Saved" column
			await userEvent.click(arrow("Saved", "Next"));

			// @then [features/board-pagination.feature] the "Saved" column shows cards 11 to 20 without sliding
			expect(cardsIn("Saved")).toEqual(names(11, 20));
			expect(column("Saved").querySelector("[data-slide]")).toBeNull();
		});
	});

	describe("moving cards between pages", () => {
		// @scenario [#move-card-onto-page]
		it("places a moved card at a chosen position on a later page", () => {
			// @given [features/board-pagination.feature] the "Applied" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Applied" column is on page 2, showing the jobs at "Globex" and "Initech" next to each other
			// @and [features/board-pagination.feature] I have a job at "Acme" in "Saved"
			const applied = many(25, 101);
			applied[10] = card(2, "Globex");
			applied[11] = card(3, "Initech");
			const { onMove } = renderBoard(board({ saved: [acme], applied }), undefined, {
				pages: { applied: 2 },
			});

			// @when [features/board-pagination.feature] I move the "Acme" card into "Applied" between "Globex" and "Initech"
			drag(acme.id, 3, acme.id);

			expect(onMove).toHaveBeenCalledWith(
				expect.objectContaining({ jobId: 1, status: "applied", index: 11 }),
			);
			// @then [features/board-pagination.feature] the "Applied" column shows "Globex", "Acme", and "Initech" in that order on page 2
			expect(within(column("Applied")).getByText("Page 2 of 3")).toBeInTheDocument();
			expect(cardsIn("Applied").slice(0, 3)).toEqual([
				"Software Engineer at Globex",
				"Software Engineer at Acme",
				"Software Engineer at Initech",
			]);
		});

		it("reorders a card within a later page at its place in the whole column", () => {
			const { onMove } = renderBoard(board({ saved: many(25) }), undefined, {
				pages: { saved: 2 },
			});
			drag(15, 11);
			expect(onMove).toHaveBeenCalledWith(
				expect.objectContaining({ jobId: 15, status: "saved", index: 10 }),
			);
			expect(cardsIn("Saved").slice(0, 2)).toEqual([
				"Software Engineer at Company 15",
				"Software Engineer at Company 11",
			]);
		});

		// @scenario [#last-card-leaves-page]
		it("shows the page before when the only card on the last page moves away", () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 21 jobs
			// @and [features/board-pagination.feature] the "Saved" column is on page 3, showing 1 card
			renderBoard(board({ saved: many(21) }), undefined, { pages: { saved: 3 } });
			expect(cardsIn("Saved")).toEqual(names(21, 21));

			// @when [features/board-pagination.feature] I move that card to "Applied"
			drag(21, "applied");

			// @then [features/board-pagination.feature] the "Saved" column is on page 2 of 2
			expect(within(column("Saved")).getByText("Page 2 of 2")).toBeInTheDocument();
			expect(cardsIn("Saved")).toEqual(names(11, 20));
		});

		// @scenario [#page-filled-after-move]
		it("fills a page back up after a card moves out of it", () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			// @and [features/board-pagination.feature] the "Saved" column is on page 1
			renderBoard(board({ saved: many(25) }));

			// @when [features/board-pagination.feature] I move a card on page 1 to "Applied"
			drag(1, "applied");

			// @then [features/board-pagination.feature] the "Saved" column still shows 10 cards on page 1
			expect(cardsIn("Saved")).toHaveLength(10);
			// @and [features/board-pagination.feature] the card that was first on page 2 is now last on page 1
			expect(cardsIn("Saved").at(-1)).toBe("Software Engineer at Company 11");
		});

		// @scenario [#search-results-paged]
		it("shows matching cards a page at a time", () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 15 jobs at "Acme" and 10 jobs at "Globex"
			const acmes = Array.from({ length: 15 }, (_, i) => card(i + 1, `Acme ${i + 1}`));
			const globexes = Array.from({ length: 10 }, (_, i) => card(i + 16, `Globex ${i + 1}`));
			// @when [features/board-pagination.feature] I search the board for "Acme"
			renderBoard(board({ saved: [...globexes, ...acmes] }), undefined, { search: "Acme" });

			// @then [features/board-pagination.feature] the "Saved" column shows that it is on page 1 of 2
			expect(within(column("Saved")).getByText("Page 1 of 2")).toBeInTheDocument();
			expect(cardsIn("Saved")[0]).toBe("Software Engineer at Acme 1");
			// @and [features/board-pagination.feature] the "Saved" column shows a count of 15
			expect(within(column("Saved")).getByText("15 jobs")).toBeInTheDocument();
		});
	});

	describe("filtering to overdue jobs", () => {
		const overdueCard = (id: number, company: string) =>
			card(id, company, {
				nextAction: { text: "Follow up", expectedDate: "2026-10-10", due: "overdue" },
			});

		it("doesn't reorder a card within its column while filtered", () => {
			const globex = overdueCard(2, "Globex");
			const { onMove, showAll } = renderBoard(
				board({ applied: [overdueCard(1, "Acme"), card(3, "Initech"), globex] }),
				undefined,
				{ filter: "overdue" },
			);

			drag(globex.id, 1);

			expect(onMove).not.toHaveBeenCalled();
			expect(screen.getByRole("alert")).toHaveTextContent(
				"Cards can't be reordered while the board is filtered.",
			);
			showAll();
			expect(cardsIn("Applied")).toEqual([
				"Software Engineer at Acme",
				"Software Engineer at Initech",
				"Software Engineer at Globex",
			]);
		});

		it("puts a card moved while filtered at the top of its new column", () => {
			const { onMove } = renderBoard(
				board({
					applied: [overdueCard(1, "Acme")],
					offer: [card(2, "Globex"), card(3, "Initech")],
				}),
				undefined,
				{ filter: "overdue" },
			);

			drag(1, "offer");

			expect(onMove).toHaveBeenCalledWith(
				expect.objectContaining({ jobId: 1, status: "offer", index: 0 }),
			);
		});
	});
});
