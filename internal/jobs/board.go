package jobs

import (
	"context"
	"fmt"

	"callback/internal/store"
)

// Board is every job, laid out in status columns.
type Board struct {
	Columns []Column `json:"columns"`
}

// Column is one status on the board with its cards, top to bottom.
type Column struct {
	Status Status `json:"status"`
	Label  string `json:"label"`
	Count  int    `json:"count"`
	Cards  []Card `json:"cards"`
}

// Card is a job as shown on the board.
type Card struct {
	ID                int64         `json:"id"`
	Company           string        `json:"company"`
	Role              string        `json:"role"`
	Location          string        `json:"location"`
	DateApplied       string        `json:"dateApplied"`
	Interest          *int          `json:"interest"`
	ClosingReason     ClosingReason `json:"closingReason"`
	ClosingReasonText string        `json:"closingReasonText"`
	DocumentCount     int           `json:"documentCount"`
	// NextInterview is the job's soonest interview that hasn't passed, or nil when it has none.
	NextInterview *NextInterview `json:"nextInterview"`
	// NextAction is what's next on the job, or nil when nothing is.
	NextAction *NextAction `json:"nextAction"`
	// DaysInStatus is how many days the job has been in its current status, from 0 for today, or
	// nil when it has no status history.
	DaysInStatus *int `json:"daysInStatus"`
}

// Board returns every job, laid out in status columns.
// @scenario [#card-shows-days-in-status] [#days-in-status-from-added] [#days-in-status-any-column]
func (s *Service) Board(ctx context.Context) (Board, error) {
	rows, err := s.store.ListJobs(ctx)
	if err != nil {
		return Board{}, err
	}
	next, err := s.nextInterviews(ctx)
	if err != nil {
		return Board{}, err
	}
	days, err := s.daysInStatus(ctx)
	if err != nil {
		return Board{}, err
	}

	today := s.today()
	var board Board
	index := map[Status]int{}
	for i, status := range Statuses {
		index[status] = i
		board.Columns = append(board.Columns, Column{Status: status, Label: status.Label(), Cards: []Card{}})
	}

	for _, row := range rows {
		i, ok := index[Status(row.Status)]
		if !ok {
			return Board{}, fmt.Errorf("job %d has unknown status %q", row.ID, row.Status)
		}
		card := cardFromRow(row, today)
		card.NextInterview = next[row.ID]
		if d, ok := days[row.ID]; ok {
			card.DaysInStatus = &d
		}
		board.Columns[i].Cards = append(board.Columns[i].Cards, card)
	}

	for i := range board.Columns {
		board.Columns[i].Count = len(board.Columns[i].Cards)
	}
	return board, nil
}

func cardFromRow(row store.ListJobsRow, today string) Card {
	reason := ClosingReason(row.ClosingReason.String)
	return Card{
		ID:                row.ID,
		Company:           row.Company,
		Role:              row.Role,
		Location:          row.Location,
		DateApplied:       row.DateApplied.String,
		Interest:          interestFromRow(row.Interest),
		ClosingReason:     reason,
		ClosingReasonText: closingReasonText(reason, row.ClosingReasonOther.String),
		DocumentCount:     int(row.DocumentCount),
		NextAction:        nextActionFromRow(row.NextAction, row.NextActionDate, Status(row.Status), today),
	}
}

// closingReasonText is the closing reason as the user sees it: the reason's label, or for Other,
// the text the user wrote.
func closingReasonText(reason ClosingReason, other string) string {
	if reason == ReasonOther {
		return other
	}
	return reason.Label()
}
