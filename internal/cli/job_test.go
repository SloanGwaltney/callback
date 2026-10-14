package cli

import (
	"fmt"
	"strings"
	"testing"

	"callback/internal/jobs"
)

func TestListLineHasEveryDetail(t *testing.T) {
	h := newHarness()
	job := h.acme()
	stars := 4
	job.Status, job.Location, job.DateApplied, job.Interest = jobs.StatusApplied, "Remote, US", "2026-10-05", &stars
	job.NextAction = &jobs.NextAction{Text: "Follow up", ExpectedDate: "2026-10-10", Due: jobs.DueOverdue}
	h.jobs.days[job.ID] = 9

	r := h.run("job", "list")

	want := fmt.Sprintf("#%d | Applied for 9 days | Acme | Software Engineer | Remote, US | applied 2026-10-05 | 4 stars | next: Follow up, expected 2026-10-10 (overdue)\n", job.ID)
	if r.stdout != want {
		t.Errorf("list = %q\nwant  %q", r.stdout, want)
	}
}

func TestListDaysAndStars(t *testing.T) {
	h := newHarness()
	job := h.acme()
	one := 1
	job.Interest = &one
	job.NextAction = &jobs.NextAction{Text: "Call", Due: jobs.DueToday}
	h.jobs.days[job.ID] = 0
	if got := h.run("job", "list").stdout; !strings.Contains(got, "| Saved since today |") || !strings.Contains(got, "| 1 star | next: Call (due today)\n") {
		t.Errorf("list = %q", got)
	}
	h.jobs.days[job.ID] = 1
	if got := h.run("job", "list").stdout; !strings.Contains(got, "| Saved for 1 day |") {
		t.Errorf("list = %q", got)
	}
}

func TestListLeavesOutClosedJobsUnlessAll(t *testing.T) {
	h := newHarness()
	h.acme()
	globex := h.jobs.add("Globex", "Designer", jobs.StatusClosed)
	globex.ClosingReasonText = "Rejected"

	if got := h.run("job", "list").stdout; strings.Contains(got, "Globex") || !strings.Contains(got, "Acme") {
		t.Errorf("list = %q", got)
	}
	got := h.run("job", "list", "--all").stdout
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], "| Closed (Rejected) | Globex |") {
		t.Errorf("list --all = %q", got)
	}
}

func TestListWithNothingToList(t *testing.T) {
	h := newHarness()
	if got := h.run("job", "list").stdout; got != "No jobs to list.\n" {
		t.Errorf("list = %q", got)
	}
	if got := h.run("job", "list", "--json").stdout; got != "[]\n" {
		t.Errorf("list --json = %q", got)
	}
	h.jobs.add("Globex", "Designer", jobs.StatusClosed)
	if got := h.run("job", "list").stdout; got != "No open jobs to list. Add --all to list the 1 closed ones.\n" {
		t.Errorf("list = %q", got)
	}
}

func TestListTakesNoArguments(t *testing.T) {
	r := newHarness().run("job", "list", "acme")
	r.wantCode(t, ExitUsage)
}

func TestShowWritesEveryDetail(t *testing.T) {
	h := newHarness()
	job := h.acme()
	job.Status, job.ClosingReasonText = jobs.StatusClosed, "Relocated"
	job.Description, job.Notes = "Line one\n\nLine three\n", "Call Jane"
	job.Salary, job.Link = "$120k", "https://acme.example/jobs/1"
	job.Contacts = []jobs.Contact{{ID: 7, Name: "Jane Doe", Relationship: jobs.RelationshipRecruiter, Email: "jane@acme.example", Notes: "Prefers email"}}
	job.Interviews = []jobs.Interview{{ID: 8, Date: "2026-10-01", Kind: jobs.InterviewPhoneScreen, Past: true}}
	job.StatusHistory = []jobs.StatusChange{{Status: jobs.StatusSaved, Date: "2026-09-30"}, {Status: jobs.StatusClosed, Date: "2026-10-02", ClosingReasonText: "Relocated"}}

	got := h.run("job", "show", fmt.Sprint(job.ID)).stdout

	for _, want := range []string{
		"Status: Closed\nClosing reason: Relocated\n",
		"Salary: $120k\nPosting link: https://acme.example/jobs/1\nDate applied: (none)\nInterest: (none)\nNext action: (none)\n",
		"\nDescription:\n  Line one\n\n  Line three\n\nNotes:\n  Call Jane\n",
		"  #7 Jane Doe | Recruiter | email jane@acme.example\n    Prefers email\n",
		"  #8 2026-10-01 | Phone screen (past)\n",
		"\nDocuments:\n  (none)\n",
		"\nStatus history:\n  2026-09-30 Saved\n  2026-10-02 Closed (Relocated)\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("show doesn't have %q:\n%s", want, got)
		}
	}
}

