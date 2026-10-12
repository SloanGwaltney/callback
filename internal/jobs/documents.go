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
func (s *Service) AttachDocument(ctx context.Context, jobID int64, src string, kind DocumentKind) (Document, error) {
	if !kind.valid() {
		return Document{}, invalid("choose whether the document is a resume, a cover letter, or other")
	}
	if !allowedType(src) {
		return Document{}, invalid("only these document types can be attached: " + strings.Join(AllowedDocumentTypes, ", "))
	}
	size, err := s.files.Size(src)
	if err != nil {
		return Document{}, err
	}
	if size > MaxDocumentBytes {
		return Document{}, invalid("files can be at most 10 MB")
	}
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return Document{}, notFound(err, ErrJobNotFound)
	}

	// The file is copied before its row is written, so a failure leaves at worst an orphaned copy,
	// which is removed below, never a row pointing at a missing file.
	stored, err := s.files.Save(src)
	if err != nil {
		return Document{}, err
	}
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
	return slices.Contains(AllowedDocumentTypes, strings.ToLower(filepath.Ext(src)))
}

// SetDocumentKind changes what kind of document an attachment is.
func (s *Service) SetDocumentKind(ctx context.Context, id int64, kind DocumentKind) error {
	if !kind.valid() {
		return invalid("choose whether the document is a resume, a cover letter, or other")
	}
	if _, err := s.store.GetDocument(ctx, id); err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	return s.store.SetDocumentKind(ctx, store.SetDocumentKindParams{ID: id, Kind: string(kind)})
}

// RemoveDocument removes an attachment from its job and deletes the app's copy. The file the
// user originally attached is never touched.
func (s *Service) RemoveDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	// The row goes first, so a failure leaves an orphaned copy rather than a row pointing at a
	// missing file.
	if err := s.store.DeleteDocument(ctx, id); err != nil {
		return err
	}
	// Each attachment has its own copy, so this deletes nothing another attachment uses.
	if err := s.files.Remove(doc.StoredPath); err != nil {
		return fmt.Errorf("the document was removed, but its file couldn't be deleted: %w", err)
	}
	return nil
}

// OpenDocument opens an attachment's stored copy in the OS default app for its file type.
func (s *Service) OpenDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	// Opens the app's copy, never the original, which may have moved.
	return s.opener.Open(s.files.Path(doc.StoredPath))
}
