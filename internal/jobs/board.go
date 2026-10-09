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
	ClosingReason     ClosingReason `json:"closingReason"`
	ClosingReasonText string        `json:"closingReasonText"`
	DocumentCount     int           `json:"documentCount"`
}

// Board returns every job, laid out in status columns.
func (s *Service) Board(ctx context.Context) (Board, error) {
	// @given [features/job-board.feature] I open the job search board
	// @when [features/job-board.feature] I open the job search board
	// @given [features/job-cards.feature] I open the job search board
	// @when [features/job-documents.feature] I look at the card on the board
	// @given [features/job-board.feature] I have not added any jobs
	// @given [features/job-board.feature] I have 3 jobs in "Saved" and 1 job in "Applied"
	// @given [features/job-board.feature] the "Saved" column holds the jobs at "Initech", "Acme", and "Globex" in that order
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme" with 2 attached documents
	rows, err := s.store.ListJobs(ctx)
	if err != nil {
		return Board{}, err
	}

	var board Board
	index := map[Status]int{}
	// @then [features/job-board.feature] I see the columns "Saved", "Applied", "Interviewing", "Offer", and "Closed" in that order
	for i, status := range Statuses {
		index[status] = i
		// @then [features/job-board.feature] every status column is empty
		board.Columns = append(board.Columns, Column{Status: status, Label: status.Label(), Cards: []Card{}})
	}

	for _, row := range rows {
		i, ok := index[Status(row.Status)]
		if !ok {
			return Board{}, fmt.Errorf("job %d has unknown status %q", row.ID, row.Status)
		}
		// @then [features/job-board.feature] the card for "Software Engineer" at "Acme" is in the "Interviewing" column
		board.Columns[i].Cards = append(board.Columns[i].Cards, cardFromRow(row))
	}

	for i := range board.Columns {
		// @then [features/job-board.feature] the "Saved" column shows a count of 3
		// @and [features/job-board.feature] the "Applied" column shows a count of 1
		// @and [features/job-board.feature] the "Interviewing" column shows a count of 0
		board.Columns[i].Count = len(board.Columns[i].Cards)
	}
	return board, nil
}

func cardFromRow(row store.ListJobsRow) Card {
	reason := ClosingReason(row.ClosingReason.String)
	return Card{
		ID: row.ID,
		// @and [features/job-cards.feature] the card shows the company, the role, and the location
		// @then [features/job-board.feature] the card still shows the company "Acme" and the role "Software Engineer"
		Company:  row.Company,
		Role:     row.Role,
		Location: row.Location,
		// @and [features/job-cards.feature] the card shows the date applied
		DateApplied:       row.DateApplied.String,
		ClosingReason:     reason,
		ClosingReasonText: closingReasonText(reason, row.ClosingReasonOther.String),
		// @then [features/job-documents.feature] the card shows that it has 2 documents
		DocumentCount: int(row.DocumentCount),
	}
}

// closingReasonText is the closing reason as the user sees it: the reason's label, or for Other,
// the text the user wrote.
func closingReasonText(reason ClosingReason, other string) string {
	if reason == ReasonOther {
		// @and [features/job-board.feature] the card shows the closing reason "Company went through layoffs"
		return other
	}
	// @and [features/job-board.feature] the card shows the closing reason "<reason>"
	// @then [features/job-board.feature] the card shows the closing reason "Rejected"
	return reason.Label()
}
