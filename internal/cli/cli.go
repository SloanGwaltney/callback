// Package cli is Callback's command line for AI agents: users drive it from their own agent,
// outside the app, over the same data and services as the app. See
// docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md and
// docs/adr/0024-give-the-agent-cli-a-prose-overview-alongside-its-help.md. cmd/callback-cli wires
// it to the user's data.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"callback/internal/db"
	"callback/internal/jobs"
)

// Name is the command line's binary name, as the help and the overview write it.
const Name = "callback-cli"

// Jobs is what the command line does with jobs. *jobs.Service satisfies it.
type Jobs interface {
	Board(ctx context.Context) (jobs.Board, error)
	Job(ctx context.Context, id int64) (jobs.Job, error)
	JobDocumentFiles(ctx context.Context, jobID int64) ([]jobs.DocumentFile, error)
	AddJob(ctx context.Context, d jobs.Details, status jobs.Status) (jobs.Job, error)
	UpdateJob(ctx context.Context, id int64, d jobs.Details) (jobs.Job, error)
	MoveJob(ctx context.Context, m jobs.Move) error
	SetClosingReason(ctx context.Context, id int64, reason jobs.ClosingReason, other string) error
	DeleteJob(ctx context.Context, id int64) error
	SetNextAction(ctx context.Context, jobID int64, text, expectedDate string) (jobs.Job, error)
	ClearNextAction(ctx context.Context, jobID int64) (jobs.Job, error)

	Contact(ctx context.Context, id int64) (jobs.Contact, error)
	AddContact(ctx context.Context, jobID int64, d jobs.ContactDetails) (jobs.Contact, error)
	UpdateContact(ctx context.Context, id int64, d jobs.ContactDetails) (jobs.Contact, error)
	RemoveContact(ctx context.Context, id int64) error

	Interview(ctx context.Context, id int64) (jobs.Interview, error)
	AddInterview(ctx context.Context, jobID int64, d jobs.InterviewDetails) (jobs.Interview, error)
	UpdateInterview(ctx context.Context, id int64, d jobs.InterviewDetails) (jobs.Interview, error)
	RemoveInterview(ctx context.Context, id int64) error

	Document(ctx context.Context, id int64) (jobs.Document, error)
	AttachDocument(ctx context.Context, jobID int64, src string, kind jobs.DocumentKind) (jobs.Document, error)
	SetDocumentKind(ctx context.Context, id int64, kind jobs.DocumentKind) error
	RemoveDocument(ctx context.Context, id int64) error

	Library(ctx context.Context) ([]jobs.LibraryDocument, error)
	LibraryDocument(ctx context.Context, id int64) (jobs.LibraryDocument, error)
	LibraryFiles(ctx context.Context) ([]jobs.DocumentFile, error)
	AddLibraryDocument(ctx context.Context, src, description string) (jobs.LibraryDocument, error)
	SetLibraryDescription(ctx context.Context, id int64, description string) (jobs.LibraryDocument, error)
	RemoveLibraryDocument(ctx context.Context, id int64) error
}

// DailyBackups takes the daily backup. *backups.Service satisfies it.
type DailyBackups interface {
	TakeDaily(ctx context.Context) (bool, error)
}

// Services are the services a command runs over.
type Services struct {
	Jobs    Jobs
	Backups DailyBackups
}

// Env is everything Run reaches outside itself.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// ReadFile reads a file that holds long text, like a job's description.
	ReadFile func(name string) ([]byte, error)
	// Open opens the user's data, bringing data from an older version of Callback up to date, and
	// returns the services over it and a func that closes it. For data a newer version of
	// Callback has updated, it returns db.ErrNewerData and leaves the data unchanged.
	Open func(ctx context.Context) (Services, func() error, error)
}

// Exit codes. Each kind of failure has its own, listed in exitCodes.
const (
	ExitOK         = 0
	ExitUnexpected = 1
	ExitUsage      = 2
	ExitValidation = 3
	ExitNotFound   = 4
	ExitNewerData  = 5
)

// exitCode is an exit code and the kind of failure it means, for errors and for the overview.
type exitCode struct {
	Code    int
	Kind    string
	Meaning string
}

// exitCodes lists every exit code. The overview prints this list.
var exitCodes = []exitCode{
	{ExitOK, "", "the command worked"},
	{ExitUnexpected, "unexpected", "anything else went wrong, like the data can't be opened"},
	{ExitUsage, "usage", "there's no such command, or an option or argument is wrong or missing"},
	{ExitValidation, "validation", "a value breaks one of Callback's rules, like a job with no role; nothing was changed"},
	{ExitNotFound, "not_found", "there's nothing with that ID"},
	{ExitNewerData, "newer_data", "the data was last opened by a newer version of Callback; update Callback"},
}

// cliError is a failure with its exit code and, for a value that isn't one of a fixed set, the
// values that can be used.
type cliError struct {
	code    int
	msg     string
	choices []string
}

func (e *cliError) Error() string { return e.msg }

func usageErr(format string, args ...any) error {
	return &cliError{code: ExitUsage, msg: fmt.Sprintf(format, args...)}
}

func validationErr(format string, args ...any) error {
	return &cliError{code: ExitValidation, msg: fmt.Sprintf(format, args...)}
}

func notFoundErr(format string, args ...any) error {
	return &cliError{code: ExitNotFound, msg: fmt.Sprintf(format, args...)}
}

// newerDataMessage is what the agent is told about data from a newer version of Callback.
const newerDataMessage = "your data is from a newer version of Callback than this command line; update Callback, then run the command again"

