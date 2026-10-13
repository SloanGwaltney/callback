import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Details, Job } from "../hooks/types";
import { useContacts } from "../hooks/useContacts";
import { useDocuments } from "../hooks/useDocuments";
import { useInterviews } from "../hooks/useInterviews";
import { useJob } from "../hooks/useJob";
import { fakeUseJob, job } from "../test/fixtures";
import { JobDetails } from "./JobDetails";

vi.mock("../hooks/useJob", () => ({ useJob: vi.fn() }));
vi.mock("../hooks/useDocuments", () => ({ useDocuments: vi.fn() }));
vi.mock("../hooks/useContacts", () => ({ useContacts: vi.fn() }));
vi.mock("../hooks/useInterviews", () => ({ useInterviews: vi.fn() }));

// The mocked useJob keeps the job in state, like the real one: saving details replaces it with
// what the backend returns, here the saved details themselves.
const backend = {
	update: vi.fn<(details: Details) => Promise<void>>(),
	setClosingReason: vi.fn<(reason: string, other: string) => Promise<Job | undefined>>(),
	openLink: vi.fn<() => Promise<void>>(),
	setNextAction: vi.fn<(text: string, expectedDate: string) => Promise<void>>(),
	clearNextAction: vi.fn<() => Promise<void>>(),
};

function mockJob(start: Job) {
	vi.mocked(useJob).mockImplementation(() => {
		const [current, setCurrent] = useState(start);
		return {
			...fakeUseJob(),
			job: current,
			error: "",
			reload: async () => {},
			update: async (details: Details) => {
				await backend.update(details);
				const updated = { ...current, ...details };
				setCurrent(updated);
				return updated;
			},
			setClosingReason: async (reason: string, other: string) => {
				const reloaded = await backend.setClosingReason(reason, other);
				if (reloaded) {
					setCurrent(reloaded);
				}
			},
			// Setting or clearing the next action replaces the job with it changed, as the backend
			// returns it. Due marks are the backend's, so a set next action has none here.
			setNextAction: async (text: string, expectedDate: string) => {
				await backend.setNextAction(text, expectedDate);
				setCurrent({ ...current, nextAction: { text, expectedDate, due: "" } });
			},
			clearNextAction: async () => {
				await backend.clearNextAction();
				setCurrent({ ...current, nextAction: null });
			},
			openLink: backend.openLink,
		};
	});
}

beforeEach(() => {
	backend.update.mockResolvedValue();
	backend.setClosingReason.mockResolvedValue(undefined);
	backend.openLink.mockResolvedValue();
	backend.setNextAction.mockResolvedValue();
	backend.clearNextAction.mockResolvedValue();
	vi.mocked(useContacts).mockReturnValue({
		add: vi.fn(),
		update: vi.fn(),
		remove: vi.fn(),
		openLink: vi.fn(),
	});
	vi.mocked(useInterviews).mockReturnValue({ add: vi.fn(), update: vi.fn(), remove: vi.fn() });
	vi.mocked(useDocuments).mockReturnValue({
		attach: vi.fn(),
		setKind: vi.fn(),
		remove: vi.fn(),
		open: vi.fn(),
	});
});

function renderDetails(start: Job) {
	mockJob(start);
	const onClose = vi.fn();
	const onChanged = vi.fn();
	const onDelete = vi.fn<(id: number) => Promise<void>>().mockResolvedValue();
	render(
		<JobDetails jobId={start.id} onClose={onClose} onChanged={onChanged} onDelete={onDelete} />,
	);
	return { onClose, onChanged, onDelete };
}

const field = (label: RegExp) => screen.getByLabelText(label);
const save = () => userEvent.click(screen.getByRole("button", { name: "Save" }));
const setDate = (date: string) =>
	fireEvent.change(field(/^Date applied/), { target: { value: date } });
const rate = (stars: number) =>
	userEvent.click(screen.getByRole("radio", { name: stars === 1 ? "1 star" : `${stars} stars` }));
const savedDetails = () => backend.update.mock.calls.at(-1)?.[0];

const filledIn = job({
	company: "Acme",
	role: "Software Engineer",
	description: "Build and maintain internal tools.",
	link: "https://acme.example.com/jobs/123",
	salary: "$120k - $140k",
	location: "Remote (US)",
	notes: "Referred by Dana from the meetup.",
	dateApplied: "2026-10-01",
	status: "applied",
	documents: [{ id: 5, jobId: 1, kind: "resume", fileName: "resume-2026.pdf" }],
});

