// The data the hooks hand to components, matching the Go structs in internal/jobs. They're the
// Wails-generated models (frontend/wailsjs/go/models.ts) without the classes' helper methods, so
// components and tests can build them as plain objects and never import the bindings
// (docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md).
import type { jobs } from "../../wailsjs/go/models";

// Go sends a job with no interest rating as null, which the generated models leave out of the type.
type Rating = { interest?: number | null };

export type Card = Omit<jobs.Card, "interest"> & Rating;
export type Column = Omit<jobs.Column, "convertValues" | "cards"> & { cards: Card[] };
export type Board = { columns: Column[] };
export type JobDocument = jobs.Document;
export type Contact = jobs.Contact;
export type ContactDetails = jobs.ContactDetails;
export type Job = Omit<jobs.Job, "convertValues" | "documents" | "contacts" | "interest"> &
	Rating & { documents: JobDocument[]; contacts: Contact[] };
export type Details = jobs.Details;
export type Move = jobs.Move;
