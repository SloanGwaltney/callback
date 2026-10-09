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
			// @then [features/job-documents.feature] the file is not attached
			// @and [features/job-documents.feature] I am told which document types are allowed
			// @and [features/job-documents.feature] I am told files can be at most 10 MB
			setError(errorMessage(err));
		}
	};

	return (
		<section className={styles.documents} aria-labelledby={`${kindId}-title`}>
			<h3 id={`${kindId}-title`} className={styles.title}>
				Documents
			</h3>
			{/* @and [features/job-cards.feature] I see the documents attached to the job */}
			{/* @then [features/job-documents.feature] the job lists "<file>" as a <kind> */}
			{/* @then [features/job-documents.feature] the job lists all three documents with their kinds */}
			{/* @then [features/job-documents.feature] the job lists "<file>" as other */}
			{/* @then [features/job-documents.feature] the job lists "RESUME.PDF" as other */}
			{/* @then [features/job-documents.feature] the job lists "portfolio.pdf" as other */}
			{/* @then [features/job-documents.feature] the job lists "acme-cover.docx" as a cover letter */}
			{/* @then [features/job-documents.feature] the job no longer lists "resume-2026.pdf" */}
			{/* @then [features/job-documents.feature] both jobs list "resume-2026.pdf" as a resume */}
			{documents.length === 0 ? (
				<p className={styles.none}>No documents attached.</p>
			) : (
				<ul className={styles.list}>
					{documents.map((doc) => (
						<li key={doc.id} className={styles.document}>
							{/* @when [features/job-documents.feature] I open the attached "resume-2026.pdf" */}
							<button
								type="button"
								className={styles.fileName}
								title="Open"
								onClick={() => run(() => open(doc.id), false)}
							>
								{doc.fileName}
							</button>
							{/* @when [features/job-documents.feature] I change its kind to cover letter */}
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
							{/* @when [features/job-documents.feature] I remove the attached "resume-2026.pdf" */}
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
				{/* @when [features/job-documents.feature] I attach the file "<file>" as a <kind> */}
				{/* @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume */}
				{/* @and [features/job-documents.feature] I attach "acme-cover.docx" as a cover letter */}
				{/* @and [features/job-documents.feature] I attach "take-home-notes.md" as other */}
				{/* @when [features/job-documents.feature] I attach the file "<file>" as other */}
				{/* @when [features/job-documents.feature] I attach the file "RESUME.PDF" as other */}
				{/* @when [features/job-documents.feature] I try to attach the file "<file>" */}
				{/* @when [features/job-documents.feature] I try to attach a 12 MB file "portfolio.pdf" */}
				{/* @when [features/job-documents.feature] I attach a 10 MB file "portfolio.pdf" as other */}
				{/* @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume to both jobs */}
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
