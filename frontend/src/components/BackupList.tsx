import { useState } from "react";
import type { Backup, BackupRef } from "../hooks/types";
import { useBackups } from "../hooks/useBackups";
import styles from "./BackupList.module.css";
import { ConfirmDialog } from "./ConfirmDialog";
import { ErrorMessage } from "./ErrorMessage";

type Props = {
	// onBack goes back to the job search board.
	onBack: () => void;
	// onRestored shows the board again with the restored data.
	onRestored: () => Promise<void>;
};

// madeOn is the day a backup was made, from the RFC 3339 time Go sends in the local time zone.
function madeOn(backup: Backup): string {
	return backup.made.slice(0, 10);
}

// restoredLabel names the backup an app backup was made before restoring.
function restoredLabel(ref: BackupRef): string {
	switch (ref.kind) {
		case "daily":
			return ref.name;
		case "agent":
			return `“${ref.name}”`;
		default:
			return "";
	}
}

// backupTitle is what a backup is called in the list and the confirmation.
export function backupTitle(backup: Backup): string {
	switch (backup.kind) {
		case "daily":
			return `Daily backup from ${backup.name}`;
		case "agent":
			return `“${backup.name}”`;
		default: {
			const restored = backup.restored ? restoredLabel(backup.restored) : "";
			return restored !== ""
				? `Made before restoring ${restored}`
				: "Made before undoing a restore";
		}
	}
}

// madeBy says who made a backup.
function madeBy(backup: Backup): string {
	switch (backup.kind) {
		case "daily":
			return "Daily backup";
		case "agent":
			return "Made by your agent";
		default:
			return "Made before a restore";
	}
}

// confirmTitle asks whether to restore a backup, naming it within the sentence.
function confirmTitle(backup: Backup): string {
	const title = backupTitle(backup);
	switch (backup.kind) {
		case "daily":
			return `Restore the daily backup from ${backup.name}?`;
		case "agent":
			return `Restore ${title}?`;
		default:
			return `Restore the backup ${title.charAt(0).toLowerCase()}${title.slice(1)}?`;
	}
}

// dataFrom is when a backup's data is from, as the confirmation says it: a daily backup's date,
// or the day any other was made.
function dataFrom(backup: Backup): string {
	return backup.kind === "daily" ? backup.name : madeOn(backup);
}

// BackupList is the backups page: every backup that's still kept, newest first, each saying who
// made it and, for one that expires, the day it's kept until. Restoring one asks first, says a
// backup of the current data is made so the restore can be undone, and goes back to the board.
// @scenario [#app-backups-open] [#app-backups-back-to-board] [#app-backups-newest-first] [#app-backups-none] [#app-backups-list-every-kind] [#app-restore-daily-backup] [#app-restore-agent-backup] [#app-restore-confirm-says-what-changes] [#app-restore-cancel] [#app-restore-keeps-daily-backups] [#app-restore-backs-up-first] [#app-undo-restore] [#app-restore-newer-version] [#app-restore-unreadable] [#app-restore-missing]
export function BackupList({ onBack, onRestored }: Props) {
	const { backups, error, restore } = useBackups();
	const [restoring, setRestoring] = useState<Backup | null>(null);

	return (
		<div className={styles.page}>
			<header className={styles.header}>
				<button type="button" className={styles.button} onClick={onBack}>
					← Back to board
				</button>
				<h1 className={styles.title}>Backups</h1>
			</header>
			<main className={styles.main}>
				<p className={styles.intro}>
					Callback backs up your data on each day it runs and keeps the last 15. Restoring a backup
					replaces your jobs and their details with the backup's. Documents' files aren't part of a
					backup.
				</p>
				<ErrorMessage message={error} className={styles.pageError} />
				{backups?.length === 0 && (
					<p className={styles.empty}>
						There are no backups yet. Callback takes one on each day it runs.
					</p>
				)}
				{backups !== null && backups.length > 0 && (
					<ul className={styles.list} aria-label="Backups">
						{backups.map((backup) => (
							<li
								key={`${backup.kind} ${backup.name}`}
								className={styles.backup}
								aria-label={backupTitle(backup)}
							>
								<div className={styles.info}>
									<p className={styles.name}>{backupTitle(backup)}</p>
									<p className={styles.details}>
										{madeBy(backup)}
										{backup.kind !== "daily" && ` on ${madeOn(backup)}`}
										{backup.keptUntil && ` · Kept until ${backup.keptUntil}`}
									</p>
								</div>
								<button
									type="button"
									className={styles.button}
									aria-label={`Restore ${backupTitle(backup)}`}
									onClick={() => setRestoring(backup)}
								>
									Restore…
								</button>
							</li>
						))}
					</ul>
				)}
			</main>
			{restoring && (
				<ConfirmDialog
					title={confirmTitle(restoring)}
					message={`Your current data will be replaced with the data from ${dataFrom(restoring)}. A backup of your current data will be made first, so you can undo this by restoring it.`}
					confirmLabel="Restore"
					onConfirm={async () => {
						await restore(restoring);
						setRestoring(null);
						await onRestored();
					}}
					onCancel={() => setRestoring(null)}
				/>
			)}
		</div>
	);
}
