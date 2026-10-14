import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Interview, InterviewDetails } from "../hooks/types";
import { useInterviews } from "../hooks/useInterviews";
import { InterviewList } from "./InterviewList";

vi.mock("../hooks/useInterviews", () => ({ useInterviews: vi.fn() }));

// TODAY is the date the mocked backend decides which interviews have passed against.
const TODAY = "2026-10-09";

// backend stands in for the jobs service: every job's interviews, listed in date order with each
// marked past or not, like the real one.
let backend: { interviews: Interview[]; nextId: number };
let api: ReturnType<typeof mockInterviews>;

function mockInterviews() {
	return {
		add: vi.fn(async (jobId: number, details: InterviewDetails): Promise<Interview> => {
			const interview = { id: backend.nextId++, jobId, ...details, past: details.date < TODAY };
			backend.interviews.push(interview);
			return interview;
		}),
		update: vi.fn(async (id: number, details: InterviewDetails): Promise<Interview> => {
			backend.interviews = backend.interviews.map((i) =>
				i.id === id ? { ...i, ...details, past: details.date < TODAY } : i,
			);
			return backend.interviews.find((i) => i.id === id) as Interview;
		}),
		remove: vi.fn(async (id: number) => {
			backend.interviews = backend.interviews.filter((i) => i.id !== id);
		}),
	};
}

beforeEach(() => {
	backend = { interviews: [], nextId: 1 };
	api = mockInterviews();
	vi.mocked(useInterviews).mockReturnValue(api);
});

// listed is the backend's order: by date, then time with untimed last, then as added.
function listed(jobId: number): Interview[] {
	const untimed = (i: Interview) => (i.time === "" ? 1 : 0);
	return backend.interviews
		.filter((i) => i.jobId === jobId)
		.sort(
			(a, b) =>
				a.date.localeCompare(b.date) ||
				untimed(a) - untimed(b) ||
				a.time.localeCompare(b.time) ||
				a.id - b.id,
		);
}

// List shows a job's interviews and reloads them from the backend when they change, like
// JobDetails does.
function List({ jobId }: { jobId: number }) {
	const [shown, setShown] = useState(() => listed(jobId));
	return (
		<InterviewList jobId={jobId} interviews={shown} onChanged={() => setShown(listed(jobId))} />
	);
}

function renderList() {
	render(<List jobId={1} />);
	return screen.getByRole("region", { name: "Interviews" });
}

// seed puts an interview straight into the backend, before the list is rendered.
function seed(kind: string, date: string, fields: Partial<InterviewDetails> = {}) {
	backend.interviews.push({
		id: backend.nextId++,
		jobId: 1,
		date,
		time: "",
		kind,
		notes: "",
		past: date < TODAY,
		...fields,
	});
}

type Entry = { date?: string; time?: string; kind?: string; notes?: string };

// fill enters details into the open interview dialog, replacing what's there, and returns it.
async function fill(entry: Entry) {
	const dialog = screen.getByRole("dialog", { name: /^(Add an interview|Edit the )/ });
	// Date and time inputs take their value whole, as a date or time picker sets it.
	if (entry.date !== undefined) {
		fireEvent.change(within(dialog).getByLabelText(/^Date/), { target: { value: entry.date } });
	}
	if (entry.time !== undefined) {
		fireEvent.change(within(dialog).getByLabelText("Time"), { target: { value: entry.time } });
	}
	if (entry.kind !== undefined) {
		await userEvent.selectOptions(
			within(dialog).getByLabelText(/^Kind/),
			entry.kind === "" ? "Choose…" : entry.kind,
		);
	}
	if (entry.notes !== undefined) {
		const notes = within(dialog).getByLabelText("Notes");
		await userEvent.clear(notes);
		if (entry.notes !== "") {
			await userEvent.type(notes, entry.notes);
		}
	}
	return dialog;
}

async function addInterview(scope: HTMLElement, entry: Entry) {
	await userEvent.click(within(scope).getByRole("button", { name: "Add interview" }));
	const dialog = await fill(entry);
	await userEvent.click(within(dialog).getByRole("button", { name: "Save interview" }));
	return dialog;
}

