# {{.Name}}: an overview for agents

Callback is a desktop app that keeps track of a person's job search. `{{.Name}}` is its command
line, for you, their agent. It works on the same data as the app, with the same rules, so whatever
you change is on the person's board the next time they look. This overview explains what Callback
keeps track of and how to work with it. `{{.Name}} --help` lists the commands, and
`{{.Name}} <command> --help` lists a command's options and the values they take.

## What Callback keeps track of

- **Jobs.** Each job is a role at a company the person is interested in or has applied to. It has
  a company and a role, and can have a location, a salary written however the posting gives it, a
  posting link, the date they applied, an interest rating from 1 to 5 stars, a description, and
  notes.
- **Statuses.** The board has a column for each status a job can be in, in this order: Saved,
  Applied, Interviewing, Offer, and Closed. Each column has its own order, top to bottom. A closed
  job has a reason it was closed: Rejected, Withdrew, No response, Declined offer, Position filled,
  or Other, with the reason in the person's own words.
- **Status history.** Every status a job has been in, and the date it moved there. It's a record:
  nothing changes or removes an entry once it's added. The days a job has been in its status
  count from its latest entry.
- **Next actions.** What the person is waiting for or needs to do next on a job, like "Follow up
  with the recruiter", with the date it's expected, if any. One that's expected today is due, and
  one whose date has passed is overdue, except on a closed job.
- **Contacts.** The people at a job: recruiters, hiring managers, referrers, interviewers, and
  others, with their email, phone, profile link, and notes.
- **Interviews.** A job's interviews, each with its date, time if known, kind, and notes.
- **Documents.** Files attached to a job, like the resume or cover letter sent for it. Callback
  keeps its own stored copy of each file in its data folder.
- **The document library.** Documents that aren't attached to any job, like a general resume,
  each with a description of what it is. Callback keeps a stored copy of each of these too.

## How the commands fit together

1. Find the job. `{{.Name}} job list` lists the jobs that aren't closed, one line each, in board
   order, starting with each job's ID. Add `--all` for closed jobs too. Its options narrow the
   list: to some statuses, to a search of the role and company, to jobs whose next action is
   overdue, to jobs that have been in their status for some days, or to jobs rated at least some
   stars. Every option you give applies.
2. Read the job. `{{.Name}} job show <job-id>` shows everything about it: its details,
   description, notes, contacts, interviews, next action, status history, and documents.
3. Change it, by its ID:
   - `{{.Name}} job update <job-id>` changes the details you name and keeps the rest.
   - `{{.Name}} job move <job-id> --status <status>` moves it to the top of another status, or
     with `--bottom` or `--position <place>`, to the bottom or a place in the column.
   - `{{.Name}} job close <job-id> --reason <reason>` closes it, or changes a closed job's
     reason. A job is moved to Closed only this way.
   - `{{.Name}} job set-next-action <job-id>` and `{{.Name}} job clear-next-action <job-id>`
     set and clear its next action. Add `--only-date` to clear only its expected date.
   - `{{.Name}} job delete <job-id>` deletes it.
4. Keep its people, interviews, and documents up to date. `job show` lists each with its ID.
   - `{{.Name}} contact add <job-id>` adds a contact, `{{.Name}} contact update <contact-id>`
     changes the details you name and keeps the rest, and `{{.Name}} contact remove <contact-id>`
     removes one.
   - `{{.Name}} interview add <job-id>`, `{{.Name}} interview update <interview-id>`, and
     `{{.Name}} interview remove <interview-id>` do the same for interviews.
   - `{{.Name}} document attach <job-id> --file <path> --kind <kind>` attaches a copy of a file
     on the person's computer. `{{.Name}} document set-kind <document-id>` changes whether it's a
     resume, a cover letter, or other, and `{{.Name}} document remove <document-id>` removes it
     and deletes its stored copy.
5. `{{.Name}} job add` adds a new job.
6. Keep the document library up to date. `{{.Name}} library list` lists it, newest first, with
   each document's ID. `{{.Name}} library add --file <path>` adds a copy of a file on the
   person's computer, `{{.Name}} library describe <library-document-id>` changes or clears its
   description, and `{{.Name}} library remove <library-document-id>` removes it and deletes its
   stored copy.

