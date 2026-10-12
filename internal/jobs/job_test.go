package jobs

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"callback/internal/store"
)

// @scenario [features/job-cards.feature#add-job] Add a job with all its details
func TestAddJobWithAllDetails(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I open the job search board
	h.board(t)

	// @when [features/job-cards.feature] I add a job with:
	added, err := h.svc.AddJob(h.ctx, Details{
		Company:     "Acme",
		Role:        "Software Engineer",
		Description: "Build and maintain internal tools.",
		Link:        "https://acme.example.com/jobs/123",
		Salary:      "$120k - $140k",
		Location:    "Remote (US)",
		Notes:       "Referred by Dana from the meetup.",
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Saved" column
	card, status := h.card(t, added.ID)
	if status != StatusSaved {
		t.Errorf("card is in %q, want Saved", status)
	}
	// @and [features/job-cards.feature] the card shows the company, the role, and the location
	if card.Company != "Acme" || card.Role != "Software Engineer" || card.Location != "Remote (US)" {
		t.Errorf("card = %+v", card)
	}

	got := h.job(t, added.ID)
	want := Job{
		ID: added.ID, Company: "Acme", Role: "Software Engineer", Description: "Build and maintain internal tools.",
		Link: "https://acme.example.com/jobs/123", Salary: "$120k - $140k", Location: "Remote (US)",
		Notes: "Referred by Dana from the meetup.", Status: StatusSaved, Documents: []Document{},
		Contacts: []Contact{},
	}
	if !jobsEqual(got, want) {
		t.Errorf("job = %+v, want %+v", got, want)
	}
}

// @scenario [features/job-cards.feature#add-job-minimum] Add a job with only a company and a role
func TestAddJobWithOnlyCompanyAndRole(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I open the job search board
	h.board(t)

	// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" and nothing else
	added, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, "")
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Saved" column
	card, status := h.card(t, added.ID)
	if status != StatusSaved || card.Company != "Acme" || card.Role != "Software Engineer" {
		t.Errorf("card %+v in %q, want Acme Software Engineer in Saved", card, status)
	}
}

// @scenario [features/job-cards.feature#add-job-in-column] Add a job directly to a status column
func TestAddJobToColumn(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I open the job search board
	h.board(t)

	// @when [features/job-cards.feature] I add a job with company "Acme" and role "Software Engineer" to the "Applied" column
	added, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, StatusApplied)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] a card for "Software Engineer" at "Acme" appears in the "Applied" column
	if _, status := h.card(t, added.ID); status != StatusApplied {
		t.Errorf("card is in %q, want Applied", status)
	}
}

// @scenario [features/job-cards.feature#add-job-missing-required] A job can't be saved without a company or a role
func TestAddJobMissingRequired(t *testing.T) {
	tests := []struct{ company, role, missing string }{
		{"", "Software Engineer", "company"},
		{"Acme", "", "role"},
		{"   ", "Software Engineer", "company"},
	}
	for _, tt := range tests {
		t.Run(tt.missing, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-cards.feature] I open the job search board
			h.board(t)

			// @when [features/job-cards.feature] I try to add a job with company "<company>" and role "<role>"
			_, err := h.svc.AddJob(h.ctx, Details{Company: tt.company, Role: tt.role}, "")

			// @then [features/job-cards.feature] the job is not added
			if n := len(h.store.jobs); n != 0 {
				t.Errorf("%d jobs added, want 0", n)
			}
			// @and [features/job-cards.feature] I am told that the <missing> is required
			wantValidation(t, err, tt.missing+" is required")
		})
	}
}

// @scenario [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
func TestNewCardGoesToTop(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-board.feature] the "Saved" column holds the jobs at "Globex" and "Initech" in that order
	h.seed(store.Job{Company: "Globex"})
	h.seed(store.Job{Company: "Initech"})

	// @when [features/job-board.feature] I add a job at "Acme" to the "Saved" column
	if _, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, StatusSaved); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-board.feature] the "Saved" column shows "Acme", "Globex", and "Initech" in that order
	if got, want := h.column(t, StatusSaved), []string{"Acme", "Globex", "Initech"}; !slices.Equal(got, want) {
		t.Errorf("Saved = %v, want %v", got, want)
	}
	assertPositions(t, h, StatusSaved)
}

