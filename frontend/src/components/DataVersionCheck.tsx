import type { ReactNode } from "react";
import { useDataVersion } from "../hooks/useDataVersion";
import styles from "./DataVersionCheck.module.css";

type Props = {
	// children is the app, rendered only once the data is known not to be from a newer version.
	children: ReactNode;
};

// DataVersionCheck shows the app, or, when a newer version of Callback has updated the data, tells
// the user to update Callback in its place. The app isn't rendered at all then, so nothing asks the
// backend for the board. See
// docs/adr/0025-restore-backups-with-sqlites-online-backup-api-and-check-data-versions-first.md.
// @scenario [#app-newer-data] [#app-older-data] [#app-same-version-data]
export function DataVersionCheck({ children }: Props) {
	const { newerData } = useDataVersion();

	if (newerData === null) {
		return null;
	}
	if (newerData) {
		return (
			<main className={styles.page}>
				<section className={styles.message} role="alert" aria-labelledby="newer-data-title">
					<h1 id="newer-data-title" className={styles.title}>
						Update Callback
					</h1>
					<p className={styles.text}>
						Your data is from a newer version of Callback. Update Callback to open it.
					</p>
					<p className={styles.text}>Nothing in your data has been changed.</p>
				</section>
			</main>
		);
	}
	return children;
}
