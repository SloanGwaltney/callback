// Builders for the data the hooks return, for component and hook tests.
import type { Board, Card, Job } from "../hooks/types";

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
