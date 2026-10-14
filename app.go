package main

import (
	"context"
	"log"
	"strings"

	"callback/internal/jobs"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is bound to the frontend by Wails. Its methods only call the jobs and backups services,
// which hold the business rules. See
// docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md.
type App struct {
	ctx     context.Context
	jobs    *jobs.Service
	backups dailyBackups
	// newerData is set when a newer version of Callback has updated the data. The app then has no
	// services, and only tells the user to update Callback.
	newerData bool
	// logf reports what went wrong where the user can't be shown it.
	logf func(format string, args ...any)
}

// dailyBackups takes the daily backup. *backups.Service satisfies it.
type dailyBackups interface {
	TakeDaily(ctx context.Context) (bool, error)
}

// NewApp creates a new App application struct
func NewApp(jobs *jobs.Service, backups dailyBackups) *App {
	return &App{jobs: jobs, backups: backups, logf: log.Printf}
}

// NewNewerDataApp returns an App for data a newer version of Callback has updated
// (db.ErrNewerData). It leaves the data alone: it takes no backup, and the frontend calls none of
// the bindings that reach the services, only DataIsNewer. See
// docs/adr/0025-restore-backups-with-sqlites-online-backup-api-and-check-data-versions-first.md.
func NewNewerDataApp() *App {
	return &App{newerData: true, logf: log.Printf}
}

// startup is called when the app starts. It saves the context for the runtime methods, then takes
// the day's backup if none has been taken yet. A failed backup is logged and never stops the app
// from starting. Data from a newer version of Callback gets no backup, so none is removed either.
// @scenario [features/daily-backups.feature#backup-on-first-start] The first start of the day takes a backup
// @scenario [features/daily-backups.feature#backup-once-per-day] Starting Callback again the same day doesn't take another backup
// @scenario [features/daily-backups.feature#backup-no-catch-up] Days when Callback didn't run get no backup
// @scenario [features/daily-backups.feature#backup-holds-data-not-documents] A backup holds my data but not the attached documents' files
// @scenario [features/daily-backups.feature#backup-keeps-last-15] Taking a 16th daily backup removes the oldest
// @scenario [features/daily-backups.feature#backup-count-not-days] Backups are kept by count, not by age
// @scenario [features/daily-backups.feature#backup-failure-still-starts] Callback still starts when the daily backup fails
// @scenario [features/data-version.feature#app-newer-data-no-backup] The app takes no daily backup of data from a newer Callback
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.newerData {
		return
	}
	if _, err := a.backups.TakeDaily(ctx); err != nil {
		a.logf("callback: the daily backup failed: %v", err)
	}
}

// DataIsNewer reports whether a newer version of Callback has updated the data, in which case the
// frontend tells the user to update Callback and doesn't show the board.
// @scenario [features/data-version.feature#app-newer-data] The app refuses data a newer Callback has updated
func (a *App) DataIsNewer() bool {
	return a.newerData
}

