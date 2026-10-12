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
				<button type="button" className={styles.open} onClick={onOpen}>
					<span className={styles.role}>{card.role}</span>
					<span className={styles.company}>{card.company}</span>
				</button>
				{handle}
			</div>
			{card.location !== "" && <p className={styles.detail}>{card.location}</p>}
			{card.dateApplied !== "" && <p className={styles.detail}>Applied {card.dateApplied}</p>}
			{card.closingReasonText !== "" && (
				<p className={styles.reason}>Closed: {card.closingReasonText}</p>
			)}
			{card.documentCount > 0 && <p className={styles.documents}>{documents}</p>}
		</article>
	);
}
