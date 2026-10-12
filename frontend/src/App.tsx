import { useState } from "react";
import styles from "./App.module.css";
import type { Pages } from "./board/boardView";
import { AddJobDialog } from "./components/AddJobDialog";
import { Board } from "./components/Board";
import { JobDetails } from "./components/JobDetails";
import { SearchBar } from "./components/SearchBar";
import { useBoard } from "./hooks/useBoard";

// App is the job search board, with a search bar and dialogs for adding a job and opening one.
// The search and each column's page are kept only while the app is open
// (docs/adr/0020-search-and-paginate-the-board-in-the-frontend.md).
// @scenario [features/board-search.feature#search-not-kept] Search is cleared after reopening the app
// @scenario [features/board-search.feature#add-job-clears-search] Adding a job clears the search
// @scenario [features/board-pagination.feature#new-card-shows-first-page] Adding a job shows the first page of its column
// @scenario [features/board-pagination.feature#search-resets-pages] Changing the search returns every column to its first page
// @scenario [features/board-pagination.feature#pages-reset-on-reopen] Every column starts on its first page after reopening the app
export default function App() {
	const { board, error, reload, addJob, moveJob, deleteJob } = useBoard();
	// adding is the column a new job is being added to, "" for Saved, or null when not adding.
	const [adding, setAdding] = useState<string | null>(null);
	const [openJob, setOpenJob] = useState<number | null>(null);
	const [search, setSearch] = useState("");
	const [pages, setPages] = useState<Pages>({});

	const changeSearch = (value: string) => {
		setSearch(value);
		setPages({});
	};

	return (
		<div className={styles.app}>
			<header className={styles.header}>
				<h1 className={styles.title}>Job search</h1>
				<SearchBar value={search} onChange={changeSearch} />
				<button type="button" className={styles.primary} onClick={() => setAdding("")}>
					Add job
				</button>
			</header>
			<main className={styles.main}>
				{error !== "" && (
					<p role="alert" className={styles.error}>
						{error}
					</p>
				)}
				{board && (
					<Board
						board={board}
						onMove={moveJob}
						onOpenCard={setOpenJob}
						onAddJob={setAdding}
						search={search}
						pages={pages}
						onPageChange={(status, page) => setPages((current) => ({ ...current, [status]: page }))}
					/>
				)}
			</main>
			{board && adding !== null && (
				<AddJobDialog
					columns={board.columns}
					initialStatus={adding}
					onAdd={async (details, status) => {
						await addJob(details, status);
						// Clearing the search shows every column's first page, where the new job is at the top.
						changeSearch("");
						setAdding(null);
					}}
					onCancel={() => setAdding(null)}
				/>
			)}
			{openJob !== null && (
				<JobDetails
					key={openJob}
					jobId={openJob}
					onClose={() => setOpenJob(null)}
					onChanged={() => void reload()}
					onDelete={async (id) => {
						await deleteJob(id);
						setOpenJob(null);
					}}
				/>
			)}
		</div>
	);
}