// @scenario [features/job-cards.feature#add-job-not-closed] A job can't be added straight to Closed
func TestAddJobNotStraightToClosed(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I open the job search board
	h.board(t)

	// @when [features/job-cards.feature] I try to add a job with company "Acme" and role "Software Engineer" to the "Closed" column
	_, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Software Engineer"}, StatusClosed)

	// @then [features/job-cards.feature] the job is not added
	if n := len(h.store.jobs); n != 0 {
		t.Errorf("%d jobs added, want 0", n)
	}
	// @and [features/job-cards.feature] I am told to add the job to another column and move it to Closed
	wantValidation(t, err, "a job can't be added straight to Closed; add it to another column, then move it to Closed")
}

func TestAddJobRejectsUnknownColumns(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Engineer"}, "ghosted"); err == nil {
		t.Error("adding to an unknown column succeeded")
	}
	if len(h.store.jobs) != 0 {
		t.Errorf("%d jobs added", len(h.store.jobs))
	}
}

func TestAddJobTrimsCompanyAndRole(t *testing.T) {
	h := newHarness(t)
	added, err := h.svc.AddJob(h.ctx, Details{Company: "  Acme ", Role: "\tEngineer\n"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if added.Company != "Acme" || added.Role != "Engineer" {
		t.Errorf("company %q, role %q", added.Company, added.Role)
	}
}

func TestAddJobRollsBackWhenAQueryFails(t *testing.T) {
	h := newHarness(t)
	h.seed(store.Job{Company: "Globex"})
	h.store.fail = "CreateJob"

	if _, err := h.svc.AddJob(h.ctx, Details{Company: "Acme", Role: "Engineer"}, ""); !errors.Is(err, errFake) {
		t.Fatalf("err = %v, want errFake", err)
	}
	h.store.fail = ""
	if got := h.column(t, StatusSaved); !slices.Equal(got, []string{"Globex"}) {
		t.Errorf("Saved = %v, want [Globex]", got)
	}
}

// @scenario [features/job-cards.feature#view-job-details] View a job's full details
func TestViewJobDetails(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with every detail filled in
	id := h.seed(store.Job{
		Company: "Acme", Role: "Software Engineer", Description: "Build and maintain internal tools.",
		Link: "https://acme.example.com/jobs/123", Salary: "$120k - $140k", Location: "Remote (US)",
		Notes: "Referred by Dana.", Status: string(StatusApplied), DateApplied: nullString("2026-10-01"),
	})
	resume := h.seedDoc(id, "resume-2026.pdf", KindResume)

	// @when [features/job-cards.feature] I open the card
	got, err := h.svc.Job(h.ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] I see the company, role, job description, link, salary, location, notes, and date applied
	want := Job{
		ID: id, Company: "Acme", Role: "Software Engineer", Description: "Build and maintain internal tools.",
		Link: "https://acme.example.com/jobs/123", Salary: "$120k - $140k", Location: "Remote (US)",
		Notes: "Referred by Dana.", DateApplied: "2026-10-01", Status: StatusApplied,
		// @and [features/job-cards.feature] I see the documents attached to the job
		Documents: []Document{{ID: resume, JobID: id, Kind: KindResume, FileName: "resume-2026.pdf"}},
		Contacts:  []Contact{},
	}
	if !jobsEqual(got, want) {
		t.Errorf("job = %+v, want %+v", got, want)
	}
}

func TestJobNotFound(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.Job(h.ctx, 99); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("Job: err = %v, want ErrJobNotFound", err)
	}
	if _, err := h.svc.UpdateJob(h.ctx, 99, Details{Company: "Acme", Role: "Engineer"}); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("UpdateJob: err = %v, want ErrJobNotFound", err)
	}
	if err := h.svc.DeleteJob(h.ctx, 99); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("DeleteJob: err = %v, want ErrJobNotFound", err)
	}
	if err := h.svc.MoveJob(h.ctx, Move{JobID: 99, Status: StatusApplied}); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("MoveJob: err = %v, want ErrJobNotFound", err)
	}
}

