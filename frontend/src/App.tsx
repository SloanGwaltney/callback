import { useState } from "react";
import styles from "./App.module.css";
import type { DueFilter, Pages } from "./board/boardView";
import { AddJobDialog } from "./components/AddJobDialog";
import { BackupList } from "./components/BackupList";
import { Board } from "./components/Board";
import { DocumentLibrary } from "./components/DocumentLibrary";
import { DueFilterSelect } from "./components/DueFilterSelect";
import { ErrorMessage } from "./components/ErrorMessage";
import { JobDetails } from "./components/JobDetails";
import { SearchBar } from "./components/SearchBar";
import { useBoard } from "./hooks/useBoard";

// App is the job search board, with a search bar, an overdue filter, and dialogs for adding a job
// and opening one, or the document library or the backups in its place. The search, the filter,
// and each column's page are kept only while the app is open
// (docs/adr/0020-search-and-paginate-the-board-in-the-frontend.md), and they're still there on
// coming back from the library or the backups, or after the board reloads. Restoring a backup
// reloads the board and goes back to it.
// @scenario [#search-not-kept] [#add-job-clears-search] [#new-card-shows-first-page] [#search-resets-pages] [#pages-reset-on-reopen] [#filter-resets-pages] [#add-job-clears-filter] [#filter-not-kept] [#open-library] [#back-to-board] [#refresh-keeps-search-and-pages] [#app-backups-open] [#app-backups-back-to-board] [#app-restore-daily-backup] [#app-restore-agent-backup]
export default function App() {
	const { board, error, reload, addJob, moveJob, deleteJob } = useBoard();
	// adding is the column a new job is being added to, "" for Saved, or null when not adding.
	const [adding, setAdding] = useState<string | null>(null);
	const [openJob, setOpenJob] = useState<number | null>(null);
	const [search, setSearch] = useState("");
	const [filter, setFilter] = useState<DueFilter>("all");
	const [pages, setPages] = useState<Pages>({});
	// page is the page shown: the board, the document library, or the backups.
	const [page, setPage] = useState<"board" | "library" | "backups">("board");

	const changeSearch = (value: string) => {
		setSearch(value);
		setPages({});
	};

	const changeFilter = (value: DueFilter) => {
		setFilter(value);
		setPages({});
	};

	if (page === "library") {
		return <DocumentLibrary onBack={() => setPage("board")} />;
	}
	if (page === "backups") {
		return (
			<BackupList
				onBack={() => setPage("board")}
				onRestored={async () => {
					await reload();
					setPage("board");
				}}
			/>
		);
	}

	return (
		<div className={styles.app}>
			<header className={styles.header}>
				{/* The search stands in for a visible title, so the heading is only read out. */}
				<h1 className={styles.title}>Job search</h1>
				<SearchBar value={search} onChange={changeSearch} />
				<div className={styles.controls}>
					<DueFilterSelect value={filter} onChange={changeFilter} />
					<button type="button" className={styles.button} onClick={() => setPage("library")}>
						Document library
					</button>
					<button type="button" className={styles.button} onClick={() => setPage("backups")}>
						Backups
					</button>
					<button type="button" className={styles.primary} onClick={() => setAdding("")}>
						Add job
					</button>
				</div>
			</header>
			<main className={styles.main}>
				<ErrorMessage message={error} className={styles.error} />
				{board && (
					<Board
						board={board}
						onMove={moveJob}
						onOpenCard={setOpenJob}
						onAddJob={setAdding}
						search={search}
						filter={filter}
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
						// Clearing the search and the filter shows every column's first page, where the new
						// job is at the top.
						changeSearch("");
						changeFilter("all");
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
