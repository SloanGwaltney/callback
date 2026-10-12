import { useCallback } from "react";
import { AddInterview, RemoveInterview, UpdateInterview } from "../../wailsjs/go/main/App";
import type { Interview, InterviewDetails } from "./types";

// useInterviews adds, edits, and removes a job's interviews. Each call rejects with the backend's
// message, for the caller to show.
export function useInterviews() {
	const add = useCallback(
		(jobId: number, details: InterviewDetails): Promise<Interview> => AddInterview(jobId, details),
		[],
	);
	const update = useCallback(
		(id: number, details: InterviewDetails): Promise<Interview> => UpdateInterview(id, details),
		[],
	);
	const remove = useCallback((id: number) => RemoveInterview(id), []);

	return { add, update, remove };
}
