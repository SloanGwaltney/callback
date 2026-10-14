package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"callback/internal/db"
	"callback/internal/jobs"
)

// The messages for values that break the app's rules are the jobs service's, and are tested
// there: "role is required" (TestAddJobMissingRequired), "the interest rating must be from 1 to 5
// stars" (TestInterestRatingRange), and "the link must be a web address" (TestInvalidJobLink), all
// in internal/jobs. The fake hands them back as the service would.
var (
	errNeedsRole    = &jobs.ValidationError{Msg: "role is required"}
	errRatingRange  = &jobs.ValidationError{Msg: "the interest rating must be from 1 to 5 stars"}
	errNotWebAdress = &jobs.ValidationError{Msg: "the link must be a web address, like https://example.com/jobs/123"}
)

// @scenario [features/agent-cli.feature#cli-text-output] Output is plain text by default
func TestListIsPlainTextByDefault(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()

	// @when [features/agent-cli.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli.feature] the list is plain text, one line per job
	want := fmt.Sprintf("#%d | Saved | Acme | Software Engineer\n", job.ID)
	if r.stdout != want {
		t.Errorf("list = %q, want %q", r.stdout, want)
	}
}

// @scenario [features/agent-cli.feature#cli-json-output] My agent can ask for any command's output as JSON
func TestListAsJSON(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()
	job.Location, job.DateApplied = "Remote, US", "2026-10-05"
	h.jobs.days[job.ID] = 9
	text := h.run("job", "list").stdout

	// @when [features/agent-cli.feature] my agent lists my jobs as JSON
	r := h.run("job", "list", "--json")

	// @then [features/agent-cli.feature] the list is JSON with the same details as the plain text list, and more
	var listed []map[string]any
	if err := json.Unmarshal([]byte(r.stdout), &listed); err != nil || len(listed) != 1 {
		t.Fatalf("list isn't a JSON list of one job: %v\n%s", err, r.stdout)
	}
	got := listed[0]
	want := map[string]any{"id": float64(job.ID), "statusLabel": "Saved", "daysInStatus": float64(9), "company": "Acme",
		"role": "Software Engineer", "location": "Remote, US", "dateApplied": "2026-10-05"}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %v, want %v", key, got[key], value)
		}
		if !strings.Contains(text, strings.TrimSuffix(fmt.Sprint(value), ".0")) {
			t.Errorf("the plain text list doesn't have %v: %q", value, text)
		}
	}
	for _, more := range []string{"status", "documentCount", "nextInterview"} {
		if _, ok := got[more]; !ok {
			t.Errorf("the JSON doesn't have %q: %v", more, got)
		}
	}
}

func TestEveryCommandPrintsJSON(t *testing.T) {
	h := newHarness()
	job := h.acme()
	id := fmt.Sprint(job.ID)
	for _, args := range [][]string{
		{"job", "show", id},
		{"job", "add", "--company", "Globex", "--role", "Designer"},
		{"job", "update", id, "--salary", "$130k"},
		{"job", "move", id, "--status", "applied"},
		{"job", "set-next-action", id, "--text", "Follow up"},
		{"job", "clear-next-action", id},
		{"job", "close", id, "--reason", "rejected"},
		{"job", "delete", id},
	} {
		r := h.run(append(args, "--json")...)
		if r.code != ExitOK || !json.Valid([]byte(r.stdout)) {
			t.Errorf("%q printed %q, exit %d", args, r.stdout, r.code)
		}
	}
}

