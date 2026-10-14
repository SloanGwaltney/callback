package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"callback/internal/jobs"
)

// recordCommands are the commands for a job's contacts, interviews, and attached documents, in
// the order the help lists them. The rules are the jobs service's, as in the app.
var recordCommands = []*command{
	{
		path:    []string{"contact", "add"},
		takesID: "job",
		summary: "Add a contact to a job",
		about: "Adds a person to a job's contacts, after the ones it has, and shows the contact with its " +
			"new ID. It needs a name and a relationship. Each job keeps its own contacts, so the same " +
			"person on two jobs is added to each.",
		options: contactOptions,
		run:     addContact,
	},
	{
		path:    []string{"contact", "update"},
		takesID: "contact",
		summary: "Change some of a contact's details and keep the rest",
		about: "Changes only the details you give and keeps the rest. Give an option an empty value, " +
			"like --phone \"\", to clear it. Shows the contact as it is afterwards.",
		options: contactOptions,
		run:     updateContact,
	},
	{
		path:    []string{"contact", "remove"},
		takesID: "contact",
		summary: "Remove a contact from its job",
		about:   "Removes a contact from its job at once, without asking to confirm. This can't be undone.",
		run:     removeContact,
	},
	{
		path:    []string{"interview", "add"},
		takesID: "job",
		summary: "Add an interview to a job",
		about: "Adds an interview to a job and shows it with its new ID. It needs a date and a kind. The " +
			"date can be in the past, to record an interview that's already happened.",
		options: interviewOptions,
		run:     addInterview,
	},
	{
		path:    []string{"interview", "update"},
		takesID: "interview",
		summary: "Change some of an interview's details and keep the rest",
		about: "Changes only the details you give and keeps the rest. Give --time or --notes an empty " +
			"value, like --time \"\", to clear it. Shows the interview as it is afterwards.",
		options: interviewOptions,
		run:     updateInterview,
	},
	{
		path:    []string{"interview", "remove"},
		takesID: "interview",
		summary: "Remove an interview from its job",
		about:   "Removes an interview from its job at once, without asking to confirm. This can't be undone.",
		run:     removeInterview,
	},
	{
		path:    []string{"document", "attach"},
		takesID: "job",
		summary: "Attach a copy of a file on this computer to a job",
		about: "Attaches a file to a job as a resume, a cover letter, or another kind of document, and " +
			"shows it with its new ID and the path of its stored copy. Callback keeps its own stored " +
			"copy, so the file you give is never changed, and moving or deleting it later doesn't " +
			"affect the job's document.\n\n" +
			"To change a job's document, attach the new file and remove the old one. Never edit a " +
			"stored copy.",
		options: []option{
			{name: "file", value: "<path>", usage: fmt.Sprintf("the file to attach, at most %d MB, of one of these types: %s",
				jobs.MaxDocumentBytes>>20, strings.Join(jobs.AllowedDocumentTypes, ", "))},
			documentKindOption,
		},
		run: attachDocument,
	},
	{
		path:    []string{"document", "set-kind"},
		takesID: "document",
		summary: "Change what kind of document an attachment is",
		about:   "Changes whether an attached document is a resume, a cover letter, or other. Shows the document as it is afterwards.",
		options: []option{documentKindOption},
		run:     setDocumentKind,
	},
	{
		path:    []string{"document", "remove"},
		takesID: "document",
		summary: "Remove a document from its job and delete its stored copy",
		about: "Removes an attached document from its job at once, without asking to confirm, and " +
			"deletes Callback's stored copy. The file it was attached from isn't touched. This can't " +
			"be undone.",
		run: removeDocument,
	},
}

var contactOptions = []option{
	{name: "name", value: "<text>", usage: "the person's name"},
	{name: "relationship", value: "<relationship>", usage: "how they relate to the job", choices: described(jobs.Relationships, jobs.Relationship.Label)},
	{name: "email", value: "<email>", usage: "their email address, like dana@example.com"},
	{name: "phone", value: "<text>", usage: "their phone number, written however you have it"},
	{name: "profile-link", value: "<url>", usage: "a web address for their profile, like https://www.linkedin.com/in/dana-smith"},
	{name: "notes", value: "<text>", usage: "your notes on them"},
	{name: "notes-file", value: "<path>", usage: "read the notes from a file, or from what's piped in with -"},
}

