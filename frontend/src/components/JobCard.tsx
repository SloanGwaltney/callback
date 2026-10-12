import type { ReactNode } from "react";
import type { Card } from "../hooks/types";
import { MAX_INTEREST } from "../jobs/constants";
import styles from "./JobCard.module.css";

type Props = {
	card: Card;
	onOpen: () => void;
	// handle is the drag handle the board puts on the card.
	handle?: ReactNode;
};

// JobCard is a job as shown on the board.
// @scenario [features/job-documents.feature#card-shows-document-count] The card shows that documents are attached
// @scenario [features/job-interest.feature#card-shows-interest] The card shows the interest rating as stars
// @scenario [features/job-interest.feature#card-unrated] An unrated card shows no stars
// @scenario [features/job-interest.feature#interest-persists] The interest rating is the same after reopening the app
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
			{card.interest != null && (
				<p
					className={styles.interest}
					role="img"
					aria-label={`${card.interest} of ${MAX_INTEREST} stars`}
				>
					{"★".repeat(card.interest)}
					<span className={styles.unfilled}>{"☆".repeat(MAX_INTEREST - card.interest)}</span>
				</p>
			)}
			{card.location !== "" && <p className={styles.detail}>{card.location}</p>}
			{card.dateApplied !== "" && <p className={styles.detail}>Applied {card.dateApplied}</p>}
			{card.closingReasonText !== "" && (
				<p className={styles.reason}>Closed: {card.closingReasonText}</p>
			)}
			{card.documentCount > 0 && <p className={styles.documents}>{documents}</p>}
		</article>
	);
}
