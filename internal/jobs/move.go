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
// transaction. See docs/adr/0015-store-card-order-as-an-integer-position.md.
func (s *Service) MoveJob(ctx context.Context, m Move) error {
	if !m.Status.valid() {
		return invalid("there's no such column")
	}
	return s.store.InTx(ctx, func(q store.Querier) error {
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "<from>"
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Applied"
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
		// @given [features/job-board.feature#reopen-clears-reason] I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
		// @and [features/job-board.feature] the job has a job description, a link, and an attached resume
		// @and [features/job-board.feature] I have a job at "Acme" in "Saved"
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
		// @and [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Saved" with no date applied
		case changesStatus && m.Status == StatusApplied && !date.Valid:
			// @then [features/job-board.feature] the job's date applied is 2026-10-09
			date = nullString(s.today())
		// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "<from>" with a date applied of 2026-10-01
		case changesStatus && m.Status == StatusSaved:
			// @then [features/job-board.feature] the job has no date applied
			date = sql.NullString{}
		// @and [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing" with a date applied of 2026-10-01
		// @and [features/job-board.feature] the job at "Acme" has a date applied of 2026-10-01
		// @and [features/job-board.feature] the job at "Acme" has no date applied
		default:
			// Any other move, including reordering within Saved or Applied, keeps the date applied
			// as it is.
			// @then [features/job-board.feature] the job's date applied is still 2026-10-01
			// @then [features/job-board.feature#reorder-applied-no-date] the job has no date applied
		}

		// @given [features/job-board.feature] the "Saved" column holds the jobs at "Acme", "Globex", and "Initech" in that order
		// @given [features/job-board.feature] the "Applied" column holds the jobs at "Globex" and "Initech" in that order
		// @given [features/job-board.feature] the "Saved" column holds the jobs at "Globex" and "Acme" in that order
		// @given [features/job-board.feature] the "Applied" column holds the jobs at "Globex" and "Acme" in that order
		// @given [features/job-board.feature] the "Closed" column holds the jobs at "Globex" and "Acme" in that order
		column, err := q.ListJobIDsInStatus(ctx, string(m.Status))
		if err != nil {
			return err
		}
		column = slices.DeleteFunc(column, func(id int64) bool { return id == job.ID })
		// @when [features/job-board.feature] I move the "Initech" card above the "Acme" card
		// @when [features/job-board.feature] I move the "Acme" card above the "Globex" card
		// @when [features/job-board.feature] I move the "Acme" card into "Applied" between "Globex" and "Initech"
		index := min(max(m.Index, 0), len(column))
		column = slices.Insert(column, index, job.ID)

		// Only the placement columns are written, so the job's details and documents stay as they
		// were.
		// @when [features/job-board.feature] I move the card to "Applied"
		// @when [features/job-board.feature] I move the card to "<to>"
		// @when [features/job-board.feature] I move the card to "Interviewing"
		// @when [features/job-board.feature] I move the card to "Closed"
		// @when [features/job-board.feature] I move the card to "Saved"
		// @and [features/job-board.feature] the job description, link, and attached resume are unchanged
		err = q.UpdateJobPlacement(ctx, store.UpdateJobPlacementParams{
			ID: job.ID,
			// @then [features/job-board.feature] the card appears in the "Applied" column
			// @then [features/job-board.feature] the card appears in the "<to>" column
			// @then [features/job-board.feature] the card appears in the "Closed" column
			// @then [features/job-board.feature] the card appears in the "Interviewing" column
			Status:             string(m.Status),
			Position:           int64(index),
			DateApplied:        date,
			ClosingReason:      reason,
			ClosingReasonOther: other,
		})
		if err != nil {
			return err
		}
		// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
		// @then [features/job-board.feature] the "Applied" column shows "Globex", "Acme", and "Initech" in that order
		if err := renumber(ctx, q, column, 0); err != nil {
			return err
		}

		if changesStatus {
			// Close the gap the card left in its old column.
			old, err := q.ListJobIDsInStatus(ctx, string(from))
			if err != nil {
				return err
			}
			// @and [features/job-board.feature] the card no longer appears in the "Saved" column
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
		// @and [features/job-board.feature] the job has no closing reason
		return sql.NullString{}, sql.NullString{}, nil
	// @and [features/job-board.feature] the job at "Acme" has the closing reason "Rejected"
	case !changesStatus:
		// Reordering within Closed keeps the reason the job already has.
		// @then [features/job-board.feature#reorder-closed-keeps-reason] the card shows the closing reason "Rejected"
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
		// @and [features/job-board.feature] I choose the closing reason "Other" and write "Company went through layoffs"
		other = strings.TrimSpace(other)
		// @and [features/job-board.feature] I choose the closing reason "Other" without writing a reason
		if other == "" {
			// The error ends the move's transaction before anything is written.
			// @then [features/job-board.feature#close-other-needs-text] the card stays in the "Interviewing" column
			// @and [features/job-board.feature] I am told to write why the job was closed
			return sql.NullString{}, sql.NullString{}, invalid("write why the job was closed")
		}
		return nullString(string(reason)), nullString(other), nil
	}
	// @and [features/job-board.feature] I choose the closing reason "<reason>"
	return nullString(string(reason)), sql.NullString{}, nil
}

// SetClosingReason changes the closing reason of a closed job.
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
		// @given [features/job-board.feature#change-closing-reason] I have a job for "Software Engineer" at "Acme" in "Closed" with the closing reason "No response"
		if Status(job.Status) != StatusClosed {
			return invalid("only a closed job has a closing reason")
		}
		// @when [features/job-board.feature] I change the closing reason to "Rejected"
		return q.SetClosingReason(ctx, store.SetClosingReasonParams{ID: id, ClosingReason: r, ClosingReasonOther: o})
	})
}