// @scenario [features/agent-cli.feature#cli-output-ids] Everything listed comes with its ID
func TestShowGivesEveryID(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme" with a contact, an interview, and a document
	job := h.acme()
	job.Contacts = []jobs.Contact{{ID: 7, JobID: job.ID, Name: "Jane Doe", Relationship: jobs.RelationshipRecruiter}}
	job.Interviews = []jobs.Interview{{ID: 8, JobID: job.ID, Date: "2026-10-20", Time: "14:00", Kind: jobs.InterviewTechnical}}
	job.Documents = []jobs.Document{{ID: 9, JobID: job.ID, Kind: jobs.KindResume, FileName: "resume.pdf"}}
	h.jobs.files[job.ID] = []jobs.DocumentFile{{DocumentID: 9, Path: "/data/documents/abc.pdf"}}

	// @when [features/agent-cli.feature] my agent shows the job
	r := h.run("job", "show", fmt.Sprint(job.ID))

	// @then [features/agent-cli.feature] the job, its contact, its interview, and its document each come with their ID
	for _, want := range []string{
		fmt.Sprintf("Job #%d: Software Engineer at Acme\n", job.ID),
		"  #7 Jane Doe | Recruiter\n",
		"  #8 2026-10-20 14:00 | Technical\n",
		"  #9 resume.pdf | Resume | stored copy: /data/documents/abc.pdf\n",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("show doesn't have %q:\n%s", want, r.stdout)
		}
	}
	var shown jobJSON
	if err := json.Unmarshal([]byte(h.run("job", "show", fmt.Sprint(job.ID), "--json").stdout), &shown); err != nil {
		t.Fatal(err)
	}
	if shown.ID != job.ID || shown.Contacts[0].ID != 7 || shown.Interviews[0].ID != 8 || shown.Documents[0].ID != 9 {
		t.Errorf("the JSON is missing an ID: %+v", shown)
	}
}

func TestShowMarksAMissingStoredCopy(t *testing.T) {
	h := newHarness()
	job := h.acme()
	job.Documents = []jobs.Document{{ID: 9, JobID: job.ID, Kind: jobs.KindCoverLetter, FileName: "letter.pdf"}}
	h.jobs.files[job.ID] = []jobs.DocumentFile{{DocumentID: 9, Path: "/data/documents/abc.pdf", Missing: true}}

	r := h.run("job", "show", fmt.Sprint(job.ID), "--json")

	var shown jobJSON
	if err := json.Unmarshal([]byte(r.stdout), &shown); err != nil {
		t.Fatal(err)
	}
	if d := shown.Documents[0]; !d.Missing || d.StoredPath != "/data/documents/abc.pdf" || d.FileName != "letter.pdf" {
		t.Errorf("document = %+v", d)
	}
	text := h.run("job", "show", fmt.Sprint(job.ID)).stdout
	if !strings.Contains(text, "| Cover letter | stored copy: /data/documents/abc.pdf (missing: the stored copy is gone)\n") {
		t.Errorf("show doesn't mark the missing copy:\n%s", text)
	}
}

// @scenario [features/agent-cli.feature#cli-dates] Dates are written as year, month, and day
func TestDatesAreYearMonthDay(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme" that I applied to on October 1, 2026
	job := h.acme()
	job.DateApplied = "2026-10-01"

	// @when [features/agent-cli.feature] my agent shows the job
	r := h.run("job", "show", fmt.Sprint(job.ID))

	// @then [features/agent-cli.feature] the date applied is written as 2026-10-01
	if !strings.Contains(r.stdout, "Date applied: 2026-10-01\n") {
		t.Errorf("show doesn't write the date applied as 2026-10-01:\n%s", r.stdout)
	}
}

// @scenario [features/agent-cli.feature#cli-change-prints-result] A change prints what it changed
func TestAChangePrintsTheResult(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme" in "Saved"
	job := h.acme()

	// @when [features/agent-cli.feature] my agent moves the job to "Applied"
	r := h.run("job", "move", fmt.Sprint(job.ID), "--status", "Applied")

	// @then [features/agent-cli.feature] my agent is shown the job as it is after the move
	r.wantCode(t, ExitOK)
	if want := h.run("job", "show", fmt.Sprint(job.ID)).stdout; r.stdout != want || !strings.Contains(r.stdout, "Status: Applied\n") {
		t.Errorf("move printed:\n%s\nwant the job in Applied:\n%s", r.stdout, want)
	}
}

// @scenario [features/agent-cli.feature#cli-error] A command that can't be done says why and changes nothing
func TestACommandThatCantBeDoneSaysWhy(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()
	h.jobs.fail = errNeedsRole

	// @when [features/agent-cli.feature] my agent tries to change the job's role to nothing
	r := h.run("job", "update", fmt.Sprint(job.ID), "--role", "")

	if h.jobs.updated.Role != "" {
		t.Errorf("the service was asked for the role %q, not nothing", h.jobs.updated.Role)
	}
	// @then [features/agent-cli.feature] the job's role is still "Software Engineer"
	if job.Role != "Software Engineer" {
		t.Errorf("role = %q", job.Role)
	}
	// @and [features/agent-cli.feature] my agent is told the job needs a role
	if r.stderr != "callback-cli: role is required\n" || r.stdout != "" {
		t.Errorf("stdout %q, stderr %q", r.stdout, r.stderr)
	}
	// @and [features/agent-cli.feature] the command ends with the exit code for a validation error
	r.wantCode(t, ExitValidation)
}

