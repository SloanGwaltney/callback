package cli

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"

	"callback/internal/jobs"
)

// commands is every command, in the order the help lists them. None opens anything on the user's
// screen; `job show` gives the links and stored copies to read instead.
// @scenario [features/agent-cli-jobs.feature#cli-no-opening] My agent can't open anything on my screen
var commands = slices.Concat(jobCommands, recordCommands)

// jobCommands are the commands for jobs themselves.
var jobCommands = []*command{
	{
		path:    []string{"job", "list"},
		summary: "List your jobs, one line each, in board order",
		about: "Lists your jobs column by column, Saved to Offer, in each column's order, one line per job " +
			"with its ID, status, days in that status, company, role, location, date applied, interest " +
			"rating, and next action. A next action that's due today or overdue is marked. Closed jobs " +
			"are left out unless you give --all, or closed with --status. Descriptions and notes aren't " +
			"listed; `job show` has them.\n\n" +
			"The other options narrow the list, and every one you give applies.",
		options: []option{
			{name: "all", usage: "list closed jobs too, after the others"},
			{name: "status", value: "<statuses>", usage: "list only the jobs in these statuses, separated by commas, like applied,interviewing", choices: described(jobs.Statuses, jobs.Status.Label)},
			{name: "search", value: "<text>", usage: "list only the jobs whose role or company contains this text, ignoring case, as the board's search does"},
			{name: "overdue", usage: "list only the jobs whose next action is overdue; one due today isn't overdue yet"},
			{name: "days-in-status", value: "<days>", usage: "list only the jobs that have been in their status for at least this many days"},
			{name: "min-interest", value: "<stars>", usage: "list only the jobs rated at least this many stars; jobs with no rating are left out"},
		},
		run: listJobs,
	},
	{
		path:    []string{"job", "show"},
		takesID: "job",
		summary: "Show everything about one job",
		about: "Shows every detail of a job, including its description, notes, and posting link, with its " +
			"contacts, interviews, next action, status history, and attached documents, each with its ID.\n\n" +
			"Each attached document is listed with the path of its stored copy, which you can read " +
			"whenever the document helps with what you're doing. Never edit a stored copy: to change a " +
			"job's document, attach a new file in its place.",
		run: showJob,
	},
	{
		path:    []string{"job", "add"},
		summary: "Add a job, at the top of Saved or of the status you give",
		about:   "Adds a job at the top of its column and shows it with its new ID. It needs a company and a role.",
		options: append([]option{
			{name: "status", value: "<status>", usage: "the column to add it to; Saved if not given. A job can't be added straight to Closed", choices: described(addStatuses, jobs.Status.Label)},
		}, detailOptions...),
		run: addJob,
	},
	{
		path:    []string{"job", "update"},
		takesID: "job",
		summary: "Change some of a job's details and keep the rest",
		about: "Changes only the details you give and keeps the rest. Give an option an empty value, " +
			"like --salary \"\", to clear it. Shows the job as it is afterwards.",
		options: detailOptions,
		run:     updateJob,
	},
	{
		path:    []string{"job", "move"},
		takesID: "job",
		summary: "Move a job to another status, or to another place in its column",
		about: "Moves a job to the top of a status column, or to the bottom or a place you give, following " +
			"the app's rules: moving to Applied fills in today as the date applied if none is set, and " +
			"moving back to Saved clears it. A move to another status is added to the job's status " +
			"history. To move a job to Closed, close it with `job close` instead, which takes a reason. " +
			"Shows the job as it is afterwards.",
		options: []option{
			{name: "status", value: "<status>", usage: "the status to move it to", choices: described(jobs.Statuses, jobs.Status.Label)},
			{name: "bottom", usage: "put it at the bottom of the column instead of the top"},
			{name: "position", value: "<place>", usage: "put it at this place in the column, counting from 1 at the top; past the bottom puts it at the bottom"},
		},
		run: moveJob,
	},
	{
		path:    []string{"job", "close"},
		takesID: "job",
		summary: "Close a job with a reason, or change a closed job's reason",
		about: "Moves a job to the top of Closed with the reason it was closed. For a job that's already " +
			"closed, changes its reason and leaves it where it is. Shows the job as it is afterwards.",
		options: []option{
			{name: "reason", value: "<reason>", usage: "why the job was closed", choices: described(jobs.ClosingReasons, jobs.ClosingReason.Label)},
			{name: "other", value: "<text>", usage: "the reason in your own words, needed with --reason other"},
		},
		run: closeJob,
	},
	{
		path:    []string{"job", "delete"},
		takesID: "job",
		summary: "Delete a job with its contacts, interviews, documents, and status history",
		about: "Deletes a job at once, without asking to confirm, along with its contacts, interviews, " +
			"attached documents and their stored copies, and status history. The document library " +
			"isn't touched. This can't be undone.",
		run: deleteJob,
	},
	{
		path:    []string{"job", "set-next-action"},
		takesID: "job",
		summary: "Set what's next on a job, and when it's expected",
		about:   "Sets a job's next action, replacing any it had. Shows the job as it is afterwards.",
		options: []option{
			{name: "text", value: "<text>", usage: "what the next action is, like \"Follow up with the recruiter\""},
			{name: "date", value: "<date>", usage: "when it's expected, like 2026-10-20; no expected date if not given"},
		},
		run: setNextAction,
	},
	{
		path:    []string{"job", "clear-next-action"},
		takesID: "job",
		summary: "Clear a job's next action, once it's done, or only its expected date",
		about:   "Removes a job's next action, or with --only-date, only its expected date. Shows the job as it is afterwards.",
		options: []option{
			{name: "only-date", usage: "clear only the expected date and keep the next action"},
		},
		run: clearNextAction,
	},
}