// GetBoard returns every job, laid out in status columns.
// @scenario [features/job-board.feature#board-status-columns] Board shows the status columns in order
// @scenario [features/job-board.feature#empty-board] Empty board when no jobs have been added
// @scenario [features/job-board.feature#column-card-count] Each column shows how many cards it holds
// @scenario [features/job-board.feature#board-persists] Board is the same after reopening the app
// @scenario [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
// @scenario [features/job-documents.feature#card-shows-document-count] The card shows that documents are attached
// @scenario [features/job-interest.feature#card-shows-interest] The card shows the interest rating as stars
// @scenario [features/job-interest.feature#card-unrated] An unrated card shows no stars
// @scenario [features/job-interest.feature#interest-persists] The interest rating is the same after reopening the app
// @scenario [features/job-interviews.feature#card-shows-next-interview] The card shows the date of the next upcoming interview
// @scenario [features/job-interviews.feature#card-interview-today] An interview today counts as upcoming
// @scenario [features/job-interviews.feature#card-no-upcoming-interview] The card shows no interview when every interview has passed
// @scenario [features/job-next-action.feature#card-shows-next-action] The card shows the next action and its expected date
// @scenario [features/job-next-action.feature#card-next-action-no-date] The card shows a next action with no expected date
// @scenario [features/job-next-action.feature#card-next-action-due-today] A next action expected today is marked as due
// @scenario [features/job-next-action.feature#card-next-action-overdue] A next action whose expected date has passed is marked as overdue
// @scenario [features/job-next-action.feature#card-next-action-upcoming] A next action expected later isn't marked
// @scenario [features/job-next-action.feature#closed-card-no-due-marks] A closed job's next action is never marked as due or overdue
// @scenario [features/job-next-action.feature#card-no-next-action] A job with no next action shows none on its card
// @scenario [features/job-next-action.feature#filter-overdue] Show only overdue jobs
// @scenario [features/job-next-action.feature#filter-not-overdue] Show only jobs that aren't overdue
// @scenario [features/job-next-action.feature#filter-due-today-not-overdue] A job due today isn't overdue yet
// @scenario [features/job-next-action.feature#filter-closed-never-overdue] A closed job is never overdue
// @scenario [features/job-next-action.feature#next-action-persists] The next action is the same after reopening the app
func (a *App) GetBoard() (jobs.Board, error) {
	return a.jobs.Board(a.ctx)
}

// AddJob adds a job to the top of a status column, or of Saved when status is "".
// @scenario [features/job-cards.feature#add-job] Add a job with all its details
// @scenario [features/job-cards.feature#add-job-minimum] Add a job with only a company and a role
// @scenario [features/job-cards.feature#add-job-in-column] Add a job directly to a status column
// @scenario [features/job-cards.feature#add-job-missing-required] A job can't be saved without a company or a role
// @scenario [features/job-board.feature#new-card-at-top] A newly added job goes to the top of its column
// @scenario [features/job-cards.feature#add-job-not-closed] A job can't be added straight to Closed
// @scenario [features/job-interest.feature#rate-interest-when-adding] Rate my interest while adding a job
// @scenario [features/job-interest.feature#unrated-by-default] A job has no interest rating until I give it one
func (a *App) AddJob(details jobs.Details, status jobs.Status) (jobs.Job, error) {
	return a.jobs.AddJob(a.ctx, details, status)
}

// GetJob returns a job with all its details, next action, documents, contacts, and interviews.
// @scenario [features/job-cards.feature#view-job-details] View a job's full details
// @scenario [features/job-documents.feature#attach-multiple-documents] Attach several documents to one job
// @scenario [features/job-interest.feature#view-interest] See the interest rating when opening the card
// @scenario [features/job-contacts.feature#view-contacts] See a job's contacts when opening its card
// @scenario [features/job-contacts.feature#no-contacts] A job with no contacts says so
// @scenario [features/job-contacts.feature#contacts-persist] Contacts are the same after reopening the app
// @scenario [features/job-interviews.feature#view-interviews] See a job's interviews when opening its card
// @scenario [features/job-interviews.feature#interviews-in-date-order] Interviews are listed from earliest to latest
// @scenario [features/job-interviews.feature#interviews-same-day-by-time] Interviews on the same day are listed by time, with untimed ones last
// @scenario [features/job-interviews.feature#past-interviews-marked] Interviews that have already happened are shown as past
// @scenario [features/job-interviews.feature#no-interviews] A job with no interviews says so
// @scenario [features/job-interviews.feature#interviews-persist] Interviews are the same after reopening the app
// @scenario [features/job-next-action.feature#view-next-action] See the next action when opening the card
// @scenario [features/document-library.feature#library-not-on-jobs] Library documents aren't attached to any job
func (a *App) GetJob(id int64) (jobs.Job, error) {
	return a.jobs.Job(a.ctx, id)
}

