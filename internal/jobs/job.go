package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"callback/internal/store"
)

// Details are the fields of a job the user fills in.
type Details struct {
	Company     string `json:"company"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Link        string `json:"link"`
	Salary      string `json:"salary"`
	Location    string `json:"location"`
	Notes       string `json:"notes"`
	// DateApplied is YYYY-MM-DD, or "" for no date applied.
	DateApplied string `json:"dateApplied"`
	// Interest is how interested the user is in the job, from 1 to 5 stars, or nil when it isn't
	// rated.
	Interest *int `json:"interest"`
}

// Job is everything about one job, as shown when its card is opened.
type Job struct {
	ID          int64  `json:"id"`
	Company     string `json:"company"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Link        string `json:"link"`
	Salary      string `json:"salary"`
	Location    string `json:"location"`
	Notes       string `json:"notes"`
	DateApplied string `json:"dateApplied"`
	Interest    *int   `json:"interest"`
	Status      Status `json:"status"`
	// ClosingReason and ClosingReasonOther are set only when the job is closed.
	ClosingReason      ClosingReason `json:"closingReason"`
	ClosingReasonOther string        `json:"closingReasonOther"`
	ClosingReasonText  string        `json:"closingReasonText"`
	// NextAction is what's next on the job, or nil when nothing is.
	NextAction *NextAction `json:"nextAction"`
	Documents  []Document  `json:"documents"`
	// Contacts are in the order they were added.
	Contacts []Contact `json:"contacts"`
	// Interviews are from earliest to latest, by date and then time, with untimed interviews last
	// on their day.
	Interviews []Interview `json:"interviews"`
	// StatusHistory is every status the job has been in, from oldest to newest.
	StatusHistory []StatusChange `json:"statusHistory"`
}

// AddJob adds a job to the top of the status column, or of Saved when status is "", and starts its
// status history with that status today.
// @scenario [features/job-status-history.feature#history-starts-when-added] A new job's history starts with the status it was added to
func (s *Service) AddJob(ctx context.Context, d Details, status Status) (Job, error) {
	if status == "" {
		status = StatusSaved
	}
	if !status.valid() {
		return Job{}, invalid(fmt.Sprintf("there's no %q column", status))
	}
	// A closed job needs a closing reason, which is chosen when a card is moved to Closed.
	if status == StatusClosed {
		return Job{}, invalid("a job can't be added straight to Closed; add it to another column, then move it to Closed")
	}
	d, err := s.validateDetails(d)
	if err != nil {
		return Job{}, err
	}

	var id int64
	err = s.store.InTx(ctx, func(q store.Querier) error {
		below, err := q.ListJobIDsInStatus(ctx, string(status))
		if err != nil {
			return err
		}
		id, err = q.CreateJob(ctx, store.CreateJobParams{
			Company:     d.Company,
			Role:        d.Role,
			Description: d.Description,
			Link:        d.Link,
			Salary:      d.Salary,
			Location:    d.Location,
			Notes:       d.Notes,
			Status:      string(status),
			Position:    0,
			DateApplied: nullString(d.DateApplied),
			Interest:    nullInterest(d.Interest),
		})
		if err != nil {
			return err
		}
		if err := s.recordStatusChange(ctx, q, id, status, sql.NullString{}, sql.NullString{}); err != nil {
			return err
		}
		// Every card already in the column moves down one.
		return renumber(ctx, q, below, 1)
	})
	if err != nil {
		return Job{}, err
	}
	return s.Job(ctx, id)
}

// Job returns a job with all its details, next action, documents, contacts, interviews, and
// status history.
// @scenario [features/job-status-history.feature#history-persists] The status history is the same after reopening the app
func (s *Service) Job(ctx context.Context, id int64) (Job, error) {
	row, err := s.store.GetJob(ctx, id)
	if err != nil {
		return Job{}, notFound(err, ErrJobNotFound)
	}
	docs, err := s.store.ListDocumentsForJob(ctx, id)
	if err != nil {
		return Job{}, err
	}
	contacts, err := s.store.ListContactsForJob(ctx, id)
	if err != nil {
		return Job{}, err
	}
	interviews, err := s.store.ListInterviewsForJob(ctx, id)
	if err != nil {
		return Job{}, err
	}
	history, err := s.store.ListStatusChangesForJob(ctx, id)
	if err != nil {
		return Job{}, err
	}

	reason := ClosingReason(row.ClosingReason.String)
	today := s.today()
	job := Job{
		ID:                 row.ID,
		Company:            row.Company,
		Role:               row.Role,
		Description:        row.Description,
		Link:               row.Link,
		Salary:             row.Salary,
		Location:           row.Location,
		Notes:              row.Notes,
		DateApplied:        row.DateApplied.String,
		Interest:           interestFromRow(row.Interest),
		Status:             Status(row.Status),
		ClosingReason:      reason,
		ClosingReasonOther: row.ClosingReasonOther.String,
		ClosingReasonText:  closingReasonText(reason, row.ClosingReasonOther.String),
		NextAction:         nextActionFromRow(row.NextAction, row.NextActionDate, Status(row.Status), today),
		Documents:          make([]Document, 0, len(docs)),
		Contacts:           make([]Contact, 0, len(contacts)),
		Interviews:         make([]Interview, 0, len(interviews)),
		StatusHistory:      make([]StatusChange, 0, len(history)),
	}
	for _, doc := range docs {
		job.Documents = append(job.Documents, documentFromRow(doc))
	}
	for _, c := range contacts {
		job.Contacts = append(job.Contacts, contactFromRow(c))
	}
	for _, i := range interviews {
		job.Interviews = append(job.Interviews, interviewFromRow(i, today))
	}
	for _, change := range history {
		job.StatusHistory = append(job.StatusHistory, statusChangeFromRow(change))
	}
	return job, nil
}

