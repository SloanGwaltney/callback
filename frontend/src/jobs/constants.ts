// Values the backend uses for statuses, closing reasons, and document kinds, with their labels.
// They mirror the constants in internal/jobs/jobs.go; keep the two in step.

export const STATUS_SAVED = "saved";
export const STATUS_APPLIED = "applied";
export const STATUS_CLOSED = "closed";

// Column titles by status, matching jobs.Status.Label. The board gets them with its columns;
// these are for showing a single job's status.
export const STATUS_LABELS: Record<string, string> = {
	saved: "Saved",
	applied: "Applied",
	interviewing: "Interviewing",
	offer: "Offer",
	closed: "Closed",
};

export const REASON_OTHER = "other";

// The closing reasons in the order they're offered, matching jobs.ClosingReason.
export const CLOSING_REASONS = [
	{ value: "rejected", label: "Rejected" },
	{ value: "withdrew", label: "Withdrew" },
	{ value: "no_response", label: "No response" },
	{ value: "declined_offer", label: "Declined offer" },
	{ value: "position_filled", label: "Position filled" },
	{ value: REASON_OTHER, label: "Other" },
];

// The kinds a document can be, matching jobs.DocumentKind.
export const DOCUMENT_KINDS = [
	{ value: "resume", label: "Resume" },
	{ value: "cover_letter", label: "Cover letter" },
	{ value: "other", label: "Other" },
];

// today returns the local date as YYYY-MM-DD, the format of dates applied.
export function today(now: Date = new Date()): string {
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}
