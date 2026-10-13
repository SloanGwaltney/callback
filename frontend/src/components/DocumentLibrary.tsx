import { type FormEvent, useId, useState } from "react";
import { errorMessage } from "../hooks/errors";
import type { LibraryDocument } from "../hooks/types";
import { useLibrary } from "../hooks/useLibrary";
import { Dialog } from "./Dialog";
import styles from "./DocumentLibrary.module.css";

type Props = {
	// onBack goes back to the job search board.
	onBack: () => void;
};

// fileNameOf returns the last part of a path from the native file dialog, on any OS.
export function fileNameOf(path: string): string {
	return path.split(/[\\/]/).pop() ?? path;
}

// DocumentLibrary is the document library page: the documents kept for the whole job search,
// newest first, each with an optional description. It adds, opens, describes, and removes them,
// asking before a removal. Library documents are the app's own copies and are never attached to a
// job.
// @scenario [features/document-library.feature#open-library] Go from the board to the document library
// @scenario [features/document-library.feature#back-to-board] Go back to the board from the document library
// @scenario [features/document-library.feature#empty-library] An empty library prompts me to add my first document
// @scenario [features/document-library.feature#add-library-document] Add a document with a description
// @scenario [features/document-library.feature#add-library-document-no-description] Add a document without a description
// @scenario [features/document-library.feature#library-newest-first] The newest documents are listed first
// @scenario [features/document-library.feature#library-same-file-twice] The same file can be added more than once
// @scenario [features/document-library.feature#library-document-kept-if-original-moved] A library document still opens after the original file is moved
// @scenario [features/document-library.feature#library-allowed-file-types] Text-like document types can be added
// @scenario [features/document-library.feature#library-file-type-any-case] A document's type is recognized whatever the letter case of its extension
// @scenario [features/document-library.feature#library-disallowed-file-types] Other file types can't be added
// @scenario [features/document-library.feature#library-file-size-limit] A file larger than 10 MB can't be added
// @scenario [features/document-library.feature#library-file-size-at-limit] A file of exactly 10 MB can be added
// @scenario [features/document-library.feature#open-library-document] Open a library document
// @scenario [features/document-library.feature#open-missing-library-document] Told when a library document's file is missing
// @scenario [features/document-library.feature#edit-library-description] Change a document's description
// @scenario [features/document-library.feature#clear-library-description] Clear a document's description
// @scenario [features/document-library.feature#remove-library-document] Remove a library document after confirming
// @scenario [features/document-library.feature#cancel-remove-library-document] Cancel removing a library document
// @scenario [features/document-library.feature#library-persists] The library is the same after reopening the app
export function DocumentLibrary({ onBack }: Props) {
	const library = useLibrary();
	// adding is the path of the file being added, chosen in the native dialog, or null.
	const [adding, setAdding] = useState<string | null>(null);
	const [describing, setDescribing] = useState<LibraryDocument | null>(null);
	const [removing, setRemoving] = useState<LibraryDocument | null>(null);
	const [error, setError] = useState("");

	// run calls the backend, or shows why it failed.
	const run = async (action: () => Promise<unknown>) => {
		setError("");
		try {
			await action();
		} catch (err) {
			setError(errorMessage(err));
		}
	};

	const chooseFile = () =>
		run(async () => {
			const path = await library.chooseFile();
			if (path !== "") {
				setAdding(path);
			}
		});

	const shownError = error !== "" ? error : library.error;

	return (
		<div className={styles.page}>
			<header className={styles.header}>
				<button type="button" className={styles.button} onClick={onBack}>
					← Back to board
				</button>
				<h1 className={styles.title}>Document library</h1>
				<button type="button" className={styles.primary} onClick={chooseFile}>
					Add a document…
				</button>
			</header>
			<main className={styles.main}>
				<p className={styles.intro}>
					Documents you use across your job search, like your base resume and cover letter
					templates. Each is the app's own copy, so it opens even if you move the original.
				</p>
				{shownError !== "" && (
					<p role="alert" className={styles.pageError}>
						{shownError}
					</p>
				)}
				{library.documents?.length === 0 && (
					<div className={styles.empty}>
						<p className={styles.emptyText}>Your library is empty.</p>
						<button type="button" className={styles.primary} onClick={chooseFile}>
							Add your first document…
						</button>
					</div>
				)}
				{library.documents !== null && library.documents.length > 0 && (
					<ul className={styles.list} aria-label="Library documents">
						{library.documents.map((doc) => (
							<li key={doc.id} className={styles.document} aria-label={doc.fileName}>
								<div className={styles.info}>
									<button
										type="button"
										className={styles.fileName}
										title={`Open ${doc.fileName}`}
										onClick={() => run(() => library.open(doc.id))}
									>
										{doc.fileName}
									</button>
									{doc.description !== "" && (
										<p className={styles.description}>{doc.description}</p>
									)}
								</div>
								<button
									type="button"
									className={styles.button}
									aria-label={`Edit the description of ${doc.fileName}`}
									onClick={() => setDescribing(doc)}
								>
									Edit description
								</button>
								<button
									type="button"
									className={styles.button}
									aria-label={`Remove ${doc.fileName}`}
									onClick={() => setRemoving(doc)}
								>
									Remove
								</button>
							</li>
						))}
					</ul>
				)}
			</main>
			{adding !== null && (
				<DescriptionForm
					title="Add to the library"
					fileName={fileNameOf(adding)}
					initial=""
					submitLabel="Add document"
					onSave={async (description) => {
						await library.add(adding, description);
						setAdding(null);
					}}
					onCancel={() => setAdding(null)}
				/>
			)}
			{describing && (
				<DescriptionForm
					title={`Describe ${describing.fileName}`}
					fileName={describing.fileName}
					initial={describing.description}
					submitLabel="Save description"
					onSave={async (description) => {
						await library.setDescription(describing.id, description);
						setDescribing(null);
					}}
					onCancel={() => setDescribing(null)}
				/>
			)}
			{removing && (
				<ConfirmRemove
					fileName={removing.fileName}
					onConfirm={async () => {
						await library.remove(removing.id);
						setRemoving(null);
					}}
					onCancel={() => setRemoving(null)}
				/>
			)}
		</div>
	);
}

