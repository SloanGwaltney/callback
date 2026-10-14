package cli

import (
	"fmt"
	"io"
	"strings"

	"callback/internal/jobs"
)

// Dates are written as the services give them, YYYY-MM-DD, in text and in JSON.

// cardJSON is a job as `job list` lists it.
type cardJSON struct {
	jobs.Card
	Status      jobs.Status `json:"status"`
	StatusLabel string      `json:"statusLabel"`
}

// jobJSON is a job as `job show` shows it: every detail, with its documents' stored copies.
type jobJSON struct {
	jobs.Job
	StatusLabel string `json:"statusLabel"`
	// Documents replaces the embedded Job's, adding where each stored copy is.
	Documents []documentJSON `json:"documents"`
}

// documentJSON is an attached document with where its stored copy is.
type documentJSON struct {
	jobs.Document
	StoredPath string `json:"storedPath"`
	// Missing is set when the stored copy is gone.
	Missing bool `json:"missing"`
}

func jobJSONFrom(job jobs.Job, files []jobs.DocumentFile) jobJSON {
	byID := map[int64]jobs.DocumentFile{}
	for _, f := range files {
		byID[f.DocumentID] = f
	}
	out := jobJSON{Job: job, StatusLabel: job.Status.Label(), Documents: make([]documentJSON, 0, len(job.Documents))}
	for _, doc := range job.Documents {
		f := byID[doc.ID]
		out.Documents = append(out.Documents, documentJSON{Document: doc, StoredPath: f.Path, Missing: f.Missing})
	}
	return out
}

// writeCardLine writes a job as one line of `job list`, leaving out what it doesn't have.
func writeCardLine(w io.Writer, card cardJSON) {
	status := card.StatusLabel
	if card.ClosingReasonText != "" {
		status += " (" + card.ClosingReasonText + ")"
	}
	switch d := card.DaysInStatus; {
	case d == nil:
	case *d == 0:
		status += " since today"
	default:
		status += " for " + plural(*d, "day")
	}
	fields := []string{fmt.Sprintf("#%d", card.ID), status, card.Company, card.Role}
	if card.Location != "" {
		fields = append(fields, card.Location)
	}
	if card.DateApplied != "" {
		fields = append(fields, "applied "+card.DateApplied)
	}
	if card.Interest != nil {
		fields = append(fields, plural(*card.Interest, "star"))
	}
	if card.NextAction != nil {
		fields = append(fields, "next: "+nextActionText(card.NextAction))
	}
	fmt.Fprintln(w, strings.Join(fields, " | "))
}

// nextActionText is a next action with its expected date and whether it's due or overdue.
func nextActionText(a *jobs.NextAction) string {
	text := a.Text
	if a.ExpectedDate != "" {
		text += ", expected " + a.ExpectedDate
	}
	switch a.Due {
	case jobs.DueToday:
		text += " (due today)"
	case jobs.DueOverdue:
		text += " (overdue)"
	}
	return text
}

