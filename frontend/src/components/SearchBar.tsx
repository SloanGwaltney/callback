import styles from "./SearchBar.module.css";

type Props = {
	value: string;
	onChange: (value: string) => void;
};

// SearchBar searches the board by role and company. The search is the app's state, not saved, so
// it starts empty each time the app opens.
// @scenario [#search-by-role] [#search-by-company] [#search-partial-any-case] [#search-ignores-other-details] [#search-blank-shows-all] [#clear-search]
export function SearchBar({ value, onChange }: Props) {
	return (
		<search className={styles.search}>
			<input
				type="search"
				className={styles.input}
				aria-label="Search jobs"
				placeholder="Search by role or company"
				value={value}
				onChange={(e) => onChange(e.target.value)}
				onKeyDown={(e) => {
					if (e.key === "Escape") {
						onChange("");
					}
				}}
			/>
			{value !== "" && (
				<button
					type="button"
					className={styles.clear}
					aria-label="Clear search"
					title="Clear search"
					onClick={() => onChange("")}
				>
					×
				</button>
			)}
		</search>
	);
}
