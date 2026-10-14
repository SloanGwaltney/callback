package cli

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"callback/internal/jobs"
)

// The scenarios in features/agent-cli-jobs.feature. The fake jobs service keeps what it's given
// and puts a job where a move or an add asks; the rules behind them, like today's date, due
// marks, days in status, and what a value must be, are the jobs service's and are tested in
// internal/jobs.

// companies returns the company on each line of a `job list`, in order.
func companies(stdout string) []string {
	var out []string
	for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
		if fields := strings.Split(line, " | "); len(fields) > 2 {
			out = append(out, fields[2])
		}
	}
	return out
}

// lineFor returns the `job list` line for company.
func lineFor(t *testing.T, stdout, company string) string {
	t.Helper()
	for line := range strings.SplitSeq(stdout, "\n") {
		if slices.Contains(strings.Split(line, " | "), company) {
			return line
		}
	}
	t.Fatalf("no line for %s in:\n%s", company, stdout)
	return ""
}

// wantCompanies fails the test unless the list shows exactly want, in order.
func wantCompanies(t *testing.T, r result, want ...string) {
	t.Helper()
	r.wantCode(t, ExitOK)
	if got := companies(r.stdout); !slices.Equal(got, want) {
		t.Errorf("listed %q, want %q\n%s", got, want, r.stdout)
	}
}

// rate gives a job an interest rating.
func rate(job *jobs.Job, stars int) { job.Interest = &stars }

// @scenario [features/agent-cli-jobs.feature#cli-list-jobs] List my jobs
func TestAgentListsJobs(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job for "Software Engineer" at "Acme" in "Applied"
	job := h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)
	// @and [features/agent-cli-jobs.feature] it has been in "Applied" for 9 days
	h.jobs.days[job.ID] = 9
	// @and [features/agent-cli-jobs.feature] its location is "Remote, US" and I applied on 2026-10-05
	job.Location, job.DateApplied = "Remote, US", "2026-10-05"
	// @and [features/agent-cli-jobs.feature] I rated it 4 stars
	rate(job, 4)
	// @and [features/agent-cli-jobs.feature] its next action is "Follow up" expected on 2026-10-15
	job.NextAction = &jobs.NextAction{Text: "Follow up", ExpectedDate: "2026-10-15"}

	// @when [features/agent-cli-jobs.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli-jobs.feature] it sees one line for the job
	lines := strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("listed %d lines, want 1:\n%s", len(lines), r.stdout)
	}
	// @and [features/agent-cli-jobs.feature] the line shows its ID, "Applied", 9 days in status, "Acme", "Software Engineer", "Remote, US", applied 2026-10-05, 4 stars, and "Follow up" expected on 2026-10-15
	want := fmt.Sprintf("#%d | Applied for 9 days | Acme | Software Engineer | Remote, US | applied 2026-10-05 | 4 stars | next: Follow up, expected 2026-10-15", job.ID)
	if lines[0] != want {
		t.Errorf("line = %q\nwant   %q", lines[0], want)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-list-board-order] Jobs are listed column by column, in each column's order
func TestAgentListsJobsInBoardOrder(t *testing.T) {
	h := newHarness()
	// @and [features/agent-cli-jobs.feature] the "Applied" column holds the job at "Globex"
	h.jobs.add("Globex", "Designer", jobs.StatusApplied)
	// @given [features/agent-cli-jobs.feature] the "Saved" column holds the jobs at "Initech" and "Acme" in that order
	h.jobs.add("Initech", "Engineer", jobs.StatusSaved)
	h.acme()

	// @when [features/agent-cli-jobs.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli-jobs.feature] it sees "Initech", "Acme", and "Globex" in that order
	wantCompanies(t, r, "Initech", "Acme", "Globex")
}

// @scenario [features/agent-cli-jobs.feature#cli-list-leaves-out-long-text] The list leaves out each job's description and notes
func TestAgentListLeavesOutLongText(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job for "Software Engineer" at "Acme" with a description and notes
	job := h.acme()
	job.Description, job.Notes = "Build the payments API", "Ask about the team size"

	// @when [features/agent-cli-jobs.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli-jobs.feature] the line for the job doesn't show its description or notes
	line := lineFor(t, r.stdout, "Acme")
	if strings.Contains(line, "payments") || strings.Contains(line, "team size") {
		t.Errorf("line = %q", line)
	}
}