// UpdateJob replaces a job's details.
// @scenario [features/job-cards.feature#edit-job-details] Edit a job's details
// @scenario [features/job-cards.feature#salary-free-text] Salary is written however the posting gives it
// @scenario [features/job-cards.feature#add-notes] Keep notes on a job
// @scenario [features/job-cards.feature#edit-notes] Add to existing notes
// @scenario [features/job-cards.feature#invalid-job-link] A job link must be a web address
// @scenario [features/job-cards.feature#set-date-applied] Set the date applied
// @scenario [features/job-cards.feature#date-applied-any-status] The date applied can be set in any status
// @scenario [features/job-cards.feature#clear-date-applied] Clear the date applied
// @scenario [features/job-cards.feature#future-date-applied] The date applied can't be in the future
// @scenario [features/job-interest.feature#rate-interest] Rate how interested I am in a job
// @scenario [features/job-interest.feature#interest-rating-range] An interest rating must be from one to five stars
// @scenario [features/job-interest.feature#interest-any-status] A job in any status can be rated
// @scenario [features/job-interest.feature#change-interest] Change my interest rating
// @scenario [features/job-interest.feature#clear-interest] Clear my interest rating
// @scenario [features/job-interest.feature#rating-doesnt-reorder] Rating a job doesn't change the order of its column
func (a *App) UpdateJob(id int64, details jobs.Details) (jobs.Job, error) {
	return a.jobs.UpdateJob(a.ctx, id, details)
}

// MoveJob moves a card to a status and position. Moving into Closed needs a closing reason.
// @scenario [features/job-board.feature#move-card-forward] Move a card to the next status
// @scenario [features/job-board.feature#move-card-any-status] Move a card to any other status
// @scenario [features/job-board.feature#move-keeps-details] Moving a card keeps its details and documents
// @scenario [features/job-board.feature#move-to-applied-sets-date] Moving a card to Applied fills in today's date when no date is set
// @scenario [features/job-board.feature#move-to-applied-keeps-date] Moving a card to Applied keeps a date applied that is already set
// @scenario [features/job-board.feature#move-to-saved-clears-date] Moving a card back to Saved clears the date applied
// @scenario [features/job-board.feature#close-asks-reason] Moving a card to Closed asks for a reason
// @scenario [features/job-board.feature#close-other-reason] Write my own reason when closing with Other
// @scenario [features/job-board.feature#reopen-clears-reason] Moving a card out of Closed clears the closing reason
// @scenario [features/job-board.feature#reorder-card] Reorder a card within its column
// @scenario [features/job-board.feature#move-card-to-position] Place a moved card at a chosen position in another column
// @scenario [features/job-board.feature#reorder-saved-keeps-date] Reordering a card within Saved keeps its date applied
// @scenario [features/job-board.feature#reorder-applied-no-date] Reordering a card within Applied doesn't fill in a date applied
// @scenario [features/job-board.feature#close-other-needs-text] Closing with Other needs a written reason
// @scenario [features/job-board.feature#reorder-closed-keeps-reason] Reordering a card within Closed keeps its closing reason
// @scenario [features/job-interest.feature#move-keeps-interest] Moving a card keeps its interest rating
// @scenario [features/job-contacts.feature#move-keeps-contacts] Moving a card keeps its contacts
// @scenario [features/job-interviews.feature#move-keeps-interviews] Moving a card keeps its interviews
// @scenario [features/job-next-action.feature#move-keeps-next-action] Moving a card keeps its next action
func (a *App) MoveJob(move jobs.Move) error {
	return a.jobs.MoveJob(a.ctx, move)
}

// SetClosingReason changes the closing reason of a closed job. other is the text for Other.
// @scenario [features/job-board.feature#change-closing-reason] Change the closing reason of a closed job
func (a *App) SetClosingReason(id int64, reason jobs.ClosingReason, other string) error {
	return a.jobs.SetClosingReason(a.ctx, id, reason, other)
}

