import { useCallback, useEffect, useState } from "react";
import { AddJob, DeleteJob, GetBoard, MoveJob } from "../../wailsjs/go/main/App";
import { applyMove } from "../board/boardDnd";
import { errorMessage } from "./errors";
import type { Board, Details, Job, Move } from "./types";

// useBoard loads the job board and changes it: adding, moving, and deleting jobs. Each change
// reloads the board, since the backend decides details like positions and dates applied.
// Changes reject with the backend's message, for the caller to show.
// @scenario [features/job-board.feature#board-persists] Board is the same after reopening the app
// @scenario [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
export function useBoard() {
	const [board, setBoard] = useState<Board | null>(null);
	const [error, setError] = useState("");

	const reload = useCallback(async () => {
		try {
			// @given [features/job-board.feature] I open the job search board
			// @when [features/job-board.feature] I open the job search board
			// @given [features/job-cards.feature] I open the job search board
			// @when [features/board-pagination.feature] I open the job search board
			setBoard(await GetBoard());
			setError("");
		} catch (err) {
			setError(errorMessage(err));
		}
	}, []);

	// The board comes from the database each time the app starts, so it's as it was left.
	// @when [features/job-board.feature] I close and reopen the app
	useEffect(() => {
		void reload();
	}, [reload]);

	const addJob = useCallback(
		async (details: Details, status: string): Promise<Job> => {
			const job = await AddJob(details, status);
			await reload();
			return job;
		},
		[reload],
	);

	const moveJob = useCallback(
		async (move: Move) => {
			setBoard((current) => current && applyMove(current, move));
			try {
				await MoveJob(move);
			} finally {
				// Picks up what the move changed on the job, or puts the card back if it failed.
				await reload();
			}
		},
		[reload],
	);

	const deleteJob = useCallback(
		async (id: number) => {
			await DeleteJob(id);
			await reload();
		},
		[reload],
	);

	return { board, error, reload, addJob, moveJob, deleteJob };
}
