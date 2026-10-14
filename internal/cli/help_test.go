package cli

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"callback/internal/jobs"
)

// @scenario [features/agent-cli.feature#cli-help] The help lists every command
func TestHelpListsEveryCommand(t *testing.T) {
	h := newHarness()

	// @when [features/agent-cli.feature] my agent asks Callback for help
	r := h.run("--help")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli.feature] it sees every command, each with a short description of what it does
	for _, cmd := range commands {
		if !regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(cmd.name()) + ` +` + regexp.QuoteMeta(cmd.summary) + `$`).MatchString(r.stdout) {
			t.Errorf("the help doesn't list %q with %q:\n%s", cmd.name(), cmd.summary, r.stdout)
		}
	}
	if h.opened != 0 {
		t.Error("the help opened the data")
	}
}

func TestEveryWayOfAskingForHelpGivesIt(t *testing.T) {
	want := newHarness().run("--help").stdout
	for _, args := range [][]string{{}, {"-h"}, {"help"}} {
		if got := newHarness().run(args...); got.stdout != want || got.code != ExitOK {
			t.Errorf("%q printed %q, exit %d", args, got.stdout, got.code)
		}
	}
}

// @scenario [features/agent-cli.feature#cli-help-points-to-overview] The help points my agent to the overview
func TestHelpPointsToTheOverview(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks Callback for help
	r := newHarness().run("--help")

	// @then [features/agent-cli.feature] it's told how to read the overview
	if !strings.Contains(r.stdout, "Read `callback-cli --overview`") {
		t.Errorf("the help doesn't point to the overview:\n%s", r.stdout)
	}
}

// @scenario [features/agent-cli.feature#cli-command-help] The help for a command lists its options and the values they take
func TestCommandHelpListsItsOptionsAndValues(t *testing.T) {
	h := newHarness()

	// @when [features/agent-cli.feature] my agent asks for help on moving a job
	r := h.run("job", "move", "--help")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli.feature] it sees every option for moving a job
	move, _, _ := findCommand([]string{"job", "move"})
	for _, o := range append(move.options, jsonOption) {
		if !strings.Contains(r.stdout, "--"+o.name) {
			t.Errorf("the help on moving a job doesn't list --%s:\n%s", o.name, r.stdout)
		}
	}
	// @and [features/agent-cli.feature] it sees every status a job can be moved to
	for _, status := range jobs.Statuses {
		if !strings.Contains(r.stdout, string(status)+" ("+status.Label()+")") {
			t.Errorf("the help on moving a job doesn't list %q:\n%s", status, r.stdout)
		}
	}
	if h.opened != 0 {
		t.Error("a command's help opened the data")
	}
}

func TestEveryWayOfAskingForACommandsHelpGivesIt(t *testing.T) {
	want := newHarness().run("job", "move", "--help").stdout
	for _, args := range [][]string{{"job", "move", "-h"}, {"help", "job", "move"}, {"--help", "job", "move"}, {"job", "move", "12", "--help"}} {
		if got := newHarness().run(args...); got.stdout != want || got.code != ExitOK {
			t.Errorf("%q printed %q, exit %d", args, got.stdout, got.code)
		}
	}
}

func TestEveryCommandHasHelp(t *testing.T) {
	for _, cmd := range commands {
		r := newHarness().run(append(cmd.path, "--help")...)
		if r.code != ExitOK || !strings.Contains(r.stdout, "Usage:\n  callback-cli "+cmd.name()) || cmd.summary == "" || cmd.about == "" {
			t.Errorf("%s has no help: %q", cmd.name(), r.stdout)
		}
	}
}

