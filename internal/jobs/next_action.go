package jobs

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"callback/internal/store"
)

// NextActionDue is whether a next action needs attention, judged from its expected date.
type NextActionDue string

const (
	// DueNone is a next action with no expected date, one expected later, or one on a closed job.
	DueNone NextActionDue = ""
	// DueToday is a next action expected today.
	DueToday NextActionDue = "today"
	// DueOverdue is a next action whose expected date has passed.
	DueOverdue NextActionDue = "overdue"
)

// NextAction is what the user is waiting for or needs to do next on a job, and when.
type NextAction struct {
	Text string `json:"text"`
	// ExpectedDate is YYYY-MM-DD, or "" when no date is expected.
	ExpectedDate string `json:"expectedDate"`
	// Due marks a next action expected today or overdue. A closed job's never is.
	Due NextActionDue `json:"due"`
}

// nextActionFromRow turns a job's stored next action into a NextAction, or nil when it has none.
func nextActionFromRow(text, date sql.NullString, status Status, today string) *NextAction {
	if !text.Valid {
		return nil
	}
	return &NextAction{
		Text:         text.String,
		ExpectedDate: date.String,
		Due:          nextActionDue(date.String, status, today),
	}
}

// nextActionDue judges whether a next action expected on date is due. A closed job is done with,
// so its next action is never marked.
func nextActionDue(date string, status Status, today string) NextActionDue {
	switch {
	case date == "" || status == StatusClosed:
		return DueNone
	// YYYY-MM-DD strings sort in date order.
	case date < today:
		return DueOverdue
	case date == today:
		return DueToday
	default:
		return DueNone
	}
}

// SetNextAction sets what's next on a job and when it's expected, replacing any next action it
// had. expectedDate is YYYY-MM-DD, or "" for no expected date, and can be in the past. A job in
// any status can have one, and setting it doesn't move the job.
func (s *Service) SetNextAction(ctx context.Context, jobID int64, text, expectedDate string) (Job, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Job{}, invalid("write what the next action is")
	}
	if expectedDate != "" {
		if _, err := time.Parse(dateLayout, expectedDate); err != nil {
			return Job{}, invalid("the expected date must be a date like 2026-10-14")
		}
	}
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return Job{}, notFound(err, ErrJobNotFound)
	}
	err := s.store.SetNextAction(ctx, store.SetNextActionParams{
		ID:             jobID,
		NextAction:     nullString(text),
		NextActionDate: nullString(expectedDate),
	})
	if err != nil {
		return Job{}, err
	}
	return s.Job(ctx, jobID)
}

// ClearNextAction removes a job's next action, once it's done.
func (s *Service) ClearNextAction(ctx context.Context, jobID int64) (Job, error) {
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return Job{}, notFound(err, ErrJobNotFound)
	}
	if err := s.store.ClearNextAction(ctx, jobID); err != nil {
		return Job{}, err
	}
	return s.Job(ctx, jobID)
}
