import { type FormEvent, useId, useState } from "react";
import type { Details } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { STATUS_SAVED } from "../jobs/constants";
import styles from "./AddJobDialog.module.css";
import { Dialog } from "./Dialog";
import { ErrorMessage } from "./ErrorMessage";
import { emptyDetails, JobFields } from "./JobFields";

type Props = {
	// columns are the board's columns, offered as where the job goes.
	columns: { status: string; label: string }[];
	// initialStatus is the column chosen first, or "" for Saved.
	initialStatus: string;
	// onAdd adds the job. If it rejects, its message is shown and nothing is added.
	onAdd: (details: Details, status: string) => Promise<void>;
	onCancel: () => void;
};

// AddJobDialog adds a job to a column on the board.
// @scenario [#add-job] [#add-job-minimum] [#add-job-in-column] [#add-job-not-closed] [#add-job-missing-required] [#new-card-at-top]
export function AddJobDialog({ columns, initialStatus, onAdd, onCancel }: Props) {
	const [details, setDetails] = useState<Details>(emptyDetails);
	const [status, setStatus] = useState(initialStatus || STATUS_SAVED);
	const { error, run } = useAction();
	const [saving, setSaving] = useState(false);
	const columnId = useId();

	const submit = async (e: FormEvent) => {
		e.preventDefault();
		setSaving(true);
		// When adding fails, the dialog stays open with what was entered.
		if (!(await run(() => onAdd(details, status)))) {
			setSaving(false);
		}
	};

	return (
		<Dialog title="Add a job" onClose={onCancel}>
			<form onSubmit={submit} noValidate>
				<JobFields details={details} onChange={setDetails} />
				<div className={styles.column}>
					<label htmlFor={columnId}>Column</label>
					<select
						id={columnId}
						className={styles.select}
						value={status}
						onChange={(e) => setStatus(e.target.value)}
					>
						{columns.map((column) => (
							<option key={column.status} value={column.status}>
								{column.label}
							</option>
						))}
					</select>
				</div>
				<ErrorMessage message={error} />
				<div className={styles.actions}>
					<button type="button" className={styles.button} onClick={onCancel}>
						Cancel
					</button>
					<button type="submit" className={styles.primary} disabled={saving}>
						Add job
					</button>
				</div>
			</form>
		</Dialog>
	);
}