var interviewOptions = []option{
	{name: "date", value: "<date>", usage: "the day of the interview, like 2026-10-20"},
	{name: "time", value: "<time>", usage: "its time on a 24-hour clock, like 14:00; no time if not given"},
	{name: "kind", value: "<kind>", usage: "what kind of interview it is", choices: described(jobs.InterviewKinds, jobs.InterviewKind.Label)},
	{name: "notes", value: "<text>", usage: "your notes on it"},
	{name: "notes-file", value: "<path>", usage: "read the notes from a file, or from what's piped in with -"},
}

var documentKindOption = option{name: "kind", value: "<kind>", usage: "what the document is for", choices: described(jobs.DocumentKinds, jobs.DocumentKind.Label)}

// missingErr names the ID when there's no what with it, like "there's no contact with ID 999".
func missingErr(what string, id int64, err, notFound error) error {
	if errors.Is(err, notFound) {
		return notFoundErr("there's no %s with ID %d", what, id)
	}
	return err
}

// required returns an error listing choices when the option name, one of a fixed set, wasn't
// given. what says what needs it, like "a contact".
func (c *call) required(name, what string, choices []string) error {
	if _, ok := c.value(name); ok {
		return nil
	}
	return &cliError{code: ExitValidation, msg: fmt.Sprintf("%s needs a %s, given with --%s", what, name, name), choices: choices}
}

// removedJSON is what removing a contact, interview, or document prints as JSON.
type removedJSON struct {
	Removed any `json:"removed"`
}

// @scenario [#cli-add-contact] [#cli-add-contact-rules]
func addContact(c *call) error {
	if err := c.required("relationship", "a contact", values(jobs.Relationships)); err != nil {
		return err
	}
	var d jobs.ContactDetails
	if err := c.applyContact(&d); err != nil {
		return err
	}
	contact, err := c.jobs.AddContact(c.ctx, c.id, d)
	if err != nil {
		return jobErr(c.id, err)
	}
	return c.output(contact, func(w io.Writer) { writeContact(w, contact) })
}

// @scenario [#cli-update-contact]
func updateContact(c *call) error {
	contact, err := c.jobs.Contact(c.ctx, c.id)
	if err != nil {
		return missingErr("contact", c.id, err, jobs.ErrContactNotFound)
	}
	d := contact.Details()
	if err := c.applyContact(&d); err != nil {
		return err
	}
	if contact, err = c.jobs.UpdateContact(c.ctx, c.id, d); err != nil {
		return missingErr("contact", c.id, err, jobs.ErrContactNotFound)
	}
	return c.output(contact, func(w io.Writer) { writeContact(w, contact) })
}

// applyContact sets the contact details given as options on d, leaving the others as they are.
func (c *call) applyContact(d *jobs.ContactDetails) error {
	for name, field := range map[string]*string{"name": &d.Name, "email": &d.Email, "phone": &d.Phone, "profile-link": &d.ProfileLink} {
		if v, ok := c.value(name); ok {
			*field = v
		}
	}
	if given, ok := c.value("relationship"); ok {
		r, err := pick("relationship", given, jobs.Relationships, jobs.Relationship.Label)
		if err != nil {
			return err
		}
		d.Relationship = r
	}
	notes, ok, err := c.longText("notes")
	if ok {
		d.Notes = notes
	}
	return err
}

// @scenario [#cli-remove-contact] [#cli-unknown-contact]
func removeContact(c *call) error {
	contact, err := c.jobs.Contact(c.ctx, c.id)
	if err != nil {
		return missingErr("contact", c.id, err, jobs.ErrContactNotFound)
	}
	if err := c.jobs.RemoveContact(c.ctx, c.id); err != nil {
		return missingErr("contact", c.id, err, jobs.ErrContactNotFound)
	}
	return c.output(removedJSON{contact}, func(w io.Writer) {
		fmt.Fprintf(w, "Removed contact #%d, %s, from job #%d.\n", contact.ID, contact.Name, contact.JobID)
	})
}

// @scenario [#cli-add-interview] [#cli-add-interview-rules]
func addInterview(c *call) error {
	if err := c.required("kind", "an interview", values(jobs.InterviewKinds)); err != nil {
		return err
	}
	var d jobs.InterviewDetails
	if err := c.applyInterview(&d); err != nil {
		return err
	}
	interview, err := c.jobs.AddInterview(c.ctx, c.id, d)
	if err != nil {
		return jobErr(c.id, err)
	}
	return c.output(interview, func(w io.Writer) { writeInterview(w, interview) })
}

