import {
	type Announcements,
	DndContext,
	type DragEndEvent,
	type DragOverEvent,
	DragOverlay,
	KeyboardSensor,
	PointerSensor,
	type UniqueIdentifier,
	useDroppable,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import {
	SortableContext,
	type SortingStrategy,
	sortableKeyboardCoordinates,
	useSortable,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useEffect, useState } from "react";
import {
	collision,
	containerOf,
	finishDrop,
	type Items,
	itemsFromBoard,
	moveAcross,
	moveFor,
	statusOf,
	toTop,
} from "../board/boardDnd";
import {
	type ColumnView,
	DUE_FILTERS,
	type DueFilter,
	filterBoard,
	type Pages,
	pageBoard,
	placeInColumn,
	searching,
} from "../board/boardView";
import type { Board as BoardData, Card, Move } from "../hooks/types";
import { useAction } from "../hooks/useAction";
import { STATUS_CLOSED } from "../jobs/constants";
import styles from "./Board.module.css";
import { ClosingReasonDialog } from "./ClosingReasonDialog";
import { ErrorMessage } from "./ErrorMessage";
import { JobCard } from "./JobCard";

type Props = {
	board: BoardData;
	// onMove saves a move. It rejects with a message to show if the move isn't allowed.
	onMove: (move: Move) => Promise<void>;
	onOpenCard: (id: number) => void;
	onAddJob: (status: string) => void;
	// search is the text the board is searched for, or "" for every job.
	search: string;
	// filter shows only overdue jobs, only jobs that aren't, or every job.
	filter: DueFilter;
	// pages is each column's current page. onPageChange changes one column's page.
	pages: Pages;
	onPageChange: (status: string, page: number) => void;
};

// REORDER_WHILE_SEARCHING is shown when a card is dropped elsewhere in its own column while
// searching. Only the matching cards are shown then, so it isn't clear where the card would land.
const REORDER_WHILE_SEARCHING = "Cards can't be reordered while searching. Clear the search first.";

// REORDER_WHILE_FILTERING is shown for the same drop while the board is filtered to overdue jobs or
// jobs that aren't, which hides cards the same way.
const REORDER_WHILE_FILTERING =
	"Cards can't be reordered while the board is filtered. Show all jobs first.";

// noSorting keeps the other cards still while a card is dragged within its column during a search
// or filter, since it can't be reordered there.
const noSorting: SortingStrategy = () => null;

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
// @scenario [features/job-status-history.feature#history-ignores-cancelled-close] Cancelling a move to Closed doesn't change the history
// @scenario [features/job-board.feature#reopen-clears-reason] Moving a card out of Closed clears the closing reason
// @scenario [features/job-board.feature#reorder-closed-keeps-reason] Reordering a card within Closed keeps its closing reason
// @scenario [features/job-board.feature#reorder-card] Reorder a card within its column
// @scenario [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
// @scenario [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
// @scenario [features/job-board.feature#board-persists] Board is the same after reopening the app
// @scenario [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
// @scenario [features/board-search.feature#search-keeps-columns] Matching cards stay in their own columns
// @scenario [features/board-search.feature#search-counts-matches] Column counts show how many cards match
// @scenario [features/board-search.feature#search-no-matches] Told when nothing matches
// @scenario [features/board-search.feature#search-move-card] Move a matching card to another status while searching
// @scenario [features/board-search.feature#search-move-to-top] A card moved to another status while searching goes to the top of that column
// @scenario [features/board-search.feature#search-no-reorder] Cards can't be reordered within a column while searching
// @scenario [features/board-pagination.feature#column-first-page] A column with more than 10 cards shows its first page
// @scenario [features/board-pagination.feature#column-fits-one-page] A column with 10 cards or fewer has no pages to move between
// @scenario [features/board-pagination.feature#count-shows-column-total] The column count shows every card in the column, not just the page
// @scenario [features/board-pagination.feature#next-page] Go to a column's next page
// @scenario [features/board-pagination.feature#previous-page] Go to a column's previous page
// @scenario [features/board-pagination.feature#page-bounds] A column can't page past its first or last page
// @scenario [features/board-pagination.feature#columns-page-independently] Paging one column leaves the other columns where they are
// @scenario [features/board-pagination.feature#page-arrows] A column with more than one page shows previous and next arrows
// @scenario [features/board-pagination.feature#page-transition] Changing page slides the cards in the direction of the arrow
// @scenario [features/board-pagination.feature#page-transition-reduced-motion] Changing page doesn't animate when my system asks for reduced motion
// @scenario [features/board-pagination.feature#one-page-per-column] A column shows only one page of cards at a time
// @scenario [features/board-pagination.feature#move-card-onto-page] Place a moved card at a chosen position on a later page
// @scenario [features/board-pagination.feature#last-card-leaves-page] Moving the only card off the last page shows the page before it
// @scenario [features/board-pagination.feature#page-filled-after-move] A page fills back up after a card moves out of it
// @scenario [features/board-pagination.feature#search-results-paged] Matching cards are shown a page at a time
// @scenario [features/job-next-action.feature#filter-overdue] Show only overdue jobs
// @scenario [features/job-next-action.feature#filter-not-overdue] Show only jobs that aren't overdue
// @scenario [features/job-next-action.feature#filter-due-today-not-overdue] A job due today isn't overdue yet
// @scenario [features/job-next-action.feature#filter-closed-never-overdue] A closed job is never overdue
// @scenario [features/job-next-action.feature#filter-show-all] Show every job again
// @scenario [features/job-next-action.feature#filter-keeps-columns] Filtered cards stay in their own columns and the counts show how many are shown
// @scenario [features/job-next-action.feature#filter-with-search] The filter and the search both apply
// @scenario [features/job-next-action.feature#filter-no-matches] Told when no jobs match the filter
export function Board({
	board,
	onMove,
	onOpenCard,
	onAddJob,
	search,
	filter,
	pages,
	onPageChange,
}: Props) {
	// preview is where the cards are while one is dragged, or null when nothing is.
	const [preview, setPreview] = useState<Items | null>(null);
	// dragged is the card being dragged, drawn in the overlay above the columns.
	const [dragged, setDragged] = useState<Card | null>(null);
	// closing is a move into Closed waiting for its closing reason.
	const [closing, setClosing] = useState<Move | null>(null);
	const { error, setError, run } = useAction();

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
	// The board shows the cards matching the search and the filter, a page at a time in each column.
	// Dragging works on what's shown, and a drop is turned back into a place in the whole column when
	// it's saved.
	const isSearching = searching(search);
	// isFiltered is whether some cards may be hidden, by the search or the filter.
	const isFiltered = isSearching || filter !== "all";
	const filtered = filterBoard(board, search, filter);
	const view = pageBoard(filtered, pages);
	const items = preview ?? itemsFromBoard(view);

	// A page past a column's last page is shown as its last page; this keeps the stored page in
	// step, so the column doesn't jump back if it grows again.
	useEffect(() => {
		for (const column of view.columns) {
			const stored = pages[column.status];
			if (stored !== undefined && stored !== column.page) {
				onPageChange(column.status, column.page);
			}
		}
	});

	const save = (move: Move) => void run(() => onMove(move));

	const dragOver = ({ active, over }: DragOverEvent) => {
		if (over) {
			setPreview((current) => {
				if (!current) {
					return current;
				}
				const moved = moveAcross(current, active.id, over.id);
				// While searching or filtering, a card moved to another column goes to its top.
				const across = containerOf(moved, active.id) !== containerOf(current, active.id);
				return isFiltered && across ? toTop(moved, active.id) : moved;
			});
		}
	};

	const dragEnd = ({ active, over }: DragEndEvent) => {
		const dropped = preview && over ? finishDrop(preview, active.id, over.id) : null;
		setPreview(null);
		setDragged(null);
		if (!dropped) {
			return;
		}
		const jobId = Number(active.id);
		const shown = moveFor(view, dropped, jobId);
		if (!shown) {
			return;
		}
		let move: Move;
		if (isFiltered) {
			if (shown.status === statusOf(board, jobId)) {
				// The drag preview is already dropped, so the card is back where it was.
				setError(isSearching ? REORDER_WHILE_SEARCHING : REORDER_WHILE_FILTERING);
				return;
			}
			move = { ...shown, index: 0 };
		} else {
			move = placeInColumn(view, shown);
		}
		// Moving into Closed waits for a closing reason. Reordering within Closed doesn't ask, so
		// the job keeps the reason it has.
		if (move.status === STATUS_CLOSED && statusOf(board, jobId) !== STATUS_CLOSED) {
			setClosing(move);
			return;
		}
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

	const empty = board.columns.every((column) => column.cards.length === 0);
	const noMatches =
		isFiltered && !empty && filtered.columns.every((column) => column.cards.length === 0);
	const filterLabel = DUE_FILTERS.find((f) => f.value === filter)?.label ?? filter;

	return (
		<div className={styles.board}>
			{empty && (
				<div className={styles.prompt}>
					<p>Track your first application by adding the job you're interested in.</p>
					<button type="button" className={styles.primary} onClick={() => onAddJob("")}>
						Add your first job
					</button>
				</div>
			)}
			{noMatches && (
				<p role="status" className={styles.noMatches}>
					{!isSearching
						? `No jobs match the "${filterLabel}" filter.`
						: filter === "all"
							? `No jobs match "${search.trim()}".`
							: `No jobs match "${search.trim()}" with the "${filterLabel}" filter.`}
				</p>
			)}
			<ErrorMessage message={error} className={styles.error} />
			<DndContext
				sensors={sensors}
				collisionDetection={collision}
				accessibility={{ announcements }}
				onDragStart={({ active }) => {
					setPreview(itemsFromBoard(view));
					setDragged(cards.get(Number(active.id)) ?? null);
				}}
				onDragOver={dragOver}
				onDragEnd={dragEnd}
				onDragCancel={() => {
					setPreview(null);
					setDragged(null);
				}}
			>
				<div className={styles.columns}>
					{view.columns.map((column) => (
						<Column
							key={column.status}
							column={column}
							cards={(items[column.status] ?? []).flatMap((id) => cards.get(id) ?? [])}
							strategy={isFiltered ? noSorting : verticalListSortingStrategy}
							onOpenCard={onOpenCard}
							onAddJob={onAddJob}
							onPageChange={onPageChange}
						/>
					))}
				</div>
				{/* The dragged card is drawn here, above the board, so a column's scroll area doesn't
				    clip it. It's only a picture of the card, so it's hidden from screen readers and
				    can't be focused; the announcements describe the drag instead. */}
				<DragOverlay>
					{dragged && (
						<div className={styles.overlay} aria-hidden="true" inert>
							<JobCard
								card={dragged}
								onOpen={() => {}}
								handle={<span className={styles.handle}>⠿</span>}
							/>
						</div>
					)}
				</DragOverlay>
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
	column: ColumnView;
	cards: Card[];
	strategy: SortingStrategy;
	onOpenCard: (id: number) => void;
	onAddJob: (status: string) => void;
	onPageChange: (status: string, page: number) => void;
};

// Direction is the arrow last used to change a column's page.
type Direction = "next" | "previous";

// reducedMotion reports whether the system asks for less motion, so pages change without sliding.
function reducedMotion(): boolean {
	return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
}

function Column({ column, cards, strategy, onOpenCard, onAddJob, onPageChange }: ColumnProps) {
	// The column itself is a drop target, so a card can be dropped into an empty column.
	const { setNodeRef } = useDroppable({ id: column.status });
	// sliding is the arrow whose page is sliding in. leaving is the page sliding out, drawn over
	// the new one until its animation ends. Each column keeps its own, so only it moves.
	const [sliding, setSliding] = useState<Direction | null>(null);
	const [leaving, setLeaving] = useState<{ cards: Card[]; direction: Direction } | null>(null);
	const headingId = `column-${column.status}`;
	const { page, pageCount } = column;

	const turn = (direction: Direction) => {
		const target = direction === "next" ? page + 1 : page - 1;
		if (target < 1 || target > pageCount) {
			return;
		}
		if (reducedMotion()) {
			setSliding(null);
			setLeaving(null);
		} else {
			setSliding(direction);
			setLeaving({ cards, direction });
		}
		onPageChange(column.status, target);
	};

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
				{/* While searching or filtering, the count is how many cards are shown (see filterBoard). */}
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
			{cards.length === 0 && <p className={styles.emptyColumn}>No jobs</p>}
			<div className={styles.pageArea}>
				{leaving && (
					// A picture of the page being left, so it can't be focused or dragged.
					<ul
						className={styles.leaving}
						data-slide={leaving.direction === "next" ? "out-left" : "out-right"}
						aria-hidden="true"
						inert
						onAnimationEnd={(e) => {
							if (e.target === e.currentTarget) {
								setLeaving(null);
							}
						}}
					>
						{leaving.cards.map((card) => (
							<li key={card.id} className={styles.item}>
								<JobCard
									card={card}
									onOpen={() => {}}
									handle={<span className={styles.handle}>⠿</span>}
								/>
							</li>
						))}
					</ul>
				)}
				<SortableContext
					id={column.status}
					items={cards.map((card) => card.id)}
					strategy={strategy}
				>
					{/* The list is new for each page, so the page sliding in animates every time. */}
					<ul
						key={page}
						ref={setNodeRef}
						className={styles.cards}
						data-slide={sliding === null ? undefined : sliding === "next" ? "in-right" : "in-left"}
						onAnimationEnd={(e) => {
							if (e.target === e.currentTarget) {
								setSliding(null);
							}
						}}
					>
						{cards.map((card) => (
							<SortableCard key={card.id} card={card} onOpen={() => onOpenCard(card.id)} />
						))}
					</ul>
				</SortableContext>
			</div>
			{pageCount > 1 && (
				<nav className={styles.pager} aria-label={`Pages of ${column.label}`}>
					{/* aria-disabled, not disabled, so the arrow keeps focus on the first or last page. */}
					<button
						type="button"
						className={styles.arrow}
						aria-label={`Previous page of ${column.label}`}
						title="Previous page"
						aria-disabled={page === 1}
						onClick={() => turn("previous")}
					>
						‹
					</button>
					<span className={styles.pageNumber} aria-live="polite">
						Page {page} of {pageCount}
					</span>
					<button
						type="button"
						className={styles.arrow}
						aria-label={`Next page of ${column.label}`}
						title="Next page"
						aria-disabled={page === pageCount}
						onClick={() => turn("next")}
					>
						›
					</button>
				</nav>
			)}
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
