import type { ReactNode } from "react";
import type { Card } from "../hooks/types";
import styles from "./JobCard.module.css";

type Props = {
	card: Card;
	onOpen: () => void;
	// handle is the drag handle the board puts on the card.
	handle?: ReactNode;
};

// JobCard is a job as shown on the board.
// @scenario [features/job-documents.feature#card-shows-document-count] The card shows that documents are attached
export function JobCard({ card, onOpen, handle }: Props) {
	const documents = card.documentCount === 1 ? "1 document" : `${card.documentCount} documents`;
	return (
		<article className={styles.card} aria-label={`${card.role} at ${card.company}`}>
			<div className={styles.header}>
				{/* @and [features/job-cards.feature] the card shows the company, the role, and the location */}
				{/* @then [features/job-board.feature] the card still shows the company "Acme" and the role "Software Engineer" */}
				{/* @then [features/job-cards.feature] the card shows the role "Senior Software Engineer" */}
				<button type="button" className={styles.open} onClick={onOpen}>
					<span className={styles.role}>{card.role}</span>
					<span className={styles.company}>{card.company}</span>
				</button>
				{handle}
			</div>
			{card.location !== "" && <p className={styles.detail}>{card.location}</p>}
			{/* @and [features/job-cards.feature] the card shows the date applied */}
			{/* @then [features/job-board.feature] the job's date applied is 2026-10-09 */}
			{/* @then [features/job-board.feature] the job's date applied is still 2026-10-01 */}
			{/* @then [features/job-board.feature#move-to-saved-clears-date] the job has no date applied */}
			{/* @then [features/job-board.feature#reorder-applied-no-date] the job has no date applied */}
			{card.dateApplied !== "" && <p className={styles.detail}>Applied {card.dateApplied}</p>}
			{/* @and [features/job-board.feature] the card shows the closing reason "<reason>" */}
			{/* @and [features/job-board.feature] the card shows the closing reason "Company went through layoffs" */}
			{/* @then [features/job-board.feature] the card shows the closing reason "Rejected" */}
			{/* @and [features/job-board.feature] the job has no closing reason */}
			{card.closingReasonText !== "" && (
				<p className={styles.reason}>Closed: {card.closingReasonText}</p>
			)}
			{/* @then [features/job-documents.feature] the card shows that it has 2 documents */}
			{card.documentCount > 0 && <p className={styles.documents}>{documents}</p>}
		</article>
	);
}
