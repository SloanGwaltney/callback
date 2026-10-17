import { useId, useState } from "react";
import type { JobDocument } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { useDocuments } from "../hooks/useDocuments";
import { DOCUMENT_KINDS } from "../jobs/constants";
import styles from "./DocumentList.module.css";
import { ErrorMessage } from "./ErrorMessage";

type Props = {
	jobId: number;
	documents: JobDocument[];
	// onChanged runs after a document is attached, relabeled, or removed.
	onChanged: () => void;
};

// DocumentList shows the documents attached to a job, one to a line, and attaches, opens, relabels,
// and removes them.
// @scenario [#attach-document] [#attach-multiple-documents] [#allowed-file-types] [#file-type-any-case] [#disallowed-file-types] [#file-size-limit] [#file-size-at-limit] [#open-document] [#document-kept-if-original-moved] [#change-document-kind] [#remove-document] [#same-document-many-jobs] [#open-missing-document] [#remove-missing-document]
export function DocumentList({ jobId, documents, onChanged }: Props) {
	const { attach, setKind, remove, open } = useDocuments();
	const [kind, setNewKind] = useState("resume");
	const { error, run } = useAction();
	const kindId = useId();

	// change calls the backend, then reloads the job, or shows why it failed.
	const change = async (action: () => Promise<unknown>) => {
		if (await run(action)) {
			onChanged();
		}
	};

	return (
		<section className={styles.documents} aria-labelledby={`${kindId}-title`}>
			<div className={styles.header}>
				<h3 id={`${kindId}-title`} className={styles.title}>
					Documents
				</h3>
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
						onClick={() => change(() => attach(jobId, kind))}
					>
						Attach a document…
					</button>
				</div>
			</div>
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
								onClick={() => run(() => open(doc.id))}
							>
								{doc.fileName}
							</button>
							<select
								className={styles.select}
								aria-label={`Kind of ${doc.fileName}`}
								value={doc.kind}
								onChange={(e) => change(() => setKind(doc.id, e.target.value))}
							>
								{DOCUMENT_KINDS.map((k) => (
									<option key={k.value} value={k.value}>
										{k.label}
									</option>
								))}
							</select>
							<button
								type="button"
								className={styles.danger}
								aria-label={`Remove ${doc.fileName}`}
								onClick={() => change(() => remove(doc.id))}
							>
								Remove
							</button>
						</li>
					))}
				</ul>
			)}
			<ErrorMessage message={error} />
		</section>
	);
}