// @scenario [features/agent-cli.feature#cli-help-lists-choices] The help lists the values for every option that takes a fixed set
func TestHelpListsTheChoices(t *testing.T) {
	// described lists choices as the help writes them, "value (Label)".
	described := func(values []string, label func(string) string) []string {
		var out []string
		for _, v := range values {
			out = append(out, v+" ("+label(v)+")")
		}
		return out
	}
	for _, tc := range []struct {
		command string
		args    []string
		choice  string
		want    []string
	}{
		{"moving a job", []string{"job", "move"}, "status",
			described(values(jobs.Statuses), func(s string) string { return jobs.Status(s).Label() })},
		{"closing a job", []string{"job", "close"}, "closing reason",
			described(values(jobs.ClosingReasons), func(s string) string { return jobs.ClosingReason(s).Label() })},
		{"adding a contact", []string{"contact", "add"}, "relationship",
			described(values(jobs.Relationships), func(s string) string { return jobs.Relationship(s).Label() })},
		{"adding an interview", []string{"interview", "add"}, "interview kind",
			described(values(jobs.InterviewKinds), func(s string) string { return jobs.InterviewKind(s).Label() })},
		{"attaching a document", []string{"document", "attach"}, "document kind",
			described(values(jobs.DocumentKinds), func(s string) string { return jobs.DocumentKind(s).Label() })},
		{"attaching a document", []string{"document", "attach"}, "file type it can attach", jobs.AllowedDocumentTypes},
	} {
		t.Run(tc.command+" lists every "+tc.choice, func(t *testing.T) {
			// @when [features/agent-cli.feature] my agent asks for help on <command>
			r := newHarness().run(append(tc.args, "--help")...)

			r.wantCode(t, ExitOK)
			// @then [features/agent-cli.feature] it sees every <choice> it can use
			for _, want := range tc.want {
				if !strings.Contains(r.stdout, want) {
					t.Errorf("the help on %s doesn't list %q:\n%s", tc.command, want, r.stdout)
				}
			}
		})
	}
}

func TestAddingAJobDoesntOfferClosed(t *testing.T) {
	r := newHarness().run("job", "add", "--help")
	if strings.Contains(r.stdout, "closed (Closed)") {
		t.Errorf("the help on adding a job offers Closed:\n%s", r.stdout)
	}
}

// @scenario [features/agent-cli.feature#cli-help-stored-files] The help says how to use a document's stored copy
func TestShowHelpSaysHowToUseStoredCopies(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks for help on showing a job
	r := newHarness().run("job", "show", "--help")

	help := strings.Join(strings.Fields(r.stdout), " ")
	// @then [features/agent-cli.feature] it's told it can read the stored copies of the job's documents
	if !strings.Contains(help, "the path of its stored copy, which you can read") {
		t.Errorf("the help on showing a job doesn't say the stored copies can be read:\n%s", r.stdout)
	}
	// @and [features/agent-cli.feature] it's told to change a document by attaching a new file, not by editing the stored copy
	if !strings.Contains(help, "Never edit a stored copy: to change a job's document, attach a new file in its place.") {
		t.Errorf("the help on showing a job doesn't say how to change a document:\n%s", r.stdout)
	}
}

// overview runs --overview and returns it with its lines joined, so phrases can be found across
// line breaks.
func overview(t *testing.T) string {
	t.Helper()
	h := newHarness()
	r := h.run("--overview")
	r.wantCode(t, ExitOK)
	if h.opened != 0 {
		t.Error("the overview opened the data")
	}
	return strings.Join(strings.Fields(r.stdout), " ")
}

// wantPhrases fails the test for each phrase text doesn't have.
func wantPhrases(t *testing.T, text string, phrases ...string) {
	t.Helper()
	for _, p := range phrases {
		if !strings.Contains(text, p) {
			t.Errorf("the overview doesn't say %q", p)
		}
	}
}

// @scenario [features/agent-cli.feature#cli-overview] My agent can read an overview of Callback and its commands
func TestOverview(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks Callback for its overview
	text := overview(t)

	// @then [features/agent-cli.feature] it reads, in prose, what Callback keeps track of: jobs and their statuses, contacts, interviews, documents, next actions, status history, and the document library
	wantPhrases(t, text, "## What Callback keeps track of", "**Jobs.**", "**Statuses.**", "**Contacts.**",
		"**Interviews.**", "**Documents.**", "**Next actions.**", "**Status history.**", "**The document library.**")
	for _, status := range jobs.Statuses {
		wantPhrases(t, text, status.Label())
	}
	for _, reason := range jobs.ClosingReasons {
		wantPhrases(t, text, reason.Label())
	}
	// @and [features/agent-cli.feature] it reads how the commands fit together, from finding a job by its ID to changing it
	wantPhrases(t, text, "## How the commands fit together", "Find the job. `callback-cli job list`",
		"Read the job. `callback-cli job show <job-id>`", "Change it, by its ID")
	// @and [features/agent-cli.feature] it reads how output, IDs, dates, long text, and errors work
	wantPhrases(t, text, "## Output", "--json", "## IDs", "`#12`", "## Dates", "2026-10-14", "## Long text",
		"--description-file job.txt", "## Errors", "standard error")
}