// addStatuses are the statuses a job can be added to: every one but Closed.
var addStatuses = jobs.Statuses[:len(jobs.Statuses)-1]

// detailOptions are a job's details, for adding and changing a job.
var detailOptions = []option{
	{name: "company", value: "<text>", usage: "the company"},
	{name: "role", value: "<text>", usage: "the role"},
	{name: "location", value: "<text>", usage: "where the job is, like \"Remote, US\""},
	{name: "salary", value: "<text>", usage: "the salary, written however the posting gives it, like \"$120k-$140k\""},
	{name: "link", value: "<url>", usage: "the posting's web address, starting with http:// or https://"},
	{name: "date-applied", value: "<date>", usage: "the date you applied, like 2026-10-01; it can't be in the future"},
	{name: "interest", value: "<stars>", usage: fmt.Sprintf("how interested you are, from %d to %d stars", jobs.MinInterest, jobs.MaxInterest)},
	{name: "description", value: "<text>", usage: "the job description"},
	{name: "description-file", value: "<path>", usage: "read the description from a file, or from what's piped in with -"},
	{name: "notes", value: "<text>", usage: "your notes on the job"},
	{name: "notes-file", value: "<path>", usage: "read the notes from a file, or from what's piped in with -"},
}

// jobErr names the job's ID when there's no job with it.
func jobErr(id int64, err error) error {
	return missingErr("job", id, err, jobs.ErrJobNotFound)
}

