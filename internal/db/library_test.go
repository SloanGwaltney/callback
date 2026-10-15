package db

import (
	"context"
	"slices"
	"testing"

	"callback/internal/store"
)

// Storage-tier tests for the library_documents table and its queries.

func createLibraryDocument(t *testing.T, q store.Querier, fileName, description string) store.LibraryDocument {
	t.Helper()
	doc, err := q.CreateLibraryDocument(context.Background(), store.CreateLibraryDocumentParams{
		FileName: fileName, Description: description, StoredPath: "documents/" + fileName,
	})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func libraryFileNames(t *testing.T, q store.Querier) []string {
	t.Helper()
	docs, err := q.ListLibraryDocuments(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range docs {
		names = append(names, d.FileName)
	}
	return names
}

func TestSchemaLibraryDocuments(t *testing.T) {
	conn := openSchema(t)

	mustExec(t, conn, `INSERT INTO library_documents (file_name, stored_path) VALUES ('resume-base-2026.docx', 'documents/a.docx')`)
	var description string
	if err := conn.QueryRow(`SELECT description FROM library_documents`).Scan(&description); err != nil {
		t.Fatal(err)
	}
	if description != "" {
		t.Errorf("description defaults to %q, want ''", description)
	}

	for _, tt := range []struct{ name, query string }{
		{"empty file name", `INSERT INTO library_documents (file_name, stored_path) VALUES ('', 'documents/b.docx')`},
		{"missing stored path", `INSERT INTO library_documents (file_name) VALUES ('cover-template.md')`},
		{"reused stored path", `INSERT INTO library_documents (file_name, stored_path) VALUES ('copy.docx', 'documents/a.docx')`},
	} {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			if _, err := conn.Exec(tt.query); err == nil {
				t.Error("insert succeeded, want a constraint error")
			}
		})
	}
}

