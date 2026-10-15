import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ListBackups, RestoreBackup } from "../../wailsjs/go/main/App";
import type { Backup } from "../hooks/types";
import { BackupList } from "./BackupList";

// The page runs over the real useBackups hook, with the bindings mocked, so a failed restore
// reloads the list as it does in the app.
vi.mock("../../wailsjs/go/main/App", () => ({
	ListBackups: vi.fn(),
	RestoreBackup: vi.fn(),
}));

const daily = (date: string): Backup => ({
	kind: "daily",
	name: date,
	made: `${date}T09:30:00-04:00`,
});
const beforeCleanup: Backup = {
	kind: "agent",
	name: "before-cleanup",
	made: "2026-10-13T10:00:00-04:00",
	keptUntil: "2026-10-20",
};
const beforeRestoring: Backup = {
	kind: "app",
	name: "20261014-093000_daily_2026-10-12",
	made: "2026-10-14T09:30:00-04:00",
	keptUntil: "2026-10-21",
	restored: { kind: "daily", name: "2026-10-12" },
};

function listed(backups: Backup[]) {
	vi.mocked(ListBackups).mockResolvedValue(backups as never);
}

async function openBackups(onRestored = vi.fn(async () => {})) {
	render(<BackupList onBack={() => {}} onRestored={onRestored} />);
	await waitFor(() => expect(vi.mocked(ListBackups)).toHaveBeenCalled());
	return { onRestored };
}

const rows = () =>
	within(screen.getByRole("list", { name: "Backups" }))
		.getAllByRole("listitem")
		.map((row) => row.getAttribute("aria-label"));
const row = (name: string) => screen.getByRole("listitem", { name });
const dialog = () => screen.getByRole("dialog");

async function chooseToRestore(name: string) {
	await userEvent.click(await screen.findByRole("button", { name: `Restore ${name}` }));
}

async function confirm() {
	await userEvent.click(within(dialog()).getByRole("button", { name: "Restore" }));
}

beforeEach(() => {
	vi.mocked(ListBackups).mockReset();
	vi.mocked(RestoreBackup)
		.mockReset()
		.mockResolvedValue(beforeRestoring as never);
});

