package jobs

import (
	"context"
	"database/sql"
	"time"

	"callback/internal/store"
)

// StatusChange is one entry in a job's status history: the status it moved to and the date it
// did. The history is a record, so nothing changes or removes an entry once it's added.
type StatusChange struct {
	Status Status `json:"status"`
	// Date is YYYY-MM-DD.
	Date string `json:"date"`
	// ClosingReason and ClosingReasonText are the reason chosen when the job moved to Closed, and
	// are empty for any other status. Changing the job's closing reason later doesn't change them.
	ClosingReason     ClosingReason `json:"closingReason"`
	ClosingReasonText string        `json:"closingReasonText"`
}

// recordStatusChange adds a job's move to status, today, to its status history. reason and other
// are the closing reason stored with a move to Closed, and NULL otherwise.
func (s *Service) recordStatusChange(ctx context.Context, q store.Querier, jobID int64, status Status, reason, other sql.NullString) error {
	return q.AddStatusChange(ctx, store.AddStatusChangeParams{
		JobID:              jobID,
		Status:             string(status),
		Date:               s.today(),
		ClosingReason:      reason,
		ClosingReasonOther: other,
	})
}

func statusChangeFromRow(row store.StatusChange) StatusChange {
	reason := ClosingReason(row.ClosingReason.String)
	return StatusChange{
		Status:            Status(row.Status),
		Date:              row.Date,
		ClosingReason:     reason,
		ClosingReasonText: closingReasonText(reason, row.ClosingReasonOther.String),
	}
}

// daysInStatus returns how many days each job has been in its current status, counted from the
// date of its latest status change to today. A job with no history isn't in the map.
func (s *Service) daysInStatus(ctx context.Context) (map[int64]int, error) {
	rows, err := s.store.ListCurrentStatusDates(ctx)
	if err != nil {
		return nil, err
	}
	today, err := time.Parse(dateLayout, s.today())
	if err != nil {
		return nil, err
	}
	days := make(map[int64]int, len(rows))
	for _, row := range rows {
		since, err := time.Parse(dateLayout, row.Date)
		if err != nil {
			return nil, err
		}
		// Both dates parse as midnight UTC, so the difference is a whole number of days.
		days[row.JobID] = max(int(today.Sub(since).Hours()/24), 0)
	}
	return days, nil
}