// @scenario [features/agent-cli.feature#cli-error-lists-choices] A value that isn't one of the choices is answered with the choices
func TestAValueThatIsntAChoiceGetsTheChoices(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()

	// @when [features/agent-cli.feature] my agent tries to move the job to "Intervewing"
	r := h.run("job", "move", fmt.Sprint(job.ID), "--status", "Intervewing")

	// @then [features/agent-cli.feature] the job doesn't move
	if len(h.jobs.moves) != 0 || job.Status != jobs.StatusSaved {
		t.Errorf("the job moved: %v", h.jobs.moves)
	}
	// @and [features/agent-cli.feature] my agent is told there's no status "Intervewing"
	if !strings.HasPrefix(r.stderr, "callback-cli: there's no status \"Intervewing\"\n") {
		t.Errorf("stderr = %q", r.stderr)
	}
	// @and [features/agent-cli.feature] my agent is told the statuses it can use
	if !strings.HasSuffix(r.stderr, "Use one of: saved, applied, interviewing, offer, closed\n") {
		t.Errorf("stderr = %q", r.stderr)
	}
	r.wantCode(t, ExitValidation)
}

// @scenario [features/agent-cli.feature#cli-unknown-command] An unknown command points my agent to the help
func TestAnUnknownCommandPointsToTheHelp(t *testing.T) {
	h := newHarness()

	// @when [features/agent-cli.feature] my agent runs a command Callback doesn't have
	r := h.run("job", "archive", "12")

	// @then [features/agent-cli.feature] my agent is told there's no such command
	if !strings.HasPrefix(r.stderr, `callback-cli: there's no command "job archive"`) {
		t.Errorf("stderr = %q", r.stderr)
	}
	// @and [features/agent-cli.feature] my agent is told how to see the help
	if !strings.Contains(r.stderr, "run `callback-cli --help` to see the commands") {
		t.Errorf("stderr = %q", r.stderr)
	}
	// @and [features/agent-cli.feature] the command ends with the exit code for a usage error
	r.wantCode(t, ExitUsage)
	if h.opened != 0 {
		t.Error("an unknown command opened the data")
	}
}

func TestAGroupAloneListsItsCommands(t *testing.T) {
	r := newHarness().run("job")
	r.wantCode(t, ExitUsage)
	if !strings.Contains(r.stderr, `"job" needs one of its commands: list, show, add, update, move, close, delete, set-next-action, clear-next-action`) {
		t.Errorf("stderr = %q", r.stderr)
	}
}

// @scenario [features/agent-cli.feature#cli-unknown-job] My agent is told when there's no job with an ID
func TestNoJobWithAnID(t *testing.T) {
	h := newHarness()

	// @when [features/agent-cli.feature] my agent tries to show the job with ID 999
	r := h.run("job", "show", "999")

	// @then [features/agent-cli.feature] my agent is told there's no job with ID 999
	if r.stderr != "callback-cli: there's no job with ID 999\n" {
		t.Errorf("stderr = %q", r.stderr)
	}
	// @and [features/agent-cli.feature] the command ends with the exit code for something not found
	r.wantCode(t, ExitNotFound)
}

func TestEveryJobCommandNamesAMissingJob(t *testing.T) {
	for _, args := range [][]string{
		{"job", "update", "999", "--salary", "x"},
		{"job", "move", "999", "--status", "applied"},
		{"job", "close", "999", "--reason", "rejected"},
		{"job", "delete", "999"},
		{"job", "set-next-action", "999", "--text", "Follow up"},
		{"job", "clear-next-action", "999"},
	} {
		r := newHarness().run(args...)
		if r.code != ExitNotFound || r.stderr != "callback-cli: there's no job with ID 999\n" {
			t.Errorf("%q: exit %d, stderr %q", args, r.code, r.stderr)
		}
	}
}

