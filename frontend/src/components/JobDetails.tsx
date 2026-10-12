import { type FormEvent, useState } from "react";
import { errorMessage } from "../hooks/errors";
import type { Details, Job } from "../hooks/types";
import { useJob } from "../hooks/useJob";
import { STATUS_CLOSED, STATUS_LABELS } from "../jobs/constants";
import { ClosingReasonDialog } from "./ClosingReasonDialog";
import { Dialog } from "./Dialog";
import { DocumentList } from "./DocumentList";
import styles from "./JobDetails.module.css";
import { JobFields } from "./JobFields";

type Props = {
	jobId: number;
	onClose: () => void;
	// onChanged runs after anything on the job changes, so the board can show it.
	onChanged: () => void;
	// onDelete deletes the job. The dialog asks for confirmation first.
	onDelete: (id: number) => Promise<void>;
};

// JobDetails shows everything about a job when its card is opened, and edits it.
// @scenario [features/job-cards.feature#view-job-details] View a job's full details
// @scenario [features/job-board.feature#change-closing-reason] Change the closing reason of a closed job
export function JobDetails({ jobId, onClose, onChanged, onDelete }: Props) {
	const { job, error, reload, update, setClosingReason, openLink } = useJob(jobId);
	const [changingReason, setChangingReason] = useState(false);

	const title = job ? `${job.role} at ${job.company}` : "Job";
	return (
		<Dialog title={title} onClose={onClose}>
			{error !== "" && (
				<p role="alert" className={styles.error}>
					{error}
				</p>
			)}
			{job && (
				<>
					<p className={styles.status}>
						In <strong>{STATUS_LABELS[job.status] ?? job.status}</strong>
						{job.status === STATUS_CLOSED && (
							<>
								{" "}
								— closed: {job.closingReasonText}{" "}
								<button
									type="button"
									className={styles.button}
									onClick={() => setChangingReason(true)}
								>
									Change reason
								</button>
							</>
						)}
					</p>
					<DetailsForm
						job={job}
						onSave={async (details) => {
							await update(details);
							onChanged();
						}}
						onOpenLink={openLink}
						onDelete={() => onDelete(job.id)}
						onClose={onClose}
					/>
					<DocumentList
						jobId={job.id}
						documents={job.documents}
						onChanged={() => {
							void reload();
							onChanged();
						}}
					/>
					{changingReason && (
						<ClosingReasonDialog
							title="Change the closing reason"
							submitLabel="Save reason"
							initialReason={job.closingReason}
							initialOther={job.closingReasonOther}
							onChoose={async (reason, other) => {
								await setClosingReason(reason, other);
								setChangingReason(false);
								onChanged();
							}}
							onCancel={() => setChangingReason(false)}
						/>
					)}
				</>
			)}
		</Dialog>
	);
}

type FormProps = {
	job: Job;
	onSave: (details: Details) => Promise<void>;
	onOpenLink: () => Promise<void>;
	onDelete: () => Promise<void>;
	onClose: () => void;
};

// DetailsForm edits a job's details. It's keyed to the loaded job, so it starts from what's saved.
// @scenario [features/job-cards.feature#edit-job-details] Edit a job's details
// @scenario [features/job-cards.feature#salary-free-text] Salary is written however the posting gives it
// @scenario [features/job-cards.feature#add-notes] Keep notes on a job
// @scenario [features/job-cards.feature#edit-notes] Add to existing notes
// @scenario [features/job-cards.feature#open-job-link] Open a job's posting link
// @scenario [features/job-cards.feature#invalid-job-link] A job link must be a web address
// @scenario [features/job-cards.feature#set-date-applied] Set the date applied
// @scenario [features/job-cards.feature#date-applied-any-status] The date applied can be set in any status
// @scenario [features/job-cards.feature#clear-date-applied] Clear the date applied
// @scenario [features/job-cards.feature#future-date-applied] The date applied can't be in the future
// @scenario [features/job-cards.feature#delete-job] Delete a job after confirming
// @scenario [features/job-cards.feature#cancel-delete-job] Cancel deleting a job
function DetailsForm({ job, onSave, onOpenLink, onDelete, onClose }: FormProps) {
	const [details, setDetails] = useState<Details>({
		company: job.company,
		role: job.role,
		description: job.description,
		link: job.link,
		salary: job.salary,
		location: job.location,
		notes: job.notes,
		dateApplied: job.dateApplied,
		// The backend sends null for no rating; the form uses undefined.
		interest: job.interest ?? undefined,
	});
	const [error, setError] = useState("");
	const [saved, setSaved] = useState(false);
	const [confirmingDelete, setConfirmingDelete] = useState(false);

	const submit = async (e: FormEvent) => {
		e.preventDefault();
		setError("");
		setSaved(false);
		try {
			await onSave(details);
			setSaved(true);
		} catch (err) {
			// The form keeps what was entered, and nothing is saved.
			setError(errorMessage(err));
		}
	};

	const openLink = async () => {
		setError("");
		try {
			await onOpenLink();
		} catch (err) {
			setError(errorMessage(err));
		}
	};

	return (
		<form onSubmit={submit} noValidate>
			<JobFields
				details={details}
				onChange={(d) => {
					setDetails(d);
					setSaved(false);
				}}
			/>
			{error !== "" && (
				<p role="alert" className={styles.error}>
					{error}
				</p>
			)}
			{saved && (
				<p role="status" className={styles.saved}>
					Saved.
				</p>
			)}
			<div className={styles.actions}>
				<button type="button" className={styles.danger} onClick={() => setConfirmingDelete(true)}>
					Delete job
				</button>
				<button
					type="button"
					className={styles.button}
					disabled={job.link === ""}
					onClick={openLink}
				>
					Open link
				</button>
				<span className={styles.spacer} />
				<button type="button" className={styles.button} onClick={onClose}>
					Close
				</button>
				<button type="submit" className={styles.primary}>
					Save
				</button>
			</div>
			{confirmingDelete && (
				<ConfirmDelete
					name={`${job.role} at ${job.company}`}
					onConfirm={onDelete}
					onCancel={() => setConfirmingDelete(false)}
				/>
			)}
		</form>
	);
}

type ConfirmProps = {
	name: string;
	onConfirm: () => Promise<void>;
	onCancel: () => void;
};

function ConfirmDelete({ name, onConfirm, onCancel }: ConfirmProps) {
	const [error, setError] = useState("");
	const confirm = async () => {
		setError("");
		try {
			await onConfirm();
		} catch (err) {
			setError(errorMessage(err));
		}
	};
	return (
		<Dialog title={`Delete ${name}?`} onClose={onCancel}>
			<p className={styles.warning}>
				The job and its attached documents will be removed. This can't be undone.
			</p>
			{error !== "" && (
				<p role="alert" className={styles.error}>
					{error}
				</p>
			)}
			<div className={styles.actions}>
				<button type="button" className={styles.button} onClick={onCancel}>
					Cancel
				</button>
				<button type="button" className={styles.danger} onClick={confirm}>
					Delete
				</button>
			</div>
		</Dialog>
	);
}
