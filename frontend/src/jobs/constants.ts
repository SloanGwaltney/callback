// Values the backend uses for statuses, closing reasons, document kinds, interest ratings, and
// contact relationships, with their labels. They mirror the constants in internal/jobs/jobs.go and
// internal/jobs/contacts.go; keep them in step.

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

// How a contact can relate to a job, in the order they're offered, matching jobs.Relationship.
export const RELATIONSHIPS = [
	{ value: "recruiter", label: "Recruiter" },
	{ value: "hiring_manager", label: "Hiring manager" },
	{ value: "referrer", label: "Referrer" },
	{ value: "interviewer", label: "Interviewer" },
	{ value: "other", label: "Other" },
];

// relationshipLabel is how a relationship is shown, or the value itself if it isn't known.
export function relationshipLabel(value: string): string {
	return RELATIONSHIPS.find((r) => r.value === value)?.label ?? value;
}

// The most stars an interest rating can have, matching jobs.MaxInterest. The fewest is 1.
export const MAX_INTEREST = 5;

export const INTEREST_QUESTION = "How interested are you in this job?";

// starsLabel names a rating for screen readers: "1 star", "4 stars".
export function starsLabel(n: number): string {
	return n === 1 ? "1 star" : `${n} stars`;
}

// today returns the local date as YYYY-MM-DD, the format of dates applied.
export function today(now: Date = new Date()): string {
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}
