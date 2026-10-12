package main

import (
	"context"
	"strings"

	"callback/internal/jobs"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is bound to the frontend by Wails. Its methods only call the jobs service, which holds the
// business rules. See
// docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md.
type App struct {
	ctx  context.Context
	jobs *jobs.Service
}

// NewApp creates a new App application struct
func NewApp(jobs *jobs.Service) *App {
	return &App{jobs: jobs}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

// GetJob returns a job with all its details and documents.
// @scenario [features/job-cards.feature#view-job-details] View a job's full details
// @scenario [features/job-documents.feature#attach-multiple-documents] Attach several documents to one job
// @scenario [features/job-interest.feature#view-interest] See the interest rating when opening the card
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
func (a *App) MoveJob(move jobs.Move) error {
	return a.jobs.MoveJob(a.ctx, move)
}

// SetClosingReason changes the closing reason of a closed job. other is the text for Other.
// @scenario [features/job-board.feature#change-closing-reason] Change the closing reason of a closed job
func (a *App) SetClosingReason(id int64, reason jobs.ClosingReason, other string) error {
	return a.jobs.SetClosingReason(a.ctx, id, reason, other)
}

// DeleteJob deletes a job and its attached documents. The frontend asks for confirmation first.
// @scenario [features/job-cards.feature#delete-job] Delete a job after confirming
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
	patterns := make([]string, len(jobs.AllowedDocumentTypes))
	for i, ext := range jobs.AllowedDocumentTypes {
		patterns[i] = "*" + ext
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Attach a document",
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

// RemoveDocument removes an attachment from its job.
// @scenario [features/job-documents.feature#remove-document] Remove an attached document
// @scenario [features/job-documents.feature#same-document-many-jobs] The same file can be attached to more than one job
func (a *App) RemoveDocument(id int64) error {
	return a.jobs.RemoveDocument(a.ctx, id)
}

// OpenDocument opens an attachment in the OS default app for its file type.
// @scenario [features/job-documents.feature#open-document] Open an attached document
// @scenario [features/job-documents.feature#document-kept-if-original-moved] An attached document still opens after the original file is moved
func (a *App) OpenDocument(id int64) error {
	return a.jobs.OpenDocument(a.ctx, id)
}