// UpdateJob replaces a job's details. Nothing is saved if any detail is invalid. The status
// history isn't touched, so changing the date applied doesn't change it.
// @scenario [features/job-status-history.feature#history-ignores-date-applied] Changing the date applied doesn't change the history
func (s *Service) UpdateJob(ctx context.Context, id int64, d Details) (Job, error) {
	d, err := s.validateDetails(d)
	if err != nil {
		return Job{}, err
	}
	if _, err := s.store.GetJob(ctx, id); err != nil {
		return Job{}, notFound(err, ErrJobNotFound)
	}
	err = s.store.UpdateJobDetails(ctx, store.UpdateJobDetailsParams{
		ID:          id,
		Company:     d.Company,
		Role:        d.Role,
		Description: d.Description,
		Link:        d.Link,
		Salary:      d.Salary,
		Location:    d.Location,
		Notes:       d.Notes,
		DateApplied: nullString(d.DateApplied),
		Interest:    nullInterest(d.Interest),
	})
	if err != nil {
		return Job{}, err
	}
	return s.Job(ctx, id)
}

// validateDetails checks the details the user entered and returns them tidied: company and role
// trimmed of surrounding space.
func (s *Service) validateDetails(d Details) (Details, error) {
	d.Company = strings.TrimSpace(d.Company)
	d.Role = strings.TrimSpace(d.Role)
	if d.Company == "" {
		return d, invalid("company is required")
	}
	if d.Role == "" {
		return d, invalid("role is required")
	}
	if d.Link != "" && !isWebAddress(d.Link) {
		return d, invalid("the link must be a web address, like https://example.com/jobs/123")
	}
	if d.DateApplied != "" {
		if _, err := time.Parse(dateLayout, d.DateApplied); err != nil {
			return d, invalid("the date applied must be a date like 2026-10-01")
		}
		// YYYY-MM-DD strings sort in date order.
		if d.DateApplied > s.today() {
			return d, invalid("the date applied can't be in the future")
		}
	}
	if d.Interest != nil && (*d.Interest < MinInterest || *d.Interest > MaxInterest) {
		return d, invalid(fmt.Sprintf("the interest rating must be from %d to %d stars", MinInterest, MaxInterest))
	}
	return d, nil
}

// isWebAddress reports whether link is an http or https URL with a host.
func isWebAddress(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// DeleteJob deletes a job, its contacts, its interviews, its status history, and its attached
// documents, both the rows and the stored copies.
func (s *Service) DeleteJob(ctx context.Context, id int64) error {
	var docs []store.Document
	err := s.store.InTx(ctx, func(q store.Querier) error {
		if _, err := q.GetJob(ctx, id); err != nil {
			return notFound(err, ErrJobNotFound)
		}
		var err error
		if docs, err = q.ListDocumentsForJob(ctx, id); err != nil {
			return err
		}
		// Deleting the job deletes its document, contact, interview, and status change rows too,
		// through ON DELETE CASCADE.
		return q.DeleteJob(ctx, id)
	})
	if err != nil {
		return err
	}

	// Files go after the rows are gone, so a failure here leaves an orphaned file rather than a row
	// pointing at a missing one. See
	// docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
	var errs []error
	for _, doc := range docs {
		if err := s.files.Remove(doc.StoredPath); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("the job was deleted, but some of its document files couldn't be removed: %w", err)
	}
	return nil
}

// OpenJobLink opens a job's posting link in the web browser.
func (s *Service) OpenJobLink(ctx context.Context, id int64) error {
	job, err := s.store.GetJob(ctx, id)
	if err != nil {
		return notFound(err, ErrJobNotFound)
	}
	if job.Link == "" {
		return invalid("this job has no link")
	}
	// Checked again at open time so a stored link can never make the OS launch a local program.
	// See docs/adr/0017-open-documents-and-links-with-a-hand-written-os-opener.md.
	if !isWebAddress(job.Link) {
		return invalid("the link must be a web address, like https://example.com/jobs/123")
	}
	return s.opener.Open(job.Link)
}

// renumber sets the positions of ids, in order, starting at start.
func renumber(ctx context.Context, q store.Querier, ids []int64, start int) error {
	for i, id := range ids {
		if err := q.SetJobPosition(ctx, store.SetJobPositionParams{ID: id, Position: int64(start + i)}); err != nil {
			return err
		}
	}
	return nil
}