// writeJob writes everything about a job, as `job show` and every change to a job print it.
func writeJob(w io.Writer, job jobJSON) {
	fmt.Fprintf(w, "Job #%d: %s at %s\n", job.ID, job.Role, job.Company)
	line := func(label, value string) {
		if value == "" {
			value = "(none)"
		}
		fmt.Fprintf(w, "%s: %s\n", label, value)
	}
	line("Status", job.StatusLabel)
	if job.Status == jobs.StatusClosed {
		line("Closing reason", job.ClosingReasonText)
	}
	line("Company", job.Company)
	line("Role", job.Role)
	line("Location", job.Location)
	line("Salary", job.Salary)
	line("Posting link", job.Link)
	line("Date applied", job.DateApplied)
	interest := ""
	if job.Interest != nil {
		interest = plural(*job.Interest, "star")
	}
	line("Interest", interest)
	next := ""
	if job.NextAction != nil {
		next = nextActionText(job.NextAction)
	}
	line("Next action", next)

	section := func(title string, empty bool, write func()) {
		fmt.Fprintf(w, "\n%s:\n", title)
		if empty {
			fmt.Fprintln(w, "  (none)")
			return
		}
		write()
	}
	section("Description", job.Description == "", func() { writeIndented(w, job.Description, "  ") })
	section("Notes", job.Notes == "", func() { writeIndented(w, job.Notes, "  ") })
	section("Contacts", len(job.Contacts) == 0, func() {
		for _, c := range job.Contacts {
			fmt.Fprintf(w, "  %s\n", contactLine(c))
			writeIndented(w, c.Notes, "    ")
		}
	})
	section("Interviews", len(job.Interviews) == 0, func() {
		for _, i := range job.Interviews {
			fmt.Fprintf(w, "  %s\n", interviewLine(i))
			writeIndented(w, i.Notes, "    ")
		}
	})
	section("Documents", len(job.Documents) == 0, func() {
		for _, d := range job.Documents {
			fmt.Fprintf(w, "  %s\n", documentLine(d))
		}
	})
	section("Status history", len(job.StatusHistory) == 0, func() {
		for _, h := range job.StatusHistory {
			entry := h.Date + " " + h.Status.Label()
			if h.ClosingReasonText != "" {
				entry += " (" + h.ClosingReasonText + ")"
			}
			fmt.Fprintf(w, "  %s\n", entry)
		}
	})
}

// contactLine is a contact on one line, starting with its ID, leaving out what it doesn't have.
func contactLine(c jobs.Contact) string {
	fields := []string{fmt.Sprintf("#%d %s", c.ID, c.Name), c.Relationship.Label()}
	for _, f := range [][2]string{{"email", c.Email}, {"phone", c.Phone}, {"profile", c.ProfileLink}} {
		if f[1] != "" {
			fields = append(fields, f[0]+" "+f[1])
		}
	}
	return strings.Join(fields, " | ")
}

// interviewLine is an interview on one line, starting with its ID.
func interviewLine(i jobs.Interview) string {
	when := i.Date
	if i.Time != "" {
		when += " " + i.Time
	}
	past := ""
	if i.Past {
		past = " (past)"
	}
	return fmt.Sprintf("#%d %s | %s%s", i.ID, when, i.Kind.Label(), past)
}

// documentLine is an attached document on one line, starting with its ID, with where its stored
// copy is.
func documentLine(d documentJSON) string {
	missing := ""
	if d.Missing {
		missing = " (missing: the stored copy is gone)"
	}
	return fmt.Sprintf("#%d %s | %s | stored copy: %s%s", d.ID, d.FileName, d.Kind.Label(), d.StoredPath, missing)
}

// writeContact writes a contact, as adding or changing one prints it.
func writeContact(w io.Writer, c jobs.Contact) {
	fmt.Fprintf(w, "Contact on job #%d: %s\n", c.JobID, contactLine(c))
	writeIndented(w, c.Notes, "  ")
}

// writeInterview writes an interview, as adding or changing one prints it.
func writeInterview(w io.Writer, i jobs.Interview) {
	fmt.Fprintf(w, "Interview on job #%d: %s\n", i.JobID, interviewLine(i))
	writeIndented(w, i.Notes, "  ")
}

// writeDocument writes an attached document, as attaching or changing one prints it.
func writeDocument(w io.Writer, d documentJSON) {
	fmt.Fprintf(w, "Document on job #%d: %s\n", d.JobID, documentLine(d))
}

// writeIndented writes text with every line indented, keeping its blank lines. It writes nothing
// for no text.
func writeIndented(w io.Writer, text, indent string) {
	if text == "" {
		return
	}
	for line := range strings.SplitSeq(strings.TrimRight(text, "\n"), "\n") {
		fmt.Fprintln(w, strings.TrimRight(indent+line, " "))
	}
}

// plural writes a count with its noun, like "1 star" or "4 stars".
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