// details returns a job's current details, for changing one field in an update.
func details(j Job) Details {
	return Details{
		Company: j.Company, Role: j.Role, Description: j.Description, Link: j.Link,
		Salary: j.Salary, Location: j.Location, Notes: j.Notes, DateApplied: j.DateApplied,
	}
}

// @scenario [features/job-cards.feature#edit-job-details] Edit a job's details
func TestEditJobDetails(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
	id := h.seed(store.Job{Company: "Acme", Description: "Build tools."})
	d := details(h.job(t, id))

	// @when [features/job-cards.feature] I change the role to "Senior Software Engineer"
	d.Role = "Senior Software Engineer"
	// @and [features/job-cards.feature] I change the job description
	d.Description = "Lead the internal tools team."
	if _, err := h.svc.UpdateJob(h.ctx, id, d); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] the card shows the role "Senior Software Engineer"
	if card, _ := h.card(t, id); card.Role != "Senior Software Engineer" {
		t.Errorf("card role = %q", card.Role)
	}
	// @and [features/job-cards.feature] the job shows the new job description
	if got := h.job(t, id).Description; got != "Lead the internal tools team." {
		t.Errorf("description = %q", got)
	}
}

// @scenario [features/job-cards.feature#salary-free-text] Salary is written however the posting gives it
func TestSalaryIsFreeText(t *testing.T) {
	for _, salary := range []string{"$120,000", "$120k - $140k", "$65/hr", "Not listed"} {
		t.Run(salary, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)

			// @when [features/job-cards.feature] I set the salary to "<salary>"
			d := details(h.job(t, id))
			d.Salary = salary
			if _, err := h.svc.UpdateJob(h.ctx, id, d); err != nil {
				t.Fatal(err)
			}

			// @then [features/job-cards.feature] the job shows the salary "<salary>"
			if got := h.job(t, id).Salary; got != salary {
				t.Errorf("salary = %q, want %q", got, salary)
			}
		})
	}
}

// @scenario [features/job-cards.feature#add-notes] Keep notes on a job
func TestKeepNotes(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-cards.feature] I write in the notes "Recruiter call went well, follow up Friday."
	d := details(h.job(t, id))
	d.Notes = "Recruiter call went well, follow up Friday."
	if _, err := h.svc.UpdateJob(h.ctx, id, d); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] the job's notes show "Recruiter call went well, follow up Friday."
	if got := h.job(t, id).Notes; got != "Recruiter call went well, follow up Friday." {
		t.Errorf("notes = %q", got)
	}
}

// @scenario [features/job-cards.feature#edit-notes] Add to existing notes
func TestAddToExistingNotes(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with the notes "Referred by Dana."
	id := h.seed(store.Job{Company: "Acme", Notes: "Referred by Dana."})

	// @when [features/job-cards.feature] I add "Phone screen booked for 10/14." on a new line in the notes
	d := details(h.job(t, id))
	d.Notes += "\nPhone screen booked for 10/14."
	if _, err := h.svc.UpdateJob(h.ctx, id, d); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] the job's notes show both lines
	if got, want := h.job(t, id).Notes, "Referred by Dana.\nPhone screen booked for 10/14."; got != want {
		t.Errorf("notes = %q, want %q", got, want)
	}
}

// @scenario [features/job-cards.feature#open-job-link] Open a job's posting link
func TestOpenJobLink(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with the link "https://acme.example.com/jobs/123"
	id := h.seed(store.Job{Company: "Acme", Link: "https://acme.example.com/jobs/123"})

	// @when [features/job-cards.feature] I open the job's link
	if err := h.svc.OpenJobLink(h.ctx, id); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] the posting opens in my web browser
	if want := []string{"https://acme.example.com/jobs/123"}; !slices.Equal(h.opener.opened, want) {
		t.Errorf("opened %v, want %v", h.opener.opened, want)
	}
}

