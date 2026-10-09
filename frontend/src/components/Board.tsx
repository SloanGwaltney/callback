import {
	type Announcements,
	closestCorners,
	DndContext,
	type DragEndEvent,
	type DragOverEvent,
	KeyboardSensor,
	PointerSensor,
	type UniqueIdentifier,
	useDroppable,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import {
	SortableContext,
	sortableKeyboardCoordinates,
	useSortable,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useState } from "react";
import {
	containerOf,
	finishDrop,
	type Items,
	itemsFromBoard,
	moveAcross,
	moveFor,
	statusOf,
} from "../board/boardDnd";
import { errorMessage } from "../hooks/errors";
import type { Board as BoardData, Card, Column as ColumnData, Move } from "../hooks/types";
import { STATUS_CLOSED } from "../jobs/constants";
import styles from "./Board.module.css";
import { ClosingReasonDialog } from "./ClosingReasonDialog";
import { JobCard } from "./JobCard";

type Props = {
	board: BoardData;
	// onMove saves a move. It rejects with a message to show if the move isn't allowed.
	onMove: (move: Move) => Promise<void>;
	onOpenCard: (id: number) => void;
	onAddJob: (status: string) => void;
};

// Board lays out the jobs in status columns. Cards are dragged with the mouse or the keyboard to
// move them between columns and reorder them (docs/adr/0013-use-dnd-kit-for-moving-and-reordering-board-cards.md).
// @scenario [features/job-board.feature#board-status-columns] Board shows the status columns in order
// @scenario [features/job-board.feature#empty-board] Empty board when no jobs have been added
// @scenario [features/job-board.feature#column-card-count] Each column shows how many cards it holds
// @scenario [features/job-board.feature#move-card-forward] Move a card to the next status
// @scenario [features/job-board.feature#move-card-any-status] Move a card to any other status
// @scenario [features/job-board.feature#move-keeps-details] Moving a card keeps its details and documents
// @scenario [features/job-board.feature#move-to-applied-sets-date] Moving a card to Applied fills in today's date when no date is set
// @scenario [features/job-board.feature#move-to-applied-keeps-date] Moving a card to Applied keeps a date applied that is already set
// @scenario [features/job-board.feature#move-to-saved-clears-date] Moving a card back to Saved clears the date applied
// @scenario [features/job-board.feature#reorder-saved-keeps-date] Reordering a card within Saved keeps its date applied
// @scenario [features/job-board.feature#reorder-applied-no-date] Reordering a card within Applied doesn't fill in a date applied
// @scenario [features/job-board.feature#close-asks-reason] Moving a card to Closed asks for a reason
// @scenario [features/job-board.feature#close-other-reason] Write my own reason when closing with Other
// @scenario [features/job-board.feature#close-other-needs-text] Closing with Other needs a written reason
// @scenario [features/job-board.feature#close-cancel] Cancelling the closing reason leaves the card where it was
// @scenario [features/job-board.feature#reopen-clears-reason] Moving a card out of Closed clears the closing reason
// @scenario [features/job-board.feature#reorder-closed-keeps-reason] Reordering a card within Closed keeps its closing reason
// @scenario [features/job-board.feature#reorder-card] Reorder a card within its column
// @scenario [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
// @scenario [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
// @scenario [features/job-board.feature#board-persists] Board is the same after reopening the app
// @scenario [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
export function Board({ board, onMove, onOpenCard, onAddJob }: Props) {
	// preview is where the cards are while one is dragged, or null when nothing is.
	const [preview, setPreview] = useState<Items | null>(null);
	// closing is a move into Closed waiting for its closing reason.
	const [closing, setClosing] = useState<Move | null>(null);
	const [error, setError] = useState("");

	const sensors = useSensors(
		// A short distance before a drag starts lets a click on the card open it.
		useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
		useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
	);

	const cards = new Map<number, Card>();
	for (const column of board.columns) {
		for (const card of column.cards) {
			cards.set(card.id, card);
		}
	}
	const items = preview ?? itemsFromBoard(board);

	const save = (move: Move) => {
		setError("");
		onMove(move).catch((err) => setError(errorMessage(err)));
	};

	const dragOver = ({ active, over }: DragOverEvent) => {
		if (over) {
			setPreview((current) => current && moveAcross(current, active.id, over.id));
		}
	};

	const dragEnd = ({ active, over }: DragEndEvent) => {
		const dropped = preview && over ? finishDrop(preview, active.id, over.id) : null;
		setPreview(null);
		if (!dropped) {
			return;
		}
		const jobId = Number(active.id);
		const move = moveFor(board, dropped, jobId);
		if (!move) {
			return;
		}
		// Moving into Closed waits for a closing reason. Reordering within Closed doesn't ask, so
		// the job keeps the reason it has.
		// @when [features/job-board.feature] I move the card to "Closed"
		// @and [features/job-board.feature] the job at "Acme" has the closing reason "Rejected"
		if (move.status === STATUS_CLOSED && statusOf(board, jobId) !== STATUS_CLOSED) {
			setClosing(move);
			return;
		}
		// @when [features/job-board.feature] I move the card to "Applied"
		// @when [features/job-board.feature] I move the card to "<to>"
		// @when [features/job-board.feature] I move the card to "Interviewing"
		// @when [features/job-board.feature] I move the card to "Saved"
		// @when [features/job-board.feature] I move the "Initech" card above the "Acme" card
		// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
		// @when [features/job-board.feature] I move the "Acme" card into "Applied" between "Globex" and "Initech"
		save(move);
	};

	const name = (id: UniqueIdentifier) => {
		const card = cards.get(Number(id));
		return card ? `${card.role} at ${card.company}` : "the card";
	};
	const place = (id: UniqueIdentifier) => {
		const status = preview ? containerLabel(board, preview, id) : undefined;
		return status ? `the ${status} column` : "the board";
	};
	const announcements: Announcements = {
		onDragStart: ({ active }) => `Picked up ${name(active.id)}.`,
		onDragOver: ({ active, over }) =>
			over
				? `${name(active.id)} is over ${place(over.id)}.`
				: `${name(active.id)} is off the board.`,
		onDragEnd: ({ active, over }) =>
			over ? `Dropped ${name(active.id)} in ${place(over.id)}.` : `Put ${name(active.id)} back.`,
		onDragCancel: ({ active }) => `Put ${name(active.id)} back where it was.`,
	};

	// @given [features/job-board.feature] I have not added any jobs
	const empty = board.columns.every((column) => column.cards.length === 0);

	return (
		<div className={styles.board}>
			{empty && (
				// @and [features/job-board.feature] I am prompted to add my first job
				<div className={styles.prompt}>
					<p>Track your first application by adding the job you're interested in.</p>
					<button type="button" className={styles.primary} onClick={() => onAddJob("")}>
						Add your first job
					</button>
				</div>
			)}
			{error !== "" && (
				<p role="alert" className={styles.error}>
					{error}
				</p>
			)}
			<DndContext
				sensors={sensors}
				collisionDetection={closestCorners}
				accessibility={{ announcements }}
				onDragStart={() => setPreview(itemsFromBoard(board))}
				onDragOver={dragOver}
				onDragEnd={dragEnd}
				onDragCancel={() => setPreview(null)}
			>
				<div className={styles.columns}>
					{/* @then [features/job-board.feature] I see the columns "Saved", "Applied", "Interviewing", "Offer", and "Closed" in that order */}
					{board.columns.map((column) => (
						<Column
							key={column.status}
							column={column}
							cards={(items[column.status] ?? []).flatMap((id) => cards.get(id) ?? [])}
							onOpenCard={onOpenCard}
							onAddJob={onAddJob}
						/>
					))}
				</div>
			</DndContext>
			{closing && (
				<ClosingReasonDialog
					title={`Close ${name(closing.jobId)}`}
					submitLabel="Close job"
					onChoose={async (reason, other) => {
						await onMove({ ...closing, closingReason: reason, closingReasonOther: other });
						setClosing(null);
					}}
					// The move is dropped, and the board still shows the card where it was.
					// @then [features/job-board.feature#close-cancel] the card stays in the "Interviewing" column
					onCancel={() => setClosing(null)}
				/>
			)}
		</div>
	);
}

// containerLabel names the column holding id, a card or a column, in the drag preview.
function containerLabel(board: BoardData, items: Items, id: UniqueIdentifier): string | undefined {
	const status = containerOf(items, id);
	return board.columns.find((column) => column.status === status)?.label;
}

type ColumnProps = {
	column: ColumnData;
	cards: Card[];
	onOpenCard: (id: number) => void;
	onAddJob: (status: string) => void;
};

function Column({ column, cards, onOpenCard, onAddJob }: ColumnProps) {
	// The column itself is a drop target, so a card can be dropped into an empty column.
	const { setNodeRef } = useDroppable({ id: column.status });
	const headingId = `column-${column.status}`;
	return (
		<section
			className={styles.column}
			aria-labelledby={headingId}
			style={{ borderTopColor: `var(--color-status-${column.status})` }}
		>
			<header className={styles.columnHeader}>
				<h2 id={headingId} className={styles.columnTitle}>
					{column.label}
				</h2>
				{/* @then [features/job-board.feature] the "Saved" column shows a count of 3 */}
				{/* @and [features/job-board.feature] the "Applied" column shows a count of 1 */}
				{/* @and [features/job-board.feature] the "Interviewing" column shows a count of 0 */}
				<span className={styles.count}>
					<span aria-hidden="true">{column.count}</span>
					<span className={styles.srOnly}>
						{column.count === 1 ? "1 job" : `${column.count} jobs`}
					</span>
				</span>
				{/* A job can't be added straight to Closed, so that column has no add button. */}
				{column.status !== STATUS_CLOSED && (
					<button
						type="button"
						className={styles.add}
						aria-label={`Add a job to ${column.label}`}
						title={`Add a job to ${column.label}`}
						onClick={() => onAddJob(column.status)}
					>
						+
					</button>
				)}
			</header>
			<SortableContext
				id={column.status}
				items={cards.map((card) => card.id)}
				strategy={verticalListSortingStrategy}
			>
				<ul ref={setNodeRef} className={styles.cards}>
					{/* @then [features/job-board.feature] the card appears in the "Applied" column */}
					{/* @then [features/job-board.feature] the card appears in the "<to>" column */}
					{/* @then [features/job-board.feature] the card appears in the "Closed" column */}
					{/* @then [features/job-board.feature] the card appears in the "Interviewing" column */}
					{/* @then [features/job-board.feature] the card for "Software Engineer" at "Acme" is in the "Interviewing" column */}
					{/* @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order */}
					{/* @then [features/job-board.feature] the "Applied" column shows "Globex", "Acme", and "Initech" in that order */}
					{/* @then [features/job-board.feature] the "Saved" column shows "Acme", "Globex", and "Initech" in that order */}
					{cards.map((card) => (
						<SortableCard key={card.id} card={card} onOpen={() => onOpenCard(card.id)} />
					))}
				</ul>
			</SortableContext>
			{/* @then [features/job-board.feature] every status column is empty */}
			{cards.length === 0 && <p className={styles.emptyColumn}>No jobs</p>}
		</section>
	);
}

function SortableCard({ card, onOpen }: { card: Card; onOpen: () => void }) {
	const {
		attributes,
		listeners,
		setNodeRef,
		setActivatorNodeRef,
		transform,
		transition,
		isDragging,
	} = useSortable({ id: card.id });
	return (
		<li
			ref={setNodeRef}
			className={isDragging ? `${styles.item} ${styles.dragging}` : styles.item}
			style={{
				transform: transform ? `translate3d(${transform.x}px, ${transform.y}px, 0)` : undefined,
				transition,
			}}
		>
			<JobCard
				card={card}
				onOpen={onOpen}
				handle={
					<button
						type="button"
						ref={setActivatorNodeRef}
						className={styles.handle}
						{...attributes}
						{...listeners}
						aria-label={`Move ${card.role} at ${card.company}`}
					>
						⠿
					</button>
				}
			/>
		</li>
	);
}
