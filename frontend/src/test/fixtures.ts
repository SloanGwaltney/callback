// Builders for the data the hooks return, for component and hook tests.
import { vi } from "vitest";
import type { Board, Card, Job } from "../hooks/types";
import type { useJob } from "../hooks/useJob";

const columns = [
	{ status: "saved", label: "Saved" },
	{ status: "applied", label: "Applied" },
	{ status: "interviewing", label: "Interviewing" },
	{ status: "offer", label: "Offer" },
	{ status: "closed", label: "Closed" },
];

export function card(id: number, company: string, fields: Partial<Card> = {}): Card {
	return {
		id,
		company,
		role: "Software Engineer",
		location: "",
		dateApplied: "",
		closingReason: "",
		closingReasonText: "",
		documentCount: 0,
		...fields,
	};
}

// board lays out cards by status, with every column present in the board's order.
export function board(cards: Partial<Record<string, Card[]>> = {}): Board {
	return {
		columns: columns.map(({ status, label }) => {
			const list = cards[status] ?? [];
			return { status, label, count: list.length, cards: list };
		}),
	};
}

export function job(fields: Partial<Job> = {}): Job {
	return {
		id: 1,
		company: "Acme",
		role: "Software Engineer",
		description: "",
		link: "",
		salary: "",
		location: "",
		notes: "",
		dateApplied: "",
		status: "saved",
		closingReason: "",
		closingReasonOther: "",
		closingReasonText: "",
		documents: [],
		contacts: [],
		interviews: [],
		...fields,
	};
}

// fakeUseJob is what a mocked useJob returns: the job, loaded, with every change a vi.fn() that
// resolves. Pass fields to replace any of them. A hook method added to useJob gets its default
// here, so the tests that mock useJob keep compiling.
export function fakeUseJob(
	fields: Partial<ReturnType<typeof useJob>> = {},
): ReturnType<typeof useJob> {
	return {
		job: job(),
		error: "",
		reload: vi.fn().mockResolvedValue(undefined),
		update: vi.fn().mockResolvedValue(job()),
		setClosingReason: vi.fn().mockResolvedValue(undefined),
		setNextAction: vi.fn().mockResolvedValue(undefined),
		clearNextAction: vi.fn().mockResolvedValue(undefined),
		openLink: vi.fn().mockResolvedValue(undefined),
		...fields,
	};
}
