import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { applyMove } from "./board/boardDnd";
import type { Board, Details } from "./hooks/types";
import { useBoard } from "./hooks/useBoard";
import { useDocuments } from "./hooks/useDocuments";
import { useJob } from "./hooks/useJob";
import { board, card, job } from "./test/fixtures";

vi.mock("./hooks/useBoard", () => ({ useBoard: vi.fn() }));
vi.mock("./hooks/useJob", () => ({ useJob: vi.fn() }));
vi.mock("./hooks/useDocuments", () => ({ useDocuments: vi.fn() }));

// mockBoard makes useBoard keep the board in state, adding a job at the top of its column and
// deleting it as the backend would.
function mockBoard(start: Board) {
	const added = vi.fn<(details: Details, status: string) => void>();
	const deleted = vi.fn<(id: number) => void>();
	vi.mocked(useBoard).mockImplementation(() => {
		const [current, setCurrent] = useState(start);
		return {
			board: current,
			error: "",
			reload: async () => {},
			addJob: async (details: Details, status: string) => {
				added(details, status);
				const columns = current.columns.map((c) => {
					if (c.status !== status) {
						return c;
					}
					const cards = [
						card(99, details.company, { role: details.role, location: details.location }),
						...c.cards,
					];
					return { ...c, cards, count: cards.length };
				});
				setCurrent({ columns });
				return job({ id: 99 });
			},
			moveJob: async (move) => setCurrent(applyMove(current, move)),
			deleteJob: async (id: number) => {
				deleted(id);
				const columns = current.columns.map((c) => {
					const cards = c.cards.filter((x) => x.id !== id);
					return { ...c, cards, count: cards.length };
				});
				setCurrent({ columns });
			},
		};
	});
	return { added, deleted };
}

beforeEach(() => {
	vi.mocked(useDocuments).mockReturnValue({
		attach: vi.fn(),
		setKind: vi.fn(),
		remove: vi.fn(),
		open: vi.fn(),
	});
});

const column = (label: string) => screen.getByRole("region", { name: label });
const field = (label: RegExp) => within(screen.getByRole("dialog")).getByLabelText(label);
const acmeIn = (label: string) =>
	within(column(label)).queryByRole("article", { name: "Software Engineer at Acme" });

