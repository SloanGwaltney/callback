package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"callback/internal/jobs"
)

// The scenarios in features/agent-cli-job-records.feature. The fake jobs service keeps what it's
// given; the rules for contacts, interviews, and documents, like what an email or a date must be,
// are the jobs service's and are tested in internal/jobs.

// contactNames lists a job's contacts as "name as relationship".
func contactNames(job *jobs.Job) []string {
	var out []string
	for _, c := range job.Contacts {
		out = append(out, c.Name+" as "+c.Relationship.Label())
	}
	return out
}

// documentNames lists a job's documents as "file as kind".
func documentNames(job *jobs.Job) []string {
	var out []string
	for _, d := range job.Documents {
		out = append(out, d.FileName+" as "+d.Kind.Label())
	}
	return out
}

// janeDoe puts "Jane Doe", a recruiter, on job.
func (h *harness) janeDoe(job *jobs.Job, c jobs.Contact) jobs.Contact {
	h.jobs.nextID++
	c.ID, c.JobID, c.Name, c.Relationship = h.jobs.nextID, job.ID, "Jane Doe", jobs.RelationshipRecruiter
	job.Contacts = append(job.Contacts, c)
	return c
}

// interview puts an interview on job.
func (h *harness) interview(job *jobs.Job, i jobs.Interview) jobs.Interview {
	h.jobs.nextID++
	i.ID, i.JobID = h.jobs.nextID, job.ID
	job.Interviews = append(job.Interviews, i)
	return i
}

// attached attaches fileName to job as kind, with a stored copy.
func (h *harness) attached(job *jobs.Job, fileName string, kind jobs.DocumentKind) jobs.Document {
	h.jobs.nextID++
	doc := jobs.Document{ID: h.jobs.nextID, JobID: job.ID, Kind: kind, FileName: fileName}
	job.Documents = append(job.Documents, doc)
	h.jobs.files[job.ID] = append(h.jobs.files[job.ID], jobs.DocumentFile{DocumentID: doc.ID, Path: storedPath(doc.ID)})
	return doc
}

// wantStderr fails the test unless r failed with code and its error says reason.
func (r result) wantStderr(t *testing.T, code int, reason string) {
	t.Helper()
	r.wantCode(t, code)
	if !strings.Contains(r.stderr, reason) {
		t.Errorf("stderr = %q, want %q", r.stderr, reason)
	}
}

// @scenario [#cli-add-contact]
func TestAgentAddsAContact(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-job-records.feature] I have a job at "Acme"
	job := h.acme()

	// @when [features/agent-cli-job-records.feature] my agent adds "Jane Doe", a recruiter, with an email and phone number, to the job
	r := h.run("contact", "add", fmt.Sprint(job.ID), "--name", "Jane Doe", "--relationship", "recruiter",
		"--email", "jane@acme.example.com", "--phone", "(555) 010-4477")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-job-records.feature] the job's contacts include "Jane Doe" as its recruiter
	if got := contactNames(job); !slices.Equal(got, []string{"Jane Doe as Recruiter"}) {
		t.Fatalf("contacts = %v", got)
	}
	// @and [features/agent-cli-job-records.feature] my agent is shown the contact with its ID
	want := fmt.Sprintf("Contact on job #%d: #%d Jane Doe | Recruiter | email jane@acme.example.com | phone (555) 010-4477\n", job.ID, job.Contacts[0].ID)
	if r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestAddedContactAsJSON(t *testing.T) {
	h := newHarness()
	job := h.acme()

	r := h.run("contact", "add", fmt.Sprint(job.ID), "--name", "Jane Doe", "--relationship", "Hiring manager", "--notes", "Met at the meetup", "--json")

	r.wantCode(t, ExitOK)
	var got jobs.Contact
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
		t.Fatalf("%v: %s", err, r.stdout)
	}
	if got != job.Contacts[0] || got.Relationship != jobs.RelationshipHiringManager || got.Notes != "Met at the meetup" {
		t.Errorf("printed %+v, saved %+v", got, job.Contacts[0])
	}
}

func TestAddingAContactToAJobThatIsntThere(t *testing.T) {
	h := newHarness()
	h.run("contact", "add", "999", "--name", "Jane Doe", "--relationship", "recruiter").wantStderr(t, ExitNotFound, "there's no job with ID 999")
}