// @scenario [features/agent-cli.feature#cli-overview-limits] The overview says what my agent can't do
func TestOverviewSaysWhatTheAgentCantDo(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks Callback for its overview
	text := overview(t)

	// @then [features/agent-cli.feature] it's told it can't open anything on my screen, delete backups, or restore a daily backup
	wantPhrases(t, text, "You can't open anything on the person's screen",
		"You can't delete backups, and you can't restore a daily backup.")
	// @and [features/agent-cli.feature] it's told to change a job's document by attaching a new file, not by editing the stored copy
	wantPhrases(t, text, "Never edit a document's stored copy. To change a job's document, attach a new file in its place.")
}

// @scenario [features/agent-cli.feature#cli-overview-read-documents] The overview tells my agent it can read my documents
func TestOverviewSaysTheAgentCanReadDocuments(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks Callback for its overview
	text := overview(t)

	// @then [features/agent-cli.feature] it's told where to find the stored copies of a job's documents and the library's
	wantPhrases(t, text, "`job show` gives the path of the stored copy of each of a job's documents",
		"`library list` gives the path of the stored copy of each document in the library")
	if strings.Contains(text, "can't reach the document library") {
		t.Error("the overview still says the library can't be reached")
	}
	// @and [features/agent-cli.feature] it's told to read them when they help with what it's doing
	wantPhrases(t, text, "Read them when they help with what you're doing")
}

// @scenario [features/agent-cli.feature#cli-overview-working-safely] The overview tells my agent how to work safely
func TestOverviewSaysHowToWorkSafely(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks Callback for its overview
	text := overview(t)

	// @then [features/agent-cli.feature] it's told to make a backup before changing many jobs at once, and how long to keep it
	wantPhrases(t, text, "Before you change many jobs at once, back up the person's data with",
		"`callback-cli backup make <backup-name> --days <days>`", "7 days suits most changes, and it can be kept from 1 to 90")
	// @and [features/agent-cli.feature] it's told how to restore that backup if the changes go wrong
	wantPhrases(t, text, "If the changes go wrong, `callback-cli backup restore <backup-name>`",
		"restoring that one undoes the restore")
}

// @scenario [features/agent-cli.feature#cli-overview-matches-commands] Every command the overview names is one Callback has
func TestEveryCommandTheOverviewNamesIsOne(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks Callback for its overview
	text := overview(t)

	named := regexp.MustCompile("`callback-cli ([^`]*)`").FindAllStringSubmatch(text, -1)
	if len(named) < len(commands) {
		t.Fatalf("the overview names only %d commands", len(named))
	}
	// @then [features/agent-cli.feature] every command it names is a command Callback has
	for _, m := range named {
		words := strings.Fields(m[1])
		if words[0] == "--help" || words[0] == "--overview" || words[0] == "<command>" {
			continue
		}
		if _, _, err := findCommand(words); err != nil {
			t.Errorf("the overview names `callback-cli %s`, which isn't a command: %v", m[1], err)
		}
	}
}

// @scenario [features/agent-cli.feature#cli-exit-codes-listed] The overview lists every exit code and what it means
func TestOverviewListsEveryExitCode(t *testing.T) {
	// @when [features/agent-cli.feature] my agent asks Callback for its overview
	text := overview(t)

	// @then [features/agent-cli.feature] it sees every exit code and the kind of failure each one means
	for _, c := range exitCodes {
		line := "**" + strconv.Itoa(c.Code) + "**"
		if c.Kind != "" {
			line += " (`" + c.Kind + "`)"
		}
		wantPhrases(t, text, line+": "+c.Meaning)
	}
	codes := []int{ExitOK, ExitUnexpected, ExitUsage, ExitValidation, ExitNotFound, ExitNewerData}
	for _, code := range codes {
		if !slices.ContainsFunc(exitCodes, func(c exitCode) bool { return c.Code == code }) {
			t.Errorf("exit code %d isn't in exitCodes", code)
		}
	}
}