describe("App", () => {
	// @scenario [features/job-cards.feature#add-job] Add a job with all its details
	it("adds a job with all its details", async () => {
		// @given [features/job-cards.feature] I open the job search board
		const { added } = mockBoard(board());
		render(<App />);

		// @when [features/job-cards.feature] I add a job with:
		await userEvent.click(screen.getByRole("button", { name: "Add job" }));
		await userEvent.type(field(/^Company/), "Acme");
		await userEvent.type(field(/^Role/), "Software Engineer");
		await userEvent.type(field(/^Job description/), "Build and maintain internal tools.");
		await userEvent.type(field(/^Link/), "https://acme.example.com/jobs/123");
		await userEvent.type(field(/^Salary/), "$120k - $140k");
		await userEvent.type(field(/^Location/), "Remote (US)");
		await userEvent.type(field(/^Notes/), "Referred by Dana from the meetup.");
		await userEvent.click(
			within(screen.getByRole("dialog")).getByRole("button", { name: "Add job" }),
		);

		expect(added).toHaveBeenCalledWith(
			{
				company: "Acme",
				role: "Software Engineer",
				description: "Build and maintain internal tools.",
				link: "https://acme.example.com/jobs/123",
				salary: "$120k - $140k",
				location: "Remote (US)",
				notes: "Referred by Dana from the meetup.",
				dateApplied: "",
			},
			"saved",
		);
		// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Saved" column
		const added_card = acmeIn("Saved");
		expect(added_card).toBeInTheDocument();
		// @and [features/job-cards.feature] the card shows the company, the role, and the location
		expect(added_card).toHaveTextContent("Acme");
		expect(added_card).toHaveTextContent("Software Engineer");
		expect(added_card).toHaveTextContent("Remote (US)");
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [features/job-cards.feature#add-job-minimum] Add a job with only a company and a role
	it("adds a job with only a company and a role", async () => {
		// @given [features/job-cards.feature] I open the job search board
		const { added } = mockBoard(board());
		render(<App />);

		// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" and nothing else
		await userEvent.click(screen.getByRole("button", { name: "Add your first job" }));
		await userEvent.type(field(/^Company/), "Acme");
		await userEvent.type(field(/^Role/), "Software Engineer");
		await userEvent.click(
			within(screen.getByRole("dialog")).getByRole("button", { name: "Add job" }),
		);

		expect(added).toHaveBeenCalledWith(
			expect.objectContaining({ company: "Acme", role: "Software Engineer", description: "" }),
			"saved",
		);
		// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Saved" column
		expect(acmeIn("Saved")).toBeInTheDocument();
	});

	// @scenario [features/job-cards.feature#add-job-in-column] Add a job directly to a status column
	it("adds a job directly to a status column", async () => {
		// @given [features/job-cards.feature] I open the job search board
		mockBoard(board());
		render(<App />);

		// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" to the "Applied" column
		await userEvent.click(screen.getByRole("button", { name: "Add a job to Applied" }));
		expect(field(/^Column/)).toHaveValue("applied");
		await userEvent.type(field(/^Company/), "Acme");
		await userEvent.type(field(/^Role/), "Software Engineer");
		await userEvent.click(
			within(screen.getByRole("dialog")).getByRole("button", { name: "Add job" }),
		);

		// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Applied" column
		expect(acmeIn("Applied")).toBeInTheDocument();
		expect(acmeIn("Saved")).not.toBeInTheDocument();
	});

	// @scenario [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
	it("puts a newly added job at the top of its column", async () => {
		// @given [features/job-board.feature] the "Saved" column holds the jobs at "Globex" and "Initech" in that order
		mockBoard(board({ saved: [card(2, "Globex"), card(3, "Initech")] }));
		render(<App />);

		// @when [features/job-board.feature] I add a job at "Acme" to the "Saved" column
		await userEvent.click(screen.getByRole("button", { name: "Add a job to Saved" }));
		await userEvent.type(field(/^Company/), "Acme");
		await userEvent.type(field(/^Role/), "Software Engineer");
		await userEvent.click(
			within(screen.getByRole("dialog")).getByRole("button", { name: "Add job" }),
		);

		// @then [features/job-board.feature] the "Saved" column shows "Acme", "Globex", and "Initech" in that order
		expect(
			within(column("Saved"))
				.getAllByRole("article")
				.map((a) => a.getAttribute("aria-label")),
		).toEqual([
			"Software Engineer at Acme",
			"Software Engineer at Globex",
			"Software Engineer at Initech",
		]);
	});

	// @scenario [features/job-cards.feature#delete-job] Delete a job after confirming
	it("removes a deleted job from the board", async () => {
		// @given [features/job-cards.feature#delete-job] I have a job for "Software Engineer" at "Acme"
		const { deleted } = mockBoard(board({ offer: [card(1, "Acme")] }));
		vi.mocked(useJob).mockReturnValue({
			job: job({ id: 1, status: "offer" }),
			error: "",
			reload: vi.fn(),
			update: vi.fn(),
			setClosingReason: vi.fn(),
			openLink: vi.fn(),
		});
		render(<App />);

		// @when [features/job-cards.feature] I delete the job
		await userEvent.click(screen.getByRole("button", { name: /^Software Engineer/ }));
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		// @and [features/job-cards.feature] I confirm the deletion
		await userEvent.click(screen.getByRole("button", { name: "Delete" }));

		expect(deleted).toHaveBeenCalledWith(1);
		// @then [features/job-cards.feature] the card for "Software Engineer" at "Acme" no longer appears on the board
		expect(
			screen.queryByRole("article", { name: "Software Engineer at Acme" }),
		).not.toBeInTheDocument();
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [features/job-cards.feature#cancel-delete-job] Cancel deleting a job
	it("keeps the job on the board when deleting is cancelled", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
		const { deleted } = mockBoard(board({ offer: [card(1, "Acme")] }));
		vi.mocked(useJob).mockReturnValue({
			job: job({ id: 1, status: "offer" }),
			error: "",
			reload: vi.fn(),
			update: vi.fn(),
			setClosingReason: vi.fn(),
			openLink: vi.fn(),
		});
		render(<App />);

		// @when [features/job-cards.feature] I delete the job
		await userEvent.click(screen.getByRole("button", { name: /^Software Engineer/ }));
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		// @but [features/job-cards.feature] I cancel the deletion
		const confirm = screen.getByRole("dialog", { name: /^Delete/ });
		await userEvent.click(within(confirm).getByRole("button", { name: "Cancel" }));

		// @then [features/job-cards.feature] the card for "Software Engineer" at "Acme" is still on the board
		expect(deleted).not.toHaveBeenCalled();
		expect(acmeIn("Offer")).toBeInTheDocument();
	});

	it("shows why the board couldn't load", () => {
		vi.mocked(useBoard).mockReturnValue({
			board: null,
			error: "database is locked",
			reload: vi.fn(),
			addJob: vi.fn(),
			moveJob: vi.fn(),
			deleteJob: vi.fn(),
		});
		render(<App />);
		expect(screen.getByRole("alert")).toHaveTextContent("database is locked");
	});
});
