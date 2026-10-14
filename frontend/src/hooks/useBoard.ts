import { useCallback, useEffect, useRef, useState } from "react";
import { AddJob, DeleteJob, GetBoard, MoveJob } from "../../wailsjs/go/main/App";
import { applyMove } from "../board/boardDnd";
import { errorMessage } from "./errors";
import type { Board, Details, Job, Move } from "./types";

// useBoard loads the job board and changes it: adding, moving, and deleting jobs. Each change
// reloads the board, since the backend decides details like positions and dates applied.
// Changes reject with the backend's message, for the caller to show. It also reloads when the
// window gets focus again, to show what the user's agent changed while they were away.
// @scenario [#board-persists] [#card-order-persists] [#refresh-on-focus] [#restore-app-open]
export function useBoard() {
	const [board, setBoard] = useState<Board | null>(null);
	const [error, setError] = useState("");
	// latest numbers each load, so a load that finishes after a later one can't show older data.
	const latest = useRef(0);

	const reload = useCallback(async () => {
		const load = ++latest.current;
		try {
			const loaded = await GetBoard();
			if (load === latest.current) {
				setBoard(loaded);
				setError("");
			}
		} catch (err) {
			if (load === latest.current) {
				setError(errorMessage(err));
			}
		}
	}, []);

	// The board comes from the database each time the app starts, so it's as it was left.
	useEffect(() => {
		void reload();
	}, [reload]);

	// Coming back to the window reloads the board, since the agent CLI changes the same database.
	useEffect(() => {
		const onFocus = () => void reload();
		window.addEventListener("focus", onFocus);
		return () => window.removeEventListener("focus", onFocus);
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
