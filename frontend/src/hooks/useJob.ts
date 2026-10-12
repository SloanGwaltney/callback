import { useCallback, useEffect, useState } from "react";
import {
	ClearNextAction,
	GetJob,
	OpenJobLink,
	SetClosingReason,
	SetNextAction,
	UpdateJob,
} from "../../wailsjs/go/main/App";
import { errorMessage } from "./errors";
import type { Details, Job } from "./types";

// useJob loads one job with all its details, next action, and documents, and changes it. Changes reject with
// the backend's message, for the caller to show.
export function useJob(id: number) {
	const [job, setJob] = useState<Job | null>(null);
	const [error, setError] = useState("");

	const reload = useCallback(async () => {
		try {
			setJob(await GetJob(id));
			setError("");
		} catch (err) {
			setError(errorMessage(err));
		}
	}, [id]);

	useEffect(() => {
		void reload();
	}, [reload]);

	const update = useCallback(
		async (details: Details): Promise<Job> => {
			const updated = await UpdateJob(id, details);
			setJob(updated);
			return updated;
		},
		[id],
	);

	const setClosingReason = useCallback(
		async (reason: string, other: string) => {
			await SetClosingReason(id, reason, other);
			await reload();
		},
		[id, reload],
	);

	// setNextAction sets what's next on the job, with expectedDate "" for no expected date.
	const setNextAction = useCallback(
		async (text: string, expectedDate: string) => {
			setJob(await SetNextAction(id, text, expectedDate));
		},
		[id],
	);

	const clearNextAction = useCallback(async () => {
		setJob(await ClearNextAction(id));
	}, [id]);

	const openLink = useCallback(() => OpenJobLink(id), [id]);

	return {
		job,
		error,
		reload,
		update,
		setClosingReason,
		setNextAction,
		clearNextAction,
		openLink,
	};
}