// @scenario [features/agent-cli.feature#cli-text-output] Output is plain text by default
// @scenario [features/agent-cli.feature#cli-json-output] My agent can ask for any command's output as JSON
// @scenario [features/agent-cli-jobs.feature#cli-list-jobs] List my jobs
// @scenario [features/agent-cli-jobs.feature#cli-list-board-order] Jobs are listed column by column, in each column's order
// @scenario [features/agent-cli-jobs.feature#cli-list-leaves-out-long-text] The list leaves out each job's description and notes
// @scenario [features/agent-cli-jobs.feature#cli-list-overdue-marked] The list marks a next action that's due or overdue
// @scenario [features/agent-cli-jobs.feature#cli-list-empty] My agent is told when no jobs are listed
// @scenario [features/agent-cli-jobs.feature#cli-list-leaves-out-closed] The list leaves out closed jobs
// @scenario [features/agent-cli-jobs.feature#cli-list-all] My agent asks for every job, closed ones too
// @scenario [features/agent-cli-jobs.feature#cli-filter-status] List only the jobs in some statuses
// @scenario [features/agent-cli-jobs.feature#cli-filter-closed-status] Asking for Closed lists closed jobs
// @scenario [features/agent-cli-jobs.feature#cli-filter-search] Search the list the way the board's search does
// @scenario [features/agent-cli-jobs.feature#cli-filter-overdue] List only the jobs whose next action is overdue
// @scenario [features/agent-cli-jobs.feature#cli-filter-stale] List only the jobs that have been in their status for some days
// @scenario [features/agent-cli-jobs.feature#cli-filter-interest] List only the jobs rated at least some stars
// @scenario [features/agent-cli-jobs.feature#cli-filter-combined] Every filter my agent gives applies
// @scenario [features/agent-cli-jobs.feature#cli-filter-no-match] My agent is told when no jobs match
func listJobs(c *call) error {
	f, err := listFilterFrom(c)
	if err != nil {
		return err
	}
	board, err := c.jobs.Board(c.ctx)
	if err != nil {
		return err
	}
	listed := []cardJSON{}
	// closedLeftOut counts the closed jobs left out for want of --all.
	closedLeftOut := 0
	for _, col := range board.Columns {
		if !f.lists(col.Status) {
			if col.Status == jobs.StatusClosed && f.statuses == nil {
				closedLeftOut = col.Count
			}
			continue
		}
		for _, card := range col.Cards {
			if f.keeps(card) {
				listed = append(listed, cardJSON{Card: card, Status: col.Status, StatusLabel: col.Label})
			}
		}
	}
	return c.output(listed, func(w io.Writer) {
		switch {
		case len(listed) > 0:
		case f.narrows() && closedLeftOut > 0:
			fmt.Fprintln(w, "No open jobs match. Add --all to look through the closed ones too.")
		case f.narrows():
			fmt.Fprintln(w, "No jobs match.")
		case closedLeftOut > 0:
			fmt.Fprintf(w, "No open jobs to list. Add --all to list the %d closed ones.\n", closedLeftOut)
		default:
			fmt.Fprintln(w, "No jobs to list.")
		}
		for _, card := range listed {
			writeCardLine(w, card)
		}
	})
}

// listFilter is what `job list` was asked to list.
type listFilter struct {
	all bool
	// statuses are the statuses to list, or nil for every one but Closed, and Closed too with all.
	statuses []jobs.Status
	// search is the text to look for in a job's role or company, or "" for every job.
	search      string
	overdue     bool
	minDays     *int
	minInterest *int
}

func listFilterFrom(c *call) (listFilter, error) {
	f := listFilter{all: c.flag("all"), overdue: c.flag("overdue")}
	if given, ok := c.value("status"); ok {
		for name := range strings.SplitSeq(given, ",") {
			status, err := pick("status", name, jobs.Statuses, jobs.Status.Label)
			if err != nil {
				return f, err
			}
			f.statuses = append(f.statuses, status)
		}
	}
	// As on the board, a search of only spaces finds every job.
	if given, ok := c.value("search"); ok {
		f.search = strings.ToLower(strings.TrimSpace(given))
	}
	var err error
	if f.minDays, err = c.count("days-in-status", "days"); err != nil {
		return f, err
	}
	if f.minInterest, err = c.count("min-interest", "stars"); err != nil {
		return f, err
	}
	return f, nil
}

