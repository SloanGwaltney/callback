import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { JobDocument } from "../hooks/types";
import { useDocuments } from "../hooks/useDocuments";
import { DocumentList } from "./DocumentList";

vi.mock("../hooks/useDocuments", () => ({ useDocuments: vi.fn() }));

// backend stands in for the jobs service: attached documents per job, and the file the native
// file dialog will return next.
let backend: { docs: JobDocument[]; chosen: string; nextId: number };
let docs: ReturnType<typeof mockDocuments>;

function mockDocuments() {
	return {
		attach: vi.fn(async (jobId: number, kind: string): Promise<JobDocument | null> => {
			const doc = { id: backend.nextId++, jobId, kind, fileName: backend.chosen };
			backend.docs.push(doc);
			return doc;
		}),
		setKind: vi.fn(async (id: number, kind: string) => {
			backend.docs = backend.docs.map((d) => (d.id === id ? { ...d, kind } : d));
		}),
		remove: vi.fn(async (id: number) => {
			backend.docs = backend.docs.filter((d) => d.id !== id);
		}),
		open: vi.fn(async (_id: number) => {}),
	};
}

beforeEach(() => {
	backend = { docs: [], chosen: "", nextId: 1 };
	docs = mockDocuments();
	vi.mocked(useDocuments).mockReturnValue(docs);
});

// List shows a job's documents and reloads them from the backend when they change, like
// JobDetails does.
function List({ jobId, label }: { jobId: number; label: string }) {
	const forJob = () => backend.docs.filter((d) => d.jobId === jobId);
	const [shown, setShown] = useState(forJob);
	return (
		<section aria-label={label}>
			<DocumentList jobId={jobId} documents={shown} onChanged={() => setShown(forJob())} />
		</section>
	);
}

function renderList() {
	render(<List jobId={1} label="Acme" />);
	return screen.getByRole("region", { name: "Acme" });
}

async function attach(scope: HTMLElement, file: string, kind: string) {
	backend.chosen = file;
	await userEvent.selectOptions(within(scope).getByLabelText("Attach as"), kind);
	await userEvent.click(within(scope).getByRole("button", { name: "Attach a document…" }));
}

// kindOf is the kind the job lists a document as, or null if it isn't listed.
function kindOf(scope: HTMLElement, file: string) {
	const select = within(scope).queryByRole<HTMLSelectElement>("combobox", {
		name: `Kind of ${file}`,
	});
	return select ? select.selectedOptions[0].textContent : null;
}

