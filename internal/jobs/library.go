package jobs

import (
	"context"
	"path/filepath"
	"strings"

	"callback/internal/store"
)

// LibraryDocument is a document kept in the document library, for use across the job search
// rather than on one job.
type LibraryDocument struct {
	ID       int64  `json:"id"`
	FileName string `json:"fileName"`
	// Description is "" when the document has none.
	Description string `json:"description"`
}

func libraryDocumentFromRow(row store.LibraryDocument) LibraryDocument {
	return LibraryDocument{ID: row.ID, FileName: row.FileName, Description: row.Description}
}

// Library returns every document in the library, newest first. Library documents are never
// attached to a job, and a job's documents are never in the library.
func (s *Service) Library(ctx context.Context) ([]LibraryDocument, error) {
	rows, err := s.store.ListLibraryDocuments(ctx)
	if err != nil {
		return nil, err
	}
	docs := make([]LibraryDocument, len(rows))
	for i, row := range rows {
		docs[i] = libraryDocumentFromRow(row)
	}
	return docs, nil
}

// LibraryDocument returns one document in the library.
func (s *Service) LibraryDocument(ctx context.Context, id int64) (LibraryDocument, error) {
	row, err := s.store.GetLibraryDocument(ctx, id)
	if err != nil {
		return LibraryDocument{}, notFound(err, ErrLibraryDocumentNotFound)
	}
	return libraryDocumentFromRow(row), nil
}

// LibraryFiles returns where the stored copy of each library document is, in the order Library
// lists them, and whether it's missing. The agent command line shows them, since an agent reads a
// document straight from its stored copy.
func (s *Service) LibraryFiles(ctx context.Context) ([]DocumentFile, error) {
	rows, err := s.store.ListLibraryDocuments(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]DocumentFile, 0, len(rows))
	for _, row := range rows {
		f, err := s.documentFile(row.ID, row.StoredPath)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

// AddLibraryDocument adds a copy of the file at src to the library, with an optional description.
// The copy is the app's own, so it keeps working if the original is moved or deleted, and adding
// the same file again makes another copy. See
// docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
func (s *Service) AddLibraryDocument(ctx context.Context, src, description string) (LibraryDocument, error) {
	if err := s.checkDocumentFile(src, "added"); err != nil {
		return LibraryDocument{}, err
	}

	var row store.LibraryDocument
	err := s.saveCopy(src, func(stored string) (err error) {
		row, err = s.store.CreateLibraryDocument(ctx, store.CreateLibraryDocumentParams{
			FileName:    filepath.Base(src),
			Description: strings.TrimSpace(description),
			StoredPath:  stored,
		})
		return err
	})
	if err != nil {
		return LibraryDocument{}, err
	}
	return libraryDocumentFromRow(row), nil
}

// SetLibraryDescription changes a library document's description, or clears it when description
// is blank.
func (s *Service) SetLibraryDescription(ctx context.Context, id int64, description string) (LibraryDocument, error) {
	row, err := s.store.GetLibraryDocument(ctx, id)
	if err != nil {
		return LibraryDocument{}, notFound(err, ErrLibraryDocumentNotFound)
	}
	row.Description = strings.TrimSpace(description)
	err = s.store.SetLibraryDocumentDescription(ctx, store.SetLibraryDocumentDescriptionParams{
		ID: id, Description: row.Description,
	})
	if err != nil {
		return LibraryDocument{}, err
	}
	return libraryDocumentFromRow(row), nil
}

// RemoveLibraryDocument removes a document from the library and deletes the app's copy. The file
// the user originally added is never touched, and neither are jobs' documents, which have their
// own copies.
func (s *Service) RemoveLibraryDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetLibraryDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrLibraryDocumentNotFound)
	}
	return s.removeCopy(doc.StoredPath, func() error { return s.store.DeleteLibraryDocument(ctx, id) })
}

// OpenLibraryDocument opens a library document's stored copy in the OS default app for its file
// type. If the copy is gone from the data directory, it returns a MissingFileError and keeps the
// document in the library.
func (s *Service) OpenLibraryDocument(ctx context.Context, id int64) error {
	doc, err := s.store.GetLibraryDocument(ctx, id)
	if err != nil {
		return notFound(err, ErrLibraryDocumentNotFound)
	}
	return s.openCopy(doc.FileName, doc.StoredPath)
}
