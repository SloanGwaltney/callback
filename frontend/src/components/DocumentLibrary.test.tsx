import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { LibraryDocument } from "../hooks/types";
import { useLibrary } from "../hooks/useLibrary";
import { DocumentLibrary, fileNameOf } from "./DocumentLibrary";

vi.mock("../hooks/useLibrary", () => ({ useLibrary: vi.fn() }));

// backend stands in for the jobs service: the library, newest first. chosen is the path the
// native file dialog returns next, "" for cancelling it.
let backend: { documents: LibraryDocument[]; nextId: number; chosen: string };

const api = {
	add: vi.fn<(path: string, description: string) => Promise<void>>(),
	setDescription: vi.fn<(id: number, description: string) => Promise<void>>(),
	remove: vi.fn<(id: number) => Promise<void>>(),
	open: vi.fn<(id: number) => Promise<void>>(),
};

// The mocked useLibrary keeps the library in state, like the real one, and reloads it from the
// backend after each change.
function mockLibrary() {
	vi.mocked(useLibrary).mockImplementation(() => {
		const [documents, setDocuments] = useState(() => [...backend.documents]);
		const reload = async () => setDocuments([...backend.documents]);
		return {
			documents,
			error: "",
			reload,
			chooseFile: async () => backend.chosen,
			add: async (path: string, description: string) => {
				await api.add(path, description);
				const doc = { id: backend.nextId++, fileName: fileNameOf(path), description };
				backend.documents.unshift(doc);
				await reload();
				return doc;
			},
			setDescription: async (id: number, description: string) => {
				await api.setDescription(id, description);
				backend.documents = backend.documents.map((d) => (d.id === id ? { ...d, description } : d));
				await reload();
			},
			remove: async (id: number) => {
				await api.remove(id);
				backend.documents = backend.documents.filter((d) => d.id !== id);
				await reload();
			},
			open: api.open,
		};
	});
}

beforeEach(() => {
	backend = { documents: [], nextId: 1, chosen: "" };
	for (const fn of Object.values(api)) {
		fn.mockReset().mockResolvedValue();
	}
	mockLibrary();
});

// seed puts a document straight into the library, before the page is rendered.
function seed(fileName: string, description = "") {
	const doc = { id: backend.nextId++, fileName, description };
	backend.documents.unshift(doc);
	return doc;
}

function renderLibrary() {
	const onBack = vi.fn();
	render(<DocumentLibrary onBack={onBack} />);
	return { onBack };
}

// listed returns what the library lists, top to bottom, as "file: description".
function listed(): string[] {
	const list = screen.queryByRole("list", { name: "Library documents" });
	if (!list) {
		return [];
	}
	return within(list)
		.getAllByRole("listitem")
		.map((item) => {
			const name = item.getAttribute("aria-label");
			const description = item.querySelector("p")?.textContent ?? "";
			return `${name}: ${description}`;
		});
}

// add chooses path in the native dialog, then types the description, if any, and adds it.
async function add(path: string, description = "") {
	backend.chosen = path;
	await userEvent.click(screen.getByRole("button", { name: "Add a document…" }));
	const dialog = screen.getByRole("dialog", { name: "Add to the library" });
	if (description !== "") {
		await userEvent.type(within(dialog).getByLabelText(/Description/), description);
	}
	await userEvent.click(within(dialog).getByRole("button", { name: "Add document" }));
}

// describe opens a document's description dialog and saves description in it, replacing the old.
async function describeAs(fileName: string, description: string) {
	await userEvent.click(
		screen.getByRole("button", { name: `Edit the description of ${fileName}` }),
	);
	const dialog = screen.getByRole("dialog", { name: `Describe ${fileName}` });
	const input = within(dialog).getByLabelText(/Description/);
	await userEvent.clear(input);
	if (description !== "") {
		await userEvent.type(input, description);
	}
	await userEvent.click(within(dialog).getByRole("button", { name: "Save description" }));
}

