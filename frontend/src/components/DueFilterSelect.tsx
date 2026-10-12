import { DUE_FILTERS, type DueFilter } from "../board/boardView";
import styles from "./DueFilterSelect.module.css";

type Props = {
	value: DueFilter;
	onChange: (value: DueFilter) => void;
};

// DueFilterSelect filters the board to overdue jobs, jobs that aren't overdue, or every job. Like
// the search, the filter is the app's state, not saved, so every job shows each time the app opens.
// @scenario [features/job-next-action.feature#filter-overdue] Show only overdue jobs
// @scenario [features/job-next-action.feature#filter-not-overdue] Show only jobs that aren't overdue
// @scenario [features/job-next-action.feature#filter-show-all] Show every job again
export function DueFilterSelect({ value, onChange }: Props) {
	return (
		<label className={styles.filter}>
			<span className={styles.label}>Show</span>
			<select
				className={styles.select}
				value={value}
				onChange={(e) => onChange(e.target.value as DueFilter)}
			>
				{DUE_FILTERS.map((f) => (
					<option key={f.value} value={f.value}>
						{f.label}
					</option>
				))}
			</select>
		</label>
	);
}