// @scenario [#cli-add-contact-rules]
func TestAddingAContactFollowsTheAppsRules(t *testing.T) {
	for _, tc := range []struct {
		problem string
		args    []string
		err     error
		reason  string
		choices bool
	}{
		{"with no name", []string{"--relationship", "recruiter"}, &jobs.ValidationError{Msg: "name is required"}, "name is required", false},
		{"with no relationship", []string{"--name", "Jane Doe"}, nil, "a contact needs a relationship", true},
		{"with the relationship \"friend\"", []string{"--name", "Jane Doe", "--relationship", "friend"}, nil, "there's no relationship \"friend\"", true},
		{"with an email that isn't an email address", []string{"--name", "Jane Doe", "--relationship", "recruiter", "--email", "jane at acme"},
			&jobs.ValidationError{Msg: "the email must be an email address, like dana@example.com"}, "the email must be an email address", false},
		{"with a profile link that isn't a web address", []string{"--name", "Jane Doe", "--relationship", "recruiter", "--profile-link", "linkedin/jane"},
			&jobs.ValidationError{Msg: "the profile link must be a web address, like https://www.linkedin.com/in/dana-smith"}, "the profile link must be a web address", false},
	} {
		t.Run(tc.problem, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli-job-records.feature] I have a job at "Acme"
			job := h.acme()
			h.jobs.fail = tc.err

			// @when [features/agent-cli-job-records.feature] my agent tries to add a contact <problem>
			r := h.run(append([]string{"contact", "add", fmt.Sprint(job.ID)}, tc.args...)...)

			// @then [features/agent-cli-job-records.feature] the job has no new contact
			if len(job.Contacts) != 0 {
				t.Errorf("contacts = %v", job.Contacts)
			}
			// @and [features/agent-cli-job-records.feature] my agent is told <reason>
			r.wantStderr(t, ExitValidation, tc.reason)
			if tc.choices {
				for _, rel := range jobs.Relationships {
					if !strings.Contains(r.stderr, string(rel)) {
						t.Errorf("stderr doesn't offer %q: %s", rel, r.stderr)
					}
				}
			}
		})
	}
}

// @scenario [#cli-update-contact]
func TestAgentChangesAContactsPhone(t *testing.T) {
	h := newHarness()
	job := h.acme()
	// @given [features/agent-cli-job-records.feature] "Jane Doe" is a contact on the job at "Acme" with an email and notes
	contact := h.janeDoe(job, jobs.Contact{Email: "jane@acme.example.com", Notes: "Prefers email."})

	// @when [features/agent-cli-job-records.feature] my agent changes the contact's phone number
	r := h.run("contact", "update", fmt.Sprint(contact.ID), "--phone", "(555) 010-4477")

	r.wantCode(t, ExitOK)
	got := job.Contacts[0]
	// @then [features/agent-cli-job-records.feature] the contact's phone number is the new one
	if got.Phone != "(555) 010-4477" {
		t.Errorf("phone = %q", got.Phone)
	}
	// @and [features/agent-cli-job-records.feature] the contact's email and notes are unchanged
	if got.Email != "jane@acme.example.com" || got.Notes != "Prefers email." || got.Name != "Jane Doe" || got.Relationship != jobs.RelationshipRecruiter {
		t.Errorf("contact = %+v", got)
	}
	if !strings.Contains(r.stdout, "phone (555) 010-4477") {
		t.Errorf("stdout = %q", r.stdout)
	}
}

func TestAgentClearsAContactsDetail(t *testing.T) {
	h := newHarness()
	job := h.acme()
	contact := h.janeDoe(job, jobs.Contact{Email: "jane@acme.example.com", Phone: "555"})

	h.run("contact", "update", fmt.Sprint(contact.ID), "--phone", "", "--relationship", "Interviewer").wantCode(t, ExitOK)

	if got := job.Contacts[0]; got.Phone != "" || got.Email != "jane@acme.example.com" || got.Relationship != jobs.RelationshipInterviewer {
		t.Errorf("contact = %+v", got)
	}
}

// @scenario [#cli-remove-contact]
func TestAgentRemovesAContact(t *testing.T) {
	h := newHarness()
	job := h.acme()
	// @given [features/agent-cli-job-records.feature] "Jane Doe" is a contact on the job at "Acme"
	contact := h.janeDoe(job, jobs.Contact{})

	// @when [features/agent-cli-job-records.feature] my agent removes the contact
	r := h.run("contact", "remove", fmt.Sprint(contact.ID))

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-job-records.feature] the job's contacts don't include "Jane Doe"
	if slices.Contains(contactNames(job), "Jane Doe as Recruiter") {
		t.Errorf("contacts = %v", contactNames(job))
	}
	if want := fmt.Sprintf("Removed contact #%d, Jane Doe, from job #%d.\n", contact.ID, job.ID); r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

