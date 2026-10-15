import { useCallback, useEffect, useState } from "react";
import { ListBackups, RestoreBackup } from "../../wailsjs/go/main/App";
import { errorMessage } from "./errors";
import type { Backup } from "./types";

// useBackups loads every backup that's still kept, newest first, and restores one. backups is null
// until the first load. A restore resolves to the backup of the data from just before, which
// undoes it, or rejects with the backend's message for the caller to show. A failed restore
// reloads the list, since the backup may have been removed since it was loaded.
export function useBackups() {
	const [backups, setBackups] = useState<Backup[] | null>(null);
	const [error, setError] = useState("");

	const reload = useCallback(async () => {
		try {
			setBackups((await ListBackups()) as Backup[]);
			setError("");
		} catch (err) {
			setError(errorMessage(err));
		}
	}, []);

	// The backups come from the backups folder each time they're shown.
	useEffect(() => {
		void reload();
	}, [reload]);

	const restore = useCallback(
		async (backup: Backup): Promise<Backup> => {
			try {
				return (await RestoreBackup(backup.kind, backup.name)) as Backup;
			} catch (err) {
				await reload();
				throw err;
			}
		},
		[reload],
	);

	return { backups, error, reload, restore };
}
