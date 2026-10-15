import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ListBackups, RestoreBackup } from "../../wailsjs/go/main/App";
import type { Backup } from "./types";
import { useBackups } from "./useBackups";

vi.mock("../../wailsjs/go/main/App", () => ({
	ListBackups: vi.fn(),
	RestoreBackup: vi.fn(),
}));

const daily: Backup = { kind: "daily", name: "2026-10-12", made: "2026-10-12T09:30:00-04:00" };
const undo: Backup = {
	kind: "app",
	name: "20261014-093000_daily_2026-10-12",
	made: "2026-10-14T09:30:00-04:00",
	keptUntil: "2026-10-21",
	restored: { kind: "daily", name: "2026-10-12" },
};

async function loaded() {
	const hook = renderHook(() => useBackups());
	await waitFor(() => expect(hook.result.current.backups).not.toBeNull());
	return hook;
}

beforeEach(() => {
	vi.mocked(ListBackups)
		.mockReset()
		.mockResolvedValue([daily] as never);
	vi.mocked(RestoreBackup)
		.mockReset()
		.mockResolvedValue(undo as never);
});

describe("useBackups", () => {
	it("loads the backups", async () => {
		const { result } = await loaded();
		expect(result.current.backups).toEqual([daily]);
		expect(result.current.error).toBe("");
	});

	it("keeps the message when the backups can't be loaded", async () => {
		vi.mocked(ListBackups).mockRejectedValue("backups: permission denied");
		const { result } = renderHook(() => useBackups());
		await waitFor(() => expect(result.current.error).toBe("backups: permission denied"));
		expect(result.current.backups).toBeNull();
	});

	it("restores a backup by its kind and name, and resolves to the backup that undoes it", async () => {
		const { result } = await loaded();

		let got: Backup | undefined;
		await act(async () => {
			got = await result.current.restore(daily);
		});

		expect(vi.mocked(RestoreBackup)).toHaveBeenCalledWith("daily", "2026-10-12");
		expect(got).toEqual(undo);
	});

	it("reloads the backups when a restore fails, and rejects with the backend's message", async () => {
		const { result } = await loaded();
		vi.mocked(RestoreBackup).mockRejectedValue("That backup no longer exists.");
		vi.mocked(ListBackups).mockResolvedValue([] as never);

		await act(async () => {
			await expect(result.current.restore(daily)).rejects.toBe("That backup no longer exists.");
		});

		expect(result.current.backups).toEqual([]);
	});
});