// count reads an option that takes a whole number of something, like days, or returns nil when
// it isn't given.
func (c *call) count(name, unit string) (*int, error) {
	given, ok := c.value(name)
	if !ok {
		return nil, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(given))
	if err != nil || n < 0 {
		return nil, usageErr("--%s takes a whole number of %s, like 4, not %q", name, unit, given)
	}
	return &n, nil
}

// lists reports whether jobs in status are listed.
func (f listFilter) lists(status jobs.Status) bool {
	if f.statuses != nil {
		return slices.Contains(f.statuses, status)
	}
	return status != jobs.StatusClosed || f.all
}

// narrows reports whether anything but the statuses narrows the list.
func (f listFilter) narrows() bool {
	return f.search != "" || f.overdue || f.minDays != nil || f.minInterest != nil
}

// keeps reports whether a card passes every filter given. The search matches the role or the
// company, ignoring case, as the board's search does (matches in frontend/src/board/boardView.ts).
// Whether a next action is overdue, and the days in status, are the jobs service's.
func (f listFilter) keeps(card jobs.Card) bool {
	switch {
	case f.search != "" && !strings.Contains(strings.ToLower(card.Role), f.search) && !strings.Contains(strings.ToLower(card.Company), f.search):
		return false
	case f.overdue && (card.NextAction == nil || card.NextAction.Due != jobs.DueOverdue):
		return false
	case f.minDays != nil && (card.DaysInStatus == nil || *card.DaysInStatus < *f.minDays):
		return false
	case f.minInterest != nil && (card.Interest == nil || *card.Interest < *f.minInterest):
		return false
	}
	return true
}

// @scenario [features/agent-cli.feature#cli-output-ids] Everything listed comes with its ID
// @scenario [features/agent-cli.feature#cli-dates] Dates are written as year, month, and day
// @scenario [features/agent-cli.feature#cli-unknown-job] My agent is told when there's no job with an ID
// @scenario [features/agent-cli-jobs.feature#cli-show-job] Show a job's full details
// @scenario [features/agent-cli-jobs.feature#cli-show-missing-document] Showing a job marks a document whose file is missing
// @scenario [features/agent-cli-jobs.feature#cli-no-opening] My agent can't open anything on my screen
func showJob(c *call) error {
	return c.printJob(c.id)
}

// printJob shows a job as `job show` does.
func (c *call) printJob(id int64) error {
	job, err := c.jobs.Job(c.ctx, id)
	if err != nil {
		return jobErr(id, err)
	}
	files, err := c.jobs.JobDocumentFiles(c.ctx, id)
	if err != nil {
		return jobErr(id, err)
	}
	out := jobJSONFrom(job, files)
	return c.output(out, func(w io.Writer) { writeJob(w, out) })
}

// @scenario [features/agent-cli-jobs.feature#cli-add-job] Add a job with its details
// @scenario [features/agent-cli-jobs.feature#cli-add-job-status] Add a job straight to a status
// @scenario [features/agent-cli-jobs.feature#cli-add-job-rules] Adding a job follows the app's rules
func addJob(c *call) error {
	var status jobs.Status
	if given, ok := c.value("status"); ok {
		var err error
		if status, err = pick("status", given, jobs.Statuses, jobs.Status.Label); err != nil {
			return err
		}
	}
	var d jobs.Details
	if err := c.applyDetails(&d); err != nil {
		return err
	}
	job, err := c.jobs.AddJob(c.ctx, d, status)
	if err != nil {
		return err
	}
	return c.printJob(job.ID)
}