// @scenario [#cli-unknown-contact]
func TestAgentIsToldThereIsNoContact(t *testing.T) {
	h := newHarness()

	// @when [features/agent-cli-job-records.feature] my agent tries to remove the contact with ID 999
	r := h.run("contact", "remove", "999")

	// @then [features/agent-cli-job-records.feature] my agent is told there's no contact with ID 999
	r.wantStderr(t, ExitNotFound, "there's no contact with ID 999")
}

func TestUpdatingAContactThatIsntThere(t *testing.T) {
	newHarness().run("contact", "update", "999", "--phone", "555").wantStderr(t, ExitNotFound, "there's no contact with ID 999")
}

// @scenario [#cli-add-interview]
func TestAgentAddsAnInterview(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-job-records.feature] I have a job at "Acme"
	job := h.acme()

	// @when [features/agent-cli-job-records.feature] my agent adds a technical interview on 2026-10-20 at 14:00 to the job
	r := h.run("interview", "add", fmt.Sprint(job.ID), "--kind", "technical", "--date", "2026-10-20", "--time", "14:00")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-job-records.feature] the job's interviews include a technical interview on 2026-10-20 at 14:00
	if len(job.Interviews) != 1 {
		t.Fatalf("interviews = %v", job.Interviews)
	}
	if got := job.Interviews[0]; got.Kind != jobs.InterviewTechnical || got.Date != "2026-10-20" || got.Time != "14:00" {
		t.Errorf("interview = %+v", got)
	}
	// @and [features/agent-cli-job-records.feature] my agent is shown the interview with its ID
	if want := fmt.Sprintf("Interview on job #%d: #%d 2026-10-20 14:00 | Technical\n", job.ID, job.Interviews[0].ID); r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

// @scenario [#cli-add-interview-rules]
func TestAddingAnInterviewFollowsTheAppsRules(t *testing.T) {
	for _, tc := range []struct {
		problem string
		args    []string
		err     error
		reason  string
		choices bool
	}{
		{"with no date", []string{"--kind", "technical"}, &jobs.ValidationError{Msg: "date is required"}, "date is required", false},
		{"on \"next Tuesday\"", []string{"--kind", "technical", "--date", "next Tuesday"},
			&jobs.ValidationError{Msg: "the date must be a date like 2026-10-14"}, "the date must be a date like 2026-10-14", false},
		{"with no kind", []string{"--date", "2026-10-20"}, nil, "an interview needs a kind", true},
		{"of the kind \"coffee\"", []string{"--date", "2026-10-20", "--kind", "coffee"}, nil, "there's no interview kind \"coffee\"", true},
	} {
		t.Run(tc.problem, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli-job-records.feature] I have a job at "Acme"
			job := h.acme()
			h.jobs.fail = tc.err

			// @when [features/agent-cli-job-records.feature] my agent tries to add an interview <problem>
			r := h.run(append([]string{"interview", "add", fmt.Sprint(job.ID)}, tc.args...)...)

			// @then [features/agent-cli-job-records.feature] the job has no new interview
			if len(job.Interviews) != 0 {
				t.Errorf("interviews = %v", job.Interviews)
			}
			// @and [features/agent-cli-job-records.feature] my agent is told <reason>
			r.wantStderr(t, ExitValidation, tc.reason)
			if tc.choices {
				for _, kind := range jobs.InterviewKinds {
					if !strings.Contains(r.stderr, string(kind)) {
						t.Errorf("stderr doesn't offer %q: %s", kind, r.stderr)
					}
				}
			}
		})
	}
}