// DeleteJob deletes a job, its contacts, its interviews, its status history, and its attached
// documents. The frontend asks for confirmation first. The document library is never touched.
// @scenario [features/job-cards.feature#delete-job] Delete a job after confirming
// @scenario [features/job-contacts.feature#contacts-deleted-with-job] Deleting a job deletes its contacts
// @scenario [features/job-interviews.feature#interviews-deleted-with-job] Deleting a job deletes its interviews
// @scenario [features/document-library.feature#delete-job-keeps-library] Deleting a job doesn't touch the library
// @scenario [features/job-status-history.feature#history-deleted-with-job] Deleting a job deletes its status history
func (a *App) DeleteJob(id int64) error {
	return a.jobs.DeleteJob(a.ctx, id)
}

// OpenJobLink opens a job's posting link in the web browser.
// @scenario [features/job-cards.feature#open-job-link] Open a job's posting link
func (a *App) OpenJobLink(id int64) error {
	return a.jobs.OpenJobLink(a.ctx, id)
}

// ChooseDocumentFile shows the native file dialog, limited to the document types that can be
// attached, and returns the chosen path, or "" if the user cancelled. The webview can't hand Go
// a file's real path, so the frontend passes this path to AttachDocument.
func (a *App) ChooseDocumentFile() (string, error) {
	return a.chooseDocument("Attach a document")
}

// chooseDocument shows the native file dialog with title, limited to jobs.AllowedDocumentTypes.
func (a *App) chooseDocument(title string) (string, error) {
	patterns := make([]string, len(jobs.AllowedDocumentTypes))
	for i, ext := range jobs.AllowedDocumentTypes {
		patterns[i] = "*" + ext
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{{
			DisplayName: "Documents (" + strings.Join(patterns, ", ") + ")",
			Pattern:     strings.Join(patterns, ";"),
		}},
	})
}

// AttachDocument attaches a copy of the file at path to a job.
// @scenario [features/job-documents.feature#attach-document] Attach a document of a given kind
// @scenario [features/job-documents.feature#attach-multiple-documents] Attach several documents to one job
// @scenario [features/job-documents.feature#allowed-file-types] Text-like document types can be attached
// @scenario [features/job-documents.feature#file-type-any-case] A document's type is recognized whatever the letter case of its extension
// @scenario [features/job-documents.feature#disallowed-file-types] Other file types can't be attached
// @scenario [features/job-documents.feature#file-size-limit] A file larger than 10 MB can't be attached
// @scenario [features/job-documents.feature#file-size-at-limit] A file of exactly 10 MB can be attached
// @scenario [features/job-documents.feature#same-document-many-jobs] The same file can be attached to more than one job
func (a *App) AttachDocument(jobID int64, path string, kind jobs.DocumentKind) (jobs.Document, error) {
	return a.jobs.AttachDocument(a.ctx, jobID, path, kind)
}

// SetDocumentKind changes what kind of document an attachment is.
// @scenario [features/job-documents.feature#change-document-kind] Change what kind of document an attachment is
func (a *App) SetDocumentKind(id int64, kind jobs.DocumentKind) error {
	return a.jobs.SetDocumentKind(a.ctx, id, kind)
}

// RemoveDocument removes an attachment from its job, even if its stored copy is already gone.
// @scenario [features/job-documents.feature#remove-document] Remove an attached document
// @scenario [features/job-documents.feature#same-document-many-jobs] The same file can be attached to more than one job
// @scenario [features/job-documents.feature#remove-missing-document] Remove an attached document whose file is missing
func (a *App) RemoveDocument(id int64) error {
	return a.jobs.RemoveDocument(a.ctx, id)
}

// OpenDocument opens an attachment in the OS default app for its file type.
// @scenario [features/job-documents.feature#open-document] Open an attached document
// @scenario [features/job-documents.feature#document-kept-if-original-moved] An attached document still opens after the original file is moved
// @scenario [features/job-documents.feature#open-missing-document] Told when an attached document's file is missing
func (a *App) OpenDocument(id int64) error {
	return a.jobs.OpenDocument(a.ctx, id)
}

