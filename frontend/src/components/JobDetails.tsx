import { type FormEvent, useId, useState } from "react";
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
	// The details form is submitted from Save in the dialog's footer, which also says how saving went.
	const formId = useId();
	const saving = useAction();
	const [saved, setSaved] = useState(false);
	// recordChanged reloads the job after one of its records changes, and lets the board know.
	const recordChanged = () => {
		void reload();
		onChanged();
	};

	const title = job ? `${job.role} at ${job.company}` : "Job";
	return (
		<Dialog
			title={title}
			wide
			onClose={onClose}
			footer={
				job && (
					<DetailsActions
						job={job}
						formId={formId}
						saved={saved}
						saveError={saving.error}
						onOpenLink={openLink}
						onDelete={() => onDelete(job.id)}
					/>
				)
			}
		>
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
						id={formId}
						onSave={async (details) => {
							setSaved(false);
							// When saving fails, the form keeps what was entered, and nothing is saved.
							setSaved(
								await saving.run(async () => {
									await update(details);
									onChanged();
								}),
							);
						}}
						onEdit={() => setSaved(false)}
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
	// id lets Save in the dialog's footer submit the form from outside it.
	id: string;
	onSave: (details: Details) => Promise<void>;
	// onEdit runs on every change, so "Saved." goes once there's something new to save.
	onEdit: () => void;
};

// DetailsForm edits a job's details. It's keyed to the loaded job, so it starts from what's saved.
// @scenario [#edit-job-details] [#salary-free-text] [#add-notes] [#edit-notes] [#invalid-job-link] [#set-date-applied] [#date-applied-any-status] [#clear-date-applied] [#future-date-applied]
function DetailsForm({ job, id, onSave, onEdit }: FormProps) {
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

	const submit = (e: FormEvent) => {
		e.preventDefault();
		void onSave(details);
	};

	return (
		<form id={id} onSubmit={submit} noValidate>
			<JobFields
				details={details}
				onChange={(d) => {
					setDetails(d);
					onEdit();
				}}
			/>
		</form>
	);
}

type ActionsProps = {
	job: Job;
	// formId is the details form's, which Save submits.
	formId: string;
	saved: boolean;
	saveError: string;
	onOpenLink: () => Promise<void>;
	onDelete: () => Promise<void>;
};

// DetailsActions are the job's buttons in the dialog's footer, which stays in view as the dialog
// scrolls: deleting the job, opening its link, and saving its details.
// @scenario [#open-job-link] [#delete-job] [#cancel-delete-job]
function DetailsActions({ job, formId, saved, saveError, onOpenLink, onDelete }: ActionsProps) {
	const { error, run } = useAction();
	const [confirmingDelete, setConfirmingDelete] = useState(false);

	return (
		<>
			<ErrorMessage message={saveError || error} />
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
				{saved && (
					<p role="status" className={styles.saved}>
						Saved.
					</p>
				)}
				<button type="submit" form={formId} className={styles.primary}>
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
		</>
	);
}
