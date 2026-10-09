package jobs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"callback/internal/store"
)

// MaxDocumentBytes is the largest file that can be attached: 10 MB.
const MaxDocumentBytes = 10 << 20

// AllowedDocumentTypes are the file extensions that can be attached, all text-like documents.
var AllowedDocumentTypes = []string{".txt", ".pdf", ".md", ".docx", ".doc", ".rtf", ".odt"}

// Document is a file attached to a job.
type Document struct {
	ID       int64        `json:"id"`
	JobID    int64        `json:"jobId"`
	Kind     DocumentKind `json:"kind"`
	FileName string       `json:"fileName"`
}

func documentFromRow(row store.Document) Document {
	return Document{ID: row.ID, JobID: row.JobID, Kind: DocumentKind(row.Kind), FileName: row.FileName}
}

// AttachDocument attaches a copy of the file at src to a job. The copy is the app's own, so it
// keeps working if the original is moved or deleted. See
// docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
// @when [features/job-documents.feature] I attach the file "<file>" as a <kind>
// @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume
// @and [features/job-documents.feature] I attach "acme-cover.docx" as a cover letter
// @and [features/job-documents.feature] I attach "take-home-notes.md" as other
// @when [features/job-documents.feature] I attach the file "<file>" as other
// @when [features/job-documents.feature] I try to attach the file "<file>"
// @when [features/job-documents.feature] I try to attach a 12 MB file "portfolio.pdf"
// @when [features/job-documents.feature] I attach a 10 MB file "portfolio.pdf" as other
// @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume to both jobs
func (s *Service) AttachDocument(ctx context.Context, jobID int64, src string, kind DocumentKind) (Document, error) {
	if !kind.valid() {
		return Document{}, invalid("choose whether the document is a resume, a cover letter, or other")
	}
	if !allowedType(src) {
		// @then [features/job-documents.feature] the file is not attached
		// @and [features/job-documents.feature] I am told which document types are allowed
		return Document{}, invalid("only these document types can be attached: " + strings.Join(AllowedDocumentTypes, ", "))
	}
	size, err := s.files.Size(src)
	if err != nil {
		return Document{}, err
	}
	if size > MaxDocumentBytes {
		// @then [features/job-documents.feature] the file is not attached
		// @and [features/job-documents.feature] I am told files can be at most 10 MB
		return Document{}, invalid("files can be at most 10 MB")
	}
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme" and a job for "Developer" at "Globex"
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return Document{}, notFound(err, ErrJobNotFound)
	}

	// The file is copied before its row is written, so a failure leaves at worst an orphaned copy,
	// which is removed below, never a row pointing at a missing file.
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	stored, err := s.files.Save(src)
	if err != nil {
		return Document{}, err
	}
	// @then [features/job-documents.feature] the job lists "<file>" as a <kind>
	// @then [features/job-documents.feature] the job lists "<file>" as other
	// @then [features/job-documents.feature] the job lists "portfolio.pdf" as other
	// @then [features/job-documents.feature] the job lists "RESUME.PDF" as other
	// @then [features/job-documents.feature] both jobs list "resume-2026.pdf" as a resume
	row, err := s.store.CreateDocument(ctx, store.CreateDocumentParams{
		JobID:      jobID,
		Kind:       string(kind),
		FileName:   filepath.Base(src),
		StoredPath: stored,
	})
	if err != nil {
		if rmErr := s.files.Remove(stored); rmErr != nil {
			return Document{}, errors.Join(err, rmErr)
		}
		return Document{}, err
	}
	return documentFromRow(row), nil
}

// allowedType reports whether the file's extension is one of AllowedDocumentTypes, ignoring case.
func allowedType(src string) bool {
	// @when [features/job-documents.feature] I attach the file "RESUME.PDF" as other
	return slices.Contains(AllowedDocumentTypes, strings.ToLower(filepath.Ext(src)))
}

// SetDocumentKind changes what kind of document an attachment is.
// @when [features/job-documents.feature] I change its kind to cover letter
func (s *Service) SetDocumentKind(ctx context.Context, id int64, kind DocumentKind) error {
	if !kind.valid() {
		return invalid("choose whether the document is a resume, a cover letter, or other")
	}
	// @given [features/job-documents.feature] I have attached "acme-cover.docx" as other to a job for "Software Engineer" at "Acme"
	if _, err := s.store.GetDocument(ctx, id); err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	// @then [features/job-documents.feature] the job lists "acme-cover.docx" as a cover letter
	return s.store.SetDocumentKind(ctx, store.SetDocumentKindParams{ID: id, Kind: string(kind)})
}

// RemoveDocument removes an attachment from its job and deletes the app's copy. The file the
// user originally attached is never touched.
// @when [features/job-documents.feature] I remove the attached "resume-2026.pdf"
func (s *Service) RemoveDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	// The row goes first, so a failure leaves an orphaned copy rather than a row pointing at a
	// missing file.
	// @then [features/job-documents.feature] the job no longer lists "resume-2026.pdf"
	if err := s.store.DeleteDocument(ctx, id); err != nil {
		return err
	}
	// Each attachment has its own copy, so this deletes nothing another attachment uses.
	// @and [features/job-documents.feature] the original "resume-2026.pdf" on my computer is not deleted
	// @and [features/job-documents.feature] removing it from one job leaves it attached to the other
	if err := s.files.Remove(doc.StoredPath); err != nil {
		return fmt.Errorf("the document was removed, but its file couldn't be deleted: %w", err)
	}
	return nil
}

// OpenDocument opens an attachment's stored copy in the OS default app for its file type.
// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
func (s *Service) OpenDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	// Opens the app's copy, never the original, which may have moved.
	// @and [features/job-documents.feature] I have since moved or deleted the original "resume-2026.pdf" on my computer
	// @then [features/job-documents.feature] the document opens with the content it had when I attached it
	// @then [features/job-documents.feature] it opens in my computer's default app for that file type
	return s.opener.Open(s.files.Path(doc.StoredPath))
}
