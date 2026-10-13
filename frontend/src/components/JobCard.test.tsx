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

	describe("next action", () => {
		const next = (expectedDate: string, due = "", text = "Waiting for a call back") => ({
			text,
			expectedDate,
			due,
		});
		const shown = () => screen.getByRole("article");

		// @scenario [features/job-next-action.feature#card-shows-next-action] The card shows the next action and its expected date
		it("shows the next action and its expected date", () => {
			// The backend decides today and sends no due mark for a date still to come
			// (TestCardShowsNextAction).
			// @when [features/job-next-action.feature] I open the job search board
			render(
				<JobCard card={card(1, "Acme", { nextAction: next("2026-10-14") })} onOpen={() => {}} />,
			);

			// @then [features/job-next-action.feature] the card shows "Waiting for a call back" expected on 2026-10-14
			expect(shown()).toHaveTextContent("Next action: Waiting for a call back");
			expect(shown()).toHaveTextContent("Expected 2026-10-14");
		});

		// @scenario [features/job-next-action.feature#card-next-action-no-date] The card shows a next action with no expected date
		it("shows a next action with no expected date", () => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Send a thank-you email" and no expected date
			const nextAction = next("", "", "Send a thank-you email");
			// @when [features/job-next-action.feature] I open the job search board
			render(<JobCard card={card(1, "Acme", { nextAction })} onOpen={() => {}} />);

			// @then [features/job-next-action.feature] the card shows "Send a thank-you email" with no expected date
			expect(shown()).toHaveTextContent("Send a thank-you email");
			expect(shown()).not.toHaveTextContent("Expected");
			// @and [features/job-next-action.feature] the next action is not marked as due or overdue
			expect(shown()).not.toHaveTextContent(/Due today|Overdue/);
		});

		// @scenario [features/job-next-action.feature#card-next-action-due-today] A next action expected today is marked as due
		it("marks a next action expected today as due", () => {
			// The backend marks it due today (TestCardMarksNextActionDue).
			// @when [features/job-next-action.feature] I open the job search board
			render(
				<JobCard
					card={card(1, "Acme", { nextAction: next("2026-10-14", "today") })}
					onOpen={() => {}}
				/>,
			);

			// @then [features/job-next-action.feature] the card marks the next action as due today
			expect(screen.getByText("Due today")).toBeInTheDocument();
			expect(shown()).not.toHaveTextContent("Overdue");
		});

		// @scenario [features/job-next-action.feature#card-next-action-overdue] A next action whose expected date has passed is marked as overdue
		it("marks a next action whose expected date has passed as overdue", () => {
			// The backend marks it overdue (TestCardMarksNextActionDue).
			// @when [features/job-next-action.feature] I open the job search board
			render(
				<JobCard
					card={card(1, "Acme", { nextAction: next("2026-10-14", "overdue") })}
					onOpen={() => {}}
				/>,
			);

			// @then [features/job-next-action.feature] the card marks the next action as overdue
			expect(screen.getByText("Overdue")).toBeInTheDocument();
			expect(shown()).not.toHaveTextContent("Due today");
		});

		// @scenario [features/job-next-action.feature#card-next-action-upcoming] A next action expected later isn't marked
		// @scenario [features/job-next-action.feature#closed-card-no-due-marks] A closed job's next action is never marked as due or overdue
		it("doesn't mark a next action the backend didn't mark", () => {
			// A date still to come, and a closed job's next action, come with no due mark
			// (TestCardMarksNextActionDue, TestClosedCardNoDueMarks).
			// @when [features/job-next-action.feature] I open the job search board
			render(
				<JobCard
					card={card(1, "Acme", { nextAction: next("2026-10-10", "", "Ask for feedback") })}
					onOpen={() => {}}
				/>,
			);

			// @then [features/job-next-action.feature] the card shows "Ask for feedback"
			expect(shown()).toHaveTextContent("Ask for feedback");
			// @then [features/job-next-action.feature] the next action is not marked as due or overdue
			expect(shown()).not.toHaveTextContent(/Due today|Overdue/);
		});

		// @scenario [features/job-next-action.feature#card-no-next-action] A job with no next action shows none on its card
		// @scenario [features/job-next-action.feature#clear-next-action] Clear the next action once it's done
		it.each([undefined, null])("shows no next action when it's %s", (nextAction) => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
			// @when [features/job-next-action.feature] I open the job search board
			render(<JobCard card={card(1, "Acme", { nextAction })} onOpen={() => {}} />);

			// @then [features/job-next-action.feature] the card doesn't show a next action
			// @and [features/job-next-action.feature#clear-next-action] the card no longer shows a next action
			expect(shown()).not.toHaveTextContent("Next action");
			expect(shown()).not.toHaveTextContent("Expected");
		});
	});

	// @scenario [features/job-status-history.feature#card-shows-days-in-status] The card shows how many days the job has been in its current status
	it.each([
		{ days: 0, shown: "Today" },
		{ days: 1, shown: "1 day" },
		{ days: 8, shown: "8 days" },
	])("shows $shown in status", ({ days, shown }) => {
		// The backend counts the days from the date the job moved and today (TestCardShowsDaysInStatus).
		render(<JobCard card={card(1, "Acme", { daysInStatus: days })} onOpen={() => {}} />);

		// @then [features/job-status-history.feature] the card shows "<shown>"
		expect(screen.getByText(shown)).toBeInTheDocument();
		expect(screen.getByText(/^In status/)).toHaveTextContent(`In status ${shown}`);
	});

	it("shows no days in status for a job with no history", () => {
		render(<JobCard card={card(1, "Acme", { daysInStatus: null })} onOpen={() => {}} />);
		expect(screen.queryByText(/^In status/)).not.toBeInTheDocument();
	});
});