// Today's date, and the due marks it gives, are the jobs service's: see
// TestDueMarksForTheCommandLine in internal/jobs.
// @scenario [features/agent-cli-jobs.feature#cli-list-overdue-marked] The list marks a next action that's due or overdue
func TestAgentListMarksDueAndOverdue(t *testing.T) {
	h := newHarness()
	// @and [features/agent-cli-jobs.feature] I have a job at "Acme" whose next action was expected on 2026-10-10
	h.acme().NextAction = &jobs.NextAction{Text: "Follow up", ExpectedDate: "2026-10-10", Due: jobs.DueOverdue}
	// @and [features/agent-cli-jobs.feature] I have a job at "Globex" whose next action is expected today
	h.jobs.add("Globex", "Designer", "").NextAction = &jobs.NextAction{Text: "Call", ExpectedDate: "2026-10-14", Due: jobs.DueToday}

	// @when [features/agent-cli-jobs.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli-jobs.feature] the line for "Acme" marks its next action as overdue
	if line := lineFor(t, r.stdout, "Acme"); !strings.HasSuffix(line, "next: Follow up, expected 2026-10-10 (overdue)") {
		t.Errorf("Acme's line = %q", line)
	}
	// @and [features/agent-cli-jobs.feature] the line for "Globex" marks its next action as due
	if line := lineFor(t, r.stdout, "Globex"); !strings.HasSuffix(line, "next: Call, expected 2026-10-14 (due today)") {
		t.Errorf("Globex's line = %q", line)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-list-empty] My agent is told when no jobs are listed
func TestAgentIsToldThereAreNoJobs(t *testing.T) {
	// @given [features/agent-cli-jobs.feature] I have no jobs
	h := newHarness()

	// @when [features/agent-cli-jobs.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli-jobs.feature] my agent is told there are no jobs to list
	if r.stdout != "No jobs to list.\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-list-leaves-out-closed] The list leaves out closed jobs
func TestAgentListLeavesOutClosedJobs(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" in "Applied"
	h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)
	// @and [features/agent-cli-jobs.feature] I have a job at "Globex" in "Closed"
	h.jobs.add("Globex", "Designer", jobs.StatusClosed)

	// @when [features/agent-cli-jobs.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli-jobs.feature] it sees "Acme"
	// @but [features/agent-cli-jobs.feature] it doesn't see "Globex"
	wantCompanies(t, r, "Acme")
}

// @scenario [features/agent-cli-jobs.feature#cli-list-all] My agent asks for every job, closed ones too
func TestAgentListsAllJobs(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" in "Applied"
	h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)
	// @and [features/agent-cli-jobs.feature] I have a job at "Globex" in "Closed"
	h.jobs.add("Globex", "Designer", jobs.StatusClosed)

	// @when [features/agent-cli-jobs.feature] my agent lists all my jobs
	r := h.run("job", "list", "--all")

	// @then [features/agent-cli-jobs.feature] it sees "Acme" and "Globex"
	wantCompanies(t, r, "Acme", "Globex")
}

// @scenario [features/agent-cli-jobs.feature#cli-filter-status] List only the jobs in some statuses
func TestAgentListsJobsInSomeStatuses(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have jobs at "Acme" in "Saved", "Globex" in "Applied", and "Initech" in "Interviewing"
	h.acme()
	h.jobs.add("Globex", "Designer", jobs.StatusApplied)
	h.jobs.add("Initech", "Engineer", jobs.StatusInterviewing)

	// @when [features/agent-cli-jobs.feature] my agent lists the jobs in "Applied" and "Interviewing"
	r := h.run("job", "list", "--status", "Applied,Interviewing")

	// @then [features/agent-cli-jobs.feature] it sees "Globex" and "Initech"
	// @but [features/agent-cli-jobs.feature] it doesn't see "Acme"
	wantCompanies(t, r, "Globex", "Initech")
}

// @scenario [features/agent-cli-jobs.feature#cli-filter-closed-status] Asking for Closed lists closed jobs
func TestAgentListsClosedJobsByStatus(t *testing.T) {
	h := newHarness()
	h.acme()
	// @given [features/agent-cli-jobs.feature] I have a job at "Globex" in "Closed"
	h.jobs.add("Globex", "Designer", jobs.StatusClosed)

	// @when [features/agent-cli-jobs.feature] my agent lists the jobs in "Closed"
	r := h.run("job", "list", "--status", "closed")

	// @then [features/agent-cli-jobs.feature] it sees "Globex"
	wantCompanies(t, r, "Globex")
}

// @scenario [features/agent-cli-jobs.feature#cli-filter-search] Search the list the way the board's search does
func TestAgentSearchesTheList(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have jobs for "Software Engineer" at "Acme" and "Designer" at "Globex"
	h.acme()
	h.jobs.add("Globex", "Designer", "")

	// @when [features/agent-cli-jobs.feature] my agent lists the jobs matching "engineer"
	r := h.run("job", "list", "--search", "engineer")

	// @then [features/agent-cli-jobs.feature] it sees "Acme"
	// @but [features/agent-cli-jobs.feature] it doesn't see "Globex"
	wantCompanies(t, r, "Acme")
}

// Today's date, and the due marks it gives, are the jobs service's: see
// TestDueMarksForTheCommandLine in internal/jobs.
// @scenario [features/agent-cli-jobs.feature#cli-filter-overdue] List only the jobs whose next action is overdue
func TestAgentListsOverdueJobs(t *testing.T) {
	h := newHarness()
	// @and [features/agent-cli-jobs.feature] I have a job at "Acme" whose next action was expected on 2026-10-10
	h.acme().NextAction = &jobs.NextAction{Text: "Follow up", ExpectedDate: "2026-10-10", Due: jobs.DueOverdue}
	// @and [features/agent-cli-jobs.feature] I have a job at "Globex" whose next action is expected today
	h.jobs.add("Globex", "Designer", "").NextAction = &jobs.NextAction{Text: "Call", ExpectedDate: "2026-10-14", Due: jobs.DueToday}
	h.jobs.add("Initech", "Engineer", "")

	// @when [features/agent-cli-jobs.feature] my agent lists the overdue jobs
	r := h.run("job", "list", "--overdue")

	// @then [features/agent-cli-jobs.feature] it sees "Acme"
	// @but [features/agent-cli-jobs.feature] it doesn't see "Globex"
	wantCompanies(t, r, "Acme")
}

