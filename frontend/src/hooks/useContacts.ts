import { useCallback } from "react";
import {
	AddContact,
	OpenContactLink,
	RemoveContact,
	UpdateContact,
} from "../../wailsjs/go/main/App";
import type { Contact, ContactDetails } from "./types";

// useContacts adds, edits, removes, and opens the profile links of a job's contacts. Each call
// rejects with the backend's message, for the caller to show.
export function useContacts() {
	const add = useCallback(
		(jobId: number, details: ContactDetails): Promise<Contact> => AddContact(jobId, details),
		[],
	);
	const update = useCallback(
		(id: number, details: ContactDetails): Promise<Contact> => UpdateContact(id, details),
		[],
	);
	const remove = useCallback((id: number) => RemoveContact(id), []);
	const openLink = useCallback((id: number) => OpenContactLink(id), []);

	return { add, update, remove, openLink };
}
