import { useCallback, useEffect, useState } from "react";
import {
	AddLibraryDocument,
	ChooseLibraryFile,
	GetLibrary,
	OpenLibraryDocument,
	RemoveLibraryDocument,
	SetLibraryDescription,
} from "../../wailsjs/go/main/App";
import { errorMessage } from "./errors";
import type { LibraryDocument } from "./types";

// useLibrary loads the document library, newest first, and changes it: adding, describing, and
// removing documents, each followed by a reload. documents is null until the first load. Changes
// and opening reject with the backend's message, for the caller to show.
// @scenario [#library-persists]
export function useLibrary() {
	const [documents, setDocuments] = useState<LibraryDocument[] | null>(null);
	const [error, setError] = useState("");

	const reload = useCallback(async () => {
		try {
			setDocuments(await GetLibrary());
			setError("");
		} catch (err) {
			setError(errorMessage(err));
		}
	}, []);

	// The library comes from the database each time it's shown, so it's as it was left.
	useEffect(() => {
		void reload();
	}, [reload]);

	// chooseFile asks for a file with the native file dialog, since the webview can't give Go a
	// file's path. It resolves to "" if the user cancels.
	const chooseFile = useCallback(() => ChooseLibraryFile(), []);

	const add = useCallback(
		async (path: string, description: string): Promise<LibraryDocument> => {
			const doc = await AddLibraryDocument(path, description);
			await reload();
			return doc;
		},
		[reload],
	);

	const setDescription = useCallback(
		async (id: number, description: string) => {
			await SetLibraryDescription(id, description);
			await reload();
		},
		[reload],
	);

	const remove = useCallback(
		async (id: number) => {
			await RemoveLibraryDocument(id);
			await reload();
		},
		[reload],
	);

	const open = useCallback((id: number) => OpenLibraryDocument(id), []);

	return { documents, error, reload, chooseFile, add, setDescription, remove, open };
}
