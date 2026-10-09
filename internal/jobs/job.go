package jobs

import (
	"context"
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
	Status      Status `json:"status"`
	// ClosingReason and ClosingReasonOther are set only when the job is closed.
	ClosingReason      ClosingReason `json:"closingReason"`
	ClosingReasonOther string        `json:"closingReasonOther"`
	ClosingReasonText  string        `json:"closingReasonText"`
	Documents          []Document    `json:"documents"`
}

// AddJob adds a job to the top of the status column, or of Saved when status is "".
func (s *Service) AddJob(ctx context.Context, d Details, status Status) (Job, error) {
	if status == "" {
		// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Saved" column
		status = StatusSaved
	}
	if !status.valid() {
		return Job{}, invalid(fmt.Sprintf("there's no %q column", status))
	}
	// A closed job needs a closing reason, which is chosen when a card is moved to Closed.
	// @when [features/job-cards.feature] I try to add a job with company "Acme" and role "Software Engineer" to the "Closed" column
	if status == StatusClosed {
		// @then [features/job-cards.feature] the job is not added
		// @and [features/job-cards.feature] I am told to add the job to another column and move it to Closed
		return Job{}, invalid("a job can't be added straight to Closed; add it to another column, then move it to Closed")
	}
	// @when [features/job-cards.feature] I try to add a job with company "<company>" and role "<role>"
	d, err := s.validateDetails(d)
	if err != nil {
		// @then [features/job-cards.feature] the job is not added
		return Job{}, err
	}

	var id int64
	err = s.store.InTx(ctx, func(q store.Querier) error {
		// @given [features/job-board.feature] the "Saved" column holds the jobs at "Globex" and "Initech" in that order
		below, err := q.ListJobIDsInStatus(ctx, string(status))
		if err != nil {
			return err
		}
		// @when [features/job-cards.feature] I add a job with:
		// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" and nothing else
		// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" to the "Applied" column
		// @when [features/job-board.feature] I add a job at "Acme" to the "Saved" column
		id, err = q.CreateJob(ctx, store.CreateJobParams{
			Company:     d.Company,
			Role:        d.Role,
			Description: d.Description,
			Link:        d.Link,
			Salary:      d.Salary,
			Location:    d.Location,
			Notes:       d.Notes,
			// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Applied" column
			Status: string(status),
			// @then [features/job-board.feature] the "Saved" column shows "Acme", "Globex", and "Initech" in that order
			Position:    0,
			DateApplied: nullString(d.DateApplied),
		})
		if err != nil {
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

// Job returns a job with all its details and documents.
// @when [features/job-cards.feature] I open the card
func (s *Service) Job(ctx context.Context, id int64) (Job, error) {
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with every detail filled in
	row, err := s.store.GetJob(ctx, id)
	if err != nil {
		return Job{}, notFound(err, ErrJobNotFound)
	}
	// @and [features/job-cards.feature] I see the documents attached to the job
	// @then [features/job-documents.feature] the job lists all three documents with their kinds
	docs, err := s.store.ListDocumentsForJob(ctx, id)
	if err != nil {
		return Job{}, err
	}

	reason := ClosingReason(row.ClosingReason.String)
	job := Job{
		ID: row.ID,
		// @then [features/job-cards.feature] I see the company, role, job description, link, salary, location, notes, and date applied
		Company: row.Company,
		Role:    row.Role,
		// @and [features/job-cards.feature] the job shows the new job description
		Description: row.Description,
		Link:        row.Link,
		// @then [features/job-cards.feature] the job shows the salary "<salary>"
		Salary:   row.Salary,
		Location: row.Location,
		// @then [features/job-cards.feature] the job's notes show "Recruiter call went well, follow up Friday."
		// @then [features/job-cards.feature] the job's notes show both lines
		Notes: row.Notes,
		// @then [features/job-cards.feature] the job's date applied is 2026-10-01
		DateApplied:        row.DateApplied.String,
		Status:             Status(row.Status),
		ClosingReason:      reason,
		ClosingReasonOther: row.ClosingReasonOther.String,
		ClosingReasonText:  closingReasonText(reason, row.ClosingReasonOther.String),
		Documents:          make([]Document, 0, len(docs)),
	}
	for _, doc := range docs {
		job.Documents = append(job.Documents, documentFromRow(doc))
	}
	return job, nil
}

// UpdateJob replaces a job's details. Nothing is saved if any detail is invalid.
func (s *Service) UpdateJob(ctx context.Context, id int64, d Details) (Job, error) {
	// @when [features/job-cards.feature] I try to set the link to "not a url"
	// @when [features/job-cards.feature] I try to set the date applied to 2026-10-20
	d, err := s.validateDetails(d)
	if err != nil {
		// @then [features/job-cards.feature] the link is not saved
		// @then [features/job-cards.feature] the date applied is not saved
		return Job{}, err
	}
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
	// @and [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with the notes "Referred by Dana."
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" in "Applied"
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with a date applied of 2026-10-01
	if _, err := s.store.GetJob(ctx, id); err != nil {
		return Job{}, notFound(err, ErrJobNotFound)
	}
	err = s.store.UpdateJobDetails(ctx, store.UpdateJobDetailsParams{
		ID:      id,
		Company: d.Company,
		// @when [features/job-cards.feature] I change the role to "Senior Software Engineer"
		// @then [features/job-cards.feature] the card shows the role "Senior Software Engineer"
		Role: d.Role,
		// @and [features/job-cards.feature] I change the job description
		Description: d.Description,
		Link:        d.Link,
		// @when [features/job-cards.feature] I set the salary to "<salary>"
		Salary:   d.Salary,
		Location: d.Location,
		// @when [features/job-cards.feature] I write in the notes "Recruiter call went well, follow up Friday."
		// @when [features/job-cards.feature] I add "Phone screen booked for 10/14." on a new line in the notes
		Notes: d.Notes,
		// @when [features/job-cards.feature] I set the date applied to 2026-10-01
		// @when [features/job-cards.feature] I clear the date applied
		// @then [features/job-cards.feature] the job has no date applied
		DateApplied: nullString(d.DateApplied),
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
	// @and [features/job-cards.feature] I am told that the <missing> is required
	if d.Company == "" {
		return d, invalid("company is required")
	}
	if d.Role == "" {
		return d, invalid("role is required")
	}
	if d.Link != "" && !isWebAddress(d.Link) {
		// @and [features/job-cards.feature] I am told the link must be a web address
		return d, invalid("the link must be a web address, like https://example.com/jobs/123")
	}
	if d.DateApplied != "" {
		if _, err := time.Parse(dateLayout, d.DateApplied); err != nil {
			return d, invalid("the date applied must be a date like 2026-10-01")
		}
		// YYYY-MM-DD strings sort in date order.
		if d.DateApplied > s.today() {
			// @and [features/job-cards.feature] I am told the date applied can't be in the future
			return d, invalid("the date applied can't be in the future")
		}
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

// DeleteJob deletes a job and its attached documents, both the rows and the stored copies.
// @when [features/job-cards.feature] I delete the job
func (s *Service) DeleteJob(ctx context.Context, id int64) error {
	var docs []store.Document
	err := s.store.InTx(ctx, func(q store.Querier) error {
		// @given [features/job-cards.feature#delete-job] I have a job for "Software Engineer" at "Acme"
		if _, err := q.GetJob(ctx, id); err != nil {
			return notFound(err, ErrJobNotFound)
		}
		var err error
		if docs, err = q.ListDocumentsForJob(ctx, id); err != nil {
			return err
		}
		// Deleting the job deletes its document rows too, through ON DELETE CASCADE.
		// @then [features/job-cards.feature] the card for "Software Engineer" at "Acme" no longer appears on the board
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
		// @and [features/job-cards.feature] its attached documents are removed too
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
// @when [features/job-cards.feature] I open the job's link
func (s *Service) OpenJobLink(ctx context.Context, id int64) error {
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with the link "https://acme.example.com/jobs/123"
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
	// @then [features/job-cards.feature] the posting opens in my web browser
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