func TestLibraryQueries(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	resume := createLibraryDocument(t, s, "resume-base-2026.docx", "Base resume")
	cover := createLibraryDocument(t, s, "cover-template.md", "")

	got, err := s.GetLibraryDocument(ctx, resume.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != resume {
		t.Errorf("GetLibraryDocument = %+v, want %+v", got, resume)
	}

	if err := s.SetLibraryDocumentDescription(ctx, store.SetLibraryDocumentDescriptionParams{ID: resume.ID, Description: "Base resume, backend roles"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetLibraryDocument(ctx, resume.ID); got.Description != "Base resume, backend roles" {
		t.Errorf("description = %q after setting it", got.Description)
	}
	if got, _ := s.GetLibraryDocument(ctx, cover.ID); got.Description != "" {
		t.Errorf("setting one description changed another's to %q", got.Description)
	}

	if err := s.DeleteLibraryDocument(ctx, resume.ID); err != nil {
		t.Fatal(err)
	}
	if got := libraryFileNames(t, s); !slices.Equal(got, []string{"cover-template.md"}) {
		t.Errorf("after delete, the library lists %v", got)
	}
}

// @scenario [#library-newest-first]
func TestListLibraryDocumentsIsNewestFirst(t *testing.T) {
	s, _ := openStore(t, t.TempDir())

	// @given [features/document-library.feature] I added "resume-base-2026.docx" to the library
	createLibraryDocument(t, s, "resume-base-2026.docx", "")
	// @and [features/document-library.feature] I then added "cover-template.md"
	createLibraryDocument(t, s, "cover-template.md", "")
	// @and [features/document-library.feature] I then added "portfolio-links.txt"
	createLibraryDocument(t, s, "portfolio-links.txt", "")

	// @when [features/document-library.feature] I go to the document library
	got := libraryFileNames(t, s)

	// @then [features/document-library.feature] the library lists "portfolio-links.txt", "cover-template.md", and "resume-base-2026.docx" in that order
	want := []string{"portfolio-links.txt", "cover-template.md", "resume-base-2026.docx"}
	if !slices.Equal(got, want) {
		t.Errorf("the library lists %v, want %v", got, want)
	}
}

// The command line prints the library in this order; internal/cli's TestAgentListsTheLibrary
// checks what it prints for each document.
// @scenario [#cli-list-library]
func TestAgentListsTheLibraryNewestFirst(t *testing.T) {
	s, _ := openStore(t, t.TempDir())

	// @given [features/agent-cli-library.feature] my library holds "resume-2026.pdf" described as "General resume"
	createLibraryDocument(t, s, "resume-2026.pdf", "General resume")
	// @and [features/agent-cli-library.feature] my library holds "cover-letter.docx" with no description
	createLibraryDocument(t, s, "cover-letter.docx", "")

	// @when [features/agent-cli-library.feature] my agent lists the library
	got := libraryFileNames(t, s)

	// @then [features/agent-cli-library.feature] it sees both documents, newest first
	if want := []string{"cover-letter.docx", "resume-2026.pdf"}; !slices.Equal(got, want) {
		t.Errorf("the library lists %v, want %v", got, want)
	}
}

func TestNewestFirstAfterRemovingTheNewest(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	createLibraryDocument(t, s, "resume-base-2026.docx", "")
	cover := createLibraryDocument(t, s, "cover-template.md", "")
	if err := s.DeleteLibraryDocument(ctx, cover.ID); err != nil {
		t.Fatal(err)
	}
	createLibraryDocument(t, s, "portfolio-links.txt", "")

	want := []string{"portfolio-links.txt", "resume-base-2026.docx"}
	if got := libraryFileNames(t, s); !slices.Equal(got, want) {
		t.Errorf("the library lists %v, want %v", got, want)
	}
}

// @scenario [#library-persists]
func TestLibraryPersistsAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume"
	createLibraryDocument(t, s, "resume-base-2026.docx", "Base resume")

	// @when [features/document-library.feature] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	// @and [features/document-library.feature] I go to the document library
	docs, err := reopened.ListLibraryDocuments(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume"
	if len(docs) != 1 || docs[0].FileName != "resume-base-2026.docx" || docs[0].Description != "Base resume" {
		t.Errorf("after reopening, the library lists %+v", docs)
	}
}

// @scenario [#delete-job-keeps-library]
func TestDeleteJobKeepsLibraryRows(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	createLibraryDocument(t, s, "resume-base-2026.docx", "")
	// @and [features/document-library.feature] I have attached "resume-base-2026.docx" to a job for "Software Engineer" at "Acme"
	acme := createJob(t, s, "Acme", "saved", 0)
	if _, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: acme, Kind: "resume", FileName: "resume-base-2026.docx", StoredPath: "documents/acme.docx"}); err != nil {
		t.Fatal(err)
	}

	// @when [features/document-library.feature] I delete the job
	if err := s.InTx(ctx, func(q store.Querier) error { return q.DeleteJob(ctx, acme) }); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature#delete-job-keeps-library] the library still lists "resume-base-2026.docx"
	if got := libraryFileNames(t, s); !slices.Equal(got, []string{"resume-base-2026.docx"}) {
		t.Errorf("after deleting the job, the library lists %v", got)
	}
}

// @scenario [#remove-library-keeps-attachments]
func TestDeleteLibraryDocumentKeepsJobDocuments(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	lib := createLibraryDocument(t, s, "resume-base-2026.docx", "")
	// @and [features/document-library.feature] I have attached "resume-base-2026.docx" to a job for "Software Engineer" at "Acme"
	acme := createJob(t, s, "Acme", "saved", 0)
	if _, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: acme, Kind: "resume", FileName: "resume-base-2026.docx", StoredPath: "documents/acme.docx"}); err != nil {
		t.Fatal(err)
	}

	// @when [features/document-library.feature] I remove "resume-base-2026.docx" from the library
	if err := s.DeleteLibraryDocument(ctx, lib.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] the job still lists "resume-base-2026.docx"
	docs, err := s.ListDocumentsForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].FileName != "resume-base-2026.docx" {
		t.Errorf("after removing it from the library, the job lists %+v", docs)
	}
}