// @scenario [features/agent-cli-jobs.feature#cli-filter-stale] List only the jobs that have been in their status for some days
func TestAgentListsJobsInTheirStatusForSomeDays(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" that has been in "Applied" for 20 days
	acme := h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)
	h.jobs.days[acme.ID] = 20
	// @and [features/agent-cli-jobs.feature] I have a job at "Globex" that has been in "Applied" for 3 days
	globex := h.jobs.add("Globex", "Designer", jobs.StatusApplied)
	h.jobs.days[globex.ID] = 3

	// @when [features/agent-cli-jobs.feature] my agent lists the jobs that have been in their status for at least 14 days
	r := h.run("job", "list", "--days-in-status", "14")

	// @then [features/agent-cli-jobs.feature] it sees "Acme"
	// @but [features/agent-cli-jobs.feature] it doesn't see "Globex"
	wantCompanies(t, r, "Acme")
}

// @scenario [features/agent-cli-jobs.feature#cli-filter-interest] List only the jobs rated at least some stars
func TestAgentListsJobsRatedAtLeastSomeStars(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" rated 5 stars
	rate(h.acme(), 5)
	// @and [features/agent-cli-jobs.feature] I have a job at "Globex" rated 3 stars
	rate(h.jobs.add("Globex", "Designer", ""), 3)
	// @and [features/agent-cli-jobs.feature] I have a job at "Initech" with no rating
	h.jobs.add("Initech", "Engineer", "")

	// @when [features/agent-cli-jobs.feature] my agent lists the jobs rated at least 4 stars
	r := h.run("job", "list", "--min-interest", "4")

	// @then [features/agent-cli-jobs.feature] it sees "Acme"
	// @but [features/agent-cli-jobs.feature] it doesn't see "Globex" or "Initech"
	wantCompanies(t, r, "Acme")
}

// @scenario [features/agent-cli-jobs.feature#cli-filter-combined] Every filter my agent gives applies
func TestEveryListFilterApplies(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job for "Software Engineer" at "Acme" in "Applied" rated 5 stars
	rate(h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied), 5)
	// @and [features/agent-cli-jobs.feature] I have a job for "Software Engineer" at "Globex" in "Saved" rated 5 stars
	rate(h.jobs.add("Globex", "Software Engineer", jobs.StatusSaved), 5)
	// @and [features/agent-cli-jobs.feature] I have a job for "Designer" at "Initech" in "Applied" rated 5 stars
	rate(h.jobs.add("Initech", "Designer", jobs.StatusApplied), 5)

	// @when [features/agent-cli-jobs.feature] my agent lists the jobs in "Applied" matching "engineer"
	r := h.run("job", "list", "--status", "applied", "--search", "engineer")

	// @then [features/agent-cli-jobs.feature] it sees "Acme"
	// @but [features/agent-cli-jobs.feature] it doesn't see "Globex" or "Initech"
	wantCompanies(t, r, "Acme")
}

// @scenario [features/agent-cli-jobs.feature#cli-filter-no-match] My agent is told when no jobs match
func TestAgentIsToldNoJobsMatch(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job for "Software Engineer" at "Acme"
	h.acme()

	// @when [features/agent-cli-jobs.feature] my agent lists the jobs matching "pilot"
	r := h.run("job", "list", "--search", "pilot")

	// @then [features/agent-cli-jobs.feature] my agent is told no jobs match
	if r.stdout != "No jobs match.\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-show-job] Show a job's full details
func TestAgentShowsAJob(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job for "Software Engineer" at "Acme" with every detail filled in
	job := h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)
	job.Location, job.Salary, job.Link, job.DateApplied = "Remote, US", "$120k", "https://acme.example/jobs/1", "2026-10-05"
	job.Description, job.Notes = "Build the payments API", "Ask about the team size"
	rate(job, 4)
	// @and [features/agent-cli-jobs.feature] it has a contact, an interview, a next action, and an attached document
	job.Contacts = []jobs.Contact{{ID: 7, Name: "Dana Smith", Relationship: jobs.RelationshipRecruiter}}
	job.Interviews = []jobs.Interview{{ID: 8, Date: "2026-10-20", Time: "14:00", Kind: jobs.InterviewTechnical}}
	job.NextAction = &jobs.NextAction{Text: "Prepare for the interview", ExpectedDate: "2026-10-19"}
	job.Documents = []jobs.Document{{ID: 9, JobID: job.ID, Kind: jobs.KindResume, FileName: "resume-2026.pdf"}}
	h.jobs.files[job.ID] = []jobs.DocumentFile{{DocumentID: 9, Path: "/data/documents/abc.pdf"}}
	// @and [features/agent-cli-jobs.feature] it has moved from "Saved" to "Applied"
	job.StatusHistory = []jobs.StatusChange{{Status: jobs.StatusSaved, Date: "2026-10-01"}, {Status: jobs.StatusApplied, Date: "2026-10-05"}}

	// @when [features/agent-cli-jobs.feature] my agent shows the job
	r := h.run("job", "show", fmt.Sprint(job.ID))

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-jobs.feature] it sees every detail of the job, including its description, notes, and posting link
	wantShown(t, r.stdout,
		"Status: Applied\n", "Company: Acme\n", "Role: Software Engineer\n", "Location: Remote, US\n", "Salary: $120k\n",
		"Posting link: https://acme.example/jobs/1\n", "Date applied: 2026-10-05\n", "Interest: 4 stars\n",
		"\nDescription:\n  Build the payments API\n", "\nNotes:\n  Ask about the team size\n")
	// @and [features/agent-cli-jobs.feature] it sees its contacts, its interviews, its next action, and its status history
	wantShown(t, r.stdout,
		"\nContacts:\n  #7 Dana Smith | Recruiter\n", "\nInterviews:\n  #8 2026-10-20 14:00 | Technical\n",
		"Next action: Prepare for the interview, expected 2026-10-19\n", "\nStatus history:\n  2026-10-01 Saved\n  2026-10-05 Applied\n")
	// @and [features/agent-cli-jobs.feature] it sees each attached document with its kind and where its stored copy is
	wantShown(t, r.stdout, "\nDocuments:\n  #9 resume-2026.pdf | Resume | stored copy: /data/documents/abc.pdf\n")
}