func TestOpenJobLinkRefusesMissingOrNonWebLinks(t *testing.T) {
	h := newHarness(t)
	none := h.acme(StatusSaved)
	// A link that skipped validation, such as one written before the rule existed.
	local := h.seed(store.Job{Company: "Globex", Link: `C:\Windows\System32\calc.exe`})

	if err := h.svc.OpenJobLink(h.ctx, none); err == nil {
		t.Error("opening a job with no link succeeded")
	}
	if err := h.svc.OpenJobLink(h.ctx, local); err == nil {
		t.Error("opening a non-web link succeeded")
	}
	if len(h.opener.opened) != 0 {
		t.Errorf("opened %v, want nothing", h.opener.opened)
	}
}

// @scenario [features/job-cards.feature#invalid-job-link] A job link must be a web address
func TestInvalidJobLink(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-cards.feature] I try to set the link to "not a url"
	d := details(h.job(t, id))
	d.Link = "not a url"
	_, err := h.svc.UpdateJob(h.ctx, id, d)

	// @then [features/job-cards.feature] the link is not saved
	if got := h.job(t, id).Link; got != "" {
		t.Errorf("link = %q, want it unchanged", got)
	}
	// @and [features/job-cards.feature] I am told the link must be a web address
	wantValidation(t, err, "the link must be a web address, like https://example.com/jobs/123")
}

func TestIsWebAddress(t *testing.T) {
	tests := map[string]bool{
		"https://acme.example.com/jobs/123":    true,
		"http://acme.example.com":              true,
		"https://jobs.example.com/~acme/(123)": true,
		"not a url":                            false,
		"acme.example.com/jobs":                false,
		"ftp://acme.example.com":               false,
		"file:///C:/Windows/notepad.exe":       false,
		"javascript:alert(1)":                  false,
		"https://":                             false,
		`C:\Windows\System32\calc.exe`:         false,
	}
	for link, want := range tests {
		if got := isWebAddress(link); got != want {
			t.Errorf("isWebAddress(%q) = %v, want %v", link, got, want)
		}
	}
}

// @scenario [features/job-cards.feature#set-date-applied] Set the date applied
func TestSetDateApplied(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" in "Applied"
	id := h.acme(StatusApplied)

	// @when [features/job-cards.feature] I set the date applied to 2026-10-01
	d := details(h.job(t, id))
	d.DateApplied = "2026-10-01"
	if _, err := h.svc.UpdateJob(h.ctx, id, d); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] the job's date applied is 2026-10-01
	if got := h.job(t, id).DateApplied; got != "2026-10-01" {
		t.Errorf("date applied = %q", got)
	}
	// @and [features/job-cards.feature] the card shows the date applied
	if card, _ := h.card(t, id); card.DateApplied != "2026-10-01" {
		t.Errorf("card date applied = %q", card.DateApplied)
	}
}

// @scenario [features/job-cards.feature#date-applied-any-status] The date applied can be set in any status
func TestDateAppliedInAnyStatus(t *testing.T) {
	for _, label := range []string{"Saved", "Interviewing", "Offer", "Closed"} {
		t.Run(label, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" in "<status>"
			id := h.acme(statusFromLabel(t, label))

			// @when [features/job-cards.feature] I set the date applied to 2026-10-01
			d := details(h.job(t, id))
			d.DateApplied = "2026-10-01"
			if _, err := h.svc.UpdateJob(h.ctx, id, d); err != nil {
				t.Fatal(err)
			}

			// @then [features/job-cards.feature] the job's date applied is 2026-10-01
			if got := h.job(t, id).DateApplied; got != "2026-10-01" {
				t.Errorf("date applied = %q", got)
			}
		})
	}
}

