import { type FormEvent, useId, useState } from "react";
import type { NextAction } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { DUE_LABELS, DUE_OVERDUE } from "../jobs/constants";
import { ErrorMessage } from "./ErrorMessage";
import styles from "./NextActionPanel.module.css";

type Props = {
	// nextAction is the job's next action, or null when it has none.
	nextAction: NextAction | null;
	// onSet saves the next action, with expectedDate "" for none, or rejects with the backend's
	// message.
	onSet: (text: string, expectedDate: string) => Promise<void>;
	onClear: () => Promise<void>;
};

// NextActionPanel shows what's next on a job and when it's expected, and sets, changes, and clears
// it. Setting one doesn't move the job, whatever its status.
// @scenario [features/job-next-action.feature#set-next-action] Set a next action with an expected date
// @scenario [features/job-next-action.feature#set-next-action-no-date] Set a next action without an expected date
// @scenario [features/job-next-action.feature#next-action-needs-text] A next action can't be saved without saying what it is
// @scenario [features/job-next-action.feature#next-action-past-date] The expected date can be in the past
// @scenario [features/job-next-action.feature#next-action-any-status] A next action can be set on a job in any status
// @scenario [features/job-next-action.feature#edit-next-action] Change the next action and its expected date
// @scenario [features/job-next-action.feature#clear-next-action-date] Clear only the expected date
// @scenario [features/job-next-action.feature#clear-next-action] Clear the next action once it's done
// @scenario [features/job-next-action.feature#view-next-action] See the next action when opening the card
export function NextActionPanel({ nextAction, onSet, onClear }: Props) {
	const [editing, setEditing] = useState(false);
	const { error, run } = useAction();
	const titleId = useId();

	return (
		<section className={styles.panel} aria-labelledby={titleId}>
			<div className={styles.header}>
				<h3 id={titleId} className={styles.title}>
					Next action
				</h3>
				{!editing && (
					<button type="button" className={styles.button} onClick={() => setEditing(true)}>
						{nextAction ? "Edit" : "Set next action"}
					</button>
				)}
			</div>
			{editing ? (
				<NextActionForm
					initial={nextAction}
					onSave={async (text, date) => {
						await onSet(text, date);
						setEditing(false);
					}}
					onCancel={() => setEditing(false)}
				/>
			) : nextAction ? (
				<div className={styles.current}>
					<p className={styles.text}>{nextAction.text}</p>
					<p className={styles.when}>
						<span>
							{nextAction.expectedDate === ""
								? "No expected date"
								: `Expected ${nextAction.expectedDate}`}
						</span>
						{nextAction.due !== "" && (
							<span className={nextAction.due === DUE_OVERDUE ? styles.overdue : styles.dueToday}>
								{DUE_LABELS[nextAction.due] ?? nextAction.due}
							</span>
						)}
					</p>
					<div className={styles.buttons}>
						{nextAction.expectedDate !== "" && (
							<button
								type="button"
								className={styles.button}
								onClick={() => run(() => onSet(nextAction.text, ""))}
							>
								Clear date
							</button>
						)}
						<button
							type="button"
							className={styles.button}
							title="Clear the next action once it's done"
							onClick={() => run(onClear)}
						>
							Clear next action
						</button>
					</div>
				</div>
			) : (
				<p className={styles.none}>No next action.</p>
			)}
			<ErrorMessage message={error} />
		</section>
	);
}

type FormProps = {
	initial: NextAction | null;
	onSave: (text: string, expectedDate: string) => Promise<void>;
	onCancel: () => void;
};

// NextActionForm writes the next action and picks its expected date. The backend checks what's
// written; when it refuses, the form stays with what was entered and says why.
function NextActionForm({ initial, onSave, onCancel }: FormProps) {
	const [text, setText] = useState(initial?.text ?? "");
	const [date, setDate] = useState(initial?.expectedDate ?? "");
	const { error, run } = useAction();
	const id = useId();

	const submit = (e: FormEvent) => {
		e.preventDefault();
		void run(() => onSave(text, date));
	};

	return (
		<form onSubmit={submit} noValidate>
			<div className={styles.fields}>
				<div className={`${styles.field} ${styles.wide}`}>
					<label htmlFor={`${id}-text`}>
						What's next<span className={styles.required}> (required)</span>
					</label>
					<input
						id={`${id}-text`}
						className={styles.input}
						// The form doesn't use the browser's validation (noValidate); this tells
						// assistive technology the field is needed, and the backend checks it.
						required
						placeholder="Waiting for a call back"
						value={text}
						onChange={(e) => setText(e.target.value)}
					/>
				</div>
				<div className={styles.field}>
					<label htmlFor={`${id}-date`}>Expected on</label>
					<input
						id={`${id}-date`}
						className={styles.input}
						type="date"
						value={date}
						onChange={(e) => setDate(e.target.value)}
					/>
				</div>
			</div>
			<ErrorMessage message={error} />
			<div className={styles.actions}>
				<button type="button" className={styles.button} onClick={onCancel}>
					Cancel
				</button>
				<button type="submit" className={styles.primary}>
					Save next action
				</button>
			</div>
		</form>
	);
}