// wantShown fails the test unless shown has every one of want.
func wantShown(t *testing.T, shown string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(shown, w) {
			t.Errorf("doesn't show %q:\n%s", w, shown)
		}
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-show-missing-document] Showing a job marks a document whose file is missing
func TestAgentShowMarksAMissingDocument(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" with "resume-2026.pdf" attached
	job := h.acme()
	job.Documents = []jobs.Document{{ID: 9, JobID: job.ID, Kind: jobs.KindResume, FileName: "resume-2026.pdf"}}
	// @and [features/agent-cli-jobs.feature] the stored copy of "resume-2026.pdf" is gone
	h.jobs.files[job.ID] = []jobs.DocumentFile{{DocumentID: 9, Path: "/data/documents/abc.pdf", Missing: true}}

	// @when [features/agent-cli-jobs.feature] my agent shows the job
	r := h.run("job", "show", fmt.Sprint(job.ID))

	// @then [features/agent-cli-jobs.feature] "resume-2026.pdf" is marked as missing its file
	wantShown(t, r.stdout, "#9 resume-2026.pdf | Resume | stored copy: /data/documents/abc.pdf (missing: the stored copy is gone)\n")
}

// @scenario [features/agent-cli-jobs.feature#cli-no-opening] My agent can't open anything on my screen
func TestAgentCantOpenAnything(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" with a posting link and an attached document
	job := h.acme()
	job.Link = "https://acme.example/jobs/1"
	job.Documents = []jobs.Document{{ID: 9, JobID: job.ID, Kind: jobs.KindResume, FileName: "resume-2026.pdf"}}
	h.jobs.files[job.ID] = []jobs.DocumentFile{{DocumentID: 9, Path: "/data/documents/abc.pdf"}}
	job.Contacts = []jobs.Contact{{ID: 7, Name: "Dana Smith", Relationship: jobs.RelationshipRecruiter, ProfileLink: "https://example.com/in/dana"}}

	// @when [features/agent-cli-jobs.feature] my agent looks at what it can do with the job
	help := h.run("--help").stdout
	shown := h.run("job", "show", fmt.Sprint(job.ID)).stdout

	// @then [features/agent-cli-jobs.feature] it can't open the posting link, a contact's profile link, or a document on my screen
	for _, cmd := range commands {
		if slices.ContainsFunc(cmd.path, func(word string) bool { return strings.Contains(word, "open") }) {
			t.Errorf("there's a command %q", cmd.name())
		}
	}
	if strings.Contains(strings.ToLower(help), "open") {
		t.Errorf("the help offers to open something:\n%s", help)
	}
	// @but [features/agent-cli-jobs.feature] it can read the links and where the documents' stored copies are
	wantShown(t, shown, "Posting link: https://acme.example/jobs/1\n", "profile https://example.com/in/dana", "stored copy: /data/documents/abc.pdf")
}

// The board's order is the jobs service's (TestNewCardGoesToTop in internal/jobs); the fake puts
// an added job at the top of its column as the service does.
// @scenario [features/agent-cli-jobs.feature#cli-add-job] Add a job with its details
func TestAgentAddsAJob(t *testing.T) {
	h := newHarness()
	h.jobs.add("Globex", "Designer", jobs.StatusSaved)

	// @when [features/agent-cli-jobs.feature] my agent adds a job for "Software Engineer" at "Acme" with a salary, a location, and a posting link
	r := h.run("job", "add", "--company", "Acme", "--role", "Software Engineer", "--salary", "$120k", "--location", "Remote, US", "--link", "https://acme.example/jobs/1")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-jobs.feature] the board has a job for "Software Engineer" at "Acme" at the top of "Saved"
	added := h.jobs.jobs[h.jobs.nextID]
	if added.Company != "Acme" || added.Role != "Software Engineer" || added.Status != jobs.StatusSaved ||
		added.Salary != "$120k" || added.Location != "Remote, US" || added.Link != "https://acme.example/jobs/1" {
		t.Errorf("added %+v", added)
	}
	wantCompanies(t, h.run("job", "list", "--status", "saved"), "Acme", "Globex")
	// @and [features/agent-cli-jobs.feature] my agent is shown the job with its ID
	if want := fmt.Sprintf("Job #%d: Software Engineer at Acme\n", added.ID); !strings.HasPrefix(r.stdout, want) {
		t.Errorf("stdout = %q, want it to start %q", r.stdout, want)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-add-job-status] Add a job straight to a status