const interview = (scope: HTMLElement, name: string) =>
	within(scope).queryByRole("listitem", { name });

// shown lists the interviews as the list names them, top to bottom.
const shown = (scope: HTMLElement) =>
	within(scope)
		.queryAllByRole("listitem")
		.map((item) => item.getAttribute("aria-label"));

describe("InterviewList", () => {
	// @scenario [#add-interview]
	it("adds an interview with all its details", async () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
		const list = renderList();

		// @when [features/job-interviews.feature] I add an interview with:
		await addInterview(list, {
			date: "2026-10-14",
			time: "14:00",
			kind: "Technical",
			notes: "Live coding with two engineers.",
		});

		expect(api.add).toHaveBeenCalledWith(1, {
			date: "2026-10-14",
			time: "14:00",
			kind: "technical",
			notes: "Live coding with two engineers.",
		});
		// @then [features/job-interviews.feature] the job's interviews show a "Technical" interview on 2026-10-14 at 2:00 PM
		const item = interview(list, "Technical interview on 2026-10-14 at 2:00 PM") as HTMLElement;
		expect(item).toBeInTheDocument();
		expect(item).toHaveTextContent("2026-10-14 at 2:00 PM");
		// @and [features/job-interviews.feature] the interview shows the notes "Live coding with two engineers."
		expect(item).toHaveTextContent("Live coding with two engineers.");
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#add-interview-minimum]
	it("adds an interview with only a date and a kind", async () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
		const list = renderList();

		// @when [features/job-interviews.feature] I add a "Phone screen" interview on 2026-10-14 with no time and no notes
		await addInterview(list, { date: "2026-10-14", kind: "Phone screen" });

		expect(api.add).toHaveBeenCalledWith(1, {
			date: "2026-10-14",
			time: "",
			kind: "phone_screen",
			notes: "",
		});
		// @then [features/job-interviews.feature] the job's interviews show a "Phone screen" interview on 2026-10-14
		expect(shown(list)).toEqual(["Phone screen interview on 2026-10-14"]);
	});

	// @scenario [#interview-kinds]
	it.each(["Phone screen", "Technical", "Behavioral", "Panel", "Onsite", "Other"])(
		"adds a %s interview",
		async (kind) => {
			// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
			const list = renderList();

			// @when [features/job-interviews.feature] I add a "<kind>" interview on 2026-10-14
			await addInterview(list, { date: "2026-10-14", kind });

			// @then [features/job-interviews.feature] the job's interviews show a "<kind>" interview on 2026-10-14
			expect(shown(list)).toEqual([`${kind} interview on 2026-10-14`]);
		},
	);

	// @scenario [#interview-missing-required]
	it.each([
		["", "Technical", "date"],
		["2026-10-14", "", "kind"],
	])("doesn't add an interview with date %j and kind %j", async (date, kind, missing) => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
		const list = renderList();
		api.add.mockRejectedValueOnce(`${missing} is required`);

		// @when [features/job-interviews.feature] I try to add an interview with date "<date>" and kind "<kind>"
		const dialog = await addInterview(list, { date, kind });

		// @then [features/job-interviews.feature] the interview is not added
		expect(shown(list)).toEqual([]);
		expect(dialog).toBeInTheDocument();
		// @and [features/job-interviews.feature] I am told that the <missing> is required
		expect(within(dialog).getByRole("alert")).toHaveTextContent(`${missing} is required`);
	});

	// @scenario [#interview-past-date]
	it("adds an interview that already happened", async () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
		const list = renderList();

		// @when [features/job-interviews.feature] I add a "Phone screen" interview on 2026-10-02
		await addInterview(list, { date: "2026-10-02", kind: "Phone screen" });

		// @then [features/job-interviews.feature] the job's interviews show a "Phone screen" interview on 2026-10-02
		expect(shown(list)).toEqual(["Phone screen interview on 2026-10-02"]);
	});

	// @scenario [#view-interviews]
	it("shows a job's interviews", () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
		seed("phone_screen", "2026-10-02");
		// @and [features/job-interviews.feature] a "Technical" interview on 2026-10-14
		seed("technical", "2026-10-14");

		// @when [features/job-interviews.feature] I open the card
		const list = renderList();

		// @then [features/job-interviews.feature] I see both interviews
		expect(shown(list)).toEqual([
			"Phone screen interview on 2026-10-02",
			"Technical interview on 2026-10-14",
		]);
	});

	// @scenario [#interviews-same-day-by-time]
	it("shows the times of interviews on the same day in the order the backend lists them", () => {
		// @and [features/job-interviews.feature] it has a "Panel" interview on 2026-10-14 with no time
		seed("panel", "2026-10-14");
		// @and [features/job-interviews.feature] a "Behavioral" interview on 2026-10-14 at 3:00 PM
		seed("behavioral", "2026-10-14", { time: "15:00" });
		// @and [features/job-interviews.feature] a "Technical" interview on 2026-10-14 at 10:00 AM
		seed("technical", "2026-10-14", { time: "10:00" });

		// @when [features/job-interviews.feature] I open the card
		const list = renderList();

		// @then [features/job-interviews.feature] the interviews are listed as "Technical", "Behavioral", and "Panel" in that order
		// The order comes from the backend (TestListInterviewsForJobOrdersSameDayByTime); the list
		// keeps it and shows each time on a 12-hour clock.
		expect(shown(list)).toEqual([
			"Technical interview on 2026-10-14 at 10:00 AM",
			"Behavioral interview on 2026-10-14 at 3:00 PM",
			"Panel interview on 2026-10-14",
		]);
	});

	// @scenario [#past-interviews-marked]
	it("marks the interviews that have already happened as past", () => {
		// The backend decides which interviews have passed (TestPastInterviewsMarked); the mock marks
		// the ones before TODAY.
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Phone screen" interview on 2026-10-02
		seed("phone_screen", "2026-10-02");
		// @and [features/job-interviews.feature] a "Technical" interview on 2026-10-14
		seed("technical", "2026-10-14");

		// @when [features/job-interviews.feature] I open the card
		const list = renderList();

		// @then [features/job-interviews.feature] the "Phone screen" interview is shown as past
		const phone = interview(list, "Phone screen interview on 2026-10-02") as HTMLElement;
		expect(phone).toHaveTextContent("Past");
		expect(phone).not.toHaveTextContent("Upcoming");
		// @and [features/job-interviews.feature] the "Technical" interview is shown as upcoming
		const tech = interview(list, "Technical interview on 2026-10-14") as HTMLElement;
		expect(tech).toHaveTextContent("Upcoming");
		expect(tech).not.toHaveTextContent("Past");
	});

	// @scenario [#no-interviews]
	it("says when a job has no interviews", () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with no interviews
		// @when [features/job-interviews.feature] I open the card
		const list = renderList();

		// @then [features/job-interviews.feature] I am told the job has no interviews yet
		expect(list).toHaveTextContent("No interviews yet.");
		expect(within(list).queryByRole("list")).not.toBeInTheDocument();
	});

	// @scenario [#edit-interview]
	it("edits an interview", async () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
		seed("technical", "2026-10-14");
		const list = renderList();
		await userEvent.click(
			within(list).getByRole("button", { name: "Edit Technical interview on 2026-10-14" }),
		);
		const dialog = screen.getByRole("dialog", { name: "Edit the Technical interview" });
		expect(within(dialog).getByLabelText(/^Date/)).toHaveValue("2026-10-14");

		// @when [features/job-interviews.feature] I change the interview's date to 2026-10-16
		await fill({ date: "2026-10-16" });
		// @and [features/job-interviews.feature] I write in its notes "Rescheduled by the recruiter."
		await fill({ notes: "Rescheduled by the recruiter." });
		await userEvent.click(within(dialog).getByRole("button", { name: "Save interview" }));

		expect(api.update).toHaveBeenCalledWith(1, {
			date: "2026-10-16",
			time: "",
			kind: "technical",
			notes: "Rescheduled by the recruiter.",
		});
		// @then [features/job-interviews.feature] the job's interviews show a "Technical" interview on 2026-10-16
		expect(shown(list)).toEqual(["Technical interview on 2026-10-16"]);
		// @and [features/job-interviews.feature] the interview shows the notes "Rescheduled by the recruiter."
		expect(interview(list, "Technical interview on 2026-10-16")).toHaveTextContent(
			"Rescheduled by the recruiter.",
		);
	});

	it("keeps the edit dialog open and says why when an edit is refused", async () => {
		seed("technical", "2026-10-14");
		const list = renderList();
		api.update.mockRejectedValueOnce("date is required");
		await userEvent.click(
			within(list).getByRole("button", { name: "Edit Technical interview on 2026-10-14" }),
		);
		const dialog = await fill({ date: "" });
		await userEvent.click(within(dialog).getByRole("button", { name: "Save interview" }));

		expect(within(dialog).getByRole("alert")).toHaveTextContent("date is required");
		expect(shown(list)).toEqual(["Technical interview on 2026-10-14"]);
	});

	// @scenario [#remove-interview]
	it("removes an interview after confirming", async () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
		seed("technical", "2026-10-14");
		seed("onsite", "2026-10-20");
		const list = renderList();

		// @when [features/job-interviews.feature] I remove the interview
		await userEvent.click(
			within(list).getByRole("button", { name: "Remove Technical interview on 2026-10-14" }),
		);
		const confirm = screen.getByRole("dialog", { name: "Remove this interview?" });
		expect(confirm).toHaveTextContent("The Technical interview on 2026-10-14 will be removed");
		// @and [features/job-interviews.feature] I confirm the removal
		await userEvent.click(within(confirm).getByRole("button", { name: "Remove" }));

		expect(api.remove).toHaveBeenCalledWith(1);
		// @then [features/job-interviews.feature] the job no longer shows the "Technical" interview
		expect(shown(list)).toEqual(["Onsite interview on 2026-10-20"]);
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#cancel-remove-interview]
	it("keeps an interview when removing it is cancelled", async () => {
		// @given [features/job-interviews.feature] I have a job for "Software Engineer" at "Acme" with a "Technical" interview on 2026-10-14
		seed("technical", "2026-10-14");
		const list = renderList();

		// @when [features/job-interviews.feature] I remove the interview
		await userEvent.click(
			within(list).getByRole("button", { name: "Remove Technical interview on 2026-10-14" }),
		);
		const confirm = screen.getByRole("dialog", { name: "Remove this interview?" });
		// @but [features/job-interviews.feature] I cancel the removal
		await userEvent.click(within(confirm).getByRole("button", { name: "Cancel" }));

		expect(api.remove).not.toHaveBeenCalled();
		// @then [features/job-interviews.feature] the job still shows the "Technical" interview on 2026-10-14
		expect(shown(list)).toEqual(["Technical interview on 2026-10-14"]);
	});

	it("says why an interview couldn't be removed", async () => {
		seed("technical", "2026-10-14");
		const list = renderList();
		api.remove.mockRejectedValueOnce("interview not found");
		await userEvent.click(
			within(list).getByRole("button", { name: "Remove Technical interview on 2026-10-14" }),
		);
		const confirm = screen.getByRole("dialog", { name: "Remove this interview?" });
		await userEvent.click(within(confirm).getByRole("button", { name: "Remove" }));

		expect(within(confirm).getByRole("alert")).toHaveTextContent("interview not found");
	});

	it("doesn't add an interview when adding is cancelled", async () => {
		const list = renderList();
		await userEvent.click(within(list).getByRole("button", { name: "Add interview" }));
		await fill({ date: "2026-10-14", kind: "Technical" });
		await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

		expect(api.add).not.toHaveBeenCalled();
		expect(shown(list)).toEqual([]);
	});
});