// @scenario [features/agent-cli.feature#cli-text-from-file] My agent gives long text from a file
// @scenario [features/agent-cli.feature#cli-text-piped] My agent pipes long text in
// @scenario [features/agent-cli-jobs.feature#cli-update-job] Change one detail and keep the rest
// @scenario [features/agent-cli-jobs.feature#cli-clear-job-detail] Clear a detail
func updateJob(c *call) error {
	id := c.id
	job, err := c.jobs.Job(c.ctx, id)
	if err != nil {
		return jobErr(id, err)
	}
	d := jobs.Details{
		Company:     job.Company,
		Role:        job.Role,
		Description: job.Description,
		Link:        job.Link,
		Salary:      job.Salary,
		Location:    job.Location,
		Notes:       job.Notes,
		DateApplied: job.DateApplied,
		Interest:    job.Interest,
	}
	if err := c.applyDetails(&d); err != nil {
		return err
	}
	if _, err := c.jobs.UpdateJob(c.ctx, id, d); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}

// applyDetails sets the details given as options on d, leaving the others as they are.
func (c *call) applyDetails(d *jobs.Details) error {
	for name, field := range map[string]*string{
		"company":      &d.Company,
		"role":         &d.Role,
		"location":     &d.Location,
		"salary":       &d.Salary,
		"link":         &d.Link,
		"date-applied": &d.DateApplied,
	} {
		if v, ok := c.value(name); ok {
			*field = v
		}
	}
	for name, field := range map[string]*string{"description": &d.Description, "notes": &d.Notes} {
		text, ok, err := c.longText(name)
		if err != nil {
			return err
		}
		if ok {
			*field = text
		}
	}
	if v, ok := c.value("interest"); ok {
		if v == "" {
			d.Interest = nil
			return nil
		}
		stars, err := strconv.Atoi(v)
		if err != nil {
			return usageErr("--interest takes a number of stars, like 4, or \"\" to clear the rating, not %q", v)
		}
		d.Interest = &stars
	}
	return nil
}

// @scenario [features/agent-cli.feature#cli-change-prints-result] A change prints what it changed
// @scenario [features/agent-cli.feature#cli-error-lists-choices] A value that isn't one of the choices is answered with the choices
// @scenario [features/agent-cli-jobs.feature#cli-move-top] A moved job goes to the top of its new column
// @scenario [features/agent-cli-jobs.feature#cli-move-bottom] Move a job to the bottom of a column
// @scenario [features/agent-cli-jobs.feature#cli-move-position] Move a job to a chosen place in a column
// @scenario [features/agent-cli-jobs.feature#cli-move-follows-rules] Moving a job follows the app's rules
// @scenario [features/agent-cli-jobs.feature#cli-move-to-closed-needs-close] Moving a job to Closed points my agent to closing it
func moveJob(c *call) error {
	id := c.id
	given, ok := c.value("status")
	if !ok {
		return &cliError{code: ExitUsage, msg: "`job move` needs --status", choices: values(jobs.Statuses)}
	}
	status, err := pick("status", given, jobs.Statuses, jobs.Status.Label)
	if err != nil {
		return err
	}
	job, err := c.jobs.Job(c.ctx, id)
	if err != nil {
		return jobErr(id, err)
	}
	if status == jobs.StatusClosed && job.Status != jobs.StatusClosed {
		return validationErr("a job is moved to Closed by closing it with a reason: `%s job close %d --reason <reason>`", Name, id)
	}
	index, err := moveIndex(c)
	if err != nil {
		return err
	}
	if err := c.jobs.MoveJob(c.ctx, jobs.Move{JobID: id, Status: status, Index: index}); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}

// @scenario [features/agent-cli-jobs.feature#cli-close-job] Close a job with a reason
// @scenario [features/agent-cli-jobs.feature#cli-close-other] Close a job with my own reason
// @scenario [features/agent-cli-jobs.feature#cli-close-needs-reason] Closing a job needs a reason
// @scenario [features/agent-cli-jobs.feature#cli-change-closing-reason] Change a closed job's reason
// moveIndex is where in the column a job moves to, from 0 at the top: the top unless --bottom or
// --position says otherwise. The jobs service puts an index past the bottom at the bottom.
func moveIndex(c *call) (int, error) {
	place, err := c.count("position", "places")
	switch {
	case err != nil:
		return 0, err
	case place != nil && c.flag("bottom"):
		return 0, usageErr("give --bottom or --position, not both")
	case c.flag("bottom"):
		return math.MaxInt32, nil
	case place == nil:
		return 0, nil
	case *place < 1:
		return 0, usageErr("--position counts from 1 at the top of the column, not %d", *place)
	}
	return *place - 1, nil
}

