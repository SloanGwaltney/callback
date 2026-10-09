import { type FormEvent, useState } from "react";
import { errorMessage } from "../hooks/errors";
import { CLOSING_REASONS, REASON_OTHER } from "../jobs/constants";
import styles from "./ClosingReasonDialog.module.css";
import { Dialog } from "./Dialog";

type Props = {
	title: string;
	submitLabel: string;
	initialReason?: string;
	initialOther?: string;
	// onChoose saves the reason. If it rejects, its message is shown and the dialog stays open.
	onChoose: (reason: string, other: string) => Promise<void>;
	onCancel: () => void;
};

// ClosingReasonDialog asks why a job was closed: one of the closing reasons, or Other with the
// user's own words.
export function ClosingReasonDialog({
	title,
	submitLabel,
	initialReason = "",
	initialOther = "",
	onChoose,
	onCancel,
}: Props) {
	const [reason, setReason] = useState(initialReason);
	const [other, setOther] = useState(initialOther);
	const [error, setError] = useState("");
	const [saving, setSaving] = useState(false);

	const submit = async (e: FormEvent) => {
		e.preventDefault();
		if (reason === "") {
			setError("Choose why the job was closed.");
			return;
		}
		// @and [features/job-board.feature] I choose the closing reason "Other" without writing a reason
		if (reason === REASON_OTHER && other.trim() === "") {
			// Nothing is saved, so the card stays where it was.
			// @then [features/job-board.feature#close-other-needs-text] the card stays in the "Interviewing" column
			// @and [features/job-board.feature] I am told to write why the job was closed
			setError("Write why the job was closed.");
			return;
		}
		setSaving(true);
		try {
			// @and [features/job-board.feature] I choose the closing reason "<reason>"
			// @and [features/job-board.feature] I choose the closing reason "Other" and write "Company went through layoffs"
			// @when [features/job-board.feature] I change the closing reason to "Rejected"
			await onChoose(reason, reason === REASON_OTHER ? other : "");
		} catch (err) {
			setError(errorMessage(err));
			setSaving(false);
		}
	};

	return (
		<Dialog title={title} onClose={onCancel}>
			<form onSubmit={submit} noValidate>
				<fieldset className={styles.reasons}>
					<legend className={styles.legend}>Why was the job closed?</legend>
					{CLOSING_REASONS.map((r) => (
						<label key={r.value} className={styles.reason}>
							<input
								type="radio"
								name="closing-reason"
								value={r.value}
								checked={reason === r.value}
								onChange={() => setReason(r.value)}
							/>
							{r.label}
						</label>
					))}
				</fieldset>
				{reason === REASON_OTHER && (
					<label className={styles.field}>
						What happened?
						<input
							className={styles.input}
							value={other}
							onChange={(e) => setOther(e.target.value)}
						/>
					</label>
				)}
				{error !== "" && (
					<p role="alert" className={styles.error}>
						{error}
					</p>
				)}
				<div className={styles.actions}>
					{/* @but [features/job-board.feature] I cancel instead of choosing a closing reason */}
					<button type="button" className={styles.button} onClick={onCancel}>
						Cancel
					</button>
					<button type="submit" className={styles.primary} disabled={saving}>
						{submitLabel}
					</button>
				</div>
			</form>
		</Dialog>
	);
}