func TestIDsMustBeWholeNumbers(t *testing.T) {
	for _, args := range [][]string{{"job", "show"}, {"job", "show", "acme"}, {"job", "show", "0"}, {"job", "show", "1", "2"}} {
		r := newHarness().run(args...)
		if r.code != ExitUsage {
			t.Errorf("%q: exit %d, stderr %q", args, r.code, r.stderr)
		}
	}
}

func TestOptionsCanComeBeforeOrAfterTheID(t *testing.T) {
	h := newHarness()
	job := h.acme()
	r := h.run("job", "update", "--salary", "$130k", fmt.Sprint(job.ID), "--location", "Remote")
	r.wantCode(t, ExitOK)
	if job.Salary != "$130k" || job.Location != "Remote" {
		t.Errorf("salary %q, location %q", job.Salary, job.Location)
	}
}

func TestAnUnknownOptionIsAUsageError(t *testing.T) {
	r := newHarness().run("job", "move", "1", "--colour", "red")
	r.wantCode(t, ExitUsage)
	if !strings.Contains(r.stderr, "provided but not defined: -colour; run `callback-cli job move --help` to see its options") {
		t.Errorf("stderr = %q", r.stderr)
	}
	r = newHarness().run("--colour", "job", "list")
	r.wantCode(t, ExitUsage)
}

func TestAddJob(t *testing.T) {
	h := newHarness()
	r := h.run("job", "add", "--company", "Acme", "--role", "Software Engineer", "--status", "Applied", "--interest", "5", "--link", "https://acme.example")

	r.wantCode(t, ExitOK)
	job := h.jobs.jobs[h.jobs.nextID]
	if job.Status != jobs.StatusApplied || *job.Interest != 5 || job.Link != "https://acme.example" {
		t.Errorf("added %+v", job)
	}
	if !strings.HasPrefix(r.stdout, fmt.Sprintf("Job #%d: Software Engineer at Acme\n", job.ID)) {
		t.Errorf("stdout = %q", r.stdout)
	}
}

func TestAddJobToAnUnknownStatus(t *testing.T) {
	h := newHarness()
	r := h.run("job", "add", "--company", "Acme", "--role", "Engineer", "--status", "pending")
	r.wantCode(t, ExitValidation)
	if len(h.jobs.jobs) != 0 {
		t.Error("a job was added")
	}
}

func TestUpdateKeepsWhatIsntGivenAndClearsWhatIsEmpty(t *testing.T) {
	h := newHarness()
	job := h.acme()
	stars := 3
	job.Notes, job.Salary, job.Interest = "Keep me", "$120k", &stars

	h.run("job", "update", fmt.Sprint(job.ID), "--salary", "", "--interest", "")

	if job.Notes != "Keep me" || job.Salary != "" || job.Interest != nil {
		t.Errorf("notes %q, salary %q, interest %v", job.Notes, job.Salary, job.Interest)
	}
}

func TestInterestMustBeANumber(t *testing.T) {
	h := newHarness()
	job := h.acme()
	r := h.run("job", "update", fmt.Sprint(job.ID), "--interest", "lots")
	r.wantCode(t, ExitUsage)
}

func TestMoveNeedsAStatus(t *testing.T) {
	r := newHarness().run("job", "move", "1")
	r.wantCode(t, ExitUsage)
	if !strings.Contains(r.stderr, "Use one of: saved, applied, interviewing, offer, closed") {
		t.Errorf("stderr = %q", r.stderr)
	}
}

func TestMoveToClosedPointsToClose(t *testing.T) {
	h := newHarness()
	job := h.acme()

	r := h.run("job", "move", fmt.Sprint(job.ID), "--status", "closed")

	r.wantCode(t, ExitValidation)
	if len(h.jobs.moves) != 0 || !strings.Contains(r.stderr, fmt.Sprintf("`callback-cli job close %d --reason <reason>`", job.ID)) {
		t.Errorf("moves %v, stderr %q", h.jobs.moves, r.stderr)
	}
}

