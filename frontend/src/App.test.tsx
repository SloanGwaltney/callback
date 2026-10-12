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
import { useLibrary } from "./hooks/useLibrary";
import { board, card, fakeUseJob, job } from "./test/fixtures";

vi.mock("./hooks/useBoard", () => ({ useBoard: vi.fn() }));
vi.mock("./hooks/useJob", () => ({ useJob: vi.fn() }));
vi.mock("./hooks/useDocuments", () => ({ useDocuments: vi.fn() }));
vi.mock("./hooks/useLibrary", () => ({ useLibrary: vi.fn() }));

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
	vi.mocked(useLibrary).mockReturnValue({
		documents: [{ id: 1, fileName: "resume-base-2026.docx", description: "Base resume" }],
		error: "",
		reload: vi.fn(),
		chooseFile: vi.fn(),
		add: vi.fn(),
		setDescription: vi.fn(),
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
		vi.mocked(useJob).mockReturnValue(fakeUseJob({ job: job({ id: 1, status: "offer" }) }));
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
		vi.mocked(useJob).mockReturnValue(fakeUseJob({ job: job({ id: 1, status: "offer" }) }));
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

	describe("search", () => {
		const searchBox = () => screen.getByRole("searchbox", { name: "Search jobs" });
		const shown = (name: string) => screen.queryByRole("article", { name });
		const twoJobs = () =>
			board({
				saved: [card(1, "Acme")],
				applied: [card(2, "Globex", { role: "Product Designer" })],
			});

		// @scenario [features/board-search.feature#search-by-role] Search by role
		it("searches by role", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			// @and [features/board-search.feature] I have a job for "Product Designer" at "Globex" in "Applied"
			mockBoard(twoJobs());
			render(<App />);

			// @when [features/board-search.feature] I search the board for "Engineer"
			await userEvent.type(searchBox(), "Engineer");

			// @then [features/board-search.feature#search-by-role] I see the card for "Software Engineer" at "Acme"
			expect(shown("Software Engineer at Acme")).toBeInTheDocument();
			// @and [features/board-search.feature] I don't see the card for "Product Designer" at "Globex"
			expect(shown("Product Designer at Globex")).not.toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#search-by-company] Search by company
		it("searches by company", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			// @and [features/board-search.feature] I have a job for "Product Designer" at "Globex" in "Applied"
			mockBoard(twoJobs());
			render(<App />);

			// @when [features/board-search.feature] I search the board for "Globex"
			await userEvent.type(searchBox(), "Globex");

			// @then [features/board-search.feature#search-by-company] I see the card for "Product Designer" at "Globex"
			expect(shown("Product Designer at Globex")).toBeInTheDocument();
			// @and [features/board-search.feature] I don't see the card for "Software Engineer" at "Acme"
			expect(shown("Software Engineer at Acme")).not.toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#search-partial-any-case] Search matches part of a word in any letter case
		it.each([["acme"], ["ACM"], ["soft"], ["ENGINEER"]])("matches %s", async (text) => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			mockBoard(
				board({ saved: [card(1, "Acme")], applied: [card(2, "Globex", { role: "Chef" })] }),
			);
			render(<App />);

			// @when [features/board-search.feature] I search the board for "<text>"
			await userEvent.type(searchBox(), text);

			// @then [features/board-search.feature#search-partial-any-case] I see the card for "Software Engineer" at "Acme"
			expect(shown("Software Engineer at Acme")).toBeInTheDocument();
			expect(shown("Chef at Globex")).not.toBeInTheDocument();
		});

		// Cards don't carry a job's notes or other details, so nothing but the role and the
		// company can match.
		// @scenario [features/board-search.feature#search-ignores-other-details] Search doesn't match a job's other details
		it("doesn't match a job's other details", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Globex" with the notes "Heard about it from someone at Acme."
			mockBoard(board({ saved: [card(2, "Globex", { location: "Acme Tower" })] }));
			render(<App />);

			// @when [features/board-search.feature] I search the board for "Acme"
			await userEvent.type(searchBox(), "Acme");

			// @then [features/board-search.feature] I don't see the card for "Software Engineer" at "Globex"
			expect(shown("Software Engineer at Globex")).not.toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#search-blank-shows-all] Searching for only spaces shows every job
		it("shows every job for a search of only spaces", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			// @and [features/board-search.feature] I have a job for "Product Designer" at "Globex" in "Applied"
			mockBoard(twoJobs());
			render(<App />);

			// @when [features/board-search.feature] I search the board for "   "
			await userEvent.type(searchBox(), "   ");

			// @then [features/board-search.feature#search-blank-shows-all] I see the card for "Software Engineer" at "Acme"
			expect(shown("Software Engineer at Acme")).toBeInTheDocument();
			// @and [features/board-search.feature#search-blank-shows-all] I see the card for "Product Designer" at "Globex"
			expect(shown("Product Designer at Globex")).toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#search-keeps-columns] Matching cards stay in their own columns
		it("keeps matching cards in their own columns", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			// @and [features/board-search.feature] I have a job for "Data Engineer" at "Globex" in "Interviewing"
			mockBoard(
				board({
					saved: [card(1, "Acme")],
					interviewing: [card(2, "Globex", { role: "Data Engineer" })],
				}),
			);
			render(<App />);

			// @when [features/board-search.feature] I search the board for "Engineer"
			await userEvent.type(searchBox(), "Engineer");

			// @then [features/board-search.feature] the card for "Software Engineer" at "Acme" is in the "Saved" column
			expect(acmeIn("Saved")).toBeInTheDocument();
			// @and [features/board-search.feature] the card for "Data Engineer" at "Globex" is in the "Interviewing" column
			expect(
				within(column("Interviewing")).getByRole("article", { name: "Data Engineer at Globex" }),
			).toBeInTheDocument();
			// @and [features/board-search.feature] I still see every status column
			expect(screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent)).toEqual([
				"Saved",
				"Applied",
				"Interviewing",
				"Offer",
				"Closed",
			]);
		});

		// @scenario [features/board-search.feature#search-counts-matches] Column counts show how many cards match
		it("counts the matching cards in each column", async () => {
			// @given [features/board-search.feature] I have 3 jobs in "Saved", 2 of them at "Acme"
			// @and [features/board-search.feature] I have 1 job in "Applied" at "Globex"
			mockBoard(
				board({
					saved: [card(1, "Acme"), card(2, "Initech"), card(3, "Acme", { role: "Designer" })],
					applied: [card(4, "Globex")],
				}),
			);
			render(<App />);

			// @when [features/board-search.feature] I search the board for "Acme"
			await userEvent.type(searchBox(), "Acme");

			// @then [features/board-search.feature] the "Saved" column shows a count of 2
			expect(within(column("Saved")).getByText("2 jobs")).toBeInTheDocument();
			// @and [features/board-search.feature] the "Applied" column shows a count of 0
			expect(within(column("Applied")).getByText("0 jobs")).toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#search-no-matches] Told when nothing matches
		it("says when nothing matches", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			mockBoard(board({ saved: [card(1, "Acme")] }));
			render(<App />);

			// @when [features/board-search.feature] I search the board for "Initech"
			await userEvent.type(searchBox(), "Initech");

			// @then [features/board-search.feature] every status column is empty
			expect(screen.queryAllByRole("article")).toEqual([]);
			expect(screen.getAllByText("No jobs")).toHaveLength(5);
			// @and [features/board-search.feature] I am told no jobs match "Initech"
			expect(screen.getByText('No jobs match "Initech".')).toHaveAttribute("role", "status");
			// @but [features/board-search.feature] I am not prompted to add my first job
			expect(screen.queryByRole("button", { name: "Add your first job" })).not.toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#clear-search] Clearing the search shows every job again
		it("shows every job again when the search is cleared", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			// @and [features/board-search.feature] I have a job for "Product Designer" at "Globex" in "Applied"
			mockBoard(twoJobs());
			render(<App />);
			// @and [features/board-search.feature] I have searched the board for "Acme"
			await userEvent.type(searchBox(), "Acme");
			expect(shown("Product Designer at Globex")).not.toBeInTheDocument();

			// @when [features/board-search.feature] I clear the search
			await userEvent.click(screen.getByRole("button", { name: "Clear search" }));

			expect(searchBox()).toHaveValue("");
			// @then [features/board-search.feature#clear-search] I see the card for "Software Engineer" at "Acme"
			expect(shown("Software Engineer at Acme")).toBeInTheDocument();
			// @and [features/board-search.feature#clear-search] I see the card for "Product Designer" at "Globex"
			expect(shown("Product Designer at Globex")).toBeInTheDocument();
		});

		it("clears the search with Escape", async () => {
			mockBoard(twoJobs());
			render(<App />);
			await userEvent.type(searchBox(), "Acme{Escape}");
			expect(searchBox()).toHaveValue("");
			expect(shown("Product Designer at Globex")).toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#search-not-kept] Search is cleared after reopening the app
		it("starts with an empty search when the app reopens", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			// @and [features/board-search.feature] I have a job for "Product Designer" at "Globex" in "Applied"
			mockBoard(twoJobs());
			const { unmount } = render(<App />);
			// @and [features/board-search.feature] I have searched the board for "Acme"
			await userEvent.type(searchBox(), "Acme");

			// @when [features/board-search.feature] I close and reopen the app
			unmount();
			render(<App />);

			// @then [features/board-search.feature#search-not-kept] the search is empty
			expect(searchBox()).toHaveValue("");
			// @and [features/board-search.feature#search-not-kept] I see the card for "Product Designer" at "Globex"
			expect(shown("Product Designer at Globex")).toBeInTheDocument();
		});

		// @scenario [features/board-search.feature#add-job-clears-search] Adding a job clears the search
		it("clears the search when a job is added", async () => {
			// @given [features/board-search.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
			mockBoard(board({ saved: [card(1, "Acme")] }));
			render(<App />);
			// @and [features/board-search.feature] I have searched the board for "Acme"
			await userEvent.type(searchBox(), "Acme");

			// @when [features/board-search.feature] I add a job at "Globex" to the "Saved" column
			await userEvent.click(screen.getByRole("button", { name: "Add a job to Saved" }));
			await userEvent.type(field(/^Company/), "Globex");
			await userEvent.type(field(/^Role/), "Product Designer");
			await userEvent.click(
				within(screen.getByRole("dialog")).getByRole("button", { name: "Add job" }),
			);

			// @then [features/board-search.feature#add-job-clears-search] the search is empty
			expect(searchBox()).toHaveValue("");
			// @and [features/board-search.feature#add-job-clears-search] I see the card for "Software Engineer" at "Acme"
			expect(shown("Software Engineer at Acme")).toBeInTheDocument();
			// @and [features/board-search.feature#add-job-clears-search] I see the card for the job at "Globex"
			expect(shown("Product Designer at Globex")).toBeInTheDocument();
		});
	});

	describe("pages", () => {
		const many = (n: number, company = "Company") =>
			Array.from({ length: n }, (_, i) => card(i + 1, `${company} ${i + 1}`));
		const next = () => userEvent.click(screen.getByRole("button", { name: "Next page of Saved" }));
		const pageOf = (text: string) => within(column("Saved")).getByText(text);

		// @scenario [features/board-pagination.feature#new-card-shows-first-page] Adding a job shows the first page of its column
		it("shows the first page of a column a job is added to", async () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			mockBoard(board({ saved: many(25) }));
			render(<App />);
			// @and [features/board-pagination.feature] the "Saved" column is on page 2
			await next();
			expect(pageOf("Page 2 of 3")).toBeInTheDocument();

			// @when [features/board-pagination.feature] I add a job at "Acme" to the "Saved" column
			await userEvent.click(screen.getByRole("button", { name: "Add a job to Saved" }));
			await userEvent.type(field(/^Company/), "Acme");
			await userEvent.type(field(/^Role/), "Software Engineer");
			await userEvent.click(
				within(screen.getByRole("dialog")).getByRole("button", { name: "Add job" }),
			);

			// @then [features/board-pagination.feature#new-card-shows-first-page] the "Saved" column is on page 1
			expect(pageOf("Page 1 of 3")).toBeInTheDocument();
			// @and [features/board-pagination.feature] the card for "Acme" is first in the "Saved" column
			expect(within(column("Saved")).getAllByRole("article")[0]).toHaveAccessibleName(
				"Software Engineer at Acme",
			);
		});

		// @scenario [features/board-pagination.feature#search-resets-pages] Changing the search returns every column to its first page
		it("returns to the first page when the search changes", async () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			mockBoard(board({ saved: many(25, "Acme") }));
			render(<App />);
			// @and [features/board-pagination.feature] the "Saved" column is on page 3
			await next();
			await next();
			expect(pageOf("Page 3 of 3")).toBeInTheDocument();

			// @when [features/board-pagination.feature] I search the board for "Acme"
			await userEvent.type(screen.getByRole("searchbox", { name: "Search jobs" }), "Acme");

			// @then [features/board-pagination.feature#search-resets-pages] the "Saved" column is on page 1
			expect(pageOf("Page 1 of 3")).toBeInTheDocument();
		});

		// @scenario [features/board-pagination.feature#pages-reset-on-reopen] Every column starts on its first page after reopening the app
		it("starts every column on its first page when the app reopens", async () => {
			// @given [features/board-pagination.feature] the "Saved" column holds 25 jobs
			mockBoard(board({ saved: many(25) }));
			const { unmount } = render(<App />);
			// @and [features/board-pagination.feature] the "Saved" column is on page 2
			await next();
			expect(pageOf("Page 2 of 3")).toBeInTheDocument();

			// @when [features/board-pagination.feature] I close and reopen the app
			unmount();
			render(<App />);

			// @then [features/board-pagination.feature#pages-reset-on-reopen] the "Saved" column is on page 1
			expect(pageOf("Page 1 of 3")).toBeInTheDocument();
		});
	});

	describe("next action filter", () => {
		const filterBox = () => screen.getByRole("combobox", { name: "Show" });
		const filterTo = (label: string) => userEvent.selectOptions(filterBox(), label);
		const shown = (company: string) =>
			screen.queryByRole("article", { name: new RegExp(` at ${company}$`) });
		// The backend marks which next actions are overdue (internal/jobs/next_action_test.go);
		// these cards come marked as it would mark them.
		const due = (company: string, expectedDate: string, mark: string, id: number) =>
			card(id, company, { nextAction: { text: "Follow up", expectedDate, due: mark } });
		const threeJobs = () =>
			board({
				applied: [
					due("Acme", "2026-10-10", "overdue", 1),
					due("Globex", "2026-10-20", "", 2),
					card(3, "Initech"),
				],
			});

		// @scenario [features/job-next-action.feature#filter-overdue] Show only overdue jobs
		it("shows only overdue jobs", async () => {
			// @given [features/job-next-action.feature] I have a job at "Acme" with a next action expected on 2026-10-10
			// @and [features/job-next-action.feature] a job at "Globex" with a next action expected on 2026-10-20
			// @and [features/job-next-action.feature] a job at "Initech" with no next action
			mockBoard(threeJobs());
			render(<App />);

			// @when [features/job-next-action.feature] I filter the board to overdue jobs
			await filterTo("Overdue");

			// @then [features/job-next-action.feature#filter-overdue] only the card for "Acme" is shown
			expect(screen.getAllByRole("article")).toHaveLength(1);
			expect(shown("Acme")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-not-overdue] Show only jobs that aren't overdue
		it("shows only jobs that aren't overdue", async () => {
			// @given [features/job-next-action.feature] I have a job at "Acme" with a next action expected on 2026-10-10
			// @and [features/job-next-action.feature] a job at "Globex" with a next action expected on 2026-10-20
			// @and [features/job-next-action.feature] a job at "Initech" with no next action
			mockBoard(threeJobs());
			render(<App />);

			// @when [features/job-next-action.feature] I filter the board to jobs that aren't overdue
			await filterTo("Not overdue");

			// @then [features/job-next-action.feature] the cards for "Globex" and "Initech" are shown
			expect(shown("Globex")).toBeInTheDocument();
			expect(shown("Initech")).toBeInTheDocument();
			// @and [features/job-next-action.feature] the card for "Acme" is not shown
			expect(shown("Acme")).not.toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-due-today-not-overdue] A job due today isn't overdue yet
		it("doesn't count a job due today as overdue", async () => {
			// @given [features/job-next-action.feature] I have a job at "Acme" with a next action expected on 2026-10-14
			mockBoard(
				board({
					applied: [
						due("Acme", "2026-10-14", "today", 1),
						due("Globex", "2026-10-01", "overdue", 2),
					],
				}),
			);
			render(<App />);

			// @when [features/job-next-action.feature] I filter the board to overdue jobs
			await filterTo("Overdue");

			// @then [features/job-next-action.feature] the card for "Acme" is not shown
			expect(shown("Acme")).not.toBeInTheDocument();
			expect(shown("Globex")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-closed-never-overdue] A closed job is never overdue
		it("doesn't count a closed job as overdue", async () => {
			// @given [features/job-next-action.feature] I have a job at "Acme" in "Closed" with a next action expected on 2026-10-10
			mockBoard(
				board({
					closed: [due("Acme", "2026-10-10", "", 1)],
					applied: [due("Globex", "2026-10-01", "overdue", 2)],
				}),
			);
			render(<App />);

			// @when [features/job-next-action.feature] I filter the board to overdue jobs
			await filterTo("Overdue");

			// @then [features/job-next-action.feature] the card for "Acme" is not shown
			expect(shown("Acme")).not.toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-show-all] Show every job again
		it("shows every job again", async () => {
			// @given [features/job-next-action.feature] I have a job at "Acme" with a next action expected on 2026-10-10
			// @and [features/job-next-action.feature] a job at "Globex" with no next action
			mockBoard(board({ applied: [due("Acme", "2026-10-10", "overdue", 1), card(2, "Globex")] }));
			render(<App />);
			// @and [features/job-next-action.feature] I have filtered the board to overdue jobs
			await filterTo("Overdue");
			expect(shown("Globex")).not.toBeInTheDocument();

			// @when [features/job-next-action.feature] I filter the board to all jobs
			await filterTo("All jobs");

			// @then [features/job-next-action.feature] the cards for "Acme" and "Globex" are shown
			expect(shown("Acme")).toBeInTheDocument();
			expect(shown("Globex")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-keeps-columns] Filtered cards stay in their own columns and the counts show how many are shown
		it("keeps filtered cards in their columns and counts them", async () => {
			// @given [features/job-next-action.feature] I have 2 overdue jobs in "Applied" and 1 overdue job in "Interviewing"
			// @and [features/job-next-action.feature] 3 jobs in "Applied" that aren't overdue
			mockBoard(
				board({
					applied: [
						due("Acme", "2026-10-10", "overdue", 1),
						due("Hooli", "2026-10-14", "today", 4),
						due("Globex", "2026-10-13", "overdue", 2),
						due("Umbrella", "2026-10-30", "", 5),
						card(6, "Vandelay"),
					],
					interviewing: [due("Initech", "2026-10-01", "overdue", 3)],
				}),
			);
			render(<App />);

			// @when [features/job-next-action.feature] I filter the board to overdue jobs
			await filterTo("Overdue");

			// @then [features/job-next-action.feature] the "Applied" column shows 2 cards and a count of 2
			expect(within(column("Applied")).getAllByRole("article")).toHaveLength(2);
			expect(within(column("Applied")).getByText("2 jobs")).toBeInTheDocument();
			// @and [features/job-next-action.feature] the "Interviewing" column shows 1 card and a count of 1
			expect(within(column("Interviewing")).getAllByRole("article")).toHaveLength(1);
			expect(within(column("Interviewing")).getByText("1 job")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-with-search] The filter and the search both apply
		it("applies both the filter and the search", async () => {
			// @given [features/job-next-action.feature] I have an overdue job for "Software Engineer" at "Acme"
			// @and [features/job-next-action.feature] an overdue job for "Designer" at "Globex"
			// @and [features/job-next-action.feature] a job for "Software Engineer" at "Initech" that isn't overdue
			mockBoard(
				board({
					applied: [
						due("Acme", "2026-10-10", "overdue", 1),
						{ ...due("Globex", "2026-10-10", "overdue", 2), role: "Designer" },
						card(3, "Initech"),
					],
				}),
			);
			render(<App />);

			// @when [features/job-next-action.feature] I filter the board to overdue jobs
			await filterTo("Overdue");
			// @and [features/job-next-action.feature] I search for "Engineer"
			await userEvent.type(screen.getByRole("searchbox", { name: "Search jobs" }), "Engineer");

			// @then [features/job-next-action.feature#filter-with-search] only the card for "Acme" is shown
			expect(screen.getAllByRole("article")).toHaveLength(1);
			expect(shown("Acme")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-no-matches] Told when no jobs match the filter
		it("says when no jobs match the filter", async () => {
			// @given [features/job-next-action.feature] none of my jobs are overdue
			mockBoard(board({ applied: [due("Acme", "2026-10-20", "", 1), card(2, "Globex")] }));
			render(<App />);

			// @when [features/job-next-action.feature] I filter the board to overdue jobs
			await filterTo("Overdue");

			expect(screen.queryAllByRole("article")).toEqual([]);
			// @then [features/job-next-action.feature] I am told that no jobs match
			expect(screen.getByText('No jobs match the "Overdue" filter.')).toHaveAttribute(
				"role",
				"status",
			);
			expect(screen.queryByRole("button", { name: "Add your first job" })).not.toBeInTheDocument();
		});

		it("names the search and the filter when nothing matches both", async () => {
			mockBoard(board({ applied: [due("Acme", "2026-10-10", "overdue", 1)] }));
			render(<App />);
			await filterTo("Overdue");
			await userEvent.type(screen.getByRole("searchbox", { name: "Search jobs" }), "Globex");
			expect(
				screen.getByText('No jobs match "Globex" with the "Overdue" filter.'),
			).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-resets-pages] Changing the filter returns every column to its first page
		it("returns to the first page when the filter changes", async () => {
			// @given [features/job-next-action.feature] the "Applied" column has 3 pages of cards
			mockBoard(
				board({ applied: Array.from({ length: 25 }, (_, i) => card(i + 1, `Company ${i + 1}`)) }),
			);
			render(<App />);
			// @and [features/job-next-action.feature] I am on its second page
			await userEvent.click(screen.getByRole("button", { name: "Next page of Applied" }));
			expect(within(column("Applied")).getByText("Page 2 of 3")).toBeInTheDocument();

			// @when [features/job-next-action.feature] I filter the board to jobs that aren't overdue
			await filterTo("Not overdue");

			// @then [features/job-next-action.feature] the "Applied" column shows its first page
			expect(within(column("Applied")).getByText("Page 1 of 3")).toBeInTheDocument();
			expect(shown("Company 1")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#add-job-clears-filter] Adding a job shows every job again
		it("shows every job again when a job is added", async () => {
			mockBoard(
				board({ applied: [due("Globex", "2026-10-10", "overdue", 1), card(2, "Initech")] }),
			);
			render(<App />);
			// @given [features/job-next-action.feature] I have filtered the board to overdue jobs
			await filterTo("Overdue");
			expect(shown("Initech")).not.toBeInTheDocument();

			// @when [features/job-next-action.feature] I add a job for "Software Engineer" at "Acme" to the "Saved" column
			await userEvent.click(screen.getByRole("button", { name: "Add a job to Saved" }));
			await userEvent.type(field(/^Company/), "Acme");
			await userEvent.type(field(/^Role/), "Software Engineer");
			await userEvent.click(
				within(screen.getByRole("dialog")).getByRole("button", { name: "Add job" }),
			);

			// @then [features/job-next-action.feature#add-job-clears-filter] the board shows all jobs
			expect(filterBox()).toHaveValue("all");
			expect(shown("Initech")).toBeInTheDocument();
			// @and [features/job-next-action.feature] the card for "Acme" is shown
			expect(acmeIn("Saved")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#filter-not-kept] The filter is cleared after reopening the app
		it("starts with every job shown when the app reopens", async () => {
			mockBoard(
				board({ applied: [due("Globex", "2026-10-10", "overdue", 1), card(2, "Initech")] }),
			);
			const { unmount } = render(<App />);
			// @given [features/job-next-action.feature] I have filtered the board to overdue jobs
			await filterTo("Overdue");

			// @when [features/job-next-action.feature#filter-not-kept] I close and reopen the app
			unmount();
			render(<App />);

			// @then [features/job-next-action.feature#filter-not-kept] the board shows all jobs
			expect(filterBox()).toHaveValue("all");
			expect(shown("Globex")).toBeInTheDocument();
			expect(shown("Initech")).toBeInTheDocument();
		});
	});

	describe("document library", () => {
		const boardShown = () => screen.queryByRole("heading", { name: "Job search" });
		const libraryShown = () => screen.queryByRole("heading", { name: "Document library" });

		// @scenario [features/document-library.feature#open-library] Go from the board to the document library
		it("goes from the board to the document library", async () => {
			// @given [features/document-library.feature] I open the job search board
			mockBoard(board({ saved: [card(1, "Acme")] }));
			render(<App />);
			expect(boardShown()).toBeInTheDocument();

			// @when [features/document-library.feature] I go to the document library
			await userEvent.click(screen.getByRole("button", { name: "Document library" }));

			// @then [features/document-library.feature] I see the document library
			expect(libraryShown()).toBeInTheDocument();
			expect(screen.getByRole("listitem", { name: "resume-base-2026.docx" })).toBeInTheDocument();
			expect(boardShown()).not.toBeInTheDocument();
			expect(screen.queryByRole("region", { name: "Saved" })).not.toBeInTheDocument();
		});

		// @scenario [features/document-library.feature#back-to-board] Go back to the board from the document library
		it("goes back to the board from the document library", async () => {
			mockBoard(board({ saved: [card(1, "Acme")] }));
			render(<App />);
			// @given [features/document-library.feature] I am in the document library
			await userEvent.click(screen.getByRole("button", { name: "Document library" }));

			// @when [features/document-library.feature] I go back to the board
			await userEvent.click(screen.getByRole("button", { name: "← Back to board" }));

			// @then [features/document-library.feature] I see the job search board
			expect(boardShown()).toBeInTheDocument();
			expect(acmeIn("Saved")).toBeInTheDocument();
			expect(libraryShown()).not.toBeInTheDocument();
		});

		it("keeps the board's search after visiting the library", async () => {
			mockBoard(board({ saved: [card(1, "Acme"), card(2, "Globex")] }));
			render(<App />);
			await userEvent.type(screen.getByRole("searchbox"), "Globex");

			await userEvent.click(screen.getByRole("button", { name: "Document library" }));
			await userEvent.click(screen.getByRole("button", { name: "← Back to board" }));

			expect(screen.getByRole("searchbox")).toHaveValue("Globex");
			expect(acmeIn("Saved")).not.toBeInTheDocument();
		});
	});
});
