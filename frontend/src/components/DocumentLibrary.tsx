import { type FormEvent, useId, useState } from "react";
import type { LibraryDocument } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { useLibrary } from "../hooks/useLibrary";
import { ConfirmDialog } from "./ConfirmDialog";
import { Dialog } from "./Dialog";
import styles from "./DocumentLibrary.module.css";
import { ErrorMessage } from "./ErrorMessage";

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
// @scenario [#open-library] [#back-to-board] [#empty-library] [#add-library-document] [#add-library-document-no-description] [#library-newest-first] [#library-same-file-twice] [#library-document-kept-if-original-moved] [#library-allowed-file-types] [#library-file-type-any-case] [#library-disallowed-file-types] [#library-file-size-limit] [#library-file-size-at-limit] [#open-library-document] [#open-missing-library-document] [#edit-library-description] [#clear-library-description] [#remove-library-document] [#cancel-remove-library-document] [#library-persists]
export function DocumentLibrary({ onBack }: Props) {
	const library = useLibrary();
	// adding is the path of the file being added, chosen in the native dialog, or null.
	const [adding, setAdding] = useState<string | null>(null);
	const [describing, setDescribing] = useState<LibraryDocument | null>(null);
	const [removing, setRemoving] = useState<LibraryDocument | null>(null);
	const { error, run } = useAction();

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
				<ErrorMessage message={shownError} className={styles.pageError} />
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
				<ConfirmDialog
					title={`Remove ${removing.fileName}?`}
					message={`${removing.fileName} will be removed from the library. The original file on your computer is kept.`}
					confirmLabel="Remove"
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
	const { error, run } = useAction();
	const id = useId();

	const submit = (e: FormEvent) => {
		e.preventDefault();
		void run(() => onSave(description));
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
				<ErrorMessage message={error} />
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
