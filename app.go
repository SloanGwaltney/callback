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
	backups startupBackups
	// newerData is set when a newer version of Callback has updated the data. The app then has no
	// services, and only tells the user to update Callback.
	newerData bool
	// logf reports what went wrong where the user can't be shown it.
	logf func(format string, args ...any)
}

// startupBackups takes the daily backup and removes expired agent backups. *backups.Service
// satisfies it. The app has no restore; agents restore through the command line.
type startupBackups interface {
	TakeDaily(ctx context.Context) (bool, error)
	RemoveExpired() error
}

// NewApp creates a new App application struct
func NewApp(jobs *jobs.Service, backups startupBackups) *App {
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
// the day's backup if none has been taken yet, and removes the agent backups that have expired. A
// failed backup, or an expired one that can't be removed yet, is logged and never stops the app
// from starting; the expired one is removed on a later run. Data from a newer version of Callback
// gets no backup, and no backup is removed.
// @scenario [#backup-on-first-start] [#backup-once-per-day] [#backup-no-catch-up] [#backup-holds-data-not-documents] [#backup-keeps-last-15] [#backup-count-not-days] [#backup-failure-still-starts] [#app-newer-data-no-backup] [#backup-rotation-keeps-agent-backups] [#agent-backup-kept-through-last-day] [#agent-backup-expires] [#agent-backup-expiry-retries]
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.newerData {
		return
	}
	if _, err := a.backups.TakeDaily(ctx); err != nil {
		a.logf("callback: the daily backup failed: %v", err)
	}
	if err := a.backups.RemoveExpired(); err != nil {
		a.logf("callback: removing expired backups failed, and will be tried again next time: %v", err)
	}
}

// DataIsNewer reports whether a newer version of Callback has updated the data, in which case the
// frontend tells the user to update Callback and doesn't show the board.
// @scenario [#app-newer-data]
func (a *App) DataIsNewer() bool {
	return a.newerData
}

// GetBoard returns every job, laid out in status columns.
// @scenario [#board-status-columns] [#empty-board] [#column-card-count] [#board-persists] [#card-order-persists] [#card-shows-document-count] [#card-shows-interest] [#card-unrated] [#interest-persists] [#card-shows-next-interview] [#card-interview-today] [#card-no-upcoming-interview] [#card-shows-next-action] [#card-next-action-no-date] [#card-next-action-due-today] [#card-next-action-overdue] [#card-next-action-upcoming] [#closed-card-no-due-marks] [#card-no-next-action] [#filter-overdue] [#filter-not-overdue] [#filter-due-today-not-overdue] [#filter-closed-never-overdue] [#next-action-persists]
func (a *App) GetBoard() (jobs.Board, error) {
	return a.jobs.Board(a.ctx)
}

// AddJob adds a job to the top of a status column, or of Saved when status is "".
// @scenario [#add-job] [#add-job-minimum] [#add-job-in-column] [#add-job-missing-required] [#new-card-at-top] [#add-job-not-closed] [#rate-interest-when-adding] [#unrated-by-default]
func (a *App) AddJob(details jobs.Details, status jobs.Status) (jobs.Job, error) {
	return a.jobs.AddJob(a.ctx, details, status)
}

// GetJob returns a job with all its details, next action, documents, contacts, and interviews.
// @scenario [#view-job-details] [#attach-multiple-documents] [#view-interest] [#view-contacts] [#no-contacts] [#contacts-persist] [#view-interviews] [#interviews-in-date-order] [#interviews-same-day-by-time] [#past-interviews-marked] [#no-interviews] [#interviews-persist] [#view-next-action] [#library-not-on-jobs]
func (a *App) GetJob(id int64) (jobs.Job, error) {
	return a.jobs.Job(a.ctx, id)
}

// UpdateJob replaces a job's details.
// @scenario [#edit-job-details] [#salary-free-text] [#add-notes] [#edit-notes] [#invalid-job-link] [#set-date-applied] [#date-applied-any-status] [#clear-date-applied] [#future-date-applied] [#rate-interest] [#interest-rating-range] [#interest-any-status] [#change-interest] [#clear-interest] [#rating-doesnt-reorder]
func (a *App) UpdateJob(id int64, details jobs.Details) (jobs.Job, error) {
	return a.jobs.UpdateJob(a.ctx, id, details)
}

// MoveJob moves a card to a status and position. Moving into Closed needs a closing reason.
// @scenario [#move-card-forward] [#move-card-any-status] [#move-keeps-details] [#move-to-applied-sets-date] [#move-to-applied-keeps-date] [#move-to-saved-clears-date] [#close-asks-reason] [#close-other-reason] [#reopen-clears-reason] [#reorder-card] [#move-card-to-position] [#reorder-saved-keeps-date] [#reorder-applied-no-date] [#close-other-needs-text] [#reorder-closed-keeps-reason] [#move-keeps-interest] [#move-keeps-contacts] [#move-keeps-interviews] [#move-keeps-next-action]
func (a *App) MoveJob(move jobs.Move) error {
	return a.jobs.MoveJob(a.ctx, move)
}

// SetClosingReason changes the closing reason of a closed job. other is the text for Other.
// @scenario [#change-closing-reason]
func (a *App) SetClosingReason(id int64, reason jobs.ClosingReason, other string) error {
	return a.jobs.SetClosingReason(a.ctx, id, reason, other)
}

// DeleteJob deletes a job, its contacts, its interviews, its status history, and its attached
// documents. The frontend asks for confirmation first. The document library is never touched.
// @scenario [#delete-job] [#contacts-deleted-with-job] [#interviews-deleted-with-job] [#delete-job-keeps-library] [#history-deleted-with-job]
func (a *App) DeleteJob(id int64) error {
	return a.jobs.DeleteJob(a.ctx, id)
}

