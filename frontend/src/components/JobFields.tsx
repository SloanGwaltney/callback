import { useId } from "react";
import type { Details } from "../hooks/types";
import { today } from "../jobs/constants";
import styles from "./JobFields.module.css";

type Props = {
	details: Details;
	onChange: (details: Details) => void;
};

export const emptyDetails: Details = {
	company: "",
	role: "",
	description: "",
	link: "",
	salary: "",
	location: "",
	notes: "",
	dateApplied: "",
};

// JobFields are the inputs for a job's details, shared by adding a job and editing one. The
// backend checks what's entered; its messages are shown by the form these sit in.
export function JobFields({ details, onChange }: Props) {
	const id = useId();
	const set = (field: keyof Details) => (value: string) => onChange({ ...details, [field]: value });

	return (
		<div className={styles.fields}>
			<TextField
				id={`${id}-company`}
				label="Company"
				required
				value={details.company}
				onChange={set("company")}
			/>
			<TextField
				id={`${id}-role`}
				label="Role"
				required
				value={details.role}
				onChange={set("role")}
			/>
			<TextField
				id={`${id}-location`}
				label="Location"
				value={details.location}
				onChange={set("location")}
			/>
			{/* Salary is free text, written however the posting gives it. */}
			<TextField
				id={`${id}-salary`}
				label="Salary"
				value={details.salary}
				onChange={set("salary")}
			/>
			<TextField
				id={`${id}-link`}
				label="Link"
				type="url"
				placeholder="https://"
				value={details.link}
				onChange={set("link")}
				wide
			/>
			<div className={styles.field}>
				<label htmlFor={`${id}-date`}>Date applied</label>
				<div className={styles.dateRow}>
					<input
						id={`${id}-date`}
						className={styles.input}
						type="date"
						max={today()}
						value={details.dateApplied}
						onChange={(e) => onChange({ ...details, dateApplied: e.target.value })}
					/>
					{details.dateApplied !== "" && (
						<button
							type="button"
							className={styles.button}
							onClick={() => onChange({ ...details, dateApplied: "" })}
						>
							Clear
						</button>
					)}
				</div>
			</div>
			<div className={`${styles.field} ${styles.wide}`}>
				<label htmlFor={`${id}-description`}>Job description</label>
				<textarea
					id={`${id}-description`}
					className={styles.textarea}
					value={details.description}
					onChange={(e) => onChange({ ...details, description: e.target.value })}
				/>
			</div>
			<div className={`${styles.field} ${styles.wide}`}>
				<label htmlFor={`${id}-notes`}>Notes</label>
				<textarea
					id={`${id}-notes`}
					className={styles.textarea}
					value={details.notes}
					onChange={(e) => onChange({ ...details, notes: e.target.value })}
				/>
			</div>
		</div>
	);
}

type TextFieldProps = {
	id: string;
	label: string;
	value: string;
	onChange: (value: string) => void;
	type?: "text" | "url";
	placeholder?: string;
	required?: boolean;
	wide?: boolean;
};

function TextField({
	id,
	label,
	value,
	onChange,
	type = "text",
	placeholder,
	required,
	wide,
}: TextFieldProps) {
	return (
		<div className={wide ? `${styles.field} ${styles.wide}` : styles.field}>
			<label htmlFor={id}>
				{label}
				{required && <span className={styles.required}> (required)</span>}
			</label>
			<input
				id={id}
				className={styles.input}
				type={type}
				placeholder={placeholder}
				aria-required={required}
				value={value}
				onChange={(e) => onChange(e.target.value)}
			/>
		</div>
	);
}
