import { type FormEvent, useId, useState } from "react";
import type { Contact, ContactDetails } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { useContacts } from "../hooks/useContacts";
import { RELATIONSHIPS, relationshipLabel } from "../jobs/constants";
import { ConfirmDialog } from "./ConfirmDialog";
import styles from "./ContactList.module.css";
import { Dialog } from "./Dialog";
import { ErrorMessage } from "./ErrorMessage";

type Props = {
	jobId: number;
	// contacts are in the order they were added.
	contacts: Contact[];
	// onChanged runs after a contact is added, edited, or removed.
	onChanged: () => void;
};

const emptyContact: ContactDetails = {
	name: "",
	relationship: "",
	email: "",
	phone: "",
	profileLink: "",
	notes: "",
};

// ContactList shows the people on a job, and adds, edits, and removes them, asking before a
// removal. Each job keeps its own contacts, so a change here never reaches another job's.
// @scenario [features/job-contacts.feature#add-contact] Add a contact with all their details
// @scenario [features/job-contacts.feature#add-contact-minimum] Add a contact with only a name and a relationship
// @scenario [features/job-contacts.feature#contact-relationships] Choose how a contact relates to the job
// @scenario [features/job-contacts.feature#contact-missing-required] A contact can't be saved without a name or a relationship
// @scenario [features/job-contacts.feature#contact-invalid-email] A contact's email must be an email address
// @scenario [features/job-contacts.feature#contact-invalid-profile-link] A contact's profile link must be a web address
// @scenario [features/job-contacts.feature#contact-phone-free-text] A phone number is written however I have it
// @scenario [features/job-contacts.feature#contact-any-status] Contacts can be added to a job in any status
// @scenario [features/job-contacts.feature#view-contacts] See a job's contacts when opening its card
// @scenario [features/job-contacts.feature#no-contacts] A job with no contacts says so
// @scenario [features/job-contacts.feature#open-contact-profile-link] Open a contact's profile link
// @scenario [features/job-contacts.feature#edit-contact] Edit a contact
// @scenario [features/job-contacts.feature#remove-contact] Remove a contact after confirming
// @scenario [features/job-contacts.feature#cancel-remove-contact] Cancel removing a contact
// @scenario [features/job-contacts.feature#same-person-many-jobs] The same person can be a contact on more than one job
// @scenario [features/job-contacts.feature#contacts-edited-separately] Editing a contact on one job doesn't change it on another
export function ContactList({ jobId, contacts, onChanged }: Props) {
	const { add, update, remove, openLink } = useContacts();
	// editing is the contact being edited, "new" while adding one, or null.
	const [editing, setEditing] = useState<Contact | "new" | null>(null);
	const [removing, setRemoving] = useState<Contact | null>(null);
	const { error, run } = useAction();
	const titleId = useId();

	return (
		<section className={styles.contacts} aria-labelledby={titleId}>
			<div className={styles.header}>
				<h3 id={titleId} className={styles.title}>
					Contacts
				</h3>
				<button type="button" className={styles.button} onClick={() => setEditing("new")}>
					Add contact
				</button>
			</div>
			{contacts.length === 0 ? (
				<p className={styles.none}>No contacts yet.</p>
			) : (
				<ul className={styles.list}>
					{contacts.map((contact) => (
						<li key={contact.id} className={styles.contact} aria-label={contact.name}>
							<div className={styles.heading}>
								<span className={styles.name}>{contact.name}</span>
								<span className={styles.relationship}>
									{relationshipLabel(contact.relationship)}
								</span>
								<span className={styles.spacer} />
								<button
									type="button"
									className={styles.button}
									aria-label={`Edit ${contact.name}`}
									onClick={() => setEditing(contact)}
								>
									Edit
								</button>
								<button
									type="button"
									className={styles.button}
									aria-label={`Remove ${contact.name}`}
									onClick={() => setRemoving(contact)}
								>
									Remove
								</button>
							</div>
							<ContactInfo
								contact={contact}
								onOpenProfile={() => run(() => openLink(contact.id))}
							/>
						</li>
					))}
				</ul>
			)}
			<ErrorMessage message={error} />
			{editing === "new" && (
				<ContactForm
					title="Add a contact"
					initial={emptyContact}
					onSave={async (details) => {
						await add(jobId, details);
						setEditing(null);
						onChanged();
					}}
					onCancel={() => setEditing(null)}
				/>
			)}
			{editing !== null && editing !== "new" && (
				<ContactForm
					title={`Edit ${editing.name}`}
					initial={editing}
					onSave={async (details) => {
						await update(editing.id, details);
						setEditing(null);
						onChanged();
					}}
					onCancel={() => setEditing(null)}
				/>
			)}
			{removing && (
				<ConfirmDialog
					title={`Remove ${removing.name}?`}
					message={`${removing.name} will be removed from this job's contacts.`}
					confirmLabel="Remove"
					onConfirm={async () => {
						await remove(removing.id);
						setRemoving(null);
						onChanged();
					}}
					onCancel={() => setRemoving(null)}
				/>
			)}
		</section>
	);
}