describe("DocumentLibrary", () => {
	it("goes back to the board", async () => {
		const { onBack } = renderLibrary();
		await userEvent.click(screen.getByRole("button", { name: "← Back to board" }));
		expect(onBack).toHaveBeenCalled();
	});

	// @scenario [#empty-library]
	it("prompts me to add my first document when the library is empty", async () => {
		// @given [features/document-library.feature] I have not added any documents to the library
		backend.documents = [];

		// @when [features/document-library.feature] I go to the document library
		renderLibrary();

		// @then [features/document-library.feature] I am prompted to add my first document
		expect(screen.getByText("Your library is empty.")).toBeInTheDocument();
		backend.chosen = "C:\\Users\\me\\resume-base-2026.docx";
		await userEvent.click(screen.getByRole("button", { name: "Add your first document…" }));
		expect(screen.getByRole("dialog", { name: "Add to the library" })).toHaveTextContent(
			"resume-base-2026.docx",
		);
	});

	it("doesn't prompt for a first document once there is one", () => {
		seed("resume-base-2026.docx");
		renderLibrary();
		expect(screen.queryByText("Your library is empty.")).not.toBeInTheDocument();
	});

	// @scenario [#add-library-document]
	it("adds a document with a description", async () => {
		// @given [features/document-library.feature] I am in the document library
		renderLibrary();

		// @when [features/document-library.feature] I add the file "resume-base-2026.docx" with the description "Base resume, backend roles"
		await add("C:\\Users\\me\\Documents\\resume-base-2026.docx", "Base resume, backend roles");

		// @then [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume, backend roles"
		expect(listed()).toEqual(["resume-base-2026.docx: Base resume, backend roles"]);
		expect(api.add).toHaveBeenCalledWith(
			"C:\\Users\\me\\Documents\\resume-base-2026.docx",
			"Base resume, backend roles",
		);
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#add-library-document-no-description]
	it("adds a document without a description", async () => {
		// @given [features/document-library.feature] I am in the document library
		renderLibrary();

		// @when [features/document-library.feature] I add the file "cover-template.md" with no description
		await add("/home/me/cover-template.md");

		// @then [features/document-library.feature] the library lists "cover-template.md" with no description
		expect(listed()).toEqual(["cover-template.md: "]);
	});

	it("adds nothing when the file dialog is cancelled", async () => {
		renderLibrary();
		backend.chosen = "";
		await userEvent.click(screen.getByRole("button", { name: "Add a document…" }));
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
		expect(api.add).not.toHaveBeenCalled();
	});

	it("adds nothing when adding is cancelled after choosing a file", async () => {
		renderLibrary();
		backend.chosen = "resume.pdf";
		await userEvent.click(screen.getByRole("button", { name: "Add a document…" }));
		await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
		expect(listed()).toEqual([]);
	});

	it("lists the library in the order the backend gives, newest first", () => {
		// The order itself is the backend's (TestListLibraryDocumentsIsNewestFirst in internal/db).
		seed("resume-base-2026.docx");
		seed("cover-template.md");
		seed("portfolio-links.txt");
		renderLibrary();
		expect(listed()).toEqual([
			"portfolio-links.txt: ",
			"cover-template.md: ",
			"resume-base-2026.docx: ",
		]);
	});

	// @scenario [#library-same-file-twice]
	it("adds the same file twice", async () => {
		// @given [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Before the rewrite"
		seed("resume-base-2026.docx", "Before the rewrite");
		renderLibrary();

		// @when [features/document-library.feature] I add "resume-base-2026.docx" again with the description "After the rewrite"
		await add("C:\\Users\\me\\resume-base-2026.docx", "After the rewrite");

		// @then [features/document-library.feature] the library lists "resume-base-2026.docx" twice, once with each description
		expect(listed()).toEqual([
			"resume-base-2026.docx: After the rewrite",
			"resume-base-2026.docx: Before the rewrite",
		]);
	});

	// @scenario [#library-allowed-file-types]
	it.each([
		"notes.txt",
		"resume.pdf",
		"prep.md",
		"cover.docx",
		"cover.doc",
		"resume.rtf",
		"resume.odt",
	])("adds %s", async (file) => {
		renderLibrary();

		// @when [features/document-library.feature] I add the file "<file>"
		await add(`C:\\Users\\me\\${file}`);

		// @then [features/document-library.feature] the library lists "<file>"
		expect(listed()).toEqual([`${file}: `]);
	});

	// @scenario [#library-disallowed-file-types]
	it.each(["headshot.png", "take-home.zip", "setup.exe", "salaries.xlsx"])(
		"says which types are allowed when adding %s",
		async (file) => {
			const message =
				"only these document types can be added: .txt, .pdf, .md, .docx, .doc, .rtf, .odt";
			api.add.mockRejectedValue(message);
			renderLibrary();

			// @when [features/document-library.feature] I try to add the file "<file>"
			await add(`C:\\Users\\me\\${file}`);

			// @then [features/document-library.feature] the file is not added
			expect(listed()).toEqual([]);
			// @and [features/document-library.feature] I am told which document types are allowed
			const dialog = screen.getByRole("dialog", { name: "Add to the library" });
			expect(within(dialog).getByRole("alert")).toHaveTextContent(message);
		},
	);

	// @scenario [#library-file-size-limit]
	it("says files can be at most 10 MB", async () => {
		api.add.mockRejectedValue("files can be at most 10 MB");
		renderLibrary();

		// @when [features/document-library.feature] I try to add a 12 MB file "portfolio.pdf"
		await add("C:\\Users\\me\\portfolio.pdf");

		// @then [features/document-library.feature] the file is not added
		expect(listed()).toEqual([]);
		// @and [features/document-library.feature] I am told files can be at most 10 MB
		expect(screen.getByRole("alert")).toHaveTextContent("files can be at most 10 MB");
	});

	// @scenario [#open-library-document]
	it("opens a library document", async () => {
		// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
		const doc = seed("resume-base-2026.docx");
		renderLibrary();

		// @when [features/document-library.feature] I open "resume-base-2026.docx" from the library
		await userEvent.click(screen.getByRole("button", { name: "resume-base-2026.docx" }));

		// The backend opens its stored copy in the default app (TestOpenLibraryDocument).
		expect(api.open).toHaveBeenCalledWith(doc.id);
		expect(screen.queryByRole("alert")).not.toBeInTheDocument();
	});

	// @scenario [#open-missing-library-document]
	it("says when a library document's file is missing", async () => {
		// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
		seed("resume-base-2026.docx");
		// @and [features/document-library.feature] its stored copy is no longer in the app's data directory
		api.open.mockRejectedValue('the file for "resume-base-2026.docx" can\'t be found');
		renderLibrary();

		// @when [features/document-library.feature] I open "resume-base-2026.docx" from the library
		await userEvent.click(screen.getByRole("button", { name: "resume-base-2026.docx" }));

		// @then [features/document-library.feature] I am told the file for "resume-base-2026.docx" can't be found
		expect(screen.getByRole("alert")).toHaveTextContent(
			'the file for "resume-base-2026.docx" can\'t be found',
		);
		// @and [features/document-library.feature#open-missing-library-document] the library still lists "resume-base-2026.docx"
		expect(listed()).toEqual(["resume-base-2026.docx: "]);
	});

	// @scenario [#edit-library-description]
	it("changes a document's description", async () => {
		// @given [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume"
		const doc = seed("resume-base-2026.docx", "Base resume");
		renderLibrary();

		// @when [features/document-library.feature] I change its description to "Base resume, backend roles"
		await describeAs("resume-base-2026.docx", "Base resume, backend roles");

		// @then [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume, backend roles"
		expect(listed()).toEqual(["resume-base-2026.docx: Base resume, backend roles"]);
		expect(api.setDescription).toHaveBeenCalledWith(doc.id, "Base resume, backend roles");
	});

	// @scenario [#clear-library-description]
	it("clears a document's description", async () => {
		// @given [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume"
		seed("resume-base-2026.docx", "Base resume");
		renderLibrary();

		// @when [features/document-library.feature] I clear its description
		await describeAs("resume-base-2026.docx", "");

		// @then [features/document-library.feature] the library lists "resume-base-2026.docx" with no description
		expect(listed()).toEqual(["resume-base-2026.docx: "]);
	});

	it("starts the description dialog with the current description", async () => {
		seed("resume-base-2026.docx", "Base resume");
		renderLibrary();
		await userEvent.click(
			screen.getByRole("button", { name: "Edit the description of resume-base-2026.docx" }),
		);
		expect(screen.getByLabelText(/Description/)).toHaveValue("Base resume");
	});

	// @scenario [#remove-library-document] [#remove-library-keeps-attachments]
	it("removes a library document after confirming", async () => {
		// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
		const doc = seed("resume-base-2026.docx");
		renderLibrary();

		// @when [features/document-library.feature] I remove "resume-base-2026.docx" from the library
		await userEvent.click(screen.getByRole("button", { name: "Remove resume-base-2026.docx" }));
		const confirm = screen.getByRole("dialog", { name: "Remove resume-base-2026.docx?" });
		expect(confirm).toHaveTextContent("The original file on your computer is kept.");
		// @and [features/document-library.feature] I confirm the removal
		await userEvent.click(within(confirm).getByRole("button", { name: "Remove" }));

		// @then [features/document-library.feature] the library no longer lists "resume-base-2026.docx"
		expect(listed()).toEqual([]);
		expect(api.remove).toHaveBeenCalledWith(doc.id);
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#cancel-remove-library-document]
	it("keeps a library document when removing is cancelled", async () => {
		// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
		seed("resume-base-2026.docx");
		renderLibrary();

		// @when [features/document-library.feature] I remove "resume-base-2026.docx" from the library
		await userEvent.click(screen.getByRole("button", { name: "Remove resume-base-2026.docx" }));
		// @but [features/document-library.feature] I cancel the removal
		await userEvent.click(
			within(screen.getByRole("dialog", { name: "Remove resume-base-2026.docx?" })).getByRole(
				"button",
				{ name: "Cancel" },
			),
		);

		// @then [features/document-library.feature#cancel-remove-library-document] the library still lists "resume-base-2026.docx"
		expect(listed()).toEqual(["resume-base-2026.docx: "]);
		expect(api.remove).not.toHaveBeenCalled();
	});

	it("shows why a document couldn't be removed", async () => {
		seed("resume-base-2026.docx");
		api.remove.mockRejectedValue("library document not found");
		renderLibrary();
		await userEvent.click(screen.getByRole("button", { name: "Remove resume-base-2026.docx" }));
		await userEvent.click(screen.getByRole("button", { name: "Remove" }));
		expect(screen.getByRole("alert")).toHaveTextContent("library document not found");
	});

	it("shows why the library couldn't load", () => {
		vi.mocked(useLibrary).mockReturnValue({
			documents: null,
			error: "database is locked",
			reload: vi.fn(),
			chooseFile: vi.fn(),
			add: vi.fn(),
			setDescription: vi.fn(),
			remove: vi.fn(),
			open: vi.fn(),
		});
		renderLibrary();
		expect(screen.getByRole("alert")).toHaveTextContent("database is locked");
		expect(screen.queryByText("Your library is empty.")).not.toBeInTheDocument();
	});
});

describe("fileNameOf", () => {
	it.each([
		["C:\\Users\\me\\resume.pdf", "resume.pdf"],
		["/home/me/cover-template.md", "cover-template.md"],
		["notes.txt", "notes.txt"],
	])("takes the file name of %s", (path, want) => {
		expect(fileNameOf(path)).toBe(want);
	});
});