func TestMoveGoesToTheTop(t *testing.T) {
	h := newHarness()
	job := h.acme()
	h.run("job", "move", fmt.Sprint(job.ID), "--status", "interviewing")
	if len(h.jobs.moves) != 1 || h.jobs.moves[0] != (jobs.Move{JobID: job.ID, Status: jobs.StatusInterviewing, Index: 0}) {
		t.Errorf("moves = %v", h.jobs.moves)
	}
}

func TestCloseAJob(t *testing.T) {
	h := newHarness()
	job := h.acme()

	r := h.run("job", "close", fmt.Sprint(job.ID), "--reason", "No response")

	r.wantCode(t, ExitOK)
	if job.Status != jobs.StatusClosed || job.ClosingReason != jobs.ReasonNoResponse {
		t.Errorf("status %q, reason %q", job.Status, job.ClosingReason)
	}
}

func TestCloseWithOther(t *testing.T) {
	h := newHarness()
	job := h.acme()
	h.run("job", "close", fmt.Sprint(job.ID), "--reason", "other", "--other", "Relocated")
	if m := h.jobs.moves[0]; m.ClosingReason != jobs.ReasonOther || m.ClosingReasonOther != "Relocated" {
		t.Errorf("move = %+v", m)
	}
}

func TestCloseAClosedJobChangesItsReason(t *testing.T) {
	h := newHarness()
	job := h.jobs.add("Acme", "Software Engineer", jobs.StatusClosed)
	job.ClosingReason = jobs.ReasonNoResponse

	h.run("job", "close", fmt.Sprint(job.ID), "--reason", "rejected")

	if len(h.jobs.moves) != 0 || job.ClosingReason != jobs.ReasonRejected {
		t.Errorf("moves %v, reason %q", h.jobs.moves, job.ClosingReason)
	}
}

func TestCloseNeedsAReason(t *testing.T) {
	h := newHarness()
	job := h.acme()

	r := h.run("job", "close", fmt.Sprint(job.ID))

	r.wantCode(t, ExitValidation)
	if job.Status != jobs.StatusSaved || !strings.Contains(r.stderr, "Use one of: rejected, withdrew, no_response, declined_offer, position_filled, other") {
		t.Errorf("status %q, stderr %q", job.Status, r.stderr)
	}
}

func TestNextAction(t *testing.T) {
	h := newHarness()
	job := h.acme()

	h.run("job", "set-next-action", fmt.Sprint(job.ID), "--text", "Follow up", "--date", "2026-10-20")
	if job.NextAction == nil || *job.NextAction != (jobs.NextAction{Text: "Follow up", ExpectedDate: "2026-10-20"}) {
		t.Errorf("next action = %+v", job.NextAction)
	}
	r := h.run("job", "clear-next-action", fmt.Sprint(job.ID))
	if job.NextAction != nil || !strings.Contains(r.stdout, "Next action: (none)\n") {
		t.Errorf("next action = %+v, stdout %q", job.NextAction, r.stdout)
	}
}

func TestPick(t *testing.T) {
	for _, given := range []string{"no_response", "No response", "no-response", " NO RESPONSE "} {
		if got, err := pick("closing reason", given, jobs.ClosingReasons, jobs.ClosingReason.Label); err != nil || got != jobs.ReasonNoResponse {
			t.Errorf("pick(%q) = %q, %v", given, got, err)
		}
	}
	if _, err := pick("closing reason", "bored", jobs.ClosingReasons, jobs.ClosingReason.Label); classify(err).code != ExitValidation {
		t.Errorf("err = %v", err)
	}
}

func TestDeleteAsJSON(t *testing.T) {
	h := newHarness()
	job := h.acme()
	r := h.run("job", "delete", fmt.Sprint(job.ID), "--json")
	want := fmt.Sprintf("{\n  \"deleted\": {\n    \"id\": %d,\n    \"company\": \"Acme\",\n    \"role\": \"Software Engineer\"\n  }\n}\n", job.ID)
	if r.stdout != want {
		t.Errorf("stdout = %q", r.stdout)
	}
}
