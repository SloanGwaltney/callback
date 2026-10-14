package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"callback/internal/jobs"
)

// commands is every command, in the order the help lists them.
var commands = []*command{
	{
		path:    []string{"job", "list"},
		summary: "List your jobs, one line each, in board order",
		about: "Lists your jobs column by column, Saved to Offer, in each column's order, one line per job " +
			"with its ID, status, days in that status, company, role, location, date applied, interest " +
			"rating, and next action. A next action that's due today or overdue is marked. Closed jobs " +
			"are left out unless you give --all. Descriptions and notes aren't listed; `job show` has them.",
		options: []option{{name: "all", usage: "list closed jobs too, after the others"}},
		run:     listJobs,
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
		summary: "Move a job to the top of another status",
		about: "Moves a job to the top of a status column, following the app's rules: moving to Applied " +
			"fills in today as the date applied if none is set, and moving back to Saved clears it. " +
			"The move is added to the job's status history. To move a job to Closed, close it with " +
			"`job close` instead, which takes a reason. Shows the job as it is afterwards.",
		options: []option{
			{name: "status", value: "<status>", usage: "the status to move it to", choices: described(jobs.Statuses, jobs.Status.Label)},
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
		summary: "Clear a job's next action, once it's done",
		about:   "Removes a job's next action. Shows the job as it is afterwards.",
		run:     clearNextAction,
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
	if errors.Is(err, jobs.ErrJobNotFound) {
		return notFoundErr("there's no job with ID %d", id)
	}
	return err
}

// @scenario [features/agent-cli.feature#cli-text-output] Output is plain text by default
// @scenario [features/agent-cli.feature#cli-json-output] My agent can ask for any command's output as JSON
func listJobs(c *call) error {
	board, err := c.jobs.Board(c.ctx)
	if err != nil {
		return err
	}
	all := c.flag("all")
	listed := []cardJSON{}
	closed := 0
	for _, col := range board.Columns {
		if col.Status == jobs.StatusClosed && !all {
			closed = col.Count
			continue
		}
		for _, card := range col.Cards {
			listed = append(listed, cardJSON{Card: card, Status: col.Status, StatusLabel: col.Label})
		}
	}
	return c.output(listed, func(w io.Writer) {
		switch {
		case len(listed) == 0 && closed > 0:
			fmt.Fprintf(w, "No open jobs to list. Add --all to list the %d closed ones.\n", closed)
		case len(listed) == 0:
			fmt.Fprintln(w, "No jobs to list.")
		}
		for _, card := range listed {
			writeCardLine(w, card)
		}
	})
}

// @scenario [features/agent-cli.feature#cli-output-ids] Everything listed comes with its ID
// @scenario [features/agent-cli.feature#cli-dates] Dates are written as year, month, and day
// @scenario [features/agent-cli.feature#cli-unknown-job] My agent is told when there's no job with an ID
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
	if err := c.jobs.MoveJob(c.ctx, jobs.Move{JobID: id, Status: status, Index: 0}); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
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

func setNextAction(c *call) error {
	id := c.id
	text, _ := c.value("text")
	date, _ := c.value("date")
	if _, err := c.jobs.SetNextAction(c.ctx, id, text, date); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}

func clearNextAction(c *call) error {
	id := c.id
	if _, err := c.jobs.ClearNextAction(c.ctx, id); err != nil {
		return jobErr(id, err)
	}
	return c.printJob(id)
}