func TestAgentAddsAJobToAStatus(t *testing.T) {
	h := newHarness()
	h.jobs.add("Globex", "Designer", jobs.StatusApplied)

	// @when [features/agent-cli-jobs.feature] my agent adds a job for "Software Engineer" at "Acme" to "Applied"
	r := h.run("job", "add", "--company", "Acme", "--role", "Software Engineer", "--status", "Applied")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-jobs.feature] the job is at the top of the "Applied" column
	wantCompanies(t, h.run("job", "list", "--status", "applied"), "Acme", "Globex")
}

// The messages are the jobs service's, tested in internal/jobs; the fake hands them back.
// @scenario [features/agent-cli-jobs.feature#cli-add-job-rules] Adding a job follows the app's rules
func TestAddingAJobFollowsTheAppsRules(t *testing.T) {
	for _, tc := range []struct {
		problem string
		args    []string
		err     error
		reason  string
	}{
		{"with no company", []string{"--role", "Software Engineer"}, &jobs.ValidationError{Msg: "company is required"}, "company is required"},
		{"with no role", []string{"--company", "Acme"}, errNeedsRole, "role is required"},
		{"to \"Closed\"", []string{"--company", "Acme", "--role", "Software Engineer", "--status", "Closed"},
			&jobs.ValidationError{Msg: "a job can't be added straight to Closed; add it to another column, then move it to Closed"}, "a job can't be added straight to Closed"},
		{"with a posting link that isn't a web address", []string{"--company", "Acme", "--role", "Software Engineer", "--link", "acme.example/jobs"}, errNotWebAdress, "the link must be a web address"},
		{"rated 6 stars", []string{"--company", "Acme", "--role", "Software Engineer", "--interest", "6"}, errRatingRange, "the interest rating must be from 1 to 5 stars"},
	} {
		t.Run(tc.problem, func(t *testing.T) {
			h := newHarness()
			h.jobs.fail = tc.err

			// @when [features/agent-cli-jobs.feature] my agent tries to add a job <problem>
			r := h.run(append([]string{"job", "add"}, tc.args...)...)

			// @then [features/agent-cli-jobs.feature] no job is added
			if len(h.jobs.jobs) != 0 {
				t.Errorf("a job was added: %v", h.jobs.jobs)
			}
			// @and [features/agent-cli-jobs.feature] my agent is told <reason>
			r.wantCode(t, ExitValidation)
			if !strings.Contains(r.stderr, tc.reason) {
				t.Errorf("stderr = %q, want %q", r.stderr, tc.reason)
			}
		})
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-update-job] Change one detail and keep the rest
func TestAgentChangesOneDetail(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job for "Software Engineer" at "Acme" with notes and a salary of "$120k"
	job := h.acme()
	job.Notes, job.Salary = "Ask about the team size", "$120k"

	// @when [features/agent-cli-jobs.feature] my agent changes the job's salary to "$130k"
	h.run("job", "update", fmt.Sprint(job.ID), "--salary", "$130k").wantCode(t, ExitOK)

	// @then [features/agent-cli-jobs.feature] the job's salary is "$130k"
	if job.Salary != "$130k" {
		t.Errorf("salary = %q", job.Salary)
	}
	// @and [features/agent-cli-jobs.feature] the job's notes are unchanged
	if job.Notes != "Ask about the team size" {
		t.Errorf("notes = %q", job.Notes)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-clear-job-detail] Clear a detail
func TestAgentClearsADetail(t *testing.T) {
	for _, tc := range []struct {
		detail string
		option string
		set    func(*jobs.Job)
		isSet  func(jobs.Job) bool
	}{
		{"salary", "--salary", func(j *jobs.Job) { j.Salary = "$120k" }, func(j jobs.Job) bool { return j.Salary != "" }},
		{"notes", "--notes", func(j *jobs.Job) { j.Notes = "Ask about the team size" }, func(j jobs.Job) bool { return j.Notes != "" }},
		{"date applied", "--date-applied", func(j *jobs.Job) { j.DateApplied = "2026-10-05" }, func(j jobs.Job) bool { return j.DateApplied != "" }},
		{"interest rating", "--interest", func(j *jobs.Job) { rate(j, 4) }, func(j jobs.Job) bool { return j.Interest != nil }},
	} {
		t.Run(tc.detail, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli-jobs.feature] I have a job at "Acme" with <detail> set
			job := h.acme()
			tc.set(job)

			// @when [features/agent-cli-jobs.feature] my agent clears the job's <detail>
			h.run("job", "update", fmt.Sprint(job.ID), tc.option, "").wantCode(t, ExitOK)

			// @then [features/agent-cli-jobs.feature] the job has no <detail>
			if tc.isSet(*job) {
				t.Errorf("the job still has its %s: %+v", tc.detail, job)
			}
		})
	}
}

// moveHarness has "Initech" and "Globex" in Applied, in that order, and "Acme" in Saved.
func moveHarness() (*harness, *jobs.Job) {
	h := newHarness()
	h.jobs.add("Initech", "Engineer", jobs.StatusApplied)
	h.jobs.add("Globex", "Designer", jobs.StatusApplied)
	return h, h.acme()
}

