import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { card } from "../test/fixtures";
import { JobCard } from "./JobCard";

describe("JobCard", () => {
	it("shows the company, the role, and the location", () => {
		render(<JobCard card={card(1, "Acme", { location: "Remote (US)" })} onOpen={() => {}} />);
		const shown = screen.getByRole("article", { name: "Software Engineer at Acme" });
		expect(shown).toHaveTextContent("Software Engineer");
		expect(shown).toHaveTextContent("Acme");
		expect(shown).toHaveTextContent("Remote (US)");
	});

	it("shows the date applied, or nothing when there's none", () => {
		const { rerender } = render(
			<JobCard card={card(1, "Acme", { dateApplied: "2026-10-01" })} onOpen={() => {}} />,
		);
		expect(screen.getByText("Applied 2026-10-01")).toBeInTheDocument();
		rerender(<JobCard card={card(1, "Acme")} onOpen={() => {}} />);
		expect(screen.queryByText(/^Applied/)).not.toBeInTheDocument();
	});

	it("shows the closing reason of a closed job", () => {
		render(<JobCard card={card(1, "Acme", { closingReasonText: "Rejected" })} onOpen={() => {}} />);
		expect(screen.getByText("Closed: Rejected")).toBeInTheDocument();
	});

	// @scenario [features/job-documents.feature#card-shows-document-count] The card shows that documents are attached
	it("shows how many documents are attached", () => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme" with 2 attached documents
		// @when [features/job-documents.feature] I look at the card on the board
		render(<JobCard card={card(1, "Acme", { documentCount: 2 })} onOpen={() => {}} />);

		// @then [features/job-documents.feature] the card shows that it has 2 documents
		expect(screen.getByText("2 documents")).toBeInTheDocument();
	});

	it("says 1 document, and nothing when there are none", () => {
		const { rerender } = render(
			<JobCard card={card(1, "Acme", { documentCount: 1 })} onOpen={() => {}} />,
		);
		expect(screen.getByText("1 document")).toBeInTheDocument();
		rerender(<JobCard card={card(1, "Acme")} onOpen={() => {}} />);
		expect(screen.queryByText(/document/)).not.toBeInTheDocument();
	});

	it("opens when its title is clicked", async () => {
		const onOpen = vi.fn();
		render(<JobCard card={card(1, "Acme")} onOpen={onOpen} />);
		await userEvent.click(screen.getByRole("button", { name: /Software Engineer/ }));
		expect(onOpen).toHaveBeenCalled();
	});
});