// AddContact adds a contact to a job, after its other contacts.
// @scenario [features/job-contacts.feature#add-contact] Add a contact with all their details
// @scenario [features/job-contacts.feature#add-contact-minimum] Add a contact with only a name and a relationship
// @scenario [features/job-contacts.feature#contact-relationships] Choose how a contact relates to the job
// @scenario [features/job-contacts.feature#contact-missing-required] A contact can't be saved without a name or a relationship
// @scenario [features/job-contacts.feature#contact-invalid-email] A contact's email must be an email address
// @scenario [features/job-contacts.feature#contact-invalid-profile-link] A contact's profile link must be a web address
// @scenario [features/job-contacts.feature#contact-phone-free-text] A phone number is written however I have it
// @scenario [features/job-contacts.feature#contact-any-status] Contacts can be added to a job in any status
// @scenario [features/job-contacts.feature#same-person-many-jobs] The same person can be a contact on more than one job
func (a *App) AddContact(jobID int64, details jobs.ContactDetails) (jobs.Contact, error) {
	return a.jobs.AddContact(a.ctx, jobID, details)
}

// UpdateContact replaces a contact's details.
// @scenario [features/job-contacts.feature#edit-contact] Edit a contact
// @scenario [features/job-contacts.feature#contacts-edited-separately] Editing a contact on one job doesn't change it on another
func (a *App) UpdateContact(id int64, details jobs.ContactDetails) (jobs.Contact, error) {
	return a.jobs.UpdateContact(a.ctx, id, details)
}

// RemoveContact removes a contact from its job. The frontend asks for confirmation first.
// @scenario [features/job-contacts.feature#remove-contact] Remove a contact after confirming
func (a *App) RemoveContact(id int64) error {
	return a.jobs.RemoveContact(a.ctx, id)
}

// OpenContactLink opens a contact's profile link in the web browser.
// @scenario [features/job-contacts.feature#open-contact-profile-link] Open a contact's profile link
func (a *App) OpenContactLink(id int64) error {
	return a.jobs.OpenContactLink(a.ctx, id)
}

// AddInterview adds an interview to a job.
// @scenario [features/job-interviews.feature#add-interview] Add an interview with all its details
// @scenario [features/job-interviews.feature#add-interview-minimum] Add an interview with only a date and a kind
// @scenario [features/job-interviews.feature#interview-kinds] Choose what kind of interview it is
// @scenario [features/job-interviews.feature#interview-missing-required] An interview can't be saved without a date or a kind
// @scenario [features/job-interviews.feature#interview-past-date] An interview that already happened can be added
// @scenario [features/job-interviews.feature#interview-any-status] Interviews can be added to a job in any status
func (a *App) AddInterview(jobID int64, details jobs.InterviewDetails) (jobs.Interview, error) {
	return a.jobs.AddInterview(a.ctx, jobID, details)
}

// UpdateInterview replaces an interview's details.
// @scenario [features/job-interviews.feature#edit-interview] Edit an interview
func (a *App) UpdateInterview(id int64, details jobs.InterviewDetails) (jobs.Interview, error) {
	return a.jobs.UpdateInterview(a.ctx, id, details)
}

// RemoveInterview removes an interview from its job. The frontend asks for confirmation first.
// @scenario [features/job-interviews.feature#remove-interview] Remove an interview after confirming
func (a *App) RemoveInterview(id int64) error {
	return a.jobs.RemoveInterview(a.ctx, id)
}

