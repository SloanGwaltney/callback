import styles from "./SearchBar.module.css";

type Props = {
	value: string;
	onChange: (value: string) => void;
};

// SearchBar searches the board by role and company. The search is the app's state, not saved, so
// it starts empty each time the app opens.
// @scenario [features/board-search.feature#search-by-role] Search by role
// @scenario [features/board-search.feature#search-by-company] Search by company
// @scenario [features/board-search.feature#search-partial-any-case] Search matches part of a word in any letter case
// @scenario [features/board-search.feature#search-ignores-other-details] Search doesn't match a job's other details
// @scenario [features/board-search.feature#search-blank-shows-all] Searching for only spaces shows every job
// @scenario [features/board-search.feature#clear-search] Clearing the search shows every job again
export function SearchBar({ value, onChange }: Props) {
	return (
		<search className={styles.search}>
			{/* @when [features/board-search.feature] I search the board for "Engineer" */}
			{/* @when [features/board-search.feature] I search the board for "Globex" */}
			{/* @when [features/board-search.feature] I search the board for "<text>" */}
			{/* @when [features/board-search.feature] I search the board for "Acme" */}
			{/* @when [features/board-search.feature] I search the board for "   " */}
			{/* @when [features/board-search.feature] I search the board for "Initech" */}
			{/* @and [features/board-search.feature] I have searched the board for "Acme" */}
			{/* @when [features/board-pagination.feature] I search the board for "Acme" */}
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
				// @when [features/board-search.feature] I clear the search
				// @and [features/board-search.feature] I clear the search
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
