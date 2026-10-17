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
// @scenario [#set-next-action] [#set-next-action-no-date] [#next-action-needs-text] [#next-action-past-date] [#next-action-any-status] [#edit-next-action] [#clear-next-action-date] [#clear-next-action] [#view-next-action]
export function NextActionPanel({ nextAction, onSet, onClear }: Props) {
	const [editing, setEditing] = useState(false);
	const { error, run } = useAction();
	const titleId = useId();

	return (
		<section className={styles.panel} aria-labelledby={titleId}>
			<div className={styles.row}>
				<h3 id={titleId} className={styles.title}>
					Next action
				</h3>
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
					<>
						<p className={styles.current}>
							<span className={styles.text}>{nextAction.text}</span>
							<span className={styles.when}>
								<span>
									{nextAction.expectedDate === ""
										? "No expected date"
										: `Expected ${nextAction.expectedDate}`}
								</span>
								{nextAction.expectedDate !== "" && (
									<button
										type="button"
										className={styles.clearDate}
										aria-label="Clear date"
										title="Clear the expected date"
										onClick={() => run(() => onSet(nextAction.text, ""))}
									>
										×
									</button>
								)}
							</span>
							{nextAction.due !== "" && (
								<span className={nextAction.due === DUE_OVERDUE ? styles.overdue : styles.dueToday}>
									{DUE_LABELS[nextAction.due] ?? nextAction.due}
								</span>
							)}
						</p>
						<div className={styles.buttons}>
							<button type="button" className={styles.button} onClick={() => setEditing(true)}>
								Edit
							</button>
							<button
								type="button"
								className={styles.button}
								aria-label="Clear next action"
								title="Clear the next action once it's done"
								onClick={() => run(onClear)}
							>
								Clear
							</button>
						</div>
					</>
				) : (
					<>
						<p className={styles.none}>No next action.</p>
						<div className={styles.buttons}>
							<button type="button" className={styles.button} onClick={() => setEditing(true)}>
								Set next action
							</button>
						</div>
					</>
				)}
			</div>
			<ErrorMessage message={error} />
		</section>
	);
}

type FormProps = {
	initial: NextAction | null;
	onSave: (text: string, expectedDate: string) => Promise<void>;
	onCancel: () => void;
};

// NextActionForm writes the next action and picks its expected date, on the panel's one line. The
// backend checks what's written; when it refuses, the form stays with what was entered and says why.
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
		<form className={styles.form} onSubmit={submit} noValidate>
			<div className={styles.row}>
				<label htmlFor={`${id}-text`} className={styles.srOnly}>
					What's next (required)
				</label>
				<input
					id={`${id}-text`}
					className={`${styles.input} ${styles.what}`}
					// The form doesn't use the browser's validation (noValidate); this tells
					// assistive technology the field is needed, and the backend checks it.
					required
					placeholder="What's next, like waiting for a call back"
					value={text}
					onChange={(e) => setText(e.target.value)}
				/>
				<label htmlFor={`${id}-date`} className={styles.srOnly}>
					Expected on
				</label>
				<input
					id={`${id}-date`}
					className={`${styles.input} ${styles.date}`}
					type="date"
					title="Expected on"
					value={date}
					onChange={(e) => setDate(e.target.value)}
				/>
				<div className={styles.buttons}>
					<button type="button" className={styles.button} onClick={onCancel}>
						Cancel
					</button>
					<button type="submit" className={styles.primary} aria-label="Save next action">
						Save
					</button>
				</div>
			</div>
			<ErrorMessage message={error} />
		</form>
	);
}