// Where a job lands is the jobs service's (TestMoveCardToPosition and
// TestMoveIndexOutOfRangeLandsAtTheEnds in internal/jobs); the fake places it as the service does.
func TestAgentMovesAJobInAColumn(t *testing.T) {
	// @scenario [features/agent-cli-jobs.feature#cli-move-top] A moved job goes to the top of its new column
	t.Run("top", func(t *testing.T) {
		// @given [features/agent-cli-jobs.feature] the "Applied" column holds the jobs at "Initech" and "Globex"
		// @and [features/agent-cli-jobs.feature] I have a job at "Acme" in "Saved"
		h, acme := moveHarness()

		// @when [features/agent-cli-jobs.feature] my agent moves the job at "Acme" to "Applied"
		h.run("job", "move", fmt.Sprint(acme.ID), "--status", "applied").wantCode(t, ExitOK)

		// @then [features/agent-cli-jobs.feature] the "Applied" column holds "Acme", "Initech", and "Globex" in that order
		wantCompanies(t, h.run("job", "list", "--status", "applied"), "Acme", "Initech", "Globex")
	})

	// @scenario [features/agent-cli-jobs.feature#cli-move-bottom] Move a job to the bottom of a column
	t.Run("bottom", func(t *testing.T) {
		// @given [features/agent-cli-jobs.feature] the "Applied" column holds the jobs at "Initech" and "Globex"
		// @and [features/agent-cli-jobs.feature] I have a job at "Acme" in "Saved"
		h, acme := moveHarness()

		// @when [features/agent-cli-jobs.feature] my agent moves the job at "Acme" to the bottom of "Applied"
		h.run("job", "move", fmt.Sprint(acme.ID), "--status", "applied", "--bottom").wantCode(t, ExitOK)

		// @then [features/agent-cli-jobs.feature] the "Applied" column holds "Initech", "Globex", and "Acme" in that order
		wantCompanies(t, h.run("job", "list", "--status", "applied"), "Initech", "Globex", "Acme")
	})

	// @scenario [features/agent-cli-jobs.feature#cli-move-position] Move a job to a chosen place in a column
	t.Run("position", func(t *testing.T) {
		// @given [features/agent-cli-jobs.feature] the "Applied" column holds the jobs at "Initech" and "Globex"
		// @and [features/agent-cli-jobs.feature] I have a job at "Acme" in "Saved"
		h, acme := moveHarness()

		// @when [features/agent-cli-jobs.feature] my agent moves the job at "Acme" to second place in "Applied"
		h.run("job", "move", fmt.Sprint(acme.ID), "--status", "applied", "--position", "2").wantCode(t, ExitOK)

		// @then [features/agent-cli-jobs.feature] the "Applied" column holds "Initech", "Acme", and "Globex" in that order
		wantCompanies(t, h.run("job", "list", "--status", "applied"), "Initech", "Acme", "Globex")
	})
}

func TestMovePlacement(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		index int
	}{
		{nil, 0},
		{[]string{"--position", "1"}, 0},
		{[]string{"--position", "3"}, 2},
		{[]string{"--bottom"}, 1<<31 - 1},
	} {
		h := newHarness()
		job := h.acme()
		h.run(append([]string{"job", "move", fmt.Sprint(job.ID), "--status", "applied"}, tc.args...)...).wantCode(t, ExitOK)
		if got := h.jobs.moves[0].Index; got != tc.index {
			t.Errorf("%q: index %d, want %d", tc.args, got, tc.index)
		}
	}
	for _, args := range [][]string{{"--position", "0"}, {"--position", "second"}, {"--position", "2", "--bottom"}} {
		h := newHarness()
		job := h.acme()
		r := h.run(append([]string{"job", "move", fmt.Sprint(job.ID), "--status", "applied"}, args...)...)
		r.wantCode(t, ExitUsage)
		if len(h.jobs.moves) != 0 {
			t.Errorf("%q moved the job", args)
		}
	}
}

// The date applied and the status history are the jobs service's: see
// TestAgentMoveFollowsTheAppsRules in internal/jobs, which carries this scenario's other steps.
// @scenario [features/agent-cli-jobs.feature#cli-move-follows-rules] Moving a job follows the app's rules
func TestAgentMoveFollowsTheAppsRules(t *testing.T) {
	h := newHarness()
	job := h.acme()

	// @when [features/agent-cli-jobs.feature] my agent moves the job to "Applied"
	h.run("job", "move", fmt.Sprint(job.ID), "--status", "Applied").wantCode(t, ExitOK)

	if len(h.jobs.moves) != 1 || h.jobs.moves[0] != (jobs.Move{JobID: job.ID, Status: jobs.StatusApplied}) {
		t.Errorf("moves = %v", h.jobs.moves)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-move-to-closed-needs-close] Moving a job to Closed points my agent to closing it
func TestAgentMoveToClosedPointsToClosing(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" in "Applied"
	job := h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)

	// @when [features/agent-cli-jobs.feature] my agent tries to move the job to "Closed"
	r := h.run("job", "move", fmt.Sprint(job.ID), "--status", "Closed")

	// @then [features/agent-cli-jobs.feature] the job doesn't move
	if len(h.jobs.moves) != 0 || job.Status != jobs.StatusApplied {
		t.Errorf("moves %v, status %q", h.jobs.moves, job.Status)
	}
	// @and [features/agent-cli-jobs.feature] my agent is told to close the job with a reason instead
	r.wantCode(t, ExitValidation)
	if want := fmt.Sprintf("a job is moved to Closed by closing it with a reason: `callback-cli job close %d --reason <reason>`", job.ID); !strings.Contains(r.stderr, want) {
		t.Errorf("stderr = %q", r.stderr)
	}
}

