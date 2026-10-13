import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	AddLibraryDocument,
	ChooseLibraryFile,
	GetLibrary,
	OpenLibraryDocument,
	RemoveLibraryDocument,
	SetLibraryDescription,
} from "../../wailsjs/go/main/App";
import { useLibrary } from "./useLibrary";

vi.mock("../../wailsjs/go/main/App", () => ({
	AddLibraryDocument: vi.fn(),
	ChooseLibraryFile: vi.fn(),
	GetLibrary: vi.fn(),
	OpenLibraryDocument: vi.fn(),
	RemoveLibraryDocument: vi.fn(),
	SetLibraryDescription: vi.fn(),
}));

const resume = { id: 1, fileName: "resume-base-2026.docx", description: "Base resume" };

async function loaded() {
	const hook = renderHook(() => useLibrary());
	await waitFor(() => expect(hook.result.current.documents).not.toBeNull());
	return hook;
}

beforeEach(() => {
	vi.mocked(GetLibrary).mockReset().mockResolvedValue([resume]);
});

describe("useLibrary", () => {
	it("loads the library", async () => {
		const { result } = await loaded();
		expect(result.current.documents).toEqual([resume]);
		expect(result.current.error).toBe("");
	});

	it("shows why the library couldn't load", async () => {
		vi.mocked(GetLibrary).mockRejectedValue("database is locked");
		const { result } = renderHook(() => useLibrary());
		await waitFor(() => expect(result.current.error).toBe("database is locked"));
		expect(result.current.documents).toBeNull();
	});

	it("chooses a file with the native dialog", async () => {
		vi.mocked(ChooseLibraryFile).mockResolvedValue("C:Usersmecover-template.md");
		const { result } = await loaded();
		await expect(result.current.chooseFile()).resolves.toBe("C:Usersmecover-template.md");
	});

	it("adds a document, then reloads the library", async () => {
		const cover = { id: 2, fileName: "cover-template.md", description: "" };
		vi.mocked(AddLibraryDocument).mockResolvedValue(cover);
		const { result } = await loaded();
		vi.mocked(GetLibrary).mockResolvedValue([cover, resume]);

		await expect(result.current.add("C:Usersmecover-template.md", "")).resolves.toEqual(cover);

		expect(AddLibraryDocument).toHaveBeenCalledWith("C:Usersmecover-template.md", "");
		await waitFor(() => expect(result.current.documents).toEqual([cover, resume]));
	});

	it("rejects with the backend's message when a file can't be added", async () => {
		vi.mocked(AddLibraryDocument).mockRejectedValue("files can be at most 10 MB");
		const { result } = await loaded();
		await expect(result.current.add("portfolio.pdf", "")).rejects.toBe(
			"files can be at most 10 MB",
		);
	});

	it("describes, opens, and removes a document", async () => {
		vi.mocked(SetLibraryDescription).mockResolvedValue({ ...resume, description: "New" });
		vi.mocked(OpenLibraryDocument).mockResolvedValue();
		vi.mocked(RemoveLibraryDocument).mockResolvedValue();
		const { result } = await loaded();

		await result.current.setDescription(1, "New");
		await result.current.open(1);
		await result.current.remove(1);

		expect(SetLibraryDescription).toHaveBeenCalledWith(1, "New");
		expect(OpenLibraryDocument).toHaveBeenCalledWith(1);
		expect(RemoveLibraryDocument).toHaveBeenCalledWith(1);
		// Once on mount, then after describing and removing.
		expect(GetLibrary).toHaveBeenCalledTimes(3);
	});
});