// @scenario [#cli-update-interview]
func TestAgentChangesAnInterviewsTime(t *testing.T) {
	h := newHarness()
	job := h.acme()
	// @given [features/agent-cli-job-records.feature] the job at "Acme" has a technical interview on 2026-10-20 at 14:00 with notes
	interview := h.interview(job, jobs.Interview{Date: "2026-10-20", Time: "14:00", Kind: jobs.InterviewTechnical, Notes: "Bring a laptop."})

	// @when [features/agent-cli-job-records.feature] my agent changes the interview's time to 15:30
	r := h.run("interview", "update", fmt.Sprint(interview.ID), "--time", "15:30")

	r.wantCode(t, ExitOK)
	got := job.Interviews[0]
	// @then [features/agent-cli-job-records.feature] the interview is on 2026-10-20 at 15:30
	if got.Date != "2026-10-20" || got.Time != "15:30" {
		t.Errorf("interview is on %s at %s", got.Date, got.Time)
	}
	// @and [features/agent-cli-job-records.feature] its kind and notes are unchanged
	if got.Kind != jobs.InterviewTechnical || got.Notes != "Bring a laptop." {
		t.Errorf("interview = %+v", got)
	}
	if want := fmt.Sprintf("Interview on job #%d: #%d 2026-10-20 15:30 | Technical\n  Bring a laptop.\n", job.ID, interview.ID); r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestAgentGivesAnInterviewsNotesFromAFile(t *testing.T) {
	h := newHarness()
	job := h.acme()
	interview := h.interview(job, jobs.Interview{Date: "2026-10-20", Kind: jobs.InterviewPanel})
	h.files["prep.txt"] = "Ask about on-call.\n\nRead the \"platform\" post."

	h.run("interview", "update", fmt.Sprint(interview.ID), "--notes-file", "prep.txt").wantCode(t, ExitOK)

	if got := job.Interviews[0].Notes; got != "Ask about on-call.\n\nRead the \"platform\" post." {
		t.Errorf("notes = %q", got)
	}
}

// @scenario [#cli-remove-interview]
func TestAgentRemovesAnInterview(t *testing.T) {
	h := newHarness()
	job := h.acme()
	// @given [features/agent-cli-job-records.feature] the job at "Acme" has a technical interview on 2026-10-20
	interview := h.interview(job, jobs.Interview{Date: "2026-10-20", Kind: jobs.InterviewTechnical})

	// @when [features/agent-cli-job-records.feature] my agent removes the interview
	r := h.run("interview", "remove", fmt.Sprint(interview.ID))

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-job-records.feature] the job has no interviews
	if len(job.Interviews) != 0 {
		t.Errorf("interviews = %v", job.Interviews)
	}
	if want := fmt.Sprintf("Removed interview #%d, Technical on 2026-10-20, from job #%d.\n", interview.ID, job.ID); r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

func TestChangingAnInterviewThatIsntThere(t *testing.T) {
	h := newHarness()
	h.run("interview", "update", "999", "--time", "15:30").wantStderr(t, ExitNotFound, "there's no interview with ID 999")
	h.run("interview", "remove", "999").wantStderr(t, ExitNotFound, "there's no interview with ID 999")
}

// @scenario [#cli-attach-document]
func TestAgentAttachesADocument(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-job-records.feature] I have a job at "Acme"
	job := h.acme()
	// The fake attaches any path. Checking the file is there and copying it are the jobs service's
	// and docfiles' (TestAttachMissingFile, TestSaveCopiesIntoTheDocumentsFolder).
	// @and [features/agent-cli-job-records.feature] there is a file "acme-resume.pdf" on my computer
	src := "Documents/acme-resume.pdf"

	// @when [features/agent-cli-job-records.feature] my agent attaches "acme-resume.pdf" to the job as a resume
	r := h.run("document", "attach", fmt.Sprint(job.ID), "--file", src, "--kind", "resume")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-job-records.feature] the job lists "acme-resume.pdf" as a resume
	if got := documentNames(job); !slices.Equal(got, []string{"acme-resume.pdf as Resume"}) {
		t.Fatalf("documents = %v", got)
	}
	// @and [features/agent-cli-job-records.feature] my agent is shown the document with its ID and where its stored copy is
	id := job.Documents[0].ID
	if want := fmt.Sprintf("Document on job #%d: #%d acme-resume.pdf | Resume | stored copy: %s\n", job.ID, id, storedPath(id)); r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestAttachedDocumentAsJSON(t *testing.T) {
	h := newHarness()
	job := h.acme()

	r := h.run("document", "attach", fmt.Sprint(job.ID), "--file", "acme-letter.docx", "--kind", "Cover letter", "--json")

	r.wantCode(t, ExitOK)
	var got documentJSON
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
		t.Fatalf("%v: %s", err, r.stdout)
	}
	if got.ID != job.Documents[0].ID || got.Kind != jobs.KindCoverLetter || got.StoredPath != storedPath(got.ID) || got.Missing {
		t.Errorf("printed %+v", got)
	}
}

func TestAttachingNeedsAFile(t *testing.T) {
	h := newHarness()
	job := h.acme()
	h.run("document", "attach", fmt.Sprint(job.ID), "--kind", "resume").wantStderr(t, ExitUsage, "needs --file")
	if len(job.Documents) != 0 {
		t.Errorf("documents = %v", job.Documents)
	}
}

