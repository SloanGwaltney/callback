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

	// @scenario [features/job-interest.feature#card-shows-interest] The card shows the interest rating as stars
	// @scenario [features/job-interest.feature#interest-persists] The interest rating is the same after reopening the app
	it("shows the interest rating as stars", () => {
		// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
		// @when [features/job-interest.feature] I open the job search board
		render(<JobCard card={card(1, "Acme", { interest: 4 })} onOpen={() => {}} />);

		// @then [features/job-interest.feature] the card shows 4 of 5 stars
		expect(screen.getByRole("img", { name: "4 of 5 stars" })).toHaveTextContent("★★★★☆");
	});

	// @scenario [features/job-interest.feature#card-unrated] An unrated card shows no stars
	it.each([undefined, null])("shows no stars when the interest rating is %s", (interest) => {
		// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with no interest rating
		// @when [features/job-interest.feature] I open the job search board
		render(<JobCard card={card(1, "Acme", { interest })} onOpen={() => {}} />);

		// @then [features/job-interest.feature] the card doesn't show an interest rating
		expect(screen.queryByRole("img")).not.toBeInTheDocument();
		expect(screen.queryByText(/[★☆]/)).not.toBeInTheDocument();
	});

	// @scenario [features/job-interviews.feature#card-shows-next-interview] The card shows the date of the next upcoming interview
	it("shows the next upcoming interview", () => {
		// The backend decides today and picks the next interview (TestCardShowsNextInterview), and
		// sends only it.
		const nextInterview = { date: "2026-10-14", time: "", kind: "technical", today: false };

		// @when [features/job-interviews.feature] I open the job search board
		render(<JobCard card={card(1, "Acme", { nextInterview })} onOpen={() => {}} />);

		// @then [features/job-interviews.feature] the card shows a "Technical" interview on 2026-10-14
		expect(screen.getByRole("article")).toHaveTextContent("Technical interview on 2026-10-14");
	});

	// @scenario [features/job-interviews.feature#card-interview-today] An interview today counts as upcoming
	it("shows an interview today as today", () => {
		// The backend decides the interview is today (TestCardShowsInterviewToday).
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
		const nextInterview = { date: "2026-10-14", time: "", kind: "technical", today: true };

		// @when [features/job-interviews.feature] I open the job search board
		render(<JobCard card={card(1, "Acme", { nextInterview })} onOpen={() => {}} />);

		// @then [features/job-interviews.feature] the card shows a "Technical" interview today
		const shown = screen.getByRole("article");
		expect(shown).toHaveTextContent("Technical interview today");
		expect(shown).not.toHaveTextContent("2026-10-14");
	});

	it("shows the next interview's time on a 12-hour clock", () => {
		const nextInterview = { date: "2026-10-14", time: "14:00", kind: "onsite", today: false };
		render(<JobCard card={card(1, "Acme", { nextInterview })} onOpen={() => {}} />);
		expect(screen.getByRole("article")).toHaveTextContent(
			"Onsite interview on 2026-10-14 at 2:00 PM",
		);
	});

	// @scenario [features/job-interviews.feature#card-no-upcoming-interview] The card shows no interview when every interview has passed
	it.each([undefined, null])(
		"shows no interview when the next interview is %s",
		(nextInterview) => {
			// The backend sends no next interview when every interview has passed (TestCardNoUpcomingInterview).
			// @when [features/job-interviews.feature] I open the job search board
			render(<JobCard card={card(1, "Acme", { nextInterview })} onOpen={() => {}} />);

			// @then [features/job-interviews.feature] the card doesn't show an interview
			expect(screen.queryByText(/interview/i)).not.toBeInTheDocument();
		},
	);

	it("opens when its title is clicked", async () => {
		const onOpen = vi.fn();
		render(<JobCard card={card(1, "Acme")} onOpen={onOpen} />);
		await userEvent.click(screen.getByRole("button", { name: /Software Engineer/ }));
		expect(onOpen).toHaveBeenCalled();
	});
});