// wantClosed fails the test unless job is closed with the reason text.
func wantClosed(t *testing.T, job *jobs.Job, text string) {
	t.Helper()
	if job.Status != jobs.StatusClosed || job.ClosingReasonText != text {
		t.Errorf("status %q, reason %q; want Closed, %q", job.Status, job.ClosingReasonText, text)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-close-job] Close a job with a reason
func TestAgentClosesAJob(t *testing.T) {
	for _, reason := range []string{"Rejected", "Withdrew", "No response", "Declined offer", "Position filled"} {
		t.Run(reason, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli-jobs.feature] I have a job at "Acme" in "Applied"
			job := h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)

			// @when [features/agent-cli-jobs.feature] my agent closes the job because "<reason>"
			h.run("job", "close", fmt.Sprint(job.ID), "--reason", reason).wantCode(t, ExitOK)

			// @then [features/agent-cli-jobs.feature] the job is in "Closed" with the reason "<reason>"
			wantClosed(t, job, reason)
		})
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-close-other] Close a job with my own reason
func TestAgentClosesAJobWithItsOwnReason(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" in "Applied"
	job := h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)

	// @when [features/agent-cli-jobs.feature] my agent closes the job with the reason "Other" and the text "Relocated"
	h.run("job", "close", fmt.Sprint(job.ID), "--reason", "Other", "--other", "Relocated").wantCode(t, ExitOK)

	// @then [features/agent-cli-jobs.feature] the job is in "Closed" with the reason "Relocated"
	wantClosed(t, job, "Relocated")
}

// The "Other" reason's message is the jobs service's (TestCloseWithOtherNeedsText in
// internal/jobs); the fake hands it back.
// @scenario [features/agent-cli-jobs.feature#cli-close-needs-reason] Closing a job needs a reason
func TestAgentClosingNeedsAReason(t *testing.T) {
	for _, tc := range []struct {
		problem string
		args    []string
		err     error
		reason  string
	}{
		{"without a reason", nil, nil, "Use one of: rejected, withdrew, no_response, declined_offer, position_filled, other"},
		{"with the reason \"Other\" and no text", []string{"--reason", "other"}, &jobs.ValidationError{Msg: "write why the job was closed"}, "write why the job was closed"},
	} {
		t.Run(tc.problem, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli-jobs.feature] I have a job at "Acme" in "Applied"
			job := h.jobs.add("Acme", "Software Engineer", jobs.StatusApplied)
			h.jobs.fail = tc.err

			// @when [features/agent-cli-jobs.feature] my agent tries to close the job <problem>
			r := h.run(append([]string{"job", "close", fmt.Sprint(job.ID)}, tc.args...)...)

			// @then [features/agent-cli-jobs.feature] the job is still in "Applied"
			if job.Status != jobs.StatusApplied {
				t.Errorf("status = %q", job.Status)
			}
			// @and [features/agent-cli-jobs.feature] my agent is told <reason>
			r.wantCode(t, ExitValidation)
			if !strings.Contains(r.stderr, tc.reason) {
				t.Errorf("stderr = %q, want %q", r.stderr, tc.reason)
			}
		})
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-change-closing-reason] Change a closed job's reason
func TestAgentChangesAClosedJobsReason(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" in "Closed" with the reason "No response"
	job := h.jobs.add("Acme", "Software Engineer", jobs.StatusClosed)
	job.ClosingReason, job.ClosingReasonText = jobs.ReasonNoResponse, "No response"

	// @when [features/agent-cli-jobs.feature] my agent closes the job because "Rejected"
	h.run("job", "close", fmt.Sprint(job.ID), "--reason", "Rejected").wantCode(t, ExitOK)

	// @then [features/agent-cli-jobs.feature] the job is in "Closed" with the reason "Rejected"
	wantClosed(t, job, "Rejected")
}

// What goes with the job is the storage's: see TestDeleteJobDeletesEverythingOnIt in internal/db.
// @scenario [features/agent-cli-jobs.feature#cli-delete-job] Delete a job
func TestAgentDeletesAJob(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" with a contact, an interview, and an attached document
	job := h.acme()
	job.Contacts = []jobs.Contact{{ID: 7, Name: "Dana Smith", Relationship: jobs.RelationshipRecruiter}}
	job.Interviews = []jobs.Interview{{ID: 8, Date: "2026-10-20", Kind: jobs.InterviewTechnical}}
	job.Documents = []jobs.Document{{ID: 9, JobID: job.ID, Kind: jobs.KindResume, FileName: "resume-2026.pdf"}}
	h.jobs.add("Globex", "Designer", "")

	// @when [features/agent-cli-jobs.feature] my agent deletes the job
	h.run("job", "delete", fmt.Sprint(job.ID)).wantCode(t, ExitOK)

	// @then [features/agent-cli-jobs.feature] the board has no job at "Acme"
	wantCompanies(t, h.run("job", "list", "--all"), "Globex")
}

// @scenario [features/agent-cli-jobs.feature#cli-set-next-action] Set a next action with an expected date
func TestAgentSetsANextAction(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme"
	job := h.acme()

	// @when [features/agent-cli-jobs.feature] my agent sets the job's next action to "Send thank-you note" expected on 2026-10-20
	r := h.run("job", "set-next-action", fmt.Sprint(job.ID), "--text", "Send thank-you note", "--date", "2026-10-20")

	// @then [features/agent-cli-jobs.feature] the job's next action is "Send thank-you note" expected on 2026-10-20
	r.wantCode(t, ExitOK)
	wantShown(t, r.stdout, "Next action: Send thank-you note, expected 2026-10-20\n")
}