// @scenario [features/agent-cli.feature#cli-exit-codes] A failed command ends with the exit code for its kind of failure
func TestEachKindOfFailureHasItsOwnExitCode(t *testing.T) {
	tests := []struct {
		failure string
		kind    string
		code    int
		run     func(h *harness) result
	}{
		{"it isn't a command Callback has, or an option is wrong", "a usage error", ExitUsage, func(h *harness) result {
			if r := h.run("job", "move", "1", "--colour", "red"); r.code != ExitUsage {
				return r
			}
			return h.run("job", "archive")
		}},
		{"a value breaks one of the app's rules", "a validation error", ExitValidation, func(h *harness) result {
			h.jobs.fail = errNeedsRole
			return h.run("job", "update", fmt.Sprint(h.acme().ID), "--role", "")
		}},
		{"there's no job, contact, interview, document, or backup with that ID or name", "something not found", ExitNotFound, func(h *harness) result {
			return h.run("job", "show", "999")
		}},
		{"my data was last opened by a newer Callback", "a newer version of my data", ExitNewerData, func(h *harness) result {
			h.openErr = fmt.Errorf("%w (data at version 9, this version knows up to 8)", db.ErrNewerData)
			return h.run("job", "list")
		}},
		{"anything else went wrong, like my data can't be opened", "an unexpected error", ExitUnexpected, func(h *harness) result {
			h.openErr = errUnexpected
			return h.run("job", "list")
		}},
	}
	seen := map[int]string{}
	for _, tt := range tests {
		t.Run(tt.failure, func(t *testing.T) {
			// @when [features/agent-cli.feature] my agent runs a command that fails because <failure>
			r := tt.run(newHarness())

			// @then [features/agent-cli.feature] the command ends with the exit code for <kind>
			r.wantCode(t, tt.code)
			// @and [features/agent-cli.feature] no other kind of failure ends with that exit code
			if other, ok := seen[r.code]; ok {
				t.Errorf("%s and %s both end with %d", tt.kind, other, r.code)
			}
			seen[r.code] = tt.kind
		})
	}
}

// @scenario [features/agent-cli.feature#cli-validation-one-code] Every validation error shares one exit code, with a message saying what's wrong
func TestEveryValidationErrorSharesOneExitCode(t *testing.T) {
	tests := []struct {
		change  string
		args    []string
		err     error
		message string
	}{
		{"change the job's role to nothing", []string{"--role", ""}, errNeedsRole, "role is required"},
		{"rate the job 6 stars", []string{"--interest", "6"}, errRatingRange, "the interest rating must be from 1 to 5 stars"},
		{"set a posting link that isn't a web address", []string{"--link", "acme.example/jobs"}, errNotWebAdress, "the link must be a web address"},
	}
	for _, tt := range tests {
		t.Run(tt.change, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
			job := h.acme()
			h.jobs.fail = tt.err

			// @when [features/agent-cli.feature] my agent tries to <change>
			r := h.run(append([]string{"job", "update", fmt.Sprint(job.ID)}, tt.args...)...)

			// @then [features/agent-cli.feature] the command ends with the exit code for a validation error
			r.wantCode(t, ExitValidation)
			// @and [features/agent-cli.feature] my agent is told <message>
			if !strings.HasPrefix(r.stderr, "callback-cli: "+tt.message) {
				t.Errorf("stderr = %q, want %q", r.stderr, tt.message)
			}
		})
	}
}

// @scenario [features/agent-cli.feature#cli-success-code] A command that works ends with exit code 0
func TestACommandThatWorksEndsWithZero(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	h.acme()

	// @when [features/agent-cli.feature] my agent lists my jobs
	r := h.run("job", "list")

	// @then [features/agent-cli.feature] the command ends with exit code 0
	r.wantCode(t, 0)
	if r.stderr != "" {
		t.Errorf("stderr = %q", r.stderr)
	}
}

// @scenario [features/agent-cli.feature#cli-json-errors] An error is JSON when my agent asked for JSON output
func TestAnErrorIsJSONWhenAskedFor(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()
	h.jobs.fail = errNeedsRole

	// @when [features/agent-cli.feature] my agent tries to change the job's role to nothing, asking for JSON output
	r := h.run("job", "update", fmt.Sprint(job.ID), "--role", "", "--json")

	var got errorJSON
	// @then [features/agent-cli.feature] my agent is told the job needs a role, as JSON
	if err := json.Unmarshal([]byte(r.stderr), &got); err != nil || got.Error.Message != "role is required" {
		t.Errorf("stderr = %q (%v)", r.stderr, err)
	}
	// @and [features/agent-cli.feature] the JSON says what kind of failure it was
	if got.Error.Kind != "validation" {
		t.Errorf("kind = %q, want validation", got.Error.Kind)
	}
	// @and [features/agent-cli.feature] the command ends with the exit code for a validation error
	r.wantCode(t, ExitValidation)
	if got.Error.ExitCode != ExitValidation {
		t.Errorf("exitCode = %d", got.Error.ExitCode)
	}
}

