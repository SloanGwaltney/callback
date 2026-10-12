import { useId, useState } from "react";
import { errorMessage } from "../hooks/errors";
import type { JobDocument } from "../hooks/types";
import { useDocuments } from "../hooks/useDocuments";
import { DOCUMENT_KINDS } from "../jobs/constants";
import styles from "./DocumentList.module.css";

type Props = {
	jobId: number;
	documents: JobDocument[];
	// onChanged runs after a document is attached, relabeled, or removed.
	onChanged: () => void;
};

// DocumentList shows the documents attached to a job, and attaches, opens, relabels, and removes
// them.
// @scenario [features/job-documents.feature#attach-document] Attach a document of a given kind
// @scenario [features/job-documents.feature#attach-multiple-documents] Attach several documents to one job
// @scenario [features/job-documents.feature#allowed-file-types] Text-like document types can be attached
// @scenario [features/job-documents.feature#file-type-any-case] A document's type is recognized whatever the letter case of its extension
// @scenario [features/job-documents.feature#disallowed-file-types] Other file types can't be attached
// @scenario [features/job-documents.feature#file-size-limit] A file larger than 10 MB can't be attached
// @scenario [features/job-documents.feature#file-size-at-limit] A file of exactly 10 MB can be attached
// @scenario [features/job-documents.feature#open-document] Open an attached document
// @scenario [features/job-documents.feature#document-kept-if-original-moved] An attached document still opens after the original file is moved
// @scenario [features/job-documents.feature#change-document-kind] Change what kind of document an attachment is
// @scenario [features/job-documents.feature#remove-document] Remove an attached document
// @scenario [features/job-documents.feature#same-document-many-jobs] The same file can be attached to more than one job
export function DocumentList({ jobId, documents, onChanged }: Props) {
	const { attach, setKind, remove, open } = useDocuments();
	const [kind, setNewKind] = useState("resume");
	const [error, setError] = useState("");
	const kindId = useId();

	// run calls the backend, then reloads the job, or shows why it failed.
	const run = async (action: () => Promise<unknown>, reload: boolean) => {
		setError("");
		try {
			await action();
			if (reload) {
				onChanged();
			}
		} catch (err) {
			setError(errorMessage(err));
		}
	};

	return (
		<section className={styles.documents} aria-labelledby={`${kindId}-title`}>
			<h3 id={`${kindId}-title`} className={styles.title}>
				Documents
			</h3>
			{documents.length === 0 ? (
				<p className={styles.none}>No documents attached.</p>
			) : (
				<ul className={styles.list}>
					{documents.map((doc) => (
						<li key={doc.id} className={styles.document}>
							<button
								type="button"
								className={styles.fileName}
								title={`Open ${doc.fileName}`}
								onClick={() => run(() => open(doc.id), false)}
							>
								{doc.fileName}
							</button>
							<select
								className={styles.select}
								aria-label={`Kind of ${doc.fileName}`}
								value={doc.kind}
								onChange={(e) => run(() => setKind(doc.id, e.target.value), true)}
							>
								{DOCUMENT_KINDS.map((k) => (
									<option key={k.value} value={k.value}>
										{k.label}
									</option>
								))}
							</select>
							<button
								type="button"
								className={styles.button}
								aria-label={`Remove ${doc.fileName}`}
								onClick={() => run(() => remove(doc.id), true)}
							>
								Remove
							</button>
						</li>
					))}
				</ul>
			)}
			<div className={styles.attach}>
				<label htmlFor={kindId} className={styles.kindLabel}>
					Attach as
				</label>
				<select
					id={kindId}
					className={styles.select}
					value={kind}
					onChange={(e) => setNewKind(e.target.value)}
				>
					{DOCUMENT_KINDS.map((k) => (
						<option key={k.value} value={k.value}>
							{k.label}
						</option>
					))}
				</select>
				<button
					type="button"
					className={styles.button}
					onClick={() => run(() => attach(jobId, kind), true)}
				>
					Attach a document…
				</button>
			</div>
			{error !== "" && (
				<p role="alert" className={styles.error}>
					{error}
				</p>
			)}
		</section>
	);
}
