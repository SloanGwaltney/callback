package jobs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"callback/internal/store"
)

// MaxDocumentBytes is the largest file that can be attached or added to the library: 10 MB.
const MaxDocumentBytes = 10 << 20

// AllowedDocumentTypes are the file extensions that can be attached or added to the library, all
// text-like documents.
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

// MissingFileError means an attachment's or library document's stored copy is no longer in the
// data directory. Its message is written to be shown to the user as is.
type MissingFileError struct {
	FileName string
}

func (e *MissingFileError) Error() string {
	return fmt.Sprintf("the file for %q can't be found", e.FileName)
}

// Document returns one attached document.
func (s *Service) Document(ctx context.Context, id int64) (Document, error) {
	row, err := s.store.GetDocument(ctx, id)
	if err != nil {
		return Document{}, notFound(err, ErrDocumentNotFound)
	}
	return documentFromRow(row), nil
}

// AttachDocument attaches a copy of the file at src to a job. The copy is the app's own, so it
// keeps working if the original is moved or deleted. See
// docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
func (s *Service) AttachDocument(ctx context.Context, jobID int64, src string, kind DocumentKind) (Document, error) {
	if !kind.valid() {
		return Document{}, invalid("choose whether the document is a resume, a cover letter, or other")
	}
	if err := s.checkDocumentFile(src, "attached"); err != nil {
		return Document{}, err
	}
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return Document{}, notFound(err, ErrJobNotFound)
	}

	var row store.Document
	err := s.saveCopy(src, func(stored string) (err error) {
		row, err = s.store.CreateDocument(ctx, store.CreateDocumentParams{
			JobID:      jobID,
			Kind:       string(kind),
			FileName:   filepath.Base(src),
			StoredPath: stored,
		})
		return err
	})
	if err != nil {
		return Document{}, err
	}
	return documentFromRow(row), nil
}

// checkDocumentFile checks that the file at src can be attached to a job or added to the library:
// one of AllowedDocumentTypes, and at most MaxDocumentBytes. verb says which, for the message.
// The app only offers files that exist, but the agent command line takes any path.
func (s *Service) checkDocumentFile(src, verb string) error {
	if !allowedType(src) {
		return invalid("only these document types can be " + verb + ": " + strings.Join(AllowedDocumentTypes, ", "))
	}
	size, err := s.files.Size(src)
	if errors.Is(err, fs.ErrNotExist) {
		return invalid("there's no file at " + src)
	}
	if err != nil {
		return err
	}
	if size > MaxDocumentBytes {
		return invalid("files can be at most 10 MB")
	}
	return nil
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
// user originally attached is never touched. A copy that's already gone from the data directory
// doesn't stop the removal.
func (s *Service) RemoveDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	// Each attachment has its own copy, so this deletes nothing another attachment uses.
	return s.removeCopy(doc.StoredPath, func() error { return s.store.DeleteDocument(ctx, id) })
}

// DocumentFile is where an attachment's stored copy is.
type DocumentFile struct {
	DocumentID int64
	// Path is the stored copy's absolute path.
	Path string
	// Missing is set when the stored copy is gone from the data directory.
	Missing bool
}

// JobDocumentFiles returns where the stored copy of each of a job's attachments is, in the order
// Job lists them, and whether it's missing. The agent command line shows them, since an agent
// reads a document straight from its stored copy.
func (s *Service) JobDocumentFiles(ctx context.Context, jobID int64) ([]DocumentFile, error) {
	if _, err := s.store.GetJob(ctx, jobID); err != nil {
		return nil, notFound(err, ErrJobNotFound)
	}
	docs, err := s.store.ListDocumentsForJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	files := make([]DocumentFile, 0, len(docs))
	for _, doc := range docs {
		f, err := s.documentFile(doc.ID, doc.StoredPath)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

// documentFile returns where the stored copy of the attachment or library document id is, and
// whether it's missing.
func (s *Service) documentFile(id int64, stored string) (DocumentFile, error) {
	ok, err := s.files.Exists(stored)
	if err != nil {
		return DocumentFile{}, err
	}
	return DocumentFile{DocumentID: id, Path: s.files.Path(stored), Missing: !ok}, nil
}

// OpenDocument opens an attachment's stored copy in the OS default app for its file type. If the
// copy is gone from the data directory, it returns a MissingFileError and keeps the attachment on
// its job.
func (s *Service) OpenDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrDocumentNotFound)
	}
	return s.openCopy(doc.FileName, doc.StoredPath)
}

// saveCopy copies the file at src, then has create write its row with the copy's stored path. The
// file is copied before its row is written, so a failure leaves at worst an orphaned copy, which is
// removed, never a row pointing at a missing file.
func (s *Service) saveCopy(src string, create func(stored string) error) error {
	stored, err := s.files.Save(src)
	if err != nil {
		return err
	}
	if err := create(stored); err != nil {
		if rmErr := s.files.Remove(stored); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return err
	}
	return nil
}

// removeCopy has deleteRow delete a document's row, then deletes its stored copy. The row goes
// first, so a failure leaves an orphaned copy rather than a row pointing at a missing file.
func (s *Service) removeCopy(stored string, deleteRow func() error) error {
	if err := deleteRow(); err != nil {
		return err
	}
	if err := s.files.Remove(stored); err != nil {
		return fmt.Errorf("the document was removed, but its file couldn't be deleted: %w", err)
	}
	return nil
}

// openCopy opens a document's stored copy in the OS default app for its file type, or returns a
// MissingFileError if the copy is gone from the data directory. It opens the app's copy, never the
// original, which may have moved.
func (s *Service) openCopy(fileName, stored string) error {
	ok, err := s.files.Exists(stored)
	if err != nil {
		return err
	}
	if !ok {
		return &MissingFileError{FileName: fileName}
	}
	return s.opener.Open(s.files.Path(stored))
}
