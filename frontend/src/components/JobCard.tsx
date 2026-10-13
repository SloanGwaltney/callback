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
// @scenario [features/job-documents.feature#card-shows-document-count] The card shows that documents are attached
// @scenario [features/job-interest.feature#card-shows-interest] The card shows the interest rating as stars
// @scenario [features/job-interest.feature#card-unrated] An unrated card shows no stars
// @scenario [features/job-interest.feature#interest-persists] The interest rating is the same after reopening the app
// @scenario [features/job-interviews.feature#card-shows-next-interview] The card shows the date of the next upcoming interview
// @scenario [features/job-interviews.feature#card-interview-today] An interview today counts as upcoming
// @scenario [features/job-interviews.feature#card-no-upcoming-interview] The card shows no interview when every interview has passed
// @scenario [features/job-next-action.feature#card-shows-next-action] The card shows the next action and its expected date
// @scenario [features/job-next-action.feature#card-next-action-no-date] The card shows a next action with no expected date
// @scenario [features/job-next-action.feature#card-next-action-due-today] A next action expected today is marked as due
// @scenario [features/job-next-action.feature#card-next-action-overdue] A next action whose expected date has passed is marked as overdue
// @scenario [features/job-next-action.feature#card-next-action-upcoming] A next action expected later isn't marked
// @scenario [features/job-next-action.feature#closed-card-no-due-marks] A closed job's next action is never marked as due or overdue
// @scenario [features/job-next-action.feature#card-no-next-action] A job with no next action shows none on its card
// @scenario [features/job-next-action.feature#clear-next-action] Clear the next action once it's done
// @scenario [features/job-status-history.feature#card-shows-days-in-status] The card shows how many days the job has been in its current status
// @scenario [features/job-status-history.feature#days-in-status-from-added] A job that hasn't moved counts from when it was added
// @scenario [features/job-status-history.feature#days-in-status-resets-on-move] Moving a card to another status starts the count again
// @scenario [features/job-status-history.feature#days-in-status-any-column] Every column's cards show their days in status
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