// SetNextAction sets what's next on a job and when it's expected, or "" for no expected date.
// @scenario [features/job-next-action.feature#set-next-action] Set a next action with an expected date
// @scenario [features/job-next-action.feature#set-next-action-no-date] Set a next action without an expected date
// @scenario [features/job-next-action.feature#next-action-needs-text] A next action can't be saved without saying what it is
// @scenario [features/job-next-action.feature#next-action-past-date] The expected date can be in the past
// @scenario [features/job-next-action.feature#next-action-any-status] A next action can be set on a job in any status
// @scenario [features/job-next-action.feature#edit-next-action] Change the next action and its expected date
// @scenario [features/job-next-action.feature#clear-next-action-date] Clear only the expected date
func (a *App) SetNextAction(jobID int64, text, expectedDate string) (jobs.Job, error) {
	return a.jobs.SetNextAction(a.ctx, jobID, text, expectedDate)
}

// ClearNextAction removes a job's next action, once it's done.
// @scenario [features/job-next-action.feature#clear-next-action] Clear the next action once it's done
func (a *App) ClearNextAction(jobID int64) (jobs.Job, error) {
	return a.jobs.ClearNextAction(a.ctx, jobID)
}

// GetLibrary returns every document in the document library, newest first.
// @scenario [features/document-library.feature#open-library] Go from the board to the document library
// @scenario [features/document-library.feature#empty-library] An empty library prompts me to add my first document
// @scenario [features/document-library.feature#library-newest-first] The newest documents are listed first
// @scenario [features/document-library.feature#library-persists] The library is the same after reopening the app
func (a *App) GetLibrary() ([]jobs.LibraryDocument, error) {
	return a.jobs.Library(a.ctx)
}

// ChooseLibraryFile shows the native file dialog for adding to the library, limited to the
// document types that can be added, and returns the chosen path, or "" if the user cancelled.
// The frontend passes this path to AddLibraryDocument.
func (a *App) ChooseLibraryFile() (string, error) {
	return a.chooseDocument("Add a document to the library")
}

// AddLibraryDocument adds a copy of the file at path to the library, with an optional description.
// @scenario [features/document-library.feature#add-library-document] Add a document with a description
// @scenario [features/document-library.feature#add-library-document-no-description] Add a document without a description
// @scenario [features/document-library.feature#library-same-file-twice] The same file can be added more than once
// @scenario [features/document-library.feature#library-allowed-file-types] Text-like document types can be added
// @scenario [features/document-library.feature#library-file-type-any-case] A document's type is recognized whatever the letter case of its extension
// @scenario [features/document-library.feature#library-disallowed-file-types] Other file types can't be added
// @scenario [features/document-library.feature#library-file-size-limit] A file larger than 10 MB can't be added
// @scenario [features/document-library.feature#library-file-size-at-limit] A file of exactly 10 MB can be added
func (a *App) AddLibraryDocument(path, description string) (jobs.LibraryDocument, error) {
	return a.jobs.AddLibraryDocument(a.ctx, path, description)
}

// SetLibraryDescription changes a library document's description, or clears it when it's blank.
// @scenario [features/document-library.feature#edit-library-description] Change a document's description
// @scenario [features/document-library.feature#clear-library-description] Clear a document's description
func (a *App) SetLibraryDescription(id int64, description string) (jobs.LibraryDocument, error) {
	return a.jobs.SetLibraryDescription(a.ctx, id, description)
}

// RemoveLibraryDocument removes a document from the library, never from a job. The frontend asks
// for confirmation first.
// @scenario [features/document-library.feature#remove-library-document] Remove a library document after confirming
// @scenario [features/document-library.feature#remove-library-keeps-attachments] Removing a library document doesn't remove it from jobs
func (a *App) RemoveLibraryDocument(id int64) error {
	return a.jobs.RemoveLibraryDocument(a.ctx, id)
}

// OpenLibraryDocument opens a library document in the OS default app for its file type.
// @scenario [features/document-library.feature#open-library-document] Open a library document
// @scenario [features/document-library.feature#library-document-kept-if-original-moved] A library document still opens after the original file is moved
// @scenario [features/document-library.feature#open-missing-library-document] Told when a library document's file is missing
func (a *App) OpenLibraryDocument(id int64) error {
	return a.jobs.OpenLibraryDocument(a.ctx, id)
}
