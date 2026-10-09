import { useCallback } from "react";
import {
	AttachDocument,
	ChooseDocumentFile,
	OpenDocument,
	RemoveDocument,
	SetDocumentKind,
} from "../../wailsjs/go/main/App";
import type { JobDocument } from "./types";

// useDocuments attaches, relabels, opens, and removes a job's documents. Each call rejects with
// the backend's message, for the caller to show.
export function useDocuments() {
	// attach asks for a file with the native file dialog, since the webview can't give Go a file's
	// path, then attaches it. It resolves to null if the user cancels the dialog.
	const attach = useCallback(async (jobId: number, kind: string): Promise<JobDocument | null> => {
		const path = await ChooseDocumentFile();
		if (path === "") {
			return null;
		}
		return AttachDocument(jobId, path, kind);
	}, []);

	const setKind = useCallback((id: number, kind: string) => SetDocumentKind(id, kind), []);
	const remove = useCallback((id: number) => RemoveDocument(id), []);
	const open = useCallback((id: number) => OpenDocument(id), []);

	return { attach, setKind, remove, open };
}