Commands follow the app's rules. Moving a job to Applied fills in today as its date applied if it
has none, and moving it back to Saved clears it. Every move to another status is added to the
job's status history. Contacts, interviews, and documents can be added to a job in any status,
and adding them doesn't move it.

## Reading the person's documents

The person's resumes, cover letters, and other documents are yours to read. `job show` gives the
path of the stored copy of each of a job's documents, and `library list` gives the path of the
stored copy of each document in the library. Read them when they help with what you're doing,
like the resume sent for a job when you prepare for its interview, or the library's resumes and
letters when you help with a new application. A document marked missing has lost its stored
copy, so there's nothing to read; tell the person, who can attach or add the file again.

## Output

Output is plain text unless you add `--json` to the command, which prints JSON with the same
details and more. Every command that changes a job prints the job as it is afterwards, and every
command that adds or changes a contact, interview, document, or library document prints it, with
its ID, so you can check what changed without another command.

## IDs

Every job, contact, interview, document, and library document comes with its ID. A job's
documents and the library's have IDs of their own, so give a library document's ID only to a
`library` command. In plain text an ID is written like `#12`; in JSON it's the `id` field.
Commands take the number alone: `{{.Name}} job show 12`.

## Dates

Dates are written as year, month, and day, like 2026-10-14, in output and in what you give.
Interview times are written on a 24-hour clock, like 14:00.

## Long text

A description or notes can be long, with quotes, line breaks, and blank lines. Rather than
quoting it on the command line, write it to a file and give the file's path, as in
`--description-file job.txt`, or pipe it in and give `-` for the path, as in
`--notes-file -`. The text is kept exactly as written.

## Errors

When a command can't be done, it changes nothing, writes what went wrong to standard error, and
ends with an exit code for its kind of failure. A value that isn't one of a fixed set, like a
status, is answered with the values you can use. With `--json`, the error is JSON instead:

    {"error": {"kind": "validation", "exitCode": 3, "message": "role is required"}}

with `choices` added when there are values to choose from.

The exit codes:
{{range .ExitCodes}}
- **{{.Code}}**{{if .Kind}} (`{{.Kind}}`){{end}}: {{.Meaning}}{{end}}

## Working safely

- Nothing asks you to confirm. A command does what it says at once, so be sure before you delete
  a job: it takes the job's contacts, interviews, documents, and status history with it, and it
  can't be undone. Removing a contact, interview, document, or library document can't be undone either.
- Before you change many jobs at once, back up the person's data with
  `{{.Name}} backup make <backup-name> --days <days>`, under a name that says what it's before,
  like `before-cleanup`. Keep it for as long as the person might take to notice something went
  wrong: 7 days suits most changes, and it can be kept from 1 to 90. It's removed after that, and
  nothing removes it before then.
- If the changes go wrong, `{{.Name}} backup restore <backup-name>` puts all of the person's data
  back as it was when the backup was made. It first backs up the data as it is, kept for 7 days,
  and tells you that backup's name, so restoring that one undoes the restore. Restoring replaces
  everything, including what the person changed in the app since the backup, so tell them what
  you restored. `{{.Name}} backup list` lists every backup, newest first, with who made it and,
  for one that expires, the day it's kept until.
- Callback also backs up the person's data once on each day it runs, the first time the app opens
  or a command runs that day, and the app backs it up just before the person restores a backup
  there. Those backups are listed too, but they're the person's, not yours to restore.
- If the data was last opened by a newer version of Callback, every command stops with exit code
  5 and changes nothing. Tell the person to update Callback. Data from an older version is
  brought up to date first, as the app does.

## What you can't do

- You can't open anything on the person's screen: not a posting link, a contact's profile link,
  or a document. Read the link, or the document's stored copy, instead.
- You can't delete backups, and you can't restore a daily backup. You can't restore one the app
  made before a restore either.
- Never edit a document's stored copy. To change a job's document, attach a new file in its place.
  To change a library document, add the new file and remove the old one.