// @scenario [#cli-update-interview]
func updateInterview(c *call) error {
	interview, err := c.jobs.Interview(c.ctx, c.id)
	if err != nil {
		return missingErr("interview", c.id, err, jobs.ErrInterviewNotFound)
	}
	d := interview.Details()
	if err := c.applyInterview(&d); err != nil {
		return err
	}
	if interview, err = c.jobs.UpdateInterview(c.ctx, c.id, d); err != nil {
		return missingErr("interview", c.id, err, jobs.ErrInterviewNotFound)
	}
	return c.output(interview, func(w io.Writer) { writeInterview(w, interview) })
}

// applyInterview sets the interview details given as options on d, leaving the others as they
// are.
func (c *call) applyInterview(d *jobs.InterviewDetails) error {
	for name, field := range map[string]*string{"date": &d.Date, "time": &d.Time} {
		if v, ok := c.value(name); ok {
			*field = v
		}
	}
	if given, ok := c.value("kind"); ok {
		kind, err := pick("interview kind", given, jobs.InterviewKinds, jobs.InterviewKind.Label)
		if err != nil {
			return err
		}
		d.Kind = kind
	}
	notes, ok, err := c.longText("notes")
	if ok {
		d.Notes = notes
	}
	return err
}

// @scenario [#cli-remove-interview]
func removeInterview(c *call) error {
	interview, err := c.jobs.Interview(c.ctx, c.id)
	if err != nil {
		return missingErr("interview", c.id, err, jobs.ErrInterviewNotFound)
	}
	if err := c.jobs.RemoveInterview(c.ctx, c.id); err != nil {
		return missingErr("interview", c.id, err, jobs.ErrInterviewNotFound)
	}
	return c.output(removedJSON{interview}, func(w io.Writer) {
		fmt.Fprintf(w, "Removed interview #%d, %s on %s, from job #%d.\n", interview.ID, interview.Kind.Label(), interview.Date, interview.JobID)
	})
}

// @scenario [#cli-attach-document] [#cli-attach-document-rules]
func attachDocument(c *call) error {
	path, ok := c.value("file")
	if !ok {
		return usageErr("`document attach` needs --file with the path of the file to attach")
	}
	kind, err := c.documentKind()
	if err != nil {
		return err
	}
	doc, err := c.jobs.AttachDocument(c.ctx, c.id, path, kind)
	if err != nil {
		return jobErr(c.id, err)
	}
	return c.printDocument(doc)
}

// documentKind reads --kind, which every document command but remove needs.
func (c *call) documentKind() (jobs.DocumentKind, error) {
	if err := c.required("kind", "a document", values(jobs.DocumentKinds)); err != nil {
		return "", err
	}
	given, _ := c.value("kind")
	return pick("document kind", given, jobs.DocumentKinds, jobs.DocumentKind.Label)
}

// @scenario [#cli-relabel-document]
func setDocumentKind(c *call) error {
	kind, err := c.documentKind()
	if err != nil {
		return err
	}
	if err := c.jobs.SetDocumentKind(c.ctx, c.id, kind); err != nil {
		return missingErr("document", c.id, err, jobs.ErrDocumentNotFound)
	}
	doc, err := c.jobs.Document(c.ctx, c.id)
	if err != nil {
		return missingErr("document", c.id, err, jobs.ErrDocumentNotFound)
	}
	return c.printDocument(doc)
}

// printDocument shows an attached document with where its stored copy is.
func (c *call) printDocument(doc jobs.Document) error {
	files, err := c.jobs.JobDocumentFiles(c.ctx, doc.JobID)
	if err != nil {
		return err
	}
	out := jobJSONFrom(jobs.Job{Documents: []jobs.Document{doc}}, files).Documents[0]
	return c.output(out, func(w io.Writer) { writeDocument(w, out) })
}

// @scenario [#cli-remove-document]
func removeDocument(c *call) error {
	doc, err := c.jobs.Document(c.ctx, c.id)
	if err != nil {
		return missingErr("document", c.id, err, jobs.ErrDocumentNotFound)
	}
	if err := c.jobs.RemoveDocument(c.ctx, c.id); err != nil {
		return missingErr("document", c.id, err, jobs.ErrDocumentNotFound)
	}
	return c.output(removedJSON{doc}, func(w io.Writer) {
		fmt.Fprintf(w, "Removed document #%d, %s, from job #%d, and deleted its stored copy.\n", doc.ID, doc.FileName, doc.JobID)
	})
}
