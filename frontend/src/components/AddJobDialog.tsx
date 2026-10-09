import { type FormEvent, useId, useState } from "react";
import { errorMessage } from "../hooks/errors";
import type { Details } from "../hooks/types";
import { STATUS_SAVED } from "../jobs/constants";
import styles from "./AddJobDialog.module.css";
import { Dialog } from "./Dialog";
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
// @scenario [features/job-cards.feature#add-job] Add a job with all its details
// @scenario [features/job-cards.feature#add-job-minimum] Add a job with only a company and a role
// @scenario [features/job-cards.feature#add-job-in-column] Add a job directly to a status column
// @scenario [features/job-cards.feature#add-job-not-closed] A job can't be added straight to Closed
// @scenario [features/job-cards.feature#add-job-missing-required] A job can't be saved without a company or a role
// @scenario [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
export function AddJobDialog({ columns, initialStatus, onAdd, onCancel }: Props) {
	const [details, setDetails] = useState<Details>(emptyDetails);
	// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Saved" column
	const [status, setStatus] = useState(initialStatus || STATUS_SAVED);
	const [error, setError] = useState("");
	const [saving, setSaving] = useState(false);
	const columnId = useId();

	const submit = async (e: FormEvent) => {
		e.preventDefault();
		setSaving(true);
		try {
			// @when [features/job-cards.feature] I add a job with:
			// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" and nothing else
			// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" to the "Applied" column
			// @when [features/job-board.feature] I add a job at "Acme" to the "Saved" column
			// @when [features/job-cards.feature] I try to add a job with company "Acme" and role "Software Engineer" to the "Closed" column
			// @when [features/job-cards.feature] I try to add a job with company "<company>" and role "<role>"
			await onAdd(details, status);
		} catch (err) {
			// The dialog stays open with what was entered.
			// @then [features/job-cards.feature] the job is not added
			// @and [features/job-cards.feature] I am told to add the job to another column and move it to Closed
			// @and [features/job-cards.feature] I am told that the <missing> is required
			setError(errorMessage(err));
			setSaving(false);
		}
	};

	return (
		<Dialog title="Add a job" onClose={onCancel}>
			<form onSubmit={submit} noValidate>
				<JobFields details={details} onChange={setDetails} />
				<div className={styles.column}>
					<label htmlFor={columnId}>Column</label>
					{/* @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Applied" column */}
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
				{error !== "" && (
					<p role="alert" className={styles.error}>
						{error}
					</p>
				)}
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
