import { type FormEvent, useId, useState } from "react";
import type { Interview, InterviewDetails } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { useInterviews } from "../hooks/useInterviews";
import { formatTime, INTERVIEW_KINDS, interviewKindLabel } from "../jobs/constants";
import { ConfirmDialog } from "./ConfirmDialog";
import { Dialog } from "./Dialog";
import { ErrorMessage } from "./ErrorMessage";
import styles from "./InterviewList.module.css";

type Props = {
	jobId: number;
	// interviews are from earliest to latest, as the backend lists them.
	interviews: Interview[];
	// onChanged runs after an interview is added, edited, or removed.
	onChanged: () => void;
};

const emptyInterview: InterviewDetails = { date: "", time: "", kind: "", notes: "" };

// when describes an interview's date and, when it has one, its time: "2026-10-14 at 2:00 PM".
export function when(interview: { date: string; time: string }): string {
	return interview.time === ""
		? interview.date
		: `${interview.date} at ${formatTime(interview.time)}`;
}

// InterviewList shows a job's interviews, marking the ones that have passed, and adds, edits, and
// removes them, asking before a removal.
// @scenario [#add-interview] [#add-interview-minimum] [#interview-kinds] [#interview-missing-required] [#interview-past-date] [#interview-any-status] [#view-interviews] [#interviews-same-day-by-time] [#past-interviews-marked] [#no-interviews] [#edit-interview] [#remove-interview] [#cancel-remove-interview]
export function InterviewList({ jobId, interviews, onChanged }: Props) {
	const { add, update, remove } = useInterviews();
	// editing is the interview being edited, "new" while adding one, or null.
	const [editing, setEditing] = useState<Interview | "new" | null>(null);
	const [removing, setRemoving] = useState<Interview | null>(null);
	const titleId = useId();

	return (
		<section className={styles.interviews} aria-labelledby={titleId}>
			<div className={styles.header}>
				<h3 id={titleId} className={styles.title}>
					Interviews
				</h3>
				<button type="button" className={styles.button} onClick={() => setEditing("new")}>
					Add interview
				</button>
			</div>
			{interviews.length === 0 ? (
				<p className={styles.none}>No interviews yet.</p>
			) : (
				<ul className={styles.list}>
					{interviews.map((interview) => {
						const name = `${interviewKindLabel(interview.kind)} interview on ${when(interview)}`;
						return (
							<li
								key={interview.id}
								className={`${styles.interview} ${interview.past ? styles.past : ""}`}
								aria-label={name}
							>
								<div className={styles.heading}>
									<span className={styles.summary}>
										<span className={styles.kind}>{interviewKindLabel(interview.kind)}</span>
										<span className={styles.when}>{when(interview)}</span>
										<span className={interview.past ? styles.pastBadge : styles.upcomingBadge}>
											{interview.past ? "Past" : "Upcoming"}
										</span>
									</span>
									<button
										type="button"
										className={styles.button}
										aria-label={`Edit ${name}`}
										onClick={() => setEditing(interview)}
									>
										Edit
									</button>
									<button
										type="button"
										className={styles.button}
										aria-label={`Remove ${name}`}
										onClick={() => setRemoving(interview)}
									>
										Remove
									</button>
								</div>
								{interview.notes !== "" && <p className={styles.notes}>{interview.notes}</p>}
							</li>
						);
					})}
				</ul>
			)}
			{editing === "new" && (
				<InterviewForm
					title="Add an interview"
					initial={emptyInterview}
					onSave={async (details) => {
						await add(jobId, details);
						setEditing(null);
						onChanged();
					}}
					onCancel={() => setEditing(null)}
				/>
			)}
			{editing !== null && editing !== "new" && (
				<InterviewForm
					title={`Edit the ${interviewKindLabel(editing.kind)} interview`}
					initial={editing}
					onSave={async (details) => {
						await update(editing.id, details);
						setEditing(null);
						onChanged();
					}}
					onCancel={() => setEditing(null)}
				/>
			)}
			{removing && (
				<ConfirmDialog
					title="Remove this interview?"
					message={`The ${interviewKindLabel(removing.kind)} interview on ${when(removing)} will be removed from this job.`}
					confirmLabel="Remove"
					onConfirm={async () => {
						await remove(removing.id);
						setRemoving(null);
						onChanged();
					}}
					onCancel={() => setRemoving(null)}
				/>
			)}
		</section>
	);
}

type FormProps = {
	title: string;
	initial: InterviewDetails;
	// onSave saves the interview, or rejects with the backend's message.
	onSave: (details: InterviewDetails) => Promise<void>;
	onCancel: () => void;
};

// InterviewForm adds or edits an interview in a dialog. The backend checks what's entered; when it
// refuses, the dialog stays open with what was entered and says why.
function InterviewForm({ title, initial, onSave, onCancel }: FormProps) {
	const [details, setDetails] = useState<InterviewDetails>({
		date: initial.date,
		time: initial.time,
		kind: initial.kind,
		notes: initial.notes,
	});
	const { error, run } = useAction();
	const id = useId();
	const set = (field: keyof InterviewDetails) => (value: string) =>
		setDetails({ ...details, [field]: value });

	const submit = (e: FormEvent) => {
		e.preventDefault();
		void run(() => onSave(details));
	};

	return (
		<Dialog title={title} onClose={onCancel}>
			<form onSubmit={submit} noValidate>
				<div className={styles.fields}>
					<div className={styles.field}>
						<label htmlFor={`${id}-date`}>
							Date<span className={styles.required}> (required)</span>
						</label>
						<input
							id={`${id}-date`}
							className={styles.input}
							type="date"
							// The form doesn't use the browser's validation (noValidate); this tells
							// assistive technology the field is needed, and the backend checks it.
							required
							value={details.date}
							onChange={(e) => set("date")(e.target.value)}
						/>
					</div>
					<div className={styles.field}>
						<label htmlFor={`${id}-time`}>Time</label>
						<input
							id={`${id}-time`}
							className={styles.input}
							type="time"
							value={details.time}
							onChange={(e) => set("time")(e.target.value)}
						/>
					</div>
					<div className={`${styles.field} ${styles.wide}`}>
						<label htmlFor={`${id}-kind`}>
							Kind<span className={styles.required}> (required)</span>
						</label>
						<select
							id={`${id}-kind`}
							className={styles.input}
							aria-required
							value={details.kind}
							onChange={(e) => set("kind")(e.target.value)}
						>
							<option value="">Choose…</option>
							{INTERVIEW_KINDS.map((k) => (
								<option key={k.value} value={k.value}>
									{k.label}
								</option>
							))}
						</select>
					</div>
					<div className={`${styles.field} ${styles.wide}`}>
						<label htmlFor={`${id}-notes`}>Notes</label>
						<textarea
							id={`${id}-notes`}
							className={styles.textarea}
							value={details.notes}
							onChange={(e) => set("notes")(e.target.value)}
						/>
					</div>
				</div>
				<ErrorMessage message={error} />
				<div className={styles.actions}>
					<button type="button" className={styles.button} onClick={onCancel}>
						Cancel
					</button>
					<button type="submit" className={styles.primary}>
						Save interview
					</button>
				</div>
			</form>
		</Dialog>
	);
}