// @scenario [#cli-attach-document-rules]
func TestAttachingADocumentFollowsTheAppsRules(t *testing.T) {
	for _, tc := range []struct {
		file    string
		args    []string
		err     error
		reason  string
		choices bool
	}{
		{"a file that doesn't exist", []string{"--file", "gone.pdf", "--kind", "resume"},
			&jobs.ValidationError{Msg: "there's no file at gone.pdf"}, "there's no file at gone.pdf", false},
		{"\"photo.png\"", []string{"--file", "photo.png", "--kind", "other"},
			&jobs.ValidationError{Msg: "only these document types can be attached: .txt, .pdf, .md, .docx, .doc, .rtf, .odt"}, "only these document types can be attached", false},
		{"a PDF larger than 10 MB", []string{"--file", "portfolio.pdf", "--kind", "other"},
			&jobs.ValidationError{Msg: "files can be at most 10 MB"}, "files can be at most 10 MB", false},
		{"\"notes.txt\" as a \"portfolio\"", []string{"--file", "notes.txt", "--kind", "portfolio"}, nil, "there's no document kind \"portfolio\"", true},
	} {
		t.Run(tc.file, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli-job-records.feature] I have a job at "Acme"
			job := h.acme()
			h.jobs.fail = tc.err

			// @when [features/agent-cli-job-records.feature] my agent tries to attach <file>
			r := h.run(append([]string{"document", "attach", fmt.Sprint(job.ID)}, tc.args...)...)

			// @then [features/agent-cli-job-records.feature] the job has no new document
			if len(job.Documents) != 0 || len(h.jobs.files[job.ID]) != 0 {
				t.Errorf("documents = %v", job.Documents)
			}
			// @and [features/agent-cli-job-records.feature] my agent is told <reason>
			r.wantStderr(t, ExitValidation, tc.reason)
			if tc.choices {
				for _, kind := range jobs.DocumentKinds {
					if !strings.Contains(r.stderr, string(kind)) {
						t.Errorf("stderr doesn't offer %q: %s", kind, r.stderr)
					}
				}
			}
		})
	}
}

// @scenario [#cli-relabel-document]
func TestAgentChangesADocumentsKind(t *testing.T) {
	h := newHarness()
	job := h.acme()
	// @given [features/agent-cli-job-records.feature] the job at "Acme" has "acme-letter.pdf" attached as other
	doc := h.attached(job, "acme-letter.pdf", jobs.KindOther)

	// @when [features/agent-cli-job-records.feature] my agent marks "acme-letter.pdf" as a cover letter
	r := h.run("document", "set-kind", fmt.Sprint(doc.ID), "--kind", "cover letter")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-job-records.feature] the job lists "acme-letter.pdf" as a cover letter
	if got := documentNames(job); !slices.Equal(got, []string{"acme-letter.pdf as Cover letter"}) {
		t.Errorf("documents = %v", got)
	}
	if want := fmt.Sprintf("Document on job #%d: #%d acme-letter.pdf | Cover letter | stored copy: %s\n", job.ID, doc.ID, storedPath(doc.ID)); r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestSettingADocumentsKindNeedsOne(t *testing.T) {
	h := newHarness()
	job := h.acme()
	doc := h.attached(job, "acme-letter.pdf", jobs.KindOther)

	r := h.run("document", "set-kind", fmt.Sprint(doc.ID))

	r.wantStderr(t, ExitValidation, "a document needs a kind")
	if !strings.Contains(r.stderr, "cover_letter") {
		t.Errorf("stderr doesn't offer the kinds: %s", r.stderr)
	}
	h.run("document", "set-kind", "999", "--kind", "resume").wantStderr(t, ExitNotFound, "there's no document with ID 999")
}

// @scenario [#cli-remove-document]
func TestAgentRemovesADocument(t *testing.T) {
	h := newHarness()
	job := h.acme()
	// @given [features/agent-cli-job-records.feature] the job at "Acme" has "acme-resume.pdf" attached
	doc := h.attached(job, "acme-resume.pdf", jobs.KindResume)

	// @when [features/agent-cli-job-records.feature] my agent removes the document
	r := h.run("document", "remove", fmt.Sprint(doc.ID))

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-job-records.feature] the job doesn't list "acme-resume.pdf"
	if len(documentNames(job)) != 0 {
		t.Errorf("documents = %v", documentNames(job))
	}
	if want := fmt.Sprintf("Removed document #%d, acme-resume.pdf, from job #%d, and deleted its stored copy.\n", doc.ID, job.ID); r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
	h.run("document", "remove", fmt.Sprint(doc.ID)).wantStderr(t, ExitNotFound, fmt.Sprintf("there's no document with ID %d", doc.ID))
}
