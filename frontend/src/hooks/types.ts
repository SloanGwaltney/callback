// The data the hooks hand to components, matching the Go structs in internal/jobs. They're the
// Wails-generated models (frontend/wailsjs/go/models.ts) without the classes' helper methods, so
// components and tests can build them as plain objects and never import the bindings
// (docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md).
import type { backups, jobs } from "../../wailsjs/go/models";

// Go sends a job with no interest rating as null, which the generated models leave out of the type.
type Rating = { interest?: number | null };

export type NextInterview = jobs.NextInterview;
export type NextAction = jobs.NextAction;
// Go sends a job with no next action as null.
type HasNextAction = { nextAction?: NextAction | null };
// Go sends a card with no upcoming interview, or no status history to count days from, as null.
export type Card = Omit<
	jobs.Card,
	"interest" | "nextInterview" | "nextAction" | "daysInStatus" | "convertValues"
> &
	Rating &
	HasNextAction & { nextInterview?: NextInterview | null; daysInStatus?: number | null };
export type Column = Omit<jobs.Column, "convertValues" | "cards"> & { cards: Card[] };
export type Board = { columns: Column[] };
export type JobDocument = jobs.Document;
export type Contact = jobs.Contact;
export type ContactDetails = jobs.ContactDetails;
export type Interview = jobs.Interview;
export type InterviewDetails = jobs.InterviewDetails;
export type StatusChange = jobs.StatusChange;
export type Job = Omit<
	jobs.Job,
	| "convertValues"
	| "documents"
	| "contacts"
	| "interviews"
	| "statusHistory"
	| "interest"
	| "nextAction"
> &
	Rating &
	HasNextAction & {
		documents: JobDocument[];
		contacts: Contact[];
		interviews: Interview[];
		statusHistory: StatusChange[];
	};
export type Details = jobs.Details;
export type LibraryDocument = jobs.LibraryDocument;
export type Move = jobs.Move;

// BackupKind is who made a backup: Callback on the first run of a day, the user's agent, or the
// app just before a restore (backups.Kind in internal/backups).
export type BackupKind = "daily" | "agent" | "app";
// BackupRef names the backup an app backup was made before restoring. For an app backup, name is
// when that one was made, like 20261014-093000, rather than its name.
export type BackupRef = { kind: BackupKind; name: string };
// Go leaves out a daily backup's keptUntil, and every backup's restored but an app backup's.
export type Backup = Omit<backups.Backup, "convertValues" | "kind" | "restored"> & {
	kind: BackupKind;
	restored?: BackupRef | null;
};