// @scenario [features/job-cards.feature#clear-date-applied] Clear the date applied
func TestClearDateApplied(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme" with a date applied of 2026-10-01
	id := h.seed(store.Job{Company: "Acme", Status: string(StatusApplied), DateApplied: nullString("2026-10-01")})

	// @when [features/job-cards.feature] I clear the date applied
	d := details(h.job(t, id))
	d.DateApplied = ""
	if _, err := h.svc.UpdateJob(h.ctx, id, d); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] the job has no date applied
	if got := h.store.jobs[id].DateApplied; got.Valid {
		t.Errorf("date applied = %q, want NULL", got.String)
	}
}

// @scenario [features/job-cards.feature#future-date-applied] The date applied can't be in the future
func TestFutureDateApplied(t *testing.T) {
	// @given [features/job-cards.feature] today is 2026-10-09
	h := newHarness(t)
	// @and [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-cards.feature] I try to set the date applied to 2026-10-20
	d := details(h.job(t, id))
	d.DateApplied = "2026-10-20"
	_, err := h.svc.UpdateJob(h.ctx, id, d)

	// @then [features/job-cards.feature] the date applied is not saved
	if got := h.job(t, id).DateApplied; got != "" {
		t.Errorf("date applied = %q, want it unchanged", got)
	}
	// @and [features/job-cards.feature] I am told the date applied can't be in the future
	wantValidation(t, err, "the date applied can't be in the future")
}

func TestDateAppliedEdges(t *testing.T) {
	tests := []struct {
		date string
		ok   bool
	}{
		{"2026-10-09", true}, // today
		{"2026-10-10", false},
		{"10/01/2026", false},
		{"2026-02-30", false},
		{"2026-10-1", false},
	}
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			h := newHarness(t)
			id := h.acme(StatusApplied)
			d := details(h.job(t, id))
			d.DateApplied = tt.date
			_, err := h.svc.UpdateJob(h.ctx, id, d)
			if tt.ok && err != nil {
				t.Errorf("err = %v, want nil", err)
			}
			if !tt.ok && err == nil {
				t.Error("saved, want an error")
			}
		})
	}
}

// @scenario [features/job-cards.feature#delete-job] Delete a job after confirming
func TestDeleteJob(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-cards.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)
	h.seedDoc(id, "resume-2026.pdf", KindResume)
	h.seedDoc(id, "acme-cover.docx", KindCoverLetter)
	other := h.seed(store.Job{Company: "Globex"})
	h.seedDoc(other, "resume-2026.pdf", KindResume)

	// @when [features/job-cards.feature] I delete the job
	// ("And I confirm the deletion" is the frontend's to ask.)
	if err := h.svc.DeleteJob(h.ctx, id); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-cards.feature] the card for "Software Engineer" at "Acme" no longer appears on the board
	if got := h.column(t, StatusSaved); !slices.Equal(got, []string{"Globex"}) {
		t.Errorf("Saved = %v, want [Globex]", got)
	}
	// @and [features/job-cards.feature] its attached documents are removed too
	if len(h.files.removed) != 2 || len(h.files.copies) != 1 {
		t.Errorf("removed %v, left %v; want both of Acme's copies removed and Globex's kept", h.files.removed, h.files.copies)
	}
	if docs, _ := h.store.ListDocumentsForJob(h.ctx, id); len(docs) != 0 {
		t.Errorf("%d document rows left", len(docs))
	}
}

func TestDeleteJobReportsFilesThatCouldNotBeRemoved(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	h.seedDoc(id, "resume-2026.pdf", KindResume)
	h.files.failRemove = true

	if err := h.svc.DeleteJob(h.ctx, id); !errors.Is(err, errFake) {
		t.Errorf("err = %v, want it to wrap errFake", err)
	}
	if _, ok := h.store.jobs[id]; ok {
		t.Error("job still exists; the row should go even when a file can't be removed")
	}
}

// jobsEqual compares two jobs, including their documents.
func jobsEqual(a, b Job) bool {
	return reflect.DeepEqual(a, b)
}
