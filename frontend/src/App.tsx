import { useState } from "react";
import styles from "./App.module.css";
import { AddJobDialog } from "./components/AddJobDialog";
import { Board } from "./components/Board";
import { JobDetails } from "./components/JobDetails";
import { useBoard } from "./hooks/useBoard";

// App is the job search board, with dialogs for adding a job and opening one.
export default function App() {
	const { board, error, reload, addJob, moveJob, deleteJob } = useBoard();
	// adding is the column a new job is being added to, "" for Saved, or null when not adding.
	const [adding, setAdding] = useState<string | null>(null);
	const [openJob, setOpenJob] = useState<number | null>(null);

	return (
		<div className={styles.app}>
			<header className={styles.header}>
				<h1 className={styles.title}>Job search</h1>
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
					<Board board={board} onMove={moveJob} onOpenCard={setOpenJob} onAddJob={setAdding} />
				)}
			</main>
			{board && adding !== null && (
				<AddJobDialog
					columns={board.columns}
					initialStatus={adding}
					onAdd={async (details, status) => {
						await addJob(details, status);
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
