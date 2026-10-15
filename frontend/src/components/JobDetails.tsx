import { type FormEvent, useState } from "react";
import type { Details, Job } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { useJob } from "../hooks/useJob";
import { STATUS_CLOSED, STATUS_LABELS } from "../jobs/constants";
import { ClosingReasonDialog } from "./ClosingReasonDialog";
import { ConfirmDialog } from "./ConfirmDialog";
import { ContactList } from "./ContactList";
import { Dialog } from "./Dialog";
import { DocumentList } from "./DocumentList";
import { ErrorMessage } from "./ErrorMessage";
import { InterviewList } from "./InterviewList";
import styles from "./JobDetails.module.css";
import { JobFields } from "./JobFields";
import { NextActionPanel } from "./NextActionPanel";
import { StatusHistory } from "./StatusHistory";

type Props = {
	jobId: number;
	onClose: () => void;
	// onChanged runs after anything on the job changes, so the board can show it.
	onChanged: () => void;
	// onDelete deletes the job. The dialog asks for confirmation first.
	onDelete: (id: number) => Promise<void>;
};

// JobDetails shows everything about a job when its card is opened, including its next action,
// interviews, contacts, documents, and status history, and edits it.
// @scenario [#view-job-details] [#change-closing-reason]
export function JobDetails({ jobId, onClose, onChanged, onDelete }: Props) {
	const { job, error, reload, update, setClosingReason, setNextAction, clearNextAction, openLink } =
		useJob(jobId);
	const [changingReason, setChangingReason] = useState(false);
	// recordChanged reloads the job after one of its records changes, and lets the board know.
	const recordChanged = () => {
		void reload();
		onChanged();
	};

	const title = job ? `${job.role} at ${job.company}` : "Job";
	return (
		<Dialog title={title} onClose={onClose}>
			<ErrorMessage message={error} />
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
					<NextActionPanel
						nextAction={job.nextAction ?? null}
						onSet={async (text, expectedDate) => {
							await setNextAction(text, expectedDate);
							onChanged();
						}}
						onClear={async () => {
							await clearNextAction();
							onChanged();
						}}
					/>
					<InterviewList jobId={job.id} interviews={job.interviews} onChanged={recordChanged} />
					<ContactList jobId={job.id} contacts={job.contacts} onChanged={recordChanged} />
					<DocumentList jobId={job.id} documents={job.documents} onChanged={recordChanged} />
					<StatusHistory history={job.statusHistory} />
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
// @scenario [#edit-job-details] [#salary-free-text] [#add-notes] [#edit-notes] [#open-job-link] [#invalid-job-link] [#set-date-applied] [#date-applied-any-status] [#clear-date-applied] [#future-date-applied] [#delete-job] [#cancel-delete-job]
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
	const { error, run } = useAction();
	const [saved, setSaved] = useState(false);
	const [confirmingDelete, setConfirmingDelete] = useState(false);

	const submit = async (e: FormEvent) => {
		e.preventDefault();
		setSaved(false);
		// When saving fails, the form keeps what was entered, and nothing is saved.
		setSaved(await run(() => onSave(details)));
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
			<ErrorMessage message={error} />
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
					onClick={() => run(onOpenLink)}
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
				<ConfirmDialog
					title={`Delete ${job.role} at ${job.company}?`}
					message="The job, its interviews, its contacts, its attached documents, and its status history will be removed. This can't be undone."
					confirmLabel="Delete"
					onConfirm={onDelete}
					onCancel={() => setConfirmingDelete(false)}
				/>
			)}
		</form>
	);
}