func TestEveryKindOfErrorIsJSONWhenAskedFor(t *testing.T) {
	h := newHarness()
	h.acme()
	for _, args := range [][]string{
		{"--json", "job", "archive"},
		{"job", "move", "--json"},
		{"job", "show", "999", "--json"},
		{"job", "move", "101", "--status", "nope", "--json"},
	} {
		r := h.run(args...)
		var got errorJSON
		if err := json.Unmarshal([]byte(r.stderr), &got); err != nil || got.Error.ExitCode != r.code || got.Error.Kind != kindOf(r.code) {
			t.Errorf("%q: stderr %q, exit %d", args, r.stderr, r.code)
		}
	}
	var got errorJSON
	_ = json.Unmarshal([]byte(h.run("job", "move", "101", "--status", "nope", "--json").stderr), &got)
	if strings.Join(got.Error.Choices, ",") != "saved,applied,interviewing,offer,closed" {
		t.Errorf("choices = %v", got.Error.Choices)
	}
}

// @scenario [features/agent-cli.feature#cli-text-errors] An error is plain text otherwise
func TestAnErrorIsPlainTextOtherwise(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()
	h.jobs.fail = errNeedsRole

	// @when [features/agent-cli.feature] my agent tries to change the job's role to nothing
	r := h.run("job", "update", fmt.Sprint(job.ID), "--role", "")

	// @then [features/agent-cli.feature] my agent is told the job needs a role, as plain text
	if r.stderr != "callback-cli: role is required\n" || json.Valid([]byte(r.stderr)) {
		t.Errorf("stderr = %q", r.stderr)
	}
}

// @scenario [features/agent-cli.feature#cli-newer-data] The command line refuses data a newer Callback has updated
func TestRefusesNewerData(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] my data was last updated by a newer version of Callback than the command line
	// db.Open reports it, leaving the data unchanged (TestOpenRefusesDataFromANewerVersion in
	// internal/db).
	h.openErr = fmt.Errorf("%w (data at version 9, this version knows up to 8)", db.ErrNewerData)

	// @when [features/agent-cli.feature] my agent runs any command
	r := h.run("job", "list")

	// @and [features/agent-cli.feature] no daily backup is taken and no backup is removed
	// TakeDaily is what takes the day's backup and then removes the oldest ones, and RemoveExpired
	// removes expired agent backups.
	if h.backups.calls != 0 || h.backups.removals != 0 {
		t.Errorf("TakeDaily called %d times and RemoveExpired %d, want none", h.backups.calls, h.backups.removals)
	}
	// @and [features/agent-cli.feature] my agent is told my data is from a newer version of Callback, and to update Callback
	if r.stderr != "callback-cli: "+newerDataMessage+"\n" {
		t.Errorf("stderr = %q", r.stderr)
	}
	// @and [features/agent-cli.feature] the command ends with the exit code for a newer version of my data
	r.wantCode(t, ExitNewerData)
}

// @scenario [features/agent-cli.feature#cli-older-data] The command line brings older data up to date, as the app does
func TestRunsAsUsualOnceOlderDataIsUpToDate(t *testing.T) {
	h := newHarness()
	h.acme()

	// @when [features/agent-cli.feature] my agent runs any command
	// Opening the data brings it up to date first (TestOpenBringsOlderDataUpToDate in internal/db).
	r := h.run("job", "list")

	// @and [features/agent-cli.feature] the command runs as usual
	r.wantCode(t, ExitOK)
	if !strings.Contains(r.stdout, "| Acme | Software Engineer") || h.opened != 1 || h.closed != 1 {
		t.Errorf("stdout = %q, opened %d, closed %d", r.stdout, h.opened, h.closed)
	}
}