// OpenJobLink opens a job's posting link in the web browser.
// @scenario [#open-job-link]
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
// @scenario [#attach-document] [#attach-multiple-documents] [#allowed-file-types] [#file-type-any-case] [#disallowed-file-types] [#file-size-limit] [#file-size-at-limit] [#same-document-many-jobs]
func (a *App) AttachDocument(jobID int64, path string, kind jobs.DocumentKind) (jobs.Document, error) {
	return a.jobs.AttachDocument(a.ctx, jobID, path, kind)
}

// SetDocumentKind changes what kind of document an attachment is.
// @scenario [#change-document-kind]
func (a *App) SetDocumentKind(id int64, kind jobs.DocumentKind) error {
	return a.jobs.SetDocumentKind(a.ctx, id, kind)
}

// RemoveDocument removes an attachment from its job, even if its stored copy is already gone.
// @scenario [#remove-document] [#same-document-many-jobs] [#remove-missing-document]
func (a *App) RemoveDocument(id int64) error {
	return a.jobs.RemoveDocument(a.ctx, id)
}

// OpenDocument opens an attachment in the OS default app for its file type.
// @scenario [#open-document] [#document-kept-if-original-moved] [#open-missing-document]
func (a *App) OpenDocument(id int64) error {
	return a.jobs.OpenDocument(a.ctx, id)
}

// AddContact adds a contact to a job, after its other contacts.
// @scenario [#add-contact] [#add-contact-minimum] [#contact-relationships] [#contact-missing-required] [#contact-invalid-email] [#contact-invalid-profile-link] [#contact-phone-free-text] [#contact-any-status] [#same-person-many-jobs]
func (a *App) AddContact(jobID int64, details jobs.ContactDetails) (jobs.Contact, error) {
	return a.jobs.AddContact(a.ctx, jobID, details)
}

// UpdateContact replaces a contact's details.
// @scenario [#edit-contact] [#contacts-edited-separately]
func (a *App) UpdateContact(id int64, details jobs.ContactDetails) (jobs.Contact, error) {
	return a.jobs.UpdateContact(a.ctx, id, details)
}

// RemoveContact removes a contact from its job. The frontend asks for confirmation first.
// @scenario [#remove-contact]
func (a *App) RemoveContact(id int64) error {
	return a.jobs.RemoveContact(a.ctx, id)
}

// OpenContactLink opens a contact's profile link in the web browser.
// @scenario [#open-contact-profile-link]
func (a *App) OpenContactLink(id int64) error {
	return a.jobs.OpenContactLink(a.ctx, id)
}

// AddInterview adds an interview to a job.
// @scenario [#add-interview] [#add-interview-minimum] [#interview-kinds] [#interview-missing-required] [#interview-past-date] [#interview-any-status]
func (a *App) AddInterview(jobID int64, details jobs.InterviewDetails) (jobs.Interview, error) {
	return a.jobs.AddInterview(a.ctx, jobID, details)
}

// UpdateInterview replaces an interview's details.
// @scenario [#edit-interview]
func (a *App) UpdateInterview(id int64, details jobs.InterviewDetails) (jobs.Interview, error) {
	return a.jobs.UpdateInterview(a.ctx, id, details)
}

// RemoveInterview removes an interview from its job. The frontend asks for confirmation first.
// @scenario [#remove-interview]
func (a *App) RemoveInterview(id int64) error {
	return a.jobs.RemoveInterview(a.ctx, id)
}

// SetNextAction sets what's next on a job and when it's expected, or "" for no expected date.
// @scenario [#set-next-action] [#set-next-action-no-date] [#next-action-needs-text] [#next-action-past-date] [#next-action-any-status] [#edit-next-action] [#clear-next-action-date]
func (a *App) SetNextAction(jobID int64, text, expectedDate string) (jobs.Job, error) {
	return a.jobs.SetNextAction(a.ctx, jobID, text, expectedDate)
}

// ClearNextAction removes a job's next action, once it's done.
// @scenario [#clear-next-action]
func (a *App) ClearNextAction(jobID int64) (jobs.Job, error) {
	return a.jobs.ClearNextAction(a.ctx, jobID)
}

// GetLibrary returns every document in the document library, newest first.
// @scenario [#open-library] [#empty-library] [#library-newest-first] [#library-persists]
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
// @scenario [#add-library-document] [#add-library-document-no-description] [#library-same-file-twice] [#library-allowed-file-types] [#library-file-type-any-case] [#library-disallowed-file-types] [#library-file-size-limit] [#library-file-size-at-limit]
func (a *App) AddLibraryDocument(path, description string) (jobs.LibraryDocument, error) {
	return a.jobs.AddLibraryDocument(a.ctx, path, description)
}

// SetLibraryDescription changes a library document's description, or clears it when it's blank.
// @scenario [#edit-library-description] [#clear-library-description]
func (a *App) SetLibraryDescription(id int64, description string) (jobs.LibraryDocument, error) {
	return a.jobs.SetLibraryDescription(a.ctx, id, description)
}

// RemoveLibraryDocument removes a document from the library, never from a job. The frontend asks
// for confirmation first.
// @scenario [#remove-library-document] [#remove-library-keeps-attachments]
func (a *App) RemoveLibraryDocument(id int64) error {
	return a.jobs.RemoveLibraryDocument(a.ctx, id)
}

// OpenLibraryDocument opens a library document in the OS default app for its file type.
// @scenario [#open-library-document] [#library-document-kept-if-original-moved] [#open-missing-library-document]
func (a *App) OpenLibraryDocument(id int64) error {
	return a.jobs.OpenLibraryDocument(a.ctx, id)
}