func closeJob(c *call) error {
	id := c.id
	given, ok := c.value("reason")
	if !ok {
		return &cliError{code: ExitValidation, msg: "closing a job needs a reason, given with --reason", choices: values(jobs.ClosingReasons)}
	}
	reason, err := pick("closing reason", given, jobs.ClosingReasons, jobs.ClosingReason.Label)
	if err != nil {
		return err
	}
	other, _ := c.value("other")
	job, err := c.jobs.Job(c.ctx, id)
	if err != nil {
		return jobErr(id, err)
	}
	if job.Status == jobs.StatusClosed {
		err = c.jobs.SetClosingReason(c.ctx, id, reason, other)
	} else {
		err = c.jobs.MoveJob(c.ctx, jobs.Move{JobID: id, Status: jobs.StatusClosed, Index: 0, ClosingReason: reason, ClosingReasonOther: other})
	}
	if err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}

// deletedJSON is what deleting a job prints as JSON.
type deletedJSON struct {
	Deleted struct {
		ID      int64  `json:"id"`
		Company string `json:"company"`
		Role    string `json:"role"`
	} `json:"deleted"`
}

// deleteJob deletes the job at once. Nothing asks to confirm, since an agent can't answer.
// @scenario [features/agent-cli.feature#cli-no-questions] Commands finish without waiting for an answer
// @scenario [features/agent-cli-jobs.feature#cli-delete-job] Delete a job
func deleteJob(c *call) error {
	id := c.id
	job, err := c.jobs.Job(c.ctx, id)
	if err != nil {
		return jobErr(id, err)
	}
	if err := c.jobs.DeleteJob(c.ctx, id); err != nil {
		return jobErr(id, err)
	}
	var out deletedJSON
	out.Deleted.ID, out.Deleted.Company, out.Deleted.Role = job.ID, job.Company, job.Role
	return c.output(out, func(w io.Writer) {
		fmt.Fprintf(w, "Deleted job #%d, %s at %s, with its contacts, interviews, documents, and status history.\n", job.ID, job.Role, job.Company)
	})
}

// @scenario [features/agent-cli-jobs.feature#cli-set-next-action] Set a next action with an expected date
// @scenario [features/agent-cli-jobs.feature#cli-set-next-action-no-date] Set a next action without an expected date
// @scenario [features/agent-cli-jobs.feature#cli-next-action-needs-text] A next action needs to say what it is
func setNextAction(c *call) error {
	id := c.id
	text, _ := c.value("text")
	date, _ := c.value("date")
	if _, err := c.jobs.SetNextAction(c.ctx, id, text, date); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}

// @scenario [features/agent-cli-jobs.feature#cli-clear-next-action-date] Clear only the expected date
// @scenario [features/agent-cli-jobs.feature#cli-clear-next-action] Clear the next action
func clearNextAction(c *call) error {
	id := c.id
	if c.flag("only-date") {
		return c.clearNextActionDate(id)
	}
	if _, err := c.jobs.ClearNextAction(c.ctx, id); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}

// clearNextActionDate keeps a job's next action and clears its expected date, as the app does by
// setting it again with no date.
func (c *call) clearNextActionDate(id int64) error {
	job, err := c.jobs.Job(c.ctx, id)
	if err != nil {
		return jobErr(id, err)
	}
	if job.NextAction == nil {
		return validationErr("job #%d has no next action, so it has no expected date to clear", id)
	}
	if _, err := c.jobs.SetNextAction(c.ctx, id, job.NextAction.Text, ""); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}
