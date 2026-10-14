import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Contact, ContactDetails } from "../hooks/types";
import { useContacts } from "../hooks/useContacts";
import { ContactList } from "./ContactList";

vi.mock("../hooks/useContacts", () => ({ useContacts: vi.fn() }));

// backend stands in for the jobs service: every job's contacts, each its own row.
let backend: { contacts: Contact[]; nextId: number };
let api: ReturnType<typeof mockContacts>;

function mockContacts() {
	return {
		add: vi.fn(async (jobId: number, details: ContactDetails): Promise<Contact> => {
			const contact = { id: backend.nextId++, jobId, ...details };
			backend.contacts.push(contact);
			return contact;
		}),
		update: vi.fn(async (id: number, details: ContactDetails): Promise<Contact> => {
			backend.contacts = backend.contacts.map((c) => (c.id === id ? { ...c, ...details } : c));
			return backend.contacts.find((c) => c.id === id) as Contact;
		}),
		remove: vi.fn(async (id: number) => {
			backend.contacts = backend.contacts.filter((c) => c.id !== id);
		}),
		openLink: vi.fn(async (_id: number) => {}),
	};
}

beforeEach(() => {
	backend = { contacts: [], nextId: 1 };
	api = mockContacts();
	vi.mocked(useContacts).mockReturnValue(api);
});

// List shows a job's contacts and reloads them from the backend when they change, like
// JobDetails does.
function List({ jobId, label }: { jobId: number; label: string }) {
	const forJob = () => backend.contacts.filter((c) => c.jobId === jobId);
	const [shown, setShown] = useState(forJob);
	return (
		<section aria-label={label}>
			<ContactList jobId={jobId} contacts={shown} onChanged={() => setShown(forJob())} />
		</section>
	);
}

function renderList() {
	render(<List jobId={1} label="Acme" />);
	return screen.getByRole("region", { name: "Acme" });
}

// seed puts a contact straight into the backend, before the list is rendered.
function seed(jobId: number, name: string, fields: Partial<ContactDetails> = {}) {
	backend.contacts.push({
		id: backend.nextId++,
		jobId,
		name,
		relationship: "recruiter",
		email: "",
		phone: "",
		profileLink: "",
		notes: "",
		...fields,
	});
}

type Entry = {
	name?: string;
	relationship?: string;
	email?: string;
	phone?: string;
	profileLink?: string;
	notes?: string;
};

// fill types into the open contact dialog, replacing what's there, and returns the dialog.
async function fill(entry: Entry) {
	const dialog = screen.getByRole("dialog", { name: /^(Add a contact|Edit )/ });
	const typeInto = async (label: string | RegExp, value: string | undefined) => {
		if (value === undefined) {
			return;
		}
		const input = within(dialog).getByLabelText(label);
		await userEvent.clear(input);
		if (value !== "") {
			await userEvent.type(input, value);
		}
	};
	await typeInto(/^Name/, entry.name);
	if (entry.relationship !== undefined) {
		await userEvent.selectOptions(
			within(dialog).getByLabelText(/^Relationship/),
			entry.relationship === "" ? "Choose…" : entry.relationship,
		);
	}
	await typeInto("Email", entry.email);
	await typeInto("Phone", entry.phone);
	await typeInto("Profile link", entry.profileLink);
	await typeInto("Notes", entry.notes);
	return dialog;
}

async function addContact(scope: HTMLElement, entry: Entry) {
	await userEvent.click(within(scope).getByRole("button", { name: "Add contact" }));
	const dialog = await fill(entry);
	await userEvent.click(within(dialog).getByRole("button", { name: "Save contact" }));
	return dialog;
}

const contact = (scope: HTMLElement, name: string) =>
	within(scope).queryByRole("listitem", { name });

// shownAs is the relationship the list shows for a contact, or null if it isn't listed.
function shownAs(scope: HTMLElement, name: string) {
	const item = contact(scope, name);
	return item ? item.querySelector("span + span")?.textContent : null;
}

const relationships = [
	["a recruiter", "Recruiter"],
	["a hiring manager", "Hiring manager"],
	["a referrer", "Referrer"],
	["an interviewer", "Interviewer"],
	["other", "Other"],
];

