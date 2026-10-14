package jobs

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"callback/internal/store"
)

// Move is a card being dropped into a column at a position.
type Move struct {
	JobID  int64  `json:"jobId"`
	Status Status `json:"status"`
	// Index is where the card lands in the column, from 0 at the top, counted as if the card had
	// already left its old place. Out-of-range values put it at the top or bottom.
	Index int `json:"index"`
	// ClosingReason and ClosingReasonOther are required when moving a card into Closed from
	// another column, and ignored otherwise.
	ClosingReason      ClosingReason `json:"closingReason"`
	ClosingReasonOther string        `json:"closingReasonOther"`
}

// MoveJob moves a card to a status and position, renumbering the affected columns in one
// transaction. See docs/adr/0015-store-card-order-as-an-integer-position.md. A move to another
// status is added to the job's status history, with its closing reason for Closed; reordering a
// card within its column isn't.
// @scenario [#history-records-move] [#history-records-moving-back] [#history-records-closing-reason] [#history-ignores-reorder] [#days-in-status-resets-on-move] [#days-in-status-ignores-reorder]
func (s *Service) MoveJob(ctx context.Context, m Move) error {
	if !m.Status.valid() {
		return invalid("there's no such column")
	}
	return s.store.InTx(ctx, func(q store.Querier) error {
		job, err := q.GetJob(ctx, m.JobID)
		if err != nil {
			return notFound(err, ErrJobNotFound)
		}
		from := Status(job.Status)
		changesStatus := from != m.Status

		reason, other, err := closingReasonFor(m, job, changesStatus)
		if err != nil {
			return err
		}

		date := job.DateApplied
		switch {
		case changesStatus && m.Status == StatusApplied && !date.Valid:
			date = nullString(s.today())
		case changesStatus && m.Status == StatusSaved:
			date = sql.NullString{}
		default:
			// Any other move, including reordering within Saved or Applied, keeps the date applied
			// as it is.
		}

		column, err := q.ListJobIDsInStatus(ctx, string(m.Status))
		if err != nil {
			return err
		}
		column = slices.DeleteFunc(column, func(id int64) bool { return id == job.ID })
		index := min(max(m.Index, 0), len(column))
		column = slices.Insert(column, index, job.ID)

		// Only the placement columns are written, so the job's details and documents stay as they
		// were.
		err = q.UpdateJobPlacement(ctx, store.UpdateJobPlacementParams{
			ID:                 job.ID,
			Status:             string(m.Status),
			Position:           int64(index),
			DateApplied:        date,
			ClosingReason:      reason,
			ClosingReasonOther: other,
		})
		if err != nil {
			return err
		}
		if err := renumber(ctx, q, column, 0); err != nil {
			return err
		}

		if changesStatus {
			if err := s.recordStatusChange(ctx, q, job.ID, m.Status, reason, other); err != nil {
				return err
			}
			// Close the gap the card left in its old column.
			old, err := q.ListJobIDsInStatus(ctx, string(from))
			if err != nil {
				return err
			}
			if err := renumber(ctx, q, old, 0); err != nil {
				return err
			}
		}
		return nil
	})
}

// closingReasonFor returns the closing reason a job has after a move.
func closingReasonFor(m Move, job store.Job, changesStatus bool) (sql.NullString, sql.NullString, error) {
	switch {
	case m.Status != StatusClosed:
		return sql.NullString{}, sql.NullString{}, nil
	case !changesStatus:
		// Reordering within Closed keeps the reason the job already has.
		return job.ClosingReason, job.ClosingReasonOther, nil
	default:
		return checkClosingReason(m.ClosingReason, m.ClosingReasonOther)
	}
}

// checkClosingReason validates a chosen closing reason and returns it as stored: Other with its
// trimmed text, any other reason with no text.
func checkClosingReason(reason ClosingReason, other string) (sql.NullString, sql.NullString, error) {
	if reason == "" {
		return sql.NullString{}, sql.NullString{}, invalid("choose why the job was closed")
	}
	if !reason.valid() {
		return sql.NullString{}, sql.NullString{}, invalid("that isn't one of the closing reasons")
	}
	if reason == ReasonOther {
		other = strings.TrimSpace(other)
		if other == "" {
			// The error ends the move's transaction before anything is written.
			return sql.NullString{}, sql.NullString{}, invalid("write why the job was closed")
		}
		return nullString(string(reason)), nullString(other), nil
	}
	return nullString(string(reason)), sql.NullString{}, nil
}

// SetClosingReason changes the closing reason of a closed job. The status history keeps the
// reason chosen when the job was closed.
// @scenario [#history-keeps-original-closing-reason]
func (s *Service) SetClosingReason(ctx context.Context, id int64, reason ClosingReason, other string) error {
	r, o, err := checkClosingReason(reason, other)
	if err != nil {
		return err
	}
	return s.store.InTx(ctx, func(q store.Querier) error {
		job, err := q.GetJob(ctx, id)
		if err != nil {
			return notFound(err, ErrJobNotFound)
		}
		if Status(job.Status) != StatusClosed {
			return invalid("only a closed job has a closing reason")
		}
		return q.SetClosingReason(ctx, store.SetClosingReasonParams{ID: id, ClosingReason: r, ClosingReasonOther: o})
	})
}