// @scenario [features/daily-backups.feature#backup-on-first-start] The first start of the day takes a backup
func TestACommandTakesTheDailyBackup(t *testing.T) {
	h := newHarness()

	// @when [features/daily-backups.feature] <start>
	// The "my agent runs a Callback command" row.
	h.run("job", "list")

	// Which day it's for, and whether one was already taken, is decided by the backups service
	// (TestFirstStartOfTheDayTakesABackup in internal/backups).
	if h.backups.calls != 1 {
		t.Errorf("TakeDaily called %d times, want 1", h.backups.calls)
	}
}

func TestAFailedBackupDoesntStopTheCommand(t *testing.T) {
	h := newHarness()
	h.backups.err = errUnexpected
	h.acme()

	r := h.run("job", "list")

	r.wantCode(t, ExitOK)
	if !strings.Contains(r.stdout, "Acme") || r.stderr != "callback-cli: warning: the daily backup failed: disk I/O error\n" {
		t.Errorf("stdout %q, stderr %q", r.stdout, r.stderr)
	}
}

func TestHelpAndUsageErrorsTakeNoBackup(t *testing.T) {
	h := newHarness()
	for _, args := range [][]string{{"--help"}, {"--overview"}, {"job", "show", "--help"}, {"job", "show"}, {"nope"}} {
		h.run(args...)
	}
	if h.opened != 0 || h.backups.calls != 0 {
		t.Errorf("opened %d times, took %d backups", h.opened, h.backups.calls)
	}
}

// @scenario [features/agent-cli.feature#cli-no-questions] Commands finish without waiting for an answer
func TestDeleteDoesntAskToConfirm(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()

	// @when [features/agent-cli.feature] my agent deletes the job
	r := h.run("job", "delete", fmt.Sprint(job.ID))

	// @then [features/agent-cli.feature] the job is deleted without my agent being asked to confirm
	r.wantCode(t, ExitOK)
	if _, ok := h.jobs.jobs[job.ID]; ok {
		t.Error("the job wasn't deleted")
	}
	if h.stdin.read {
		t.Error("delete read from stdin")
	}
	want := fmt.Sprintf("Deleted job #%d, Software Engineer at Acme, with its contacts, interviews, documents, and status history.\n", job.ID)
	if r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

// longText has quotes, line breaks, and blank lines.
const longText = "We're hiring a \"Software Engineer\".\r\n\nYou'll:\n  - build things\n\n\n'Remote' is fine.\n"

// @scenario [features/agent-cli.feature#cli-text-from-file] My agent gives long text from a file
func TestLongTextFromAFile(t *testing.T) {
	for _, field := range []string{"description", "notes"} {
		t.Run(field, func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
			job := h.acme()
			// @and [features/agent-cli.feature] a file holds a job description with quotes, line breaks, and blank lines
			h.files["job.txt"] = longText

			// @when [features/agent-cli.feature] my agent sets the job's <field> from that file
			r := h.run("job", "update", fmt.Sprint(job.ID), "--"+field+"-file", "job.txt")

			r.wantCode(t, ExitOK)
			got := map[string]string{"description": job.Description, "notes": job.Notes}[field]
			// @then [features/agent-cli.feature] the job's <field> is exactly what the file holds
			if got != longText {
				t.Errorf("%s = %q, want %q", field, got, longText)
			}
		})
	}
}

// @scenario [features/agent-cli.feature#cli-text-piped] My agent pipes long text in
func TestLongTextPipedIn(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli.feature] I have a job for "Software Engineer" at "Acme"
	job := h.acme()

	// @when [features/agent-cli.feature] my agent pipes a job description with quotes and line breaks into setting the job's description
	h.stdin.Reader = strings.NewReader(longText)
	r := h.run("job", "update", fmt.Sprint(job.ID), "--description-file", "-")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli.feature] the job's description is exactly what was piped in
	if job.Description != longText {
		t.Errorf("description = %q, want %q", job.Description, longText)
	}
}

func TestLongTextProblems(t *testing.T) {
	h := newHarness()
	job := h.acme()
	id := fmt.Sprint(job.ID)

	r := h.run("job", "update", id, "--notes", "short", "--notes-file", "notes.txt")
	r.wantCode(t, ExitUsage)

	r = h.run("job", "update", id, "--notes-file", "nowhere.txt")
	r.wantCode(t, ExitValidation)
	if r.stderr != "callback-cli: there's no file at nowhere.txt\n" {
		t.Errorf("stderr = %q", r.stderr)
	}
}