describe("BackupList", () => {
	// @scenario [#app-backups-newest-first]
	it("lists the backups in the order the backend gives, newest first", async () => {
		// @given [features/restore-backups.feature] daily backups were taken on 2026-10-12, 2026-10-14, and 2026-10-13
		// The backend sorts them newest first (TestDailyBackupsAreListedNewestFirst in
		// internal/backups).
		listed([daily("2026-10-14"), daily("2026-10-13"), daily("2026-10-12")]);

		// @when [features/restore-backups.feature] I open my backups
		await openBackups();

		// @then [features/restore-backups.feature] I see the backups from 2026-10-14, 2026-10-13, and 2026-10-12, in that order
		await waitFor(() =>
			expect(rows()).toEqual([
				"Daily backup from 2026-10-14",
				"Daily backup from 2026-10-13",
				"Daily backup from 2026-10-12",
			]),
		);
	});

	// @scenario [#app-backups-none]
	it("says when there are no backups yet", async () => {
		// @given [features/restore-backups.feature] no backups have been taken
		listed([]);

		// @when [features/restore-backups.feature] I open my backups
		await openBackups();

		// @then [features/restore-backups.feature] I'm told there are no backups yet, and that one is taken each day Callback runs
		expect(
			await screen.findByText("There are no backups yet. Callback takes one on each day it runs."),
		).toBeInTheDocument();
		expect(screen.queryByRole("list", { name: "Backups" })).not.toBeInTheDocument();
	});

	// @scenario [#app-backups-list-every-kind] [#app-restore-backs-up-first]
	it("says who made each backup, and the day each one that expires is kept until", async () => {
		// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12
		// @and [features/restore-backups.feature] my agent made a backup named "before-cleanup" on 2026-10-13
		// @and [features/restore-backups.feature] I restored a backup on 2026-10-14, so a backup was made before restoring it
		listed([beforeRestoring, beforeCleanup, daily("2026-10-12")]);

		// @when [features/restore-backups.feature] I open my backups
		await openBackups();

		// @then [features/restore-backups.feature] I see all 3 backups, newest first
		await waitFor(() =>
			expect(rows()).toEqual([
				"Made before restoring 2026-10-12",
				"“before-cleanup”",
				"Daily backup from 2026-10-12",
			]),
		);
		// @and [features/restore-backups.feature] each says whether it's a daily backup, made by my agent, or made before a restore
		// @and [features/restore-backups.feature] the backups made by my agent or before a restore show the day each is kept until
		expect(row("Made before restoring 2026-10-12")).toHaveTextContent(
			"Made before a restore on 2026-10-14 · Kept until 2026-10-21",
		);
		expect(row("“before-cleanup”")).toHaveTextContent(
			"Made by your agent on 2026-10-13 · Kept until 2026-10-20",
		);
		expect(row("Daily backup from 2026-10-12")).toHaveTextContent("Daily backup");
		expect(row("Daily backup from 2026-10-12")).not.toHaveTextContent("Kept until");
		// @and [features/restore-backups.feature] the backup from my agent shows its name "before-cleanup"
		expect(within(row("“before-cleanup”")).getByText("“before-cleanup”")).toBeInTheDocument();
		// @and [features/restore-backups.feature] it's listed with my backups as made before restoring 2026-10-12
		expect(screen.getByText("Made before restoring 2026-10-12")).toBeInTheDocument();
	});

	it("names a backup made before undoing a restore, and one made before restoring an agent's backup", async () => {
		listed([
			{
				...beforeRestoring,
				name: "20261014-093100_app_20261014-093000",
				restored: { kind: "app", name: "20261014-093000" },
			},
			{
				...beforeRestoring,
				name: "20261014-093000_agent_before-cleanup",
				restored: { kind: "agent", name: "before-cleanup" },
			},
		]);

		await openBackups();

		await waitFor(() =>
			expect(rows()).toEqual([
				"Made before undoing a restore",
				"Made before restoring “before-cleanup”",
			]),
		);
	});

	// @scenario [#app-restore-confirm-says-what-changes]
	it("asks before restoring, saying the current data is replaced and backed up first", async () => {
		// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12
		listed([daily("2026-10-12")]);
		await openBackups();

		// @when [features/restore-backups.feature] I choose to restore the backup from 2026-10-12
		await chooseToRestore("Daily backup from 2026-10-12");

		// @then [features/restore-backups.feature] I'm asked to confirm that my current data will be replaced with the data from 2026-10-12
		expect(
			within(dialog()).getByRole("heading", { name: "Restore the daily backup from 2026-10-12?" }),
		).toBeInTheDocument();
		expect(
			within(dialog()).getByText(
				/Your current data will be replaced with the data from 2026-10-12\./,
			),
		).toBeInTheDocument();
		// @and [features/restore-backups.feature] I'm told a backup of my current data will be made first, so I can undo it
		expect(
			within(dialog()).getByText(
				/A backup of your current data will be made first, so you can undo this/,
			),
		).toBeInTheDocument();
		expect(vi.mocked(RestoreBackup)).not.toHaveBeenCalled();
	});

	it("says the data is from the day an agent's backup was made", async () => {
		listed([beforeCleanup]);
		await openBackups();

		await chooseToRestore("“before-cleanup”");

		expect(
			within(dialog()).getByText(/replaced with the data from 2026-10-13\./),
		).toBeInTheDocument();
		expect(
			within(dialog()).getByRole("heading", { name: "Restore “before-cleanup”?" }),
		).toBeInTheDocument();
	});

	// @scenario [#app-restore-cancel]
	it("restores nothing when I cancel", async () => {
		// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
		// @and [features/restore-backups.feature] I have since moved the job to "Closed"
		listed([daily("2026-10-12")]);
		const { onRestored } = await openBackups();

		// @when [features/restore-backups.feature] I choose to restore the backup from 2026-10-12
		await chooseToRestore("Daily backup from 2026-10-12");
		// @and [features/restore-backups.feature] I cancel
		await userEvent.click(within(dialog()).getByRole("button", { name: "Cancel" }));

		// @then [features/restore-backups.feature] the job at "Acme" is still in "Closed"
		// Nothing is restored, so the data, and the job's column, stay as they are.
		expect(vi.mocked(RestoreBackup)).not.toHaveBeenCalled();
		expect(onRestored).not.toHaveBeenCalled();
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#app-restore-daily-backup] [#app-restore-keeps-daily-backups] [#app-restore-backs-up-first] [#app-restore-older-version] [#app-restore-missing-document-file] [#app-restore-keeps-document-files]
	it("restores a daily backup once I confirm, then shows the board", async () => {
		// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12
		listed([daily("2026-10-12")]);
		const { onRestored } = await openBackups();

		// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
		await chooseToRestore("Daily backup from 2026-10-12");
		await confirm();

		// What the restore brings back, and which backups it keeps, is the backend's
		// (restore_dir_test.go and restore_test.go in internal/backups); App.test.tsx shows the board.
		expect(vi.mocked(RestoreBackup)).toHaveBeenCalledWith("daily", "2026-10-12");
		expect(onRestored).toHaveBeenCalledOnce();
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
	});

	// @scenario [#app-restore-agent-backup]
	it("restores a backup my agent made", async () => {
		// @given [features/restore-backups.feature] my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
		listed([beforeCleanup]);
		const { onRestored } = await openBackups();

		// @when [features/restore-backups.feature] I restore the backup "before-cleanup" and confirm
		await chooseToRestore("“before-cleanup”");
		await confirm();

		expect(vi.mocked(RestoreBackup)).toHaveBeenCalledWith("agent", "before-cleanup");
		expect(onRestored).toHaveBeenCalledOnce();
	});

	// @scenario [#app-undo-restore]
	it("restores the backup made before a restore, which undoes it", async () => {
		// @and [features/restore-backups.feature] I restored the backup from 2026-10-12
		listed([beforeRestoring, daily("2026-10-12")]);
		const { onRestored } = await openBackups();

		// @when [features/restore-backups.feature] I restore the backup made before that restore and confirm
		await chooseToRestore("Made before restoring 2026-10-12");
		expect(
			within(dialog()).getByRole("heading", {
				name: "Restore the backup made before restoring 2026-10-12?",
			}),
		).toBeInTheDocument();
		await confirm();

		expect(vi.mocked(RestoreBackup)).toHaveBeenCalledWith(
			"app",
			"20261014-093000_daily_2026-10-12",
		);
		expect(onRestored).toHaveBeenCalledOnce();
	});

	// The messages are the backend's (TestTheAppIsToldABackupIsFromANewerVersion and
	// TestTheAppIsToldABackupCouldntBeRestored in internal/backups).
	// @scenario [#app-restore-newer-version] [#app-restore-unreadable]
	it.each([
		"That backup is from a newer version of Callback, so it can't be restored. Update Callback to restore it. Your data is unchanged.",
		"The backup couldn't be restored. Your data is unchanged.",
	])("says why a backup couldn't be restored: %s", async (message) => {
		listed([daily("2026-10-12")]);
		const { onRestored } = await openBackups();
		vi.mocked(RestoreBackup).mockRejectedValue(message);

		// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
		await chooseToRestore("Daily backup from 2026-10-12");
		await confirm();

		// @and [features/restore-backups.feature] I'm told the backup is from a newer version of Callback, and to update Callback to restore it
		// @and [features/restore-backups.feature] I'm told the backup couldn't be restored
		expect(await within(dialog()).findByRole("alert")).toHaveTextContent(message);
		expect(onRestored).not.toHaveBeenCalled();
	});

	// @scenario [#app-restore-missing]
	it("says a backup no longer exists, and stops listing it", async () => {
		// @given [features/restore-backups.feature] I am looking at my backups
		listed([daily("2026-10-13"), daily("2026-10-12")]);
		const { onRestored } = await openBackups();
		// @and [features/restore-backups.feature] the backup from 2026-10-12 has since been removed
		listed([daily("2026-10-13")]);
		const message =
			"That backup no longer exists. It may have been removed since you opened your backups. Your data is unchanged.";
		vi.mocked(RestoreBackup).mockRejectedValue(message);

		// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
		await chooseToRestore("Daily backup from 2026-10-12");
		await confirm();

		// @and [features/restore-backups.feature] I'm told that backup no longer exists
		expect(await within(dialog()).findByRole("alert")).toHaveTextContent(message);
		expect(onRestored).not.toHaveBeenCalled();
		// @and [features/restore-backups.feature] the backup from 2026-10-12 is no longer listed
		await waitFor(() => expect(rows()).toEqual(["Daily backup from 2026-10-13"]));
	});

	it("shows why the backups couldn't be loaded", async () => {
		vi.mocked(ListBackups).mockRejectedValue("backups: permission denied");

		await openBackups();

		expect(await screen.findByRole("alert")).toHaveTextContent("backups: permission denied");
	});
});
