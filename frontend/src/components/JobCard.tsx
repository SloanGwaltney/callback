import type { ReactNode } from "react";
import type { Card } from "../hooks/types";
import {
	DUE_LABELS,
	DUE_OVERDUE,
	daysInStatusLabel,
	formatTime,
	interviewKindLabel,
	MAX_INTEREST,
} from "../jobs/constants";
import styles from "./JobCard.module.css";

type Props = {
	card: Card;
	onOpen: () => void;
	// handle is the drag handle the board puts on the card.
	handle?: ReactNode;
};

// JobCard is a job as shown on the board.
// @scenario [#card-shows-document-count] [#card-shows-interest] [#card-unrated] [#interest-persists] [#card-shows-next-interview] [#card-interview-today] [#card-no-upcoming-interview] [#card-shows-next-action] [#card-next-action-no-date] [#card-next-action-due-today] [#card-next-action-overdue] [#card-next-action-upcoming] [#closed-card-no-due-marks] [#card-no-next-action] [#clear-next-action] [#card-shows-days-in-status] [#days-in-status-from-added] [#days-in-status-resets-on-move] [#days-in-status-any-column]
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
			{card.daysInStatus != null && (
				<p className={styles.inStatus}>
					In status <span className={styles.days}>{daysInStatusLabel(card.daysInStatus)}</span>
				</p>
			)}
			{card.nextInterview && (
				<p className={styles.interview}>
					{interviewKindLabel(card.nextInterview.kind)} interview{" "}
					{card.nextInterview.today ? "today" : `on ${card.nextInterview.date}`}
					{card.nextInterview.time !== "" && ` at ${formatTime(card.nextInterview.time)}`}
				</p>
			)}
			{card.nextAction && (
				<div className={styles.nextAction}>
					<p className={styles.nextText}>
						<span className={styles.srOnly}>Next action: </span>
						{card.nextAction.text}
					</p>
					{(card.nextAction.expectedDate !== "" || card.nextAction.due !== "") && (
						<p className={styles.nextWhen}>
							{card.nextAction.expectedDate !== "" && (
								<span>Expected {card.nextAction.expectedDate}</span>
							)}
							{/* The backend marks the due ones, and never a closed job's. */}
							{card.nextAction.due !== "" && (
								<span
									className={card.nextAction.due === DUE_OVERDUE ? styles.overdue : styles.dueToday}
								>
									{DUE_LABELS[card.nextAction.due] ?? card.nextAction.due}
								</span>
							)}
						</p>
					)}
				</div>
			)}
			{card.closingReasonText !== "" && (
				<p className={styles.reason}>Closed: {card.closingReasonText}</p>
			)}
			{card.documentCount > 0 && <p className={styles.documents}>{documents}</p>}
		</article>
	);
}