type FormProps = {
	title: string;
	fileName: string;
	initial: string;
	submitLabel: string;
	// onSave saves the description, or rejects with the backend's message.
	onSave: (description: string) => Promise<void>;
	onCancel: () => void;
};

// DescriptionForm asks for a document's optional description, when adding it or later. When the
// backend refuses, the dialog stays open with what was entered and says why.
function DescriptionForm({ title, fileName, initial, submitLabel, onSave, onCancel }: FormProps) {
	const [description, setDescription] = useState(initial);
	const [error, setError] = useState("");
	const id = useId();

	const submit = async (e: FormEvent) => {
		e.preventDefault();
		setError("");
		try {
			await onSave(description);
		} catch (err) {
			setError(errorMessage(err));
		}
	};

	return (
		<Dialog title={title} onClose={onCancel}>
			<form onSubmit={submit} noValidate>
				<p className={styles.chosen}>{fileName}</p>
				<div className={styles.field}>
					<label htmlFor={id}>Description (optional)</label>
					<input
						id={id}
						className={styles.input}
						placeholder="Like “Base resume, backend roles”"
						value={description}
						onChange={(e) => setDescription(e.target.value)}
					/>
				</div>
				{error !== "" && (
					<p role="alert" className={styles.error}>
						{error}
					</p>
				)}
				<div className={styles.actions}>
					<button type="button" className={styles.button} onClick={onCancel}>
						Cancel
					</button>
					<button type="submit" className={styles.primary}>
						{submitLabel}
					</button>
				</div>
			</form>
		</Dialog>
	);
}

type ConfirmProps = {
	fileName: string;
	onConfirm: () => Promise<void>;
	onCancel: () => void;
};

function ConfirmRemove({ fileName, onConfirm, onCancel }: ConfirmProps) {
	const [error, setError] = useState("");
	const confirm = async () => {
		setError("");
		try {
			await onConfirm();
		} catch (err) {
			setError(errorMessage(err));
		}
	};
	return (
		<Dialog title={`Remove ${fileName}?`} onClose={onCancel}>
			<p className={styles.warning}>
				{fileName} will be removed from the library. The original file on your computer is kept.
			</p>
			{error !== "" && (
				<p role="alert" className={styles.error}>
					{error}
				</p>
			)}
			<div className={styles.actions}>
				<button type="button" className={styles.button} onClick={onCancel}>
					Cancel
				</button>
				<button type="button" className={styles.danger} onClick={confirm}>
					Remove
				</button>
			</div>
		</Dialog>
	);
}