// classify turns any error a command returns into a cliError with its exit code.
func classify(err error) *cliError {
	var ce *cliError
	var invalid *jobs.ValidationError
	var missing *jobs.MissingFileError
	switch {
	case errors.As(err, &ce):
		return ce
	case errors.As(err, &invalid), errors.As(err, &missing):
		return &cliError{code: ExitValidation, msg: err.Error()}
	case errors.Is(err, jobs.ErrJobNotFound), errors.Is(err, jobs.ErrDocumentNotFound),
		errors.Is(err, jobs.ErrContactNotFound), errors.Is(err, jobs.ErrInterviewNotFound),
		errors.Is(err, jobs.ErrLibraryDocumentNotFound):
		return &cliError{code: ExitNotFound, msg: err.Error()}
	case errors.Is(err, db.ErrNewerData):
		return &cliError{code: ExitNewerData, msg: newerDataMessage}
	default:
		return &cliError{code: ExitUnexpected, msg: err.Error()}
	}
}

// kindOf returns the kind of failure an exit code means.
func kindOf(code int) string {
	for _, c := range exitCodes {
		if c.Code == code {
			return c.Kind
		}
	}
	return "unexpected"
}

// Run runs the command line with args, the arguments after the binary's name, and returns the
// exit code. It never asks a question: everything a command needs comes from its arguments, a
// file, or what's piped in. Help, the overview, and usage errors don't open the data; every other
// command opens it, takes the day's backup if none has been taken yet, then runs.
// @scenario [features/agent-cli.feature#cli-error] A command that can't be done says why and changes nothing
// @scenario [features/agent-cli.feature#cli-unknown-command] An unknown command points my agent to the help
// @scenario [features/agent-cli.feature#cli-exit-codes] A failed command ends with the exit code for its kind of failure
// @scenario [features/agent-cli.feature#cli-validation-one-code] Every validation error shares one exit code, with a message saying what's wrong
// @scenario [features/agent-cli.feature#cli-success-code] A command that works ends with exit code 0
// @scenario [features/agent-cli.feature#cli-json-errors] An error is JSON when my agent asked for JSON output
// @scenario [features/agent-cli.feature#cli-text-errors] An error is plain text otherwise
// @scenario [features/agent-cli.feature#cli-newer-data] The command line refuses data a newer Callback has updated
// @scenario [features/agent-cli.feature#cli-older-data] The command line brings older data up to date, as the app does
// @scenario [features/agent-cli.feature#cli-no-questions] Commands finish without waiting for an answer
// @scenario [features/daily-backups.feature#backup-on-first-start] The first start of the day takes a backup
func Run(ctx context.Context, args []string, env Env) int {
	asJSON := slices.Contains(args, "--json") || slices.Contains(args, "-json")
	if err := run(ctx, args, env); err != nil {
		return report(env.Stderr, classify(err), asJSON)
	}
	return ExitOK
}

func run(ctx context.Context, args []string, env Env) error {
	// Options before the command: --json, --help, and --overview.
	var help bool
	i := 0
	for ; i < len(args) && strings.HasPrefix(args[i], "-"); i++ {
		switch strings.TrimLeft(args[i], "-") {
		case "json":
		case "help", "h":
			help = true
		case "overview":
			return writeOverview(env.Stdout)
		default:
			return usageErr("there's no option %s; run `%s --help` to see the commands and their options", args[i], Name)
		}
	}
	words := args[i:]
	if len(words) == 0 {
		return writeHelp(env.Stdout)
	}
	if words[0] == "help" {
		if len(words) == 1 {
			return writeHelp(env.Stdout)
		}
		cmd, _, err := findCommand(words[1:])
		if err != nil {
			return err
		}
		return writeCommandHelp(env.Stdout, cmd)
	}

	cmd, rest, err := findCommand(words)
	if err != nil {
		return err
	}
	if help {
		return writeCommandHelp(env.Stdout, cmd)
	}
	c, err := parseCall(cmd, rest)
	if errors.Is(err, flag.ErrHelp) {
		return writeCommandHelp(env.Stdout, cmd)
	}
	if err != nil {
		return err
	}

	services, closeData, err := env.Open(ctx)
	if err != nil {
		return err
	}
	defer closeData()
	// A failed backup never stops the command, as it never stops the app from starting.
	if _, err := services.Backups.TakeDaily(ctx); err != nil {
		fmt.Fprintf(env.Stderr, "%s: warning: the daily backup failed: %v\n", Name, err)
	}

	c.ctx, c.env, c.jobs = ctx, env, services.Jobs
	return cmd.run(c)
}

// errorJSON is how an error is written as JSON.
type errorJSON struct {
	Error struct {
		Kind     string   `json:"kind"`
		ExitCode int      `json:"exitCode"`
		Message  string   `json:"message"`
		Choices  []string `json:"choices,omitempty"`
	} `json:"error"`
}

// report writes err to w, as JSON or as plain text, and returns its exit code.
func report(w io.Writer, err *cliError, asJSON bool) int {
	if asJSON {
		var out errorJSON
		out.Error.Kind = kindOf(err.code)
		out.Error.ExitCode = err.code
		out.Error.Message = err.msg
		out.Error.Choices = err.choices
		writeJSON(w, out)
		return err.code
	}
	fmt.Fprintf(w, "%s: %s\n", Name, err.msg)
	if len(err.choices) > 0 {
		fmt.Fprintf(w, "Use one of: %s\n", strings.Join(err.choices, ", "))
	}
	return err.code
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