type InfoProps = {
	contact: Contact;
	onOpenProfile: () => void;
};

// ContactInfo lists the details a contact has, leaving out the ones left empty.
function ContactInfo({ contact, onOpenProfile }: InfoProps) {
	if (
		contact.email === "" &&
		contact.phone === "" &&
		contact.profileLink === "" &&
		contact.notes === ""
	) {
		return null;
	}
	return (
		<dl className={styles.info}>
			{contact.email !== "" && (
				<>
					<dt>Email</dt>
					<dd>{contact.email}</dd>
				</>
			)}
			{contact.phone !== "" && (
				<>
					<dt>Phone</dt>
					<dd>{contact.phone}</dd>
				</>
			)}
			{contact.profileLink !== "" && (
				<>
					<dt>Profile</dt>
					<dd>
						<button
							type="button"
							className={styles.link}
							title={`Open ${contact.profileLink}`}
							aria-label={`Open ${contact.name}'s profile link`}
							onClick={onOpenProfile}
						>
							{contact.profileLink}
						</button>
					</dd>
				</>
			)}
			{contact.notes !== "" && (
				<>
					<dt>Notes</dt>
					<dd className={styles.notes}>{contact.notes}</dd>
				</>
			)}
		</dl>
	);
}

type FormProps = {
	title: string;
	initial: ContactDetails;
	// onSave saves the contact, or rejects with the backend's message.
	onSave: (details: ContactDetails) => Promise<void>;
	onCancel: () => void;
};

// ContactForm adds or edits a contact in a dialog. The backend checks what's entered; when it
// refuses, the dialog stays open with what was entered and says why.
function ContactForm({ title, initial, onSave, onCancel }: FormProps) {
	const [details, setDetails] = useState<ContactDetails>({
		name: initial.name,
		relationship: initial.relationship,
		email: initial.email,
		phone: initial.phone,
		profileLink: initial.profileLink,
		notes: initial.notes,
	});
	const { error, run } = useAction();
	const id = useId();
	const set = (field: keyof ContactDetails) => (value: string) =>
		setDetails({ ...details, [field]: value });

	const submit = (e: FormEvent) => {
		e.preventDefault();
		void run(() => onSave(details));
	};

	return (
		<Dialog title={title} onClose={onCancel}>
			<form onSubmit={submit} noValidate>
				<div className={styles.fields}>
					<div className={styles.field}>
						<label htmlFor={`${id}-name`}>
							Name<span className={styles.required}> (required)</span>
						</label>
						<input
							id={`${id}-name`}
							className={styles.input}
							aria-required
							value={details.name}
							onChange={(e) => set("name")(e.target.value)}
						/>
					</div>
					<div className={styles.field}>
						<label htmlFor={`${id}-relationship`}>
							Relationship<span className={styles.required}> (required)</span>
						</label>
						<select
							id={`${id}-relationship`}
							className={styles.input}
							aria-required
							value={details.relationship}
							onChange={(e) => set("relationship")(e.target.value)}
						>
							<option value="">Choose…</option>
							{RELATIONSHIPS.map((r) => (
								<option key={r.value} value={r.value}>
									{r.label}
								</option>
							))}
						</select>
					</div>
					<div className={styles.field}>
						<label htmlFor={`${id}-email`}>Email</label>
						<input
							id={`${id}-email`}
							className={styles.input}
							type="email"
							value={details.email}
							onChange={(e) => set("email")(e.target.value)}
						/>
					</div>
					{/* Phone is free text, written however the user has it. */}
					<div className={styles.field}>
						<label htmlFor={`${id}-phone`}>Phone</label>
						<input
							id={`${id}-phone`}
							className={styles.input}
							type="tel"
							value={details.phone}
							onChange={(e) => set("phone")(e.target.value)}
						/>
					</div>
					<div className={`${styles.field} ${styles.wide}`}>
						<label htmlFor={`${id}-profile`}>Profile link</label>
						<input
							id={`${id}-profile`}
							className={styles.input}
							type="url"
							placeholder="https://"
							value={details.profileLink}
							onChange={(e) => set("profileLink")(e.target.value)}
						/>
					</div>
					<div className={`${styles.field} ${styles.wide}`}>
						<label htmlFor={`${id}-notes`}>Notes</label>
						<textarea
							id={`${id}-notes`}
							className={styles.textarea}
							value={details.notes}
							onChange={(e) => set("notes")(e.target.value)}
						/>
					</div>
				</div>
				<ErrorMessage message={error} />
				<div className={styles.actions}>
					<button type="button" className={styles.button} onClick={onCancel}>
						Cancel
					</button>
					<button type="submit" className={styles.primary}>
						Save contact
					</button>
				</div>
			</form>
		</Dialog>
	);
}
