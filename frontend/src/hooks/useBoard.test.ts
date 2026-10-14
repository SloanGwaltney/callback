import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AddJob, DeleteJob, GetBoard, MoveJob } from "../../wailsjs/go/main/App";
import type { jobs } from "../../wailsjs/go/models";
import { board, card } from "../test/fixtures";
import { useBoard } from "./useBoard";

vi.mock("../../wailsjs/go/main/App", () => ({
	AddJob: vi.fn(),
	DeleteJob: vi.fn(),
	GetBoard: vi.fn(),
	MoveJob: vi.fn(),
}));

// The bindings' types are the generated model classes; the hooks only use their fields.
const asBoard = (b: ReturnType<typeof board>) => b as jobs.Board;

const details = {
	company: "Acme",
	role: "Software Engineer",
	description: "",
	link: "",
	salary: "",
	location: "",
	notes: "",
	dateApplied: "",
};

describe("useBoard", () => {
	// @scenario [#board-persists]
	it("loads the saved board when the app opens", async () => {
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ interviewing: [card(1, "Acme")] })));

		// @when [features/job-board.feature] I close and reopen the app
		const { result } = renderHook(() => useBoard());

		// @then [features/job-board.feature] the card for "Software Engineer" at "Acme" is in the "Interviewing" column
		await waitFor(() => expect(result.current.board?.columns[2].cards[0]?.company).toBe("Acme"));
		expect(result.current.board?.columns[2].status).toBe("interviewing");
	});

	// @scenario [#card-order-persists]
	it("loads cards in their saved order", async () => {
		// @given [features/job-board.feature] the "Saved" column holds the jobs at "Initech", "Acme", and "Globex" in that order
		const saved = [card(3, "Initech"), card(1, "Acme"), card(2, "Globex")];
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ saved })));

		// @when [features/job-board.feature] I close and reopen the app
		const { result } = renderHook(() => useBoard());

		// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
		await waitFor(() =>
			expect(result.current.board?.columns[0].cards.map((c) => c.company)).toEqual([
				"Initech",
				"Acme",
				"Globex",
			]),
		);
	});

	// @scenario [#refresh-on-focus]
	it("reloads the board when the window gets focus again", async () => {
		// @given [features/job-board.feature] the app is open with a job for "Software Engineer" at "Acme" in "Applied"
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ applied: [card(1, "Acme")] })));
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board?.columns[1].cards[0]?.company).toBe("Acme"));
		// @and [features/job-board.feature] I switch to another window
		act(() => {
			window.dispatchEvent(new Event("blur"));
		});
		// @and [features/job-board.feature] my agent moves the job to "Interviewing"
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ interviewing: [card(1, "Acme")] })));
		expect(result.current.board?.columns[1].cards[0]?.company).toBe("Acme");

		// @when [features/job-board.feature#refresh-on-focus] I switch back to the app
		act(() => {
			window.dispatchEvent(new Event("focus"));
		});

		// @then [features/job-board.feature] the card for "Software Engineer" at "Acme" is in the "Interviewing" column
		await waitFor(() => expect(result.current.board?.columns[2].cards[0]?.company).toBe("Acme"));
		expect(result.current.board?.columns[1].cards).toEqual([]);
	});

	// The restore itself, into the database the app has open, is TestRestoreWhileTheAppIsOpen in
	// internal/db.
	// @scenario [#restore-app-open]
	it("shows the restored data when the window gets focus again", async () => {
		// @given [features/agent-backups.feature] the app is open and I have switched to another window
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ closed: [card(1, "Acme")] })));
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board?.columns[4].cards[0]?.company).toBe("Acme"));
		act(() => {
			window.dispatchEvent(new Event("blur"));
		});
		// @and [features/agent-backups.feature] my agent restores a backup an agent made
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ applied: [card(1, "Acme")] })));

		// @when [features/agent-backups.feature] I switch back to the app
		act(() => {
			window.dispatchEvent(new Event("focus"));
		});

		// @then [features/agent-backups.feature] the board shows the restored data
		await waitFor(() => expect(result.current.board?.columns[1].cards[0]?.company).toBe("Acme"));
		expect(result.current.board?.columns[4].cards).toEqual([]);
	});

	it("stops reloading on focus once it's gone", async () => {
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board()));
		const { result, unmount } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board).not.toBeNull());
		unmount();
		vi.mocked(GetBoard).mockClear();

		window.dispatchEvent(new Event("focus"));

		expect(GetBoard).not.toHaveBeenCalled();
	});

	it("keeps the newest board when an older load finishes last", async () => {
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board()));
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board).not.toBeNull());

		let finishOlder = (_: jobs.Board) => {};
		vi.mocked(GetBoard).mockReturnValueOnce(
			new Promise<jobs.Board>((resolve) => {
				finishOlder = resolve;
			}),
		);
		const older = act(() => result.current.reload());
		vi.mocked(GetBoard).mockResolvedValueOnce(asBoard(board({ interviewing: [card(1, "Acme")] })));
		await act(() => result.current.reload());
		finishOlder(asBoard(board({ applied: [card(1, "Acme")] })));
		await older;

		expect(result.current.board?.columns[2].cards[0]?.company).toBe("Acme");
		expect(result.current.board?.columns[1].cards).toEqual([]);
	});

	it("shows why the board couldn't load", async () => {
		vi.mocked(GetBoard).mockRejectedValue("database is locked");
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.error).toBe("database is locked"));
	});

	// @scenario [#new-card-at-top]
	it("adds a job, then reloads the board", async () => {
		// @given [features/job-board.feature] the "Saved" column holds the jobs at "Globex" and "Initech" in that order
		const before = board({ saved: [card(2, "Globex"), card(3, "Initech")] });
		vi.mocked(GetBoard).mockResolvedValue(asBoard(before));
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board).not.toBeNull());

		const after = board({ saved: [card(1, "Acme"), card(2, "Globex"), card(3, "Initech")] });
		vi.mocked(GetBoard).mockResolvedValue(asBoard(after));
		vi.mocked(AddJob).mockResolvedValue({ id: 1 } as jobs.Job);
		// @when [features/job-board.feature] I add a job at "Acme" to the "Saved" column
		await act(() => result.current.addJob(details, "saved"));

		expect(AddJob).toHaveBeenCalledWith(details, "saved");
		// @then [features/job-board.feature] the "Saved" column shows "Acme", "Globex", and "Initech" in that order
		expect(result.current.board?.columns[0].cards.map((c) => c.company)).toEqual([
			"Acme",
			"Globex",
			"Initech",
		]);
	});

	it("rejects with the backend's message when a job can't be added", async () => {
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board()));
		vi.mocked(AddJob).mockRejectedValue("company is required");
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board).not.toBeNull());

		await expect(result.current.addJob({ ...details, company: "" }, "saved")).rejects.toBe(
			"company is required",
		);
		expect(GetBoard).toHaveBeenCalledTimes(1);
	});

	it("shows a move at once, then reloads the board from the backend", async () => {
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ saved: [card(1, "Acme")] })));
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board).not.toBeNull());

		let finish = () => {};
		vi.mocked(MoveJob).mockReturnValue(
			new Promise<void>((resolve) => {
				finish = resolve;
			}),
		);
		const reloaded = board({ applied: [card(1, "Acme", { dateApplied: "2026-10-09" })] });
		vi.mocked(GetBoard).mockResolvedValue(asBoard(reloaded));
		const move = {
			jobId: 1,
			status: "applied",
			index: 0,
			closingReason: "",
			closingReasonOther: "",
		};
		let done: Promise<void> = Promise.resolve();
		act(() => {
			done = result.current.moveJob(move);
		});

		expect(result.current.board?.columns[1].cards.map((c) => c.id)).toEqual([1]);
		expect(result.current.board?.columns[1].cards[0].dateApplied).toBe("");
		await act(async () => {
			finish();
			await done;
		});
		expect(MoveJob).toHaveBeenCalledWith(move);
		expect(result.current.board?.columns[1].cards[0].dateApplied).toBe("2026-10-09");
	});

	it("puts the card back when a move fails", async () => {
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ interviewing: [card(1, "Acme")] })));
		vi.mocked(MoveJob).mockRejectedValue("write why the job was closed");
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board).not.toBeNull());

		const move = {
			jobId: 1,
			status: "closed",
			index: 0,
			closingReason: "other",
			closingReasonOther: "",
		};
		await act(async () => {
			await expect(result.current.moveJob(move)).rejects.toBe("write why the job was closed");
		});
		expect(result.current.board?.columns[2].cards.map((c) => c.id)).toEqual([1]);
	});

	it("deletes a job, then reloads the board", async () => {
		vi.mocked(GetBoard).mockResolvedValue(asBoard(board({ saved: [card(1, "Acme")] })));
		const { result } = renderHook(() => useBoard());
		await waitFor(() => expect(result.current.board).not.toBeNull());

		vi.mocked(GetBoard).mockResolvedValue(asBoard(board()));
		vi.mocked(DeleteJob).mockResolvedValue();
		await act(() => result.current.deleteJob(1));

		expect(DeleteJob).toHaveBeenCalledWith(1);
		expect(result.current.board?.columns[0].cards).toEqual([]);
	});
});