describe("JobDetails", () => {
	// @scenario [features/job-cards.feature#view-job-details] View a job's full details
	it("shows a job's full details", () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with every detail filled in
		// @when [features/job-cards.feature] I open the card
		renderDetails(filledIn);

		// @then [features/job-cards.feature] I see the company, role, job description, link, salary, location, notes, and date applied
		expect(field(/^Company/)).toHaveValue("Acme");
		expect(field(/^Role/)).toHaveValue("Software Engineer");
		expect(field(/^Job description/)).toHaveValue("Build and maintain internal tools.");
		expect(field(/^Link/)).toHaveValue("https://acme.example.com/jobs/123");
		expect(field(/^Salary/)).toHaveValue("$120k - $140k");
		expect(field(/^Location/)).toHaveValue("Remote (US)");
		expect(field(/^Notes/)).toHaveValue("Referred by Dana from the meetup.");
		expect(field(/^Date applied/)).toHaveValue("2026-10-01");
		// @and [features/job-cards.feature] I see the documents attached to the job
		expect(screen.getByRole("button", { name: "resume-2026.pdf" })).toBeInTheDocument();
	});

	// @scenario [features/job-cards.feature#edit-job-details] Edit a job's details
	it("edits a job's details", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
		const { onChanged } = renderDetails(job());

		// @when [features/job-cards.feature] I change the role to "Senior Software Engineer"
		await userEvent.clear(field(/^Role/));
		await userEvent.type(field(/^Role/), "Senior Software Engineer");
		// @and [features/job-cards.feature] I change the job description
		await userEvent.type(field(/^Job description/), "Lead the platform team.");
		await save();

		// @then [features/job-cards.feature] the card shows the role "Senior Software Engineer"
		expect(savedDetails()?.role).toBe("Senior Software Engineer");
		expect(onChanged).toHaveBeenCalled();
		expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent(
			"Senior Software Engineer at Acme",
		);
		// @and [features/job-cards.feature] the job shows the new job description
		expect(field(/^Job description/)).toHaveValue("Lead the platform team.");
		expect(screen.getByRole("status")).toHaveTextContent("Saved.");
	});

	// @scenario [features/job-cards.feature#salary-free-text] Salary is written however the posting gives it
	it.each(["$120,000", "$120k - $140k", "$65/hr", "Not listed"])(
		"keeps the salary %s as written",
		async (salary) => {
			// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
			renderDetails(job());

			// @when [features/job-cards.feature] I set the salary to "<salary>"
			await userEvent.type(field(/^Salary/), salary);
			await save();

			// @then [features/job-cards.feature] the job shows the salary "<salary>"
			expect(savedDetails()?.salary).toBe(salary);
			expect(field(/^Salary/)).toHaveValue(salary);
		},
	);

	// @scenario [features/job-cards.feature#add-notes] Keep notes on a job
	it("keeps notes on a job", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
		renderDetails(job());

		// @when [features/job-cards.feature] I write in the notes "Recruiter call went well, follow up Friday."
		await userEvent.type(field(/^Notes/), "Recruiter call went well, follow up Friday.");
		await save();

		// @then [features/job-cards.feature] the job's notes show "Recruiter call went well, follow up Friday."
		expect(savedDetails()?.notes).toBe("Recruiter call went well, follow up Friday.");
		expect(field(/^Notes/)).toHaveValue("Recruiter call went well, follow up Friday.");
	});

	// @scenario [features/job-cards.feature#edit-notes] Add to existing notes
	it("adds a line to existing notes", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with the notes "Referred by Dana."
		renderDetails(job({ notes: "Referred by Dana." }));

		// @when [features/job-cards.feature] I add "Phone screen booked for 10/14." on a new line in the notes
		await userEvent.type(field(/^Notes/), "{Enter}Phone screen booked for 10/14.");
		await save();

		// @then [features/job-cards.feature] the job's notes show both lines
		expect(savedDetails()?.notes).toBe("Referred by Dana.\nPhone screen booked for 10/14.");
	});

	// @scenario [features/job-cards.feature#open-job-link] Open a job's posting link
	it("opens the job's posting link", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with the link "https://acme.example.com/jobs/123"
		renderDetails(job({ link: "https://acme.example.com/jobs/123" }));

		// @when [features/job-cards.feature] I open the job's link
		await userEvent.click(screen.getByRole("button", { name: "Open link" }));

		// @then [features/job-cards.feature] the posting opens in my web browser
		expect(backend.openLink).toHaveBeenCalled();
	});

	it("can't open a link the job doesn't have", () => {
		renderDetails(job());
		expect(screen.getByRole("button", { name: "Open link" })).toBeDisabled();
	});

	it("shows why the link couldn't be opened", async () => {
		backend.openLink.mockRejectedValue("the link must be a web address");
		renderDetails(job({ link: "https://acme.example.com" }));
		await userEvent.click(screen.getByRole("button", { name: "Open link" }));
		expect(screen.getByRole("alert")).toHaveTextContent("the link must be a web address");
	});

	// @scenario [features/job-cards.feature#invalid-job-link] A job link must be a web address
	it("doesn't save a link that isn't a web address", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
		const { onChanged } = renderDetails(job());
		backend.update.mockRejectedValue(
			"the link must be a web address, like https://example.com/jobs/123",
		);

		// @when [features/job-cards.feature] I try to set the link to "not a url"
		await userEvent.type(field(/^Link/), "not a url");
		await save();

		// @then [features/job-cards.feature] the link is not saved
		expect(onChanged).not.toHaveBeenCalled();
		expect(screen.queryByRole("status")).not.toBeInTheDocument();
		// @and [features/job-cards.feature] I am told the link must be a web address
		expect(screen.getByRole("alert")).toHaveTextContent("the link must be a web address");
	});

	// @scenario [features/job-cards.feature#set-date-applied] Set the date applied
	it("sets the date applied", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" in "Applied"
		const { onChanged } = renderDetails(job({ status: "applied" }));

		// @when [features/job-cards.feature] I set the date applied to 2026-10-01
		setDate("2026-10-01");
		await save();

		// @then [features/job-cards.feature] the job's date applied is 2026-10-01
		expect(savedDetails()?.dateApplied).toBe("2026-10-01");
		// @and [features/job-cards.feature] the card shows the date applied
		expect(onChanged).toHaveBeenCalled();
	});

	// @scenario [features/job-cards.feature#date-applied-any-status] The date applied can be set in any status
	it.each(["saved", "interviewing", "offer", "closed"])(
		"sets the date applied in %s",
		async (status) => {
			// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			renderDetails(job({ status }));

			// @when [features/job-cards.feature] I set the date applied to 2026-10-01
			setDate("2026-10-01");
			await save();

			// @then [features/job-cards.feature] the job's date applied is 2026-10-01
			expect(savedDetails()?.dateApplied).toBe("2026-10-01");
		},
	);

	// @scenario [features/job-cards.feature#clear-date-applied] Clear the date applied
	it("clears the date applied", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with a date applied of 2026-10-01
		renderDetails(job({ dateApplied: "2026-10-01" }));

		// @when [features/job-cards.feature] I clear the date applied
		await userEvent.click(screen.getByRole("button", { name: "Clear" }));
		await save();

		// @then [features/job-cards.feature#clear-date-applied] the job has no date applied
		expect(savedDetails()?.dateApplied).toBe("");
		expect(field(/^Date applied/)).toHaveValue("");
		expect(screen.queryByRole("button", { name: "Clear" })).not.toBeInTheDocument();
	});

	// @scenario [features/job-cards.feature#future-date-applied] The date applied can't be in the future
	it("doesn't save a date applied in the future", async () => {
		// @given [features/job-cards.feature] today is 2026-10-09
		// @and [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
		const { onChanged } = renderDetails(job());
		backend.update.mockRejectedValue("the date applied can't be in the future");

		// @when [features/job-cards.feature] I try to set the date applied to 2026-10-20
		setDate("2026-10-20");
		await save();

		// @then [features/job-cards.feature] the date applied is not saved
		expect(onChanged).not.toHaveBeenCalled();
		// @and [features/job-cards.feature] I am told the date applied can't be in the future
		expect(screen.getByRole("alert")).toHaveTextContent("the date applied can't be in the future");
	});

	// @scenario [features/job-cards.feature#delete-job] Delete a job after confirming
	it("deletes a job after confirming", async () => {
		// @given [features/job-cards.feature#delete-job] I have a job for "Software Engineer" at "Acme"
		const { onDelete } = renderDetails(job({ id: 4 }));

		// @when [features/job-cards.feature] I delete the job
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		const confirm = screen.getByRole("dialog", { name: "Delete Software Engineer at Acme?" });
		expect(confirm).toHaveTextContent("its attached documents");
		expect(onDelete).not.toHaveBeenCalled();
		// @and [features/job-cards.feature] I confirm the deletion
		await userEvent.click(within(confirm).getByRole("button", { name: "Delete" }));

		// @then [features/job-cards.feature] the card for "Software Engineer" at "Acme" no longer appears on the board
		// @and [features/job-cards.feature] its attached documents are removed too
		expect(onDelete).toHaveBeenCalledWith(4);
	});

	// @scenario [features/job-contacts.feature#contacts-deleted-with-job] Deleting a job deletes its contacts
	it("warns that deleting a job deletes its contacts", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
		const dana = {
			id: 7,
			jobId: 4,
			name: "Dana Smith",
			relationship: "recruiter",
			email: "",
			phone: "",
			profileLink: "",
			notes: "",
		};
		const { onDelete } = renderDetails(job({ id: 4, contacts: [dana] }));
		expect(screen.getByRole("listitem", { name: "Dana Smith" })).toBeInTheDocument();

		// @when [features/job-contacts.feature] I delete the job
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		const confirm = screen.getByRole("dialog", { name: "Delete Software Engineer at Acme?" });
		// @and [features/job-contacts.feature] I confirm the deletion
		await userEvent.click(within(confirm).getByRole("button", { name: "Delete" }));

		// @then [features/job-contacts.feature] its contacts are removed too
		// The backend deletes them with the job (TestDeleteJobDeletesItsContacts); the confirmation
		// says so.
		expect(confirm).toHaveTextContent("its contacts");
		expect(onDelete).toHaveBeenCalledWith(4);
	});

	// @scenario [features/job-interviews.feature#interviews-deleted-with-job] Deleting a job deletes its interviews
	it("warns that deleting a job deletes its interviews", async () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
		const technical = {
			id: 7,
			jobId: 4,
			date: "2026-10-14",
			time: "",
			kind: "technical",
			notes: "",
			past: false,
		};
		const { onDelete } = renderDetails(job({ id: 4, interviews: [technical] }));
		expect(
			screen.getByRole("listitem", { name: "Technical interview on 2026-10-14" }),
		).toBeInTheDocument();

		// @when [features/job-interviews.feature] I delete the job
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		const confirm = screen.getByRole("dialog", { name: "Delete Software Engineer at Acme?" });
		// @and [features/job-interviews.feature] I confirm the deletion
		await userEvent.click(within(confirm).getByRole("button", { name: "Delete" }));

		// @then [features/job-interviews.feature] its interviews are removed too
		// The backend deletes them with the job (TestDeleteJobDeletesItsInterviews); the confirmation
		// says so.
		expect(confirm).toHaveTextContent("its interviews");
		expect(onDelete).toHaveBeenCalledWith(4);
	});

	// @scenario [features/job-interviews.feature#interview-any-status] Interviews can be added to a job in any status
	it.each(["saved", "applied", "interviewing", "offer", "closed"])(
		"offers to add an interview to a job in %s",
		(status) => {
			// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			renderDetails(job({ status }));

			const interviews = screen.getByRole("region", { name: "Interviews" });
			expect(within(interviews).getByRole("button", { name: "Add interview" })).toBeEnabled();
		},
	);

	// @scenario [features/document-library.feature#library-not-on-jobs] Library documents aren't attached to any job
	it("shows no attached documents on a job, whatever the library holds", () => {
		// @given [features/document-library.feature] I have a job for "Software Engineer" at "Acme" with no attached documents
		// The library's documents are never on a job (TestLibraryDocumentsAreNotOnJobs), so the job
		// comes from the backend with none.
		const acme = job({ documents: [] });

		// @when [features/document-library.feature] I open the card
		renderDetails(acme);

		// @then [features/document-library.feature] the job has no attached documents
		const documents = screen.getByRole("region", { name: "Documents" });
		expect(within(documents).getByText("No documents attached.")).toBeInTheDocument();
		expect(within(documents).queryByRole("listitem")).not.toBeInTheDocument();
	});

	// @scenario [features/document-library.feature#delete-job-keeps-library] Deleting a job doesn't touch the library
	it("deletes a job without asking about the document library", async () => {
		const { onDelete } = renderDetails(job({ id: 4 }));

		// @when [features/document-library.feature] I delete the job
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		const confirm = screen.getByRole("dialog", { name: "Delete Software Engineer at Acme?" });
		// @and [features/document-library.feature] I confirm the deletion
		await userEvent.click(within(confirm).getByRole("button", { name: "Delete" }));

		// The backend leaves the library alone (TestDeleteJobKeepsTheLibrary).
		expect(onDelete).toHaveBeenCalledWith(4);
	});

	it("shows why a job couldn't be deleted", async () => {
		const { onDelete } = renderDetails(job());
		onDelete.mockRejectedValue("job not found");
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		await userEvent.click(screen.getByRole("button", { name: "Delete" }));
		expect(screen.getByRole("alert")).toHaveTextContent("job not found");
	});

	// @scenario [features/job-cards.feature#cancel-delete-job] Cancel deleting a job
	it("keeps the job when deleting is cancelled", async () => {
		// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
		const { onDelete } = renderDetails(job());

		// @when [features/job-cards.feature] I delete the job
		await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
		// @but [features/job-cards.feature] I cancel the deletion
		await userEvent.click(
			within(screen.getByRole("dialog", { name: /^Delete/ })).getByRole("button", {
				name: "Cancel",
			}),
		);

		// @then [features/job-cards.feature] the card for "Software Engineer" at "Acme" is still on the board
		expect(onDelete).not.toHaveBeenCalled();
		expect(screen.queryByRole("dialog", { name: /^Delete/ })).not.toBeInTheDocument();
	});

	// @scenario [features/job-board.feature#change-closing-reason] Change the closing reason of a closed job
	it("changes the closing reason of a closed job", async () => {
		// @given [features/job-board.feature#change-closing-reason] I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
		const closed = job({
			status: "closed",
			closingReason: "no_response",
			closingReasonText: "No response",
		});
		const { onChanged } = renderDetails(closed);
		expect(screen.getByText(/closed: No response/)).toBeInTheDocument();
		backend.setClosingReason.mockResolvedValue({
			...closed,
			closingReason: "rejected",
			closingReasonText: "Rejected",
		});

		// @when [features/job-board.feature] I change the closing reason to "Rejected"
		await userEvent.click(screen.getByRole("button", { name: "Change reason" }));
		expect(screen.getByRole("radio", { name: "No response" })).toBeChecked();
		await userEvent.click(screen.getByRole("radio", { name: "Rejected" }));
		await userEvent.click(screen.getByRole("button", { name: "Save reason" }));

		expect(backend.setClosingReason).toHaveBeenCalledWith("rejected", "");
		// @then [features/job-board.feature] the card shows the closing reason "Rejected"
		expect(onChanged).toHaveBeenCalled();
		expect(screen.getByText(/closed: Rejected/)).toBeInTheDocument();
	});

	it("offers no closing reason for a job that isn't closed", () => {
		renderDetails(job({ status: "offer" }));
		expect(screen.getByText("Offer")).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Change reason" })).not.toBeInTheDocument();
	});

	// @scenario [features/job-interest.feature#rate-interest] Rate how interested I am in a job
	it.each([1, 2, 3, 4, 5])("rates the job %i stars", async (stars) => {
		// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with no interest rating
		const { onChanged } = renderDetails(job());
		expect(screen.queryByRole("radio", { checked: true })).not.toBeInTheDocument();

		// @when [features/job-interest.feature] I answer "How interested are you in this job?" with <stars> stars
		await rate(stars);
		await save();

		// @then [features/job-interest.feature] the job's interest rating is <stars> stars
		expect(savedDetails()?.interest).toBe(stars);
		expect(onChanged).toHaveBeenCalled();
	});

	// @scenario [features/job-interest.feature#interest-any-status] A job in any status can be rated
	it.each(["saved", "applied", "interviewing", "offer", "closed"])(
		"rates a job in %s",
		async (status) => {
			// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			renderDetails(job({ status }));

			// @when [features/job-interest.feature] I answer "How interested are you in this job?" with 4 stars
			await rate(4);
			await save();

			// @then [features/job-interest.feature] the job's interest rating is 4 stars
			expect(savedDetails()?.interest).toBe(4);
		},
	);

	// @scenario [features/job-interest.feature#change-interest] Change my interest rating
	it("changes the interest rating", async () => {
		// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 2 stars
		renderDetails(job({ interest: 2 }));

		// @when [features/job-interest.feature] I change the interest rating to 5 stars
		await rate(5);
		await save();

		// @then [features/job-interest.feature] the job's interest rating is 5 stars
		expect(savedDetails()?.interest).toBe(5);
		expect(screen.getByRole("radio", { name: "5 stars" })).toBeChecked();
	});

	// @scenario [features/job-interest.feature#clear-interest] Clear my interest rating
	it("clears the interest rating", async () => {
		// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 3 stars
		renderDetails(job({ interest: 3 }));

		// @when [features/job-interest.feature] I clear the interest rating
		await userEvent.click(screen.getByRole("button", { name: "Clear interest rating" }));
		await save();

		// @then [features/job-interest.feature] the job has no interest rating
		expect(savedDetails()?.interest).toBeUndefined();
		expect(screen.queryByRole("radio", { checked: true })).not.toBeInTheDocument();
	});

	// @scenario [features/job-interest.feature#view-interest] See the interest rating when opening the card
	it("shows the interest rating", () => {
		// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
		// @when [features/job-interest.feature] I open the card
		renderDetails(job({ interest: 4 }));

		// @then [features/job-interest.feature] I see "How interested are you in this job?" answered with 4 stars
		const question = screen.getByRole("group", { name: "How interested are you in this job?" });
		expect(within(question).getByRole("radio", { name: "4 stars" })).toBeChecked();
	});

	it("starts unrated when the backend sends no rating", async () => {
		renderDetails(job({ interest: null }));
		expect(screen.queryByRole("radio", { checked: true })).not.toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Clear interest rating" })).not.toBeInTheDocument();
		await save();
		expect(savedDetails()?.interest).toBeUndefined();
	});

	it("keeps the interest rating when other details are saved", async () => {
		renderDetails(job({ interest: 4 }));
		await userEvent.type(field(/^Notes/), "Call back Friday.");
		await save();
		expect(savedDetails()?.interest).toBe(4);
	});

	it("shows why the interest rating wasn't saved", async () => {
		backend.update.mockRejectedValue("the interest rating must be from 1 to 5 stars");
		renderDetails(job({ interest: 3 }));
		await rate(2);
		await save();
		expect(screen.getByRole("alert")).toHaveTextContent(
			"the interest rating must be from 1 to 5 stars",
		);
	});

	describe("next action", () => {
		const panel = () => screen.getByRole("region", { name: "Next action" });
		const inPanel = () => within(panel());
		const waiting = { text: "Waiting for a call back", expectedDate: "2026-10-14", due: "" };
		// setNextAction opens the form, writes text and picks date, and saves.
		const setNextAction = async (text: string, date: string) => {
			const open = inPanel().queryByRole("button", { name: "Set next action" });
			await userEvent.click(open ?? inPanel().getByRole("button", { name: "Edit" }));
			const what = inPanel().getByLabelText(/^What's next/);
			await userEvent.clear(what);
			if (text !== "") {
				await userEvent.type(what, text);
			}
			fireEvent.change(inPanel().getByLabelText("Expected on"), { target: { value: date } });
			await userEvent.click(inPanel().getByRole("button", { name: "Save next action" }));
		};

		// @scenario [features/job-next-action.feature#set-next-action] Set a next action with an expected date
		it("sets a next action with an expected date", async () => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
			const { onChanged } = renderDetails(job({ status: "applied" }));
			expect(inPanel().getByText("No next action.")).toBeInTheDocument();

			// @when [features/job-next-action.feature] I set the next action to "Waiting for a call back from the recruiter" expected on 2026-10-14
			await setNextAction("Waiting for a call back from the recruiter", "2026-10-14");

			expect(backend.setNextAction).toHaveBeenCalledWith(
				"Waiting for a call back from the recruiter",
				"2026-10-14",
			);
			// @then [features/job-next-action.feature] the job's next action is "Waiting for a call back from the recruiter"
			expect(inPanel().getByText("Waiting for a call back from the recruiter")).toBeInTheDocument();
			// @and [features/job-next-action.feature#set-next-action] its expected date is 2026-10-14
			expect(inPanel().getByText("Expected 2026-10-14")).toBeInTheDocument();
			expect(onChanged).toHaveBeenCalled();
		});

		// @scenario [features/job-next-action.feature#set-next-action-no-date] Set a next action without an expected date
		it("sets a next action without an expected date", async () => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
			renderDetails(job({ status: "applied" }));

			// @when [features/job-next-action.feature] I set the next action to "Send a thank-you email" with no expected date
			await setNextAction("Send a thank-you email", "");

			expect(backend.setNextAction).toHaveBeenCalledWith("Send a thank-you email", "");
			// @then [features/job-next-action.feature] the job's next action is "Send a thank-you email"
			expect(inPanel().getByText("Send a thank-you email")).toBeInTheDocument();
			// @and [features/job-next-action.feature#set-next-action-no-date] it has no expected date
			expect(inPanel().getByText("No expected date")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#next-action-needs-text] A next action can't be saved without saying what it is
		it("says why a next action with nothing written isn't saved", async () => {
			// The backend refuses it (TestNextActionNeedsText); the form shows its message.
			backend.setNextAction.mockRejectedValueOnce("write what the next action is");
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with no next action
			renderDetails(job({ status: "applied" }));

			// @when [features/job-next-action.feature] I try to set a next action expected on 2026-10-14 with nothing but spaces written
			await setNextAction("   ", "2026-10-14");

			// @then [features/job-next-action.feature] the next action is not saved
			expect(inPanel().getByRole("button", { name: "Save next action" })).toBeInTheDocument();
			expect(inPanel().getByLabelText("Expected on")).toHaveValue("2026-10-14");
			// @and [features/job-next-action.feature] I am told to write what the next action is
			expect(inPanel().getByRole("alert")).toHaveTextContent("write what the next action is");
		});

		// @scenario [features/job-next-action.feature#next-action-past-date] The expected date can be in the past
		it("sets an expected date in the past", async () => {
			renderDetails(job({ status: "applied" }));

			// @when [features/job-next-action.feature] I set the next action to "Follow up on application" expected on 2026-10-05
			await setNextAction("Follow up on application", "2026-10-05");

			expect(backend.setNextAction).toHaveBeenCalledWith("Follow up on application", "2026-10-05");
			// @then [features/job-next-action.feature] the job's next action is "Follow up on application"
			expect(inPanel().getByText("Follow up on application")).toBeInTheDocument();
			// @and [features/job-next-action.feature] its expected date is 2026-10-05
			expect(inPanel().getByText("Expected 2026-10-05")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#next-action-any-status] A next action can be set on a job in any status
		it.each(["saved", "applied", "interviewing", "offer", "closed"])(
			"sets a next action on a job in %s",
			async (status) => {
				// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
				renderDetails(job({ status, closingReasonText: status === "closed" ? "Rejected" : "" }));

				// @when [features/job-next-action.feature] I set the next action to "Check the posting is still open" expected on 2026-10-14
				await setNextAction("Check the posting is still open", "2026-10-14");

				// @then [features/job-next-action.feature] the job's next action is "Check the posting is still open"
				expect(inPanel().getByText("Check the posting is still open")).toBeInTheDocument();
			},
		);

		// @scenario [features/job-next-action.feature#edit-next-action] Change the next action and its expected date
		it("changes the next action and its expected date", async () => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
			renderDetails(job({ status: "offer", nextAction: waiting }));
			await userEvent.click(inPanel().getByRole("button", { name: "Edit" }));
			expect(inPanel().getByLabelText(/^What's next/)).toHaveValue("Waiting for a call back");
			expect(inPanel().getByLabelText("Expected on")).toHaveValue("2026-10-14");
			await userEvent.click(inPanel().getByRole("button", { name: "Cancel" }));

			// @when [features/job-next-action.feature] I change the next action to "Expecting an email with the offer details" expected on 2026-10-16
			await setNextAction("Expecting an email with the offer details", "2026-10-16");

			// @then [features/job-next-action.feature] the job's next action is "Expecting an email with the offer details"
			expect(inPanel().getByText("Expecting an email with the offer details")).toBeInTheDocument();
			// @and [features/job-next-action.feature] its expected date is 2026-10-16
			expect(inPanel().getByText("Expected 2026-10-16")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#clear-next-action-date] Clear only the expected date
		it("clears only the expected date", async () => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
			renderDetails(job({ status: "applied", nextAction: waiting }));

			// @when [features/job-next-action.feature#clear-next-action-date] I clear the expected date
			await userEvent.click(inPanel().getByRole("button", { name: "Clear date" }));

			expect(backend.setNextAction).toHaveBeenCalledWith("Waiting for a call back", "");
			// @then [features/job-next-action.feature#clear-next-action-date] the job's next action is still "Waiting for a call back"
			expect(inPanel().getByText("Waiting for a call back")).toBeInTheDocument();
			// @and [features/job-next-action.feature#clear-next-action-date] it has no expected date
			expect(inPanel().getByText("No expected date")).toBeInTheDocument();
			expect(inPanel().queryByRole("button", { name: "Clear date" })).not.toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#clear-next-action] Clear the next action once it's done
		it("clears the next action once it's done", async () => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
			const { onChanged } = renderDetails(job({ status: "applied", nextAction: waiting }));

			// @when [features/job-next-action.feature#clear-next-action] I clear the next action
			await userEvent.click(inPanel().getByRole("button", { name: "Clear next action" }));

			expect(backend.clearNextAction).toHaveBeenCalled();
			// @then [features/job-next-action.feature#clear-next-action] the job has no next action
			expect(inPanel().getByText("No next action.")).toBeInTheDocument();
			// The board reloads, and the card shows none (JobCard.test.tsx).
			expect(onChanged).toHaveBeenCalled();
		});

		it("shows why the next action couldn't be cleared", async () => {
			backend.clearNextAction.mockRejectedValueOnce("job not found");
			renderDetails(job({ nextAction: waiting }));
			await userEvent.click(inPanel().getByRole("button", { name: "Clear next action" }));
			expect(inPanel().getByRole("alert")).toHaveTextContent("job not found");
			expect(inPanel().getByText("Waiting for a call back")).toBeInTheDocument();
		});

		// @scenario [features/job-next-action.feature#view-next-action] See the next action when opening the card
		it("shows the next action when the card is opened", () => {
			// @given [features/job-next-action.feature] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
			// @when [features/job-next-action.feature] I open the card
			renderDetails(job({ nextAction: waiting }));

			// @then [features/job-next-action.feature] I see the next action "Waiting for a call back" and its expected date 2026-10-14
			expect(inPanel().getByText("Waiting for a call back")).toBeInTheDocument();
			expect(inPanel().getByText("Expected 2026-10-14")).toBeInTheDocument();
		});

		it("shows the backend's due mark", () => {
			renderDetails(job({ nextAction: { ...waiting, due: "overdue" } }));
			expect(inPanel().getByText("Overdue")).toBeInTheDocument();
		});
	});

	describe("status history", () => {
		const history = [
			{ status: "saved", date: "2026-10-01", closingReason: "", closingReasonText: "" },
			{ status: "applied", date: "2026-10-09", closingReason: "", closingReasonText: "" },
		];

		// @scenario [features/job-status-history.feature#history-read-only] The status history can't be edited
		it("shows the status history with nothing to edit", () => {
			// @given [features/job-status-history.feature#history-read-only] I have a job for "Software Engineer" at "Acme" with a status history
			// @when [features/job-status-history.feature] I open the card
			renderDetails(job({ status: "applied", statusHistory: history }));

			// @then [features/job-status-history.feature] I can see the status history
			const region = screen.getByRole("region", { name: "Status history" });
			expect(within(region).getAllByRole("listitem")).toHaveLength(2);
			// @but [features/job-status-history.feature] I can't change, add, or remove its entries
			expect(within(region).queryAllByRole("button")).toHaveLength(0);
			expect(within(region).queryAllByRole("textbox")).toHaveLength(0);
			expect(within(region).queryAllByRole("combobox")).toHaveLength(0);
		});

		// @scenario [features/job-status-history.feature#history-deleted-with-job] Deleting a job deletes its status history
		it("warns that deleting a job deletes its status history", async () => {
			const { onDelete } = renderDetails(job({ id: 4, statusHistory: history }));

			// @when [features/job-status-history.feature] I delete the job
			await userEvent.click(screen.getByRole("button", { name: "Delete job" }));
			const confirm = screen.getByRole("dialog", { name: "Delete Software Engineer at Acme?" });
			// @and [features/job-status-history.feature] I confirm the deletion
			await userEvent.click(within(confirm).getByRole("button", { name: "Delete" }));

			// @then [features/job-status-history.feature] its status history is removed too
			// The backend deletes it with the job (TestDeleteJobDeletesItsStatusHistory); the
			// confirmation says so.
			expect(confirm).toHaveTextContent("its status history");
			expect(onDelete).toHaveBeenCalledWith(4);
		});
	});
});