describe("ContactList", () => {
	// @scenario [#add-contact]
	it("adds a contact with all their details", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();

		// @when [features/job-contacts.feature] I add a contact with:
		await addContact(list, {
			name: "Dana Smith",
			relationship: "Recruiter",
			email: "dana@acme.example.com",
			phone: "(555) 010-4477",
			profileLink: "https://www.linkedin.com/in/dana-smith",
			notes: "Prefers email. Out of office Fridays.",
		});

		expect(api.add).toHaveBeenCalledWith(1, {
			name: "Dana Smith",
			relationship: "recruiter",
			email: "dana@acme.example.com",
			phone: "(555) 010-4477",
			profileLink: "https://www.linkedin.com/in/dana-smith",
			notes: "Prefers email. Out of office Fridays.",
		});
		// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a recruiter
		expect(shownAs(list, "Dana Smith")).toBe("Recruiter");
		// @and [features/job-contacts.feature] the contact shows the email, phone, profile link, and notes
		const dana = contact(list, "Dana Smith") as HTMLElement;
		expect(dana).toHaveTextContent("dana@acme.example.com");
		expect(dana).toHaveTextContent("(555) 010-4477");
		expect(
			within(dana).getByRole("button", { name: "Open Dana Smith's profile link" }),
		).toHaveTextContent("https://www.linkedin.com/in/dana-smith");
		expect(dana).toHaveTextContent("Prefers email. Out of office Fridays.");
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#add-contact-minimum]
	it("adds a contact with only a name and a relationship", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();

		// @when [features/job-contacts.feature] I add the contact "Dana Smith" as a recruiter and nothing else
		await addContact(list, { name: "Dana Smith", relationship: "Recruiter" });

		// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a recruiter
		expect(shownAs(list, "Dana Smith")).toBe("Recruiter");
		expect(within(contact(list, "Dana Smith") as HTMLElement).queryByRole("term")).toBeNull();
	});

	// @scenario [#contact-relationships]
	it.each(relationships)("adds a contact as %s", async (_spec, label) => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();

		// @when [features/job-contacts.feature] I add the contact "Dana Smith" as <relationship>
		await addContact(list, { name: "Dana Smith", relationship: label });

		// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as <relationship>
		expect(shownAs(list, "Dana Smith")).toBe(label);
	});

	// @scenario [#contact-missing-required]
	it.each([
		["", "Recruiter", "name"],
		["Dana Smith", "", "relationship"],
	])("doesn't add a contact with name %j and relationship %j", async (name, rel, missing) => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();
		api.add.mockRejectedValueOnce(`${missing} is required`);

		// @when [features/job-contacts.feature] I try to add a contact with name "<name>" and relationship "<relationship>"
		const dialog = await addContact(list, { name, relationship: rel });

		// @then [features/job-contacts.feature] the contact is not added
		expect(within(list).queryAllByRole("listitem")).toHaveLength(0);
		// @and [features/job-contacts.feature] I am told that the <missing> is required
		expect(within(dialog).getByRole("alert")).toHaveTextContent(`${missing} is required`);
		// The dialog stays open with what was entered.
		expect(within(dialog).getByLabelText(/^Name/)).toHaveValue(name);
	});

	// @scenario [#contact-invalid-email]
	it("doesn't add a contact whose email isn't an email address", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();
		api.add.mockRejectedValueOnce("the email must be an email address, like dana@example.com");

		// @when [features/job-contacts.feature] I try to add the contact "Dana Smith" as a recruiter with the email "dana at acme"
		const dialog = await addContact(list, {
			name: "Dana Smith",
			relationship: "Recruiter",
			email: "dana at acme",
		});

		// @then [features/job-contacts.feature] the contact is not added
		expect(contact(list, "Dana Smith")).toBeNull();
		// @and [features/job-contacts.feature] I am told the email must be an email address
		expect(within(dialog).getByRole("alert")).toHaveTextContent(
			"the email must be an email address",
		);
	});

	// @scenario [#contact-invalid-profile-link]
	it("doesn't add a contact whose profile link isn't a web address", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();
		api.add.mockRejectedValueOnce(
			"the profile link must be a web address, like https://www.linkedin.com/in/dana-smith",
		);

		// @when [features/job-contacts.feature] I try to add the contact "Dana Smith" as a recruiter with the profile link "not a url"
		const dialog = await addContact(list, {
			name: "Dana Smith",
			relationship: "Recruiter",
			profileLink: "not a url",
		});

		// @then [features/job-contacts.feature] the contact is not added
		expect(contact(list, "Dana Smith")).toBeNull();
		// @and [features/job-contacts.feature] I am told the profile link must be a web address
		expect(within(dialog).getByRole("alert")).toHaveTextContent(
			"the profile link must be a web address",
		);
	});

	// @scenario [#contact-phone-free-text]
	it.each(["(555) 010-4477", "+44 20 7946 0958", "555-0104 ext. 12"])(
		"keeps the phone %s as written",
		async (phone) => {
			// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
			const list = renderList();

			// @when [features/job-contacts.feature] I add the contact "Dana Smith" as a recruiter with the phone "<phone>"
			await addContact(list, { name: "Dana Smith", relationship: "Recruiter", phone });

			// @then [features/job-contacts.feature] the contact shows the phone "<phone>"
			const dana = contact(list, "Dana Smith") as HTMLElement;
			expect(within(dana).getByText("Phone").nextElementSibling).toHaveTextContent(phone);
		},
	);

	it("cancels adding a contact", async () => {
		const list = renderList();
		await userEvent.click(within(list).getByRole("button", { name: "Add contact" }));
		await fill({ name: "Dana Smith", relationship: "Recruiter" });
		await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
		expect(api.add).not.toHaveBeenCalled();
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#view-contacts]
	it("lists contacts in the order they were added", () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		// @and [features/job-contacts.feature] I added the contact "Dana Smith" as a recruiter
		seed(1, "Dana Smith", { relationship: "recruiter" });
		// @and [features/job-contacts.feature] I then added the contact "Lee Park" as a hiring manager
		seed(1, "Lee Park", { relationship: "hiring_manager" });

		// @when [features/job-contacts.feature] I open the card
		const list = renderList();

		// @then [features/job-contacts.feature] I see "Dana Smith" and "Lee Park" in the order I added them
		const names = within(list)
			.getAllByRole("listitem")
			.map((li) => li.getAttribute("aria-label"));
		expect(names).toEqual(["Dana Smith", "Lee Park"]);
		expect(shownAs(list, "Lee Park")).toBe("Hiring manager");
	});

	// @scenario [#no-contacts]
	it("says when a job has no contacts", () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with no contacts
		// @when [features/job-contacts.feature] I open the card
		const list = renderList();

		// @then [features/job-contacts.feature] I am told the job has no contacts yet
		expect(within(list).getByText("No contacts yet.")).toBeInTheDocument();
	});

	// @scenario [#open-contact-profile-link]
	it("opens a contact's profile link", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
		// @and [features/job-contacts.feature] the contact "Dana Smith" has the profile link "https://www.linkedin.com/in/dana-smith"
		seed(1, "Dana Smith", { profileLink: "https://www.linkedin.com/in/dana-smith" });
		const list = renderList();

		// @when [features/job-contacts.feature] I open Dana Smith's profile link
		await userEvent.click(
			within(list).getByRole("button", { name: "Open Dana Smith's profile link" }),
		);

		// @then [features/job-contacts.feature] the profile opens in my web browser
		expect(api.openLink).toHaveBeenCalledWith(1);
	});

	it("shows why a profile link couldn't be opened", async () => {
		seed(1, "Dana Smith", { profileLink: "https://www.linkedin.com/in/dana-smith" });
		const list = renderList();
		api.openLink.mockRejectedValueOnce("contact not found");
		await userEvent.click(
			within(list).getByRole("button", { name: "Open Dana Smith's profile link" }),
		);
		expect(within(list).getByRole("alert")).toHaveTextContent("contact not found");
	});

	// @scenario [#edit-contact]
	it("edits a contact", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith" as a recruiter
		seed(1, "Dana Smith", { email: "dana@acme.example.com", phone: "(555) 010-4477" });
		const list = renderList();
		await userEvent.click(within(list).getByRole("button", { name: "Edit Dana Smith" }));
		const dialog = screen.getByRole("dialog", { name: "Edit Dana Smith" });
		expect(within(dialog).getByLabelText("Phone")).toHaveValue("(555) 010-4477");

		// @when [features/job-contacts.feature] I change Dana Smith's email to "dana.smith@acme.example.com"
		await fill({ email: "dana.smith@acme.example.com" });
		// @and [features/job-contacts.feature] I change the relationship to hiring manager
		await fill({ relationship: "Hiring manager" });
		await userEvent.click(within(dialog).getByRole("button", { name: "Save contact" }));

		expect(api.update).toHaveBeenCalledWith(1, {
			name: "Dana Smith",
			relationship: "hiring_manager",
			email: "dana.smith@acme.example.com",
			phone: "(555) 010-4477",
			profileLink: "",
			notes: "",
		});
		// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a hiring manager
		expect(shownAs(list, "Dana Smith")).toBe("Hiring manager");
		// @and [features/job-contacts.feature] the contact shows the email "dana.smith@acme.example.com"
		expect(contact(list, "Dana Smith")).toHaveTextContent("dana.smith@acme.example.com");
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	it("keeps the edit dialog open with the backend's message when an edit is refused", async () => {
		seed(1, "Dana Smith", { email: "dana@acme.example.com" });
		const list = renderList();
		api.update.mockRejectedValueOnce("the email must be an email address, like dana@example.com");
		await userEvent.click(within(list).getByRole("button", { name: "Edit Dana Smith" }));
		const dialog = await fill({ email: "dana at acme" });
		await userEvent.click(within(dialog).getByRole("button", { name: "Save contact" }));

		expect(within(dialog).getByRole("alert")).toHaveTextContent("the email must be");
		expect(contact(list, "Dana Smith")).toHaveTextContent("dana@acme.example.com");
	});

	// @scenario [#remove-contact]
	it("removes a contact after confirming", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
		seed(1, "Dana Smith");
		const list = renderList();

		// @when [features/job-contacts.feature] I remove the contact
		await userEvent.click(within(list).getByRole("button", { name: "Remove Dana Smith" }));
		const confirm = screen.getByRole("dialog", { name: "Remove Dana Smith?" });
		expect(api.remove).not.toHaveBeenCalled();
		// @and [features/job-contacts.feature] I confirm the removal
		await userEvent.click(within(confirm).getByRole("button", { name: "Remove" }));

		// @then [features/job-contacts.feature] the job no longer shows "Dana Smith"
		expect(api.remove).toHaveBeenCalledWith(1);
		expect(contact(list, "Dana Smith")).toBeNull();
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	it("shows why a contact couldn't be removed", async () => {
		seed(1, "Dana Smith");
		const list = renderList();
		api.remove.mockRejectedValueOnce("contact not found");
		await userEvent.click(within(list).getByRole("button", { name: "Remove Dana Smith" }));
		const confirm = screen.getByRole("dialog", { name: "Remove Dana Smith?" });
		await userEvent.click(within(confirm).getByRole("button", { name: "Remove" }));
		expect(within(confirm).getByRole("alert")).toHaveTextContent("contact not found");
	});

	// @scenario [#cancel-remove-contact]
	it("keeps a contact when removing is cancelled", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
		seed(1, "Dana Smith");
		const list = renderList();

		// @when [features/job-contacts.feature] I remove the contact
		await userEvent.click(within(list).getByRole("button", { name: "Remove Dana Smith" }));
		// @but [features/job-contacts.feature] I cancel the removal
		await userEvent.click(
			within(screen.getByRole("dialog", { name: "Remove Dana Smith?" })).getByRole("button", {
				name: "Cancel",
			}),
		);

		// @then [features/job-contacts.feature] the job still shows "Dana Smith"
		expect(api.remove).not.toHaveBeenCalled();
		expect(contact(list, "Dana Smith")).toBeInTheDocument();
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#same-person-many-jobs]
	it("adds the same person to two jobs", async () => {
		// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" and a job for "Developer" at "Globex"
		render(
			<>
				<List jobId={1} label="Acme" />
				<List jobId={2} label="Globex" />
			</>,
		);
		const acme = screen.getByRole("region", { name: "Acme" });
		const globex = screen.getByRole("region", { name: "Globex" });

		// @when [features/job-contacts.feature] I add the contact "Dana Smith" as a recruiter to both jobs
		await addContact(acme, { name: "Dana Smith", relationship: "Recruiter" });
		await addContact(globex, { name: "Dana Smith", relationship: "Recruiter" });

		// @then [features/job-contacts.feature] both jobs show "Dana Smith" as a recruiter
		expect(shownAs(acme, "Dana Smith")).toBe("Recruiter");
		expect(shownAs(globex, "Dana Smith")).toBe("Recruiter");
	});

	// @scenario [#contacts-edited-separately]
	it("edits a contact on one job without changing it on another", async () => {
		// @given [features/job-contacts.feature] I have a job at "Acme" and a job at "Globex" that both have the contact "Dana Smith"
		seed(1, "Dana Smith", { email: "dana@acme.example.com" });
		seed(2, "Dana Smith", { email: "dana@acme.example.com" });
		render(
			<>
				<List jobId={1} label="Acme" />
				<List jobId={2} label="Globex" />
			</>,
		);
		const acme = screen.getByRole("region", { name: "Acme" });
		const globex = screen.getByRole("region", { name: "Globex" });

		// @when [features/job-contacts.feature] I change Dana Smith's email on the "Acme" job
		await userEvent.click(within(acme).getByRole("button", { name: "Edit Dana Smith" }));
		const dialog = await fill({ email: "dana.smith@acme.example.com" });
		await userEvent.click(within(dialog).getByRole("button", { name: "Save contact" }));

		// @then [features/job-contacts.feature] the "Globex" job's contact "Dana Smith" keeps the email it had
		expect(api.update).toHaveBeenCalledWith(1, expect.anything());
		expect(contact(acme, "Dana Smith")).toHaveTextContent("dana.smith@acme.example.com");
		expect(contact(globex, "Dana Smith")).toHaveTextContent("dana@acme.example.com");
		expect(contact(globex, "Dana Smith")).not.toHaveTextContent("dana.smith@");
	});
});
