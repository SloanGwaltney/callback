import { useId } from "react";
import type { StatusChange } from "../hooks/types";
import { STATUS_LABELS } from "../jobs/constants";
import styles from "./StatusHistory.module.css";

type Props = {
	// history is from oldest to newest, as the backend lists it.
	history: StatusChange[];
};

// StatusHistory shows every status a job has been in and the date it moved there, with the
// closing reason chosen when it was closed. It's a record, so it has nothing to edit.
// @scenario [features/job-status-history.feature#history-starts-when-added] A new job's history starts with the status it was added to
// @scenario [features/job-status-history.feature#history-oldest-first] The history lists every change from oldest to newest
// @scenario [features/job-status-history.feature#history-records-closing-reason] Closing a job records its closing reason in the history
// @scenario [features/job-status-history.feature#history-read-only] The status history can't be edited
export function StatusHistory({ history }: Props) {
	const titleId = useId();
	return (
		<section className={styles.history} aria-labelledby={titleId}>
			<h3 id={titleId} className={styles.title}>
				Status history
			</h3>
			{history.length === 0 ? (
				<p className={styles.none}>No status changes recorded.</p>
			) : (
				<ol className={styles.list}>
					{history.map((change, i) => {
						const label = STATUS_LABELS[change.status] ?? change.status;
						return (
							// Entries never change or move, so their place in the list is a stable key.
							// biome-ignore lint/suspicious/noArrayIndexKey: the history is append-only
							<li key={i} className={styles.entry}>
								<span className={styles.status}>{label}</span>
								<time className={styles.date} dateTime={change.date}>
									{change.date}
								</time>
								{change.closingReasonText !== "" && (
									<span className={styles.reason}>{change.closingReasonText}</span>
								)}
							</li>
						);
					})}
				</ol>
			)}
		</section>
	);
}