// @scenario [features/agent-cli-jobs.feature#cli-set-next-action-no-date] Set a next action without an expected date
func TestAgentSetsANextActionWithoutADate(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme"
	job := h.acme()

	// @when [features/agent-cli-jobs.feature] my agent sets the job's next action to "Research the team"
	r := h.run("job", "set-next-action", fmt.Sprint(job.ID), "--text", "Research the team")

	// @then [features/agent-cli-jobs.feature] the job's next action is "Research the team" with no expected date
	r.wantCode(t, ExitOK)
	if job.NextAction == nil || *job.NextAction != (jobs.NextAction{Text: "Research the team"}) {
		t.Errorf("next action = %+v", job.NextAction)
	}
	wantShown(t, r.stdout, "Next action: Research the team\n")
}

// @scenario [features/agent-cli-jobs.feature#cli-clear-next-action-date] Clear only the expected date
func TestAgentClearsANextActionsDate(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" whose next action is "Follow up" expected on 2026-10-20
	job := h.acme()
	job.NextAction = &jobs.NextAction{Text: "Follow up", ExpectedDate: "2026-10-20"}

	// @when [features/agent-cli-jobs.feature] my agent clears the next action's expected date
	r := h.run("job", "clear-next-action", fmt.Sprint(job.ID), "--only-date")

	// @then [features/agent-cli-jobs.feature] the job's next action is "Follow up" with no expected date
	r.wantCode(t, ExitOK)
	if job.NextAction == nil || *job.NextAction != (jobs.NextAction{Text: "Follow up"}) {
		t.Errorf("next action = %+v", job.NextAction)
	}
	wantShown(t, r.stdout, "Next action: Follow up\n")
}

func TestClearingTheDateOfNoNextAction(t *testing.T) {
	h := newHarness()
	job := h.acme()
	r := h.run("job", "clear-next-action", fmt.Sprint(job.ID), "--only-date")
	r.wantCode(t, ExitValidation)
	if job.NextAction != nil || !strings.Contains(r.stderr, "has no next action") {
		t.Errorf("next action %+v, stderr %q", job.NextAction, r.stderr)
	}
}

// @scenario [features/agent-cli-jobs.feature#cli-clear-next-action] Clear the next action
func TestAgentClearsANextAction(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme" whose next action is "Follow up"
	job := h.acme()
	job.NextAction = &jobs.NextAction{Text: "Follow up"}

	// @when [features/agent-cli-jobs.feature] my agent clears the job's next action
	r := h.run("job", "clear-next-action", fmt.Sprint(job.ID))

	// @then [features/agent-cli-jobs.feature] the job has no next action
	r.wantCode(t, ExitOK)
	if job.NextAction != nil {
		t.Errorf("next action = %+v", job.NextAction)
	}
}

// The message is the jobs service's (TestNextActionNeedsText in internal/jobs); the fake hands it
// back.
// @scenario [features/agent-cli-jobs.feature#cli-next-action-needs-text] A next action needs to say what it is
func TestAgentNextActionNeedsText(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-jobs.feature] I have a job at "Acme"
	job := h.acme()
	h.jobs.fail = &jobs.ValidationError{Msg: "write what the next action is"}

	// @when [features/agent-cli-jobs.feature] my agent tries to set the job's next action to nothing
	r := h.run("job", "set-next-action", fmt.Sprint(job.ID), "--text", "  ")

	// @then [features/agent-cli-jobs.feature] the job has no next action
	if job.NextAction != nil {
		t.Errorf("next action = %+v", job.NextAction)
	}
	// @and [features/agent-cli-jobs.feature] my agent is told the next action needs to say what it is
	r.wantCode(t, ExitValidation)
	if !strings.Contains(r.stderr, "write what the next action is") {
		t.Errorf("stderr = %q", r.stderr)
	}
}

func TestListFilterProblems(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"--status", "pending"}, ExitValidation},
		{[]string{"--days-in-status", "two weeks"}, ExitUsage},
		{[]string{"--min-interest", "-1"}, ExitUsage},
	} {
		h := newHarness()
		h.run(append([]string{"job", "list"}, tc.args...)...).wantCode(t, tc.code)
	}
}

func TestListSearchMatchesTheCompanyAndIgnoresSpaces(t *testing.T) {
	h := newHarness()
	h.acme()
	h.jobs.add("Globex", "Designer", "")
	wantCompanies(t, h.run("job", "list", "--search", "  GLOB "), "Globex")
	wantCompanies(t, h.run("job", "list", "--search", "   "), "Acme", "Globex")
}

func TestListFiltersLeaveOutJobsWithoutWhatTheyAskFor(t *testing.T) {
	h := newHarness()
	h.acme()
	wantCompanies(t, h.run("job", "list", "--days-in-status", "0"))
	if got := h.run("job", "list", "--overdue").stdout; got != "No jobs match.\n" {
		t.Errorf("stdout = %q", got)
	}
}

func TestNoMatchPointsToClosedJobs(t *testing.T) {
	h := newHarness()
	h.jobs.add("Acme", "Pilot", jobs.StatusClosed)
	if got := h.run("job", "list", "--search", "pilot").stdout; got != "No open jobs match. Add --all to look through the closed ones too.\n" {
		t.Errorf("stdout = %q", got)
	}
	wantCompanies(t, h.run("job", "list", "--search", "pilot", "--all"), "Acme")
}