describe("DocumentList", () => {
	// @scenario [#attach-document]
	it.each([
		["resume-2026.pdf", "Resume"],
		["acme-cover.docx", "Cover letter"],
		["portfolio-links.txt", "Other"],
	])("attaches %s as a %s", async (file, kind) => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();

		// @when [features/job-documents.feature] I attach the file "<file>" as a <kind>
		await attach(list, file, kind);

		// @then [features/job-documents.feature] the job lists "<file>" as a <kind>
		expect(kindOf(list, file)).toBe(kind);
	});

	// @scenario [#attach-multiple-documents]
	it("attaches several documents to one job", async () => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();

		// @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume
		await attach(list, "resume-2026.pdf", "Resume");
		// @and [features/job-documents.feature] I attach "acme-cover.docx" as a cover letter
		await attach(list, "acme-cover.docx", "Cover letter");
		// @and [features/job-documents.feature] I attach "take-home-notes.md" as other
		await attach(list, "take-home-notes.md", "Other");

		// @then [features/job-documents.feature] the job lists all three documents with their kinds
		expect(kindOf(list, "resume-2026.pdf")).toBe("Resume");
		expect(kindOf(list, "acme-cover.docx")).toBe("Cover letter");
		expect(kindOf(list, "take-home-notes.md")).toBe("Other");
	});

	// @scenario [#allowed-file-types]
	it.each([
		"notes.txt",
		"resume.pdf",
		"prep.md",
		"cover.docx",
		"cover.doc",
		"resume.rtf",
		"resume.odt",
	])("attaches %s", async (file) => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();
		// @when [features/job-documents.feature] I attach the file "<file>" as other
		await attach(list, file, "Other");
		// @then [features/job-documents.feature] the job lists "<file>" as other
		expect(kindOf(list, file)).toBe("Other");
	});

	// @scenario [#file-type-any-case]
	it("attaches a file with an upper-case extension", async () => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();
		// @when [features/job-documents.feature] I attach the file "RESUME.PDF" as other
		await attach(list, "RESUME.PDF", "Other");
		// @then [features/job-documents.feature] the job lists "RESUME.PDF" as other
		expect(kindOf(list, "RESUME.PDF")).toBe("Other");
	});

	// @scenario [#file-size-at-limit]
	it("attaches a 10 MB file", async () => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();
		// @when [features/job-documents.feature] I attach a 10 MB file "portfolio.pdf" as other
		await attach(list, "portfolio.pdf", "Other");
		// @then [features/job-documents.feature] the job lists "portfolio.pdf" as other
		expect(kindOf(list, "portfolio.pdf")).toBe("Other");
	});

	// @scenario [#disallowed-file-types]
	it.each(["headshot.png", "take-home.zip", "setup.exe", "salaries.xlsx"])(
		"doesn't attach %s",
		async (file) => {
			// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
			const list = renderList();
			docs.attach.mockRejectedValueOnce(
				"only these document types can be attached: .txt, .pdf, .md, .docx, .doc, .rtf, .odt",
			);

			// @when [features/job-documents.feature] I try to attach the file "<file>"
			await attach(list, file, "Other");

			// @then [features/job-documents.feature] the file is not attached
			expect(kindOf(list, file)).toBeNull();
			// @and [features/job-documents.feature] I am told which document types are allowed
			expect(within(list).getByRole("alert")).toHaveTextContent(
				"only these document types can be attached: .txt, .pdf, .md, .docx, .doc, .rtf, .odt",
			);
		},
	);

	// @scenario [#file-size-limit]
	it("doesn't attach a file over 10 MB", async () => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
		const list = renderList();
		docs.attach.mockRejectedValueOnce("files can be at most 10 MB");

		// @when [features/job-documents.feature] I try to attach a 12 MB file "portfolio.pdf"
		await attach(list, "portfolio.pdf", "Other");

		// @then [features/job-documents.feature] the file is not attached
		expect(kindOf(list, "portfolio.pdf")).toBeNull();
		// @and [features/job-documents.feature] I am told files can be at most 10 MB
		expect(within(list).getByRole("alert")).toHaveTextContent("files can be at most 10 MB");
	});

	it("attaches nothing when the file dialog is cancelled", async () => {
		docs.attach.mockResolvedValueOnce(null);
		const list = renderList();
		await attach(list, "", "Resume");
		expect(within(list).getByText("No documents attached.")).toBeInTheDocument();
		expect(within(list).queryByRole("alert")).not.toBeInTheDocument();
	});

	// @scenario [#open-document]
	it("opens an attached document", async () => {
		// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
		const list = renderList();
		await attach(list, "resume-2026.pdf", "Resume");

		// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
		await userEvent.click(within(list).getByRole("button", { name: "resume-2026.pdf" }));

		// @then [features/job-documents.feature] it opens in my computer's default app for that file type
		expect(docs.open).toHaveBeenCalledWith(1);
	});

	it("shows the full file name on hover, since long names are truncated", async () => {
		const list = renderList();
		await attach(list, "resume-2026.pdf", "Resume");
		expect(within(list).getByRole("button", { name: "resume-2026.pdf" })).toHaveAttribute(
			"title",
			"Open resume-2026.pdf",
		);
	});

	// @scenario [#document-kept-if-original-moved]
	it("opens the attachment by its ID, not the original file's path", async () => {
		// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
		const list = renderList();
		await attach(list, "resume-2026.pdf", "Resume");
		// @and [features/job-documents.feature] I have since moved or deleted the original "resume-2026.pdf" on my computer
		backend.chosen = "";

		// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
		await userEvent.click(within(list).getByRole("button", { name: "resume-2026.pdf" }));

		// @then [features/job-documents.feature] the document opens with the content it had when I attached it
		expect(docs.open).toHaveBeenCalledWith(1);
	});

	it("shows why a document couldn't be opened", async () => {
		const list = renderList();
		await attach(list, "resume-2026.pdf", "Resume");
		docs.open.mockRejectedValueOnce("document not found");
		await userEvent.click(within(list).getByRole("button", { name: "resume-2026.pdf" }));
		expect(within(list).getByRole("alert")).toHaveTextContent("document not found");
	});

	// @scenario [#open-missing-document]
	it("says when an attached document's file is missing", async () => {
		// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
		const list = renderList();
		await attach(list, "resume-2026.pdf", "Resume");
		// @and [features/job-documents.feature] its stored copy is no longer in the app's data directory
		docs.open.mockRejectedValue('the file for "resume-2026.pdf" can\'t be found');

		// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
		await userEvent.click(within(list).getByRole("button", { name: "resume-2026.pdf" }));

		// @then [features/job-documents.feature] I am told the file for "resume-2026.pdf" can't be found
		expect(within(list).getByRole("alert")).toHaveTextContent(
			'the file for "resume-2026.pdf" can\'t be found',
		);
		// @and [features/job-documents.feature] the job still lists "resume-2026.pdf"
		expect(kindOf(list, "resume-2026.pdf")).toBe("Resume");
	});

	// @scenario [#remove-missing-document]
	it("removes an attached document whose file is missing", async () => {
		// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
		const list = renderList();
		await attach(list, "resume-2026.pdf", "Resume");
		// @and [features/job-documents.feature] its stored copy is no longer in the app's data directory
		docs.open.mockRejectedValue('the file for "resume-2026.pdf" can\'t be found');
		await userEvent.click(within(list).getByRole("button", { name: "resume-2026.pdf" }));

		// @when [features/job-documents.feature] I remove the attached "resume-2026.pdf"
		await userEvent.click(within(list).getByRole("button", { name: "Remove resume-2026.pdf" }));

		// @then [features/job-documents.feature] the job no longer lists "resume-2026.pdf"
		expect(kindOf(list, "resume-2026.pdf")).toBeNull();
		expect(within(list).queryByRole("alert")).not.toBeInTheDocument();
	});

	// @scenario [#change-document-kind]
	it("changes what kind of document an attachment is", async () => {
		// @given [features/job-documents.feature] I have attached "acme-cover.docx" as other to a job for "Software Engineer" at "Acme"
		const list = renderList();
		await attach(list, "acme-cover.docx", "Other");

		// @when [features/job-documents.feature] I change its kind to cover letter
		await userEvent.selectOptions(
			within(list).getByRole("combobox", { name: "Kind of acme-cover.docx" }),
			"Cover letter",
		);

		expect(docs.setKind).toHaveBeenCalledWith(1, "cover_letter");
		// @then [features/job-documents.feature] the job lists "acme-cover.docx" as a cover letter
		expect(kindOf(list, "acme-cover.docx")).toBe("Cover letter");
	});

	// @scenario [#remove-document]
	it("removes an attached document", async () => {
		// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
		const list = renderList();
		await attach(list, "resume-2026.pdf", "Resume");

		// @when [features/job-documents.feature] I remove the attached "resume-2026.pdf"
		await userEvent.click(within(list).getByRole("button", { name: "Remove resume-2026.pdf" }));

		// @then [features/job-documents.feature] the job no longer lists "resume-2026.pdf"
		expect(kindOf(list, "resume-2026.pdf")).toBeNull();
		// @and [features/job-documents.feature] the original "resume-2026.pdf" on my computer is not deleted
		expect(docs.remove).toHaveBeenCalledWith(1);
	});

	// @scenario [#same-document-many-jobs]
	it("attaches the same file to two jobs", async () => {
		// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme" and a job for "Developer" at "Globex"
		render(
			<>
				<List jobId={1} label="Acme" />
				<List jobId={2} label="Globex" />
			</>,
		);
		const acme = screen.getByRole("region", { name: "Acme" });
		const globex = screen.getByRole("region", { name: "Globex" });

		// @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume to both jobs
		await attach(acme, "resume-2026.pdf", "Resume");
		await attach(globex, "resume-2026.pdf", "Resume");

		// @then [features/job-documents.feature] both jobs list "resume-2026.pdf" as a resume
		expect(kindOf(acme, "resume-2026.pdf")).toBe("Resume");
		expect(kindOf(globex, "resume-2026.pdf")).toBe("Resume");

		// @and [features/job-documents.feature] removing it from one job leaves it attached to the other
		await userEvent.click(within(acme).getByRole("button", { name: "Remove resume-2026.pdf" }));
		expect(kindOf(acme, "resume-2026.pdf")).toBeNull();
		expect(kindOf(globex, "resume-2026.pdf")).toBe("Resume");
	});
});
