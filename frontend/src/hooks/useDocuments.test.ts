import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import {
	AttachDocument,
	ChooseDocumentFile,
	OpenDocument,
	RemoveDocument,
	SetDocumentKind,
} from "../../wailsjs/go/main/App";
import { useDocuments } from "./useDocuments";

vi.mock("../../wailsjs/go/main/App", () => ({
	AttachDocument: vi.fn(),
	ChooseDocumentFile: vi.fn(),
	OpenDocument: vi.fn(),
	RemoveDocument: vi.fn(),
	SetDocumentKind: vi.fn(),
}));

describe("useDocuments", () => {
	it("attaches the file chosen in the native dialog", async () => {
		vi.mocked(ChooseDocumentFile).mockResolvedValue("C:\\Users\\me\\resume-2026.pdf");
		const doc = { id: 4, jobId: 1, kind: "resume", fileName: "resume-2026.pdf" };
		vi.mocked(AttachDocument).mockResolvedValue(doc);
		const { result } = renderHook(() => useDocuments());

		await expect(result.current.attach(1, "resume")).resolves.toEqual(doc);
		expect(AttachDocument).toHaveBeenCalledWith(1, "C:\\Users\\me\\resume-2026.pdf", "resume");
	});

	it("attaches nothing when the file dialog is cancelled", async () => {
		vi.mocked(ChooseDocumentFile).mockResolvedValue("");
		const { result } = renderHook(() => useDocuments());
		await expect(result.current.attach(1, "resume")).resolves.toBeNull();
		expect(AttachDocument).not.toHaveBeenCalled();
	});

	it("rejects with the backend's message when a file can't be attached", async () => {
		vi.mocked(ChooseDocumentFile).mockResolvedValue("headshot.png");
		vi.mocked(AttachDocument).mockRejectedValue("only these document types can be attached: .txt");
		const { result } = renderHook(() => useDocuments());
		await expect(result.current.attach(1, "other")).rejects.toBe(
			"only these document types can be attached: .txt",
		);
	});

	it("relabels, opens, and removes a document", async () => {
		vi.mocked(SetDocumentKind).mockResolvedValue();
		vi.mocked(OpenDocument).mockResolvedValue();
		vi.mocked(RemoveDocument).mockResolvedValue();
		const { result } = renderHook(() => useDocuments());

		await result.current.setKind(4, "cover_letter");
		await result.current.open(4);
		await result.current.remove(4);

		expect(SetDocumentKind).toHaveBeenCalledWith(4, "cover_letter");
		expect(OpenDocument).toHaveBeenCalledWith(4);
		expect(RemoveDocument).toHaveBeenCalledWith(4);
	});
});
