package jobs

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// libraryListing returns what the library lists, newest first, as "file: description".
func libraryListing(t *testing.T, h *harness) []string {
	t.Helper()
	docs, err := h.svc.Library(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range docs {
		out = append(out, d.FileName+": "+d.Description)
	}
	return out
}

func mustAddToLibrary(t *testing.T, h *harness, src, description string) LibraryDocument {
	t.Helper()
	doc, err := h.svc.AddLibraryDocument(h.ctx, src, description)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// @scenario [#add-library-document]
func TestAddLibraryDocument(t *testing.T) {
	h := newHarness(t)

	// @when [features/document-library.feature] I add the file "resume-base-2026.docx" with the description "Base resume, backend roles"
	doc := mustAddToLibrary(t, h, filepath.Join("Documents", "resume-base-2026.docx"), "Base resume, backend roles")

	// @then [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume, backend roles"
	want := []string{"resume-base-2026.docx: Base resume, backend roles"}
	if got := libraryListing(t, h); !slices.Equal(got, want) {
		t.Errorf("the library lists %v, want %v", got, want)
	}
	if doc.FileName != "resume-base-2026.docx" || doc.Description != "Base resume, backend roles" {
		t.Errorf("returned %+v", doc)
	}
	if len(h.files.copies) != 1 {
		t.Errorf("copies = %v, want one", h.files.copies)
	}
}

// @scenario [#add-library-document-no-description]
func TestAddLibraryDocumentWithoutDescription(t *testing.T) {
	h := newHarness(t)

	// @when [features/document-library.feature] I add the file "cover-template.md" with no description
	mustAddToLibrary(t, h, "cover-template.md", "")

	// @then [features/document-library.feature] the library lists "cover-template.md" with no description
	if got := libraryListing(t, h); !slices.Equal(got, []string{"cover-template.md: "}) {
		t.Errorf("the library lists %v", got)
	}
}

func TestAddLibraryDocumentTrimsTheDescription(t *testing.T) {
	h := newHarness(t)
	mustAddToLibrary(t, h, "cover-template.md", "  Template  ")
	mustAddToLibrary(t, h, "notes.txt", "   ")
	want := []string{"notes.txt: ", "cover-template.md: Template"}
	if got := libraryListing(t, h); !slices.Equal(got, want) {
		t.Errorf("the library lists %v, want %v", got, want)
	}
}

// @scenario [#library-newest-first]
func TestLibraryIsNewestFirst(t *testing.T) {
	h := newHarness(t)
	mustAddToLibrary(t, h, "resume-base-2026.docx", "")
	mustAddToLibrary(t, h, "cover-template.md", "")
	mustAddToLibrary(t, h, "portfolio-links.txt", "")

	// The order itself comes from ListLibraryDocuments, proven against SQLite by
	// TestListLibraryDocumentsIsNewestFirst in internal/db; the service keeps it.
	want := []string{"portfolio-links.txt: ", "cover-template.md: ", "resume-base-2026.docx: "}
	if got := libraryListing(t, h); !slices.Equal(got, want) {
		t.Errorf("the library lists %v, want %v", got, want)
	}
}

// @scenario [#library-same-file-twice]
func TestAddSameFileToLibraryTwice(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Before the rewrite"
	mustAddToLibrary(t, h, "resume-base-2026.docx", "Before the rewrite")

	// @when [features/document-library.feature] I add "resume-base-2026.docx" again with the description "After the rewrite"
	mustAddToLibrary(t, h, "resume-base-2026.docx", "After the rewrite")

	// @then [features/document-library.feature] the library lists "resume-base-2026.docx" twice, once with each description
	want := []string{"resume-base-2026.docx: After the rewrite", "resume-base-2026.docx: Before the rewrite"}
	if got := libraryListing(t, h); !slices.Equal(got, want) {
		t.Errorf("the library lists %v, want %v", got, want)
	}
	if len(h.files.copies) != 2 {
		t.Errorf("copies = %v, want one for each", h.files.copies)
	}
}

// @scenario [#library-document-kept-if-original-moved]
func TestLibraryDocumentOpensItsOwnCopy(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] I added "resume-base-2026.docx" to the library
	doc := mustAddToLibrary(t, h, filepath.Join("Downloads", "resume-base-2026.docx"), "")
	stored := h.store.library[doc.ID].StoredPath
	// @and [features/document-library.feature] I have since moved or deleted the original "resume-base-2026.docx" on my computer
	// (The original's path is never read again after adding; the storage test in internal/docfiles
	// deletes a real original and reads the copy.)

	// @when [features/document-library.feature] I open "resume-base-2026.docx" from the library
	if err := h.svc.OpenLibraryDocument(h.ctx, doc.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] the document opens with the content it had when I added it
	if want := []string{h.files.Path(stored)}; !slices.Equal(h.opener.opened, want) {
		t.Errorf("opened %v, want the stored copy %v", h.opener.opened, want)
	}
}

// @scenario [#library-allowed-file-types]
func TestLibraryAllowedFileTypes(t *testing.T) {
	for _, file := range []string{"notes.txt", "resume.pdf", "prep.md", "cover.docx", "cover.doc", "resume.rtf", "resume.odt"} {
		t.Run(file, func(t *testing.T) {
			h := newHarness(t)

			// @when [features/document-library.feature] I add the file "<file>"
			mustAddToLibrary(t, h, file, "")

			// @then [features/document-library.feature] the library lists "<file>"
			if got := libraryListing(t, h); !slices.Equal(got, []string{file + ": "}) {
				t.Errorf("the library lists %v", got)
			}
		})
	}
}

// @scenario [#library-file-type-any-case]
func TestLibraryFileTypeAnyCase(t *testing.T) {
	h := newHarness(t)

	// @when [features/document-library.feature] I add the file "RESUME.PDF"
	mustAddToLibrary(t, h, "RESUME.PDF", "")

	// @then [features/document-library.feature] the library lists "RESUME.PDF"
	if got := libraryListing(t, h); !slices.Equal(got, []string{"RESUME.PDF: "}) {
		t.Errorf("the library lists %v", got)
	}
}

// @scenario [#library-disallowed-file-types]
func TestLibraryDisallowedFileTypes(t *testing.T) {
	for _, file := range []string{"headshot.png", "take-home.zip", "setup.exe", "salaries.xlsx", "no-extension"} {
		t.Run(file, func(t *testing.T) {
			h := newHarness(t)

			// @when [features/document-library.feature] I try to add the file "<file>"
			_, err := h.svc.AddLibraryDocument(h.ctx, file, "")

			// @then [features/document-library.feature] the file is not added
			if len(h.store.library) != 0 || len(h.files.copies) != 0 {
				t.Errorf("added: rows %v, copies %v", h.store.library, h.files.copies)
			}
			// @and [features/document-library.feature] I am told which document types are allowed
			wantValidation(t, err, "only these document types can be added: .txt, .pdf, .md, .docx, .doc, .rtf, .odt")
		})
	}
}

// @scenario [#library-file-size-limit]
func TestLibraryFileSizeLimit(t *testing.T) {
	h := newHarness(t)

	// @when [features/document-library.feature] I try to add a 12 MB file "portfolio.pdf"
	h.files.sizes["portfolio.pdf"] = 12 << 20
	_, err := h.svc.AddLibraryDocument(h.ctx, "portfolio.pdf", "")

	// @then [features/document-library.feature] the file is not added
	if len(h.store.library) != 0 || len(h.files.copies) != 0 {
		t.Errorf("added: rows %v, copies %v", h.store.library, h.files.copies)
	}
	// @and [features/document-library.feature] I am told files can be at most 10 MB
	wantValidation(t, err, "files can be at most 10 MB")
}

// @scenario [#library-file-size-at-limit]
func TestLibraryFileSizeAtLimit(t *testing.T) {
	h := newHarness(t)

	// @when [features/document-library.feature] I add a 10 MB file "portfolio.pdf"
	h.files.sizes["portfolio.pdf"] = 10 << 20
	mustAddToLibrary(t, h, "portfolio.pdf", "")

	// @then [features/document-library.feature] the library lists "portfolio.pdf"
	if got := libraryListing(t, h); !slices.Equal(got, []string{"portfolio.pdf: "}) {
		t.Errorf("the library lists %v", got)
	}
}

func TestLibraryOneByteOverTheLimitIsRejected(t *testing.T) {
	h := newHarness(t)
	h.files.sizes["portfolio.pdf"] = 10<<20 + 1
	if _, err := h.svc.AddLibraryDocument(h.ctx, "portfolio.pdf", ""); err == nil {
		t.Error("added a file one byte over 10 MB")
	}
}

func TestAddLibraryDocumentRemovesTheCopyIfTheRowFails(t *testing.T) {
	h := newHarness(t)
	h.store.fail = "CreateLibraryDocument"

	if _, err := h.svc.AddLibraryDocument(h.ctx, "resume.pdf", ""); !errors.Is(err, errFake) {
		t.Fatalf("err = %v, want errFake", err)
	}
	if len(h.files.copies) != 0 {
		t.Errorf("orphaned copies: %v", h.files.copies)
	}
}

// @scenario [#open-library-document]
func TestOpenLibraryDocument(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	doc := h.seedLibraryDoc("resume-base-2026.docx", "")

	// @when [features/document-library.feature] I open "resume-base-2026.docx" from the library
	if err := h.svc.OpenLibraryDocument(h.ctx, doc); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] it opens in my computer's default app for that file type
	if len(h.opener.opened) != 1 || !strings.HasSuffix(h.opener.opened[0], ".docx") {
		t.Errorf("opened %v, want the stored .docx copy", h.opener.opened)
	}
}

// @scenario [#open-missing-library-document]
func TestOpenMissingLibraryDocument(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	doc := h.seedLibraryDoc("resume-base-2026.docx", "")
	// @and [features/document-library.feature] its stored copy is no longer in the app's data directory
	delete(h.files.copies, h.store.library[doc].StoredPath)

	// @when [features/document-library.feature] I open "resume-base-2026.docx" from the library
	err := h.svc.OpenLibraryDocument(h.ctx, doc)

	// @then [features/document-library.feature] I am told the file for "resume-base-2026.docx" can't be found
	if err == nil || err.Error() != `the file for "resume-base-2026.docx" can't be found` {
		t.Errorf("err = %v", err)
	}
	if len(h.opener.opened) != 0 {
		t.Errorf("opened %v", h.opener.opened)
	}
	// @and [features/document-library.feature#open-missing-library-document] the library still lists "resume-base-2026.docx"
	if got := libraryListing(t, h); !slices.Equal(got, []string{"resume-base-2026.docx: "}) {
		t.Errorf("the library lists %v", got)
	}
}

// @scenario [#edit-library-description]
func TestEditLibraryDescription(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume"
	doc := h.seedLibraryDoc("resume-base-2026.docx", "Base resume")

	// @when [features/document-library.feature] I change its description to "Base resume, backend roles"
	got, err := h.svc.SetLibraryDescription(h.ctx, doc, "Base resume, backend roles")
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume, backend roles"
	want := []string{"resume-base-2026.docx: Base resume, backend roles"}
	if listing := libraryListing(t, h); !slices.Equal(listing, want) {
		t.Errorf("the library lists %v, want %v", listing, want)
	}
	if got.Description != "Base resume, backend roles" {
		t.Errorf("returned %+v", got)
	}
}

// @scenario [#clear-library-description]
func TestClearLibraryDescription(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx" with the description "Base resume"
	doc := h.seedLibraryDoc("resume-base-2026.docx", "Base resume")

	// @when [features/document-library.feature] I clear its description
	if _, err := h.svc.SetLibraryDescription(h.ctx, doc, "  "); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] the library lists "resume-base-2026.docx" with no description
	if got := libraryListing(t, h); !slices.Equal(got, []string{"resume-base-2026.docx: "}) {
		t.Errorf("the library lists %v", got)
	}
}

func TestLibraryDocumentNotFound(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.SetLibraryDescription(h.ctx, 99, "x"); !errors.Is(err, ErrLibraryDocumentNotFound) {
		t.Errorf("SetLibraryDescription err = %v, want ErrLibraryDocumentNotFound", err)
	}
	if err := h.svc.RemoveLibraryDocument(h.ctx, 99); !errors.Is(err, ErrLibraryDocumentNotFound) {
		t.Errorf("RemoveLibraryDocument err = %v, want ErrLibraryDocumentNotFound", err)
	}
	if err := h.svc.OpenLibraryDocument(h.ctx, 99); !errors.Is(err, ErrLibraryDocumentNotFound) {
		t.Errorf("OpenLibraryDocument err = %v, want ErrLibraryDocumentNotFound", err)
	}
}

// @scenario [#remove-library-document]
func TestRemoveLibraryDocument(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	original := filepath.Join("Documents", "resume-base-2026.docx")
	doc := mustAddToLibrary(t, h, original, "")
	stored := h.store.library[doc.ID].StoredPath

	// @when [features/document-library.feature] I remove "resume-base-2026.docx" from the library
	if err := h.svc.RemoveLibraryDocument(h.ctx, doc.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] the library no longer lists "resume-base-2026.docx"
	if got := libraryListing(t, h); len(got) != 0 {
		t.Errorf("the library lists %v", got)
	}
	// @and [features/document-library.feature] the original "resume-base-2026.docx" on my computer is not deleted
	if !slices.Equal(h.files.removed, []string{stored}) {
		t.Errorf("removed %v, want only the stored copy %q, never %q", h.files.removed, stored, original)
	}
}

func TestRemoveLibraryDocumentReportsAFileThatCouldNotBeDeleted(t *testing.T) {
	h := newHarness(t)
	doc := h.seedLibraryDoc("resume-base-2026.docx", "")
	h.files.failRemove = true

	if err := h.svc.RemoveLibraryDocument(h.ctx, doc); !errors.Is(err, errFake) {
		t.Errorf("err = %v, want errFake", err)
	}
	if got := libraryListing(t, h); len(got) != 0 {
		t.Errorf("the row stayed: %v", got)
	}
}

// @scenario [#library-not-on-jobs]
func TestLibraryDocumentsAreNotOnJobs(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	mustAddToLibrary(t, h, "resume-base-2026.docx", "")
	// @and [features/document-library.feature] I have a job for "Software Engineer" at "Acme" with no attached documents
	id := h.acme(StatusSaved)

	// @when [features/document-library.feature] I open the card
	job := h.job(t, id)

	// @then [features/document-library.feature] the job has no attached documents
	if len(job.Documents) != 0 {
		t.Errorf("the job lists %+v", job.Documents)
	}
	if card, _ := h.card(t, id); card.DocumentCount != 0 {
		t.Errorf("the card counts %d documents", card.DocumentCount)
	}
}

// @scenario [#delete-job-keeps-library]
func TestDeleteJobKeepsTheLibrary(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	lib := mustAddToLibrary(t, h, "resume-base-2026.docx", "")
	libCopy := h.store.library[lib.ID].StoredPath
	// @and [features/document-library.feature] I have attached "resume-base-2026.docx" to a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)
	mustAttach(t, h, id, "resume-base-2026.docx", KindResume)

	// @when [features/document-library.feature] I delete the job
	if err := h.svc.DeleteJob(h.ctx, id); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature#delete-job-keeps-library] the library still lists "resume-base-2026.docx"
	if got := libraryListing(t, h); !slices.Equal(got, []string{"resume-base-2026.docx: "}) {
		t.Errorf("the library lists %v", got)
	}
	if _, ok := h.files.copies[libCopy]; !ok || slices.Contains(h.files.removed, libCopy) {
		t.Errorf("the library's copy %q was deleted with the job", libCopy)
	}
}

// @scenario [#remove-library-keeps-attachments]
func TestRemoveLibraryDocumentKeepsAttachments(t *testing.T) {
	h := newHarness(t)
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	lib := mustAddToLibrary(t, h, "resume-base-2026.docx", "")
	// @and [features/document-library.feature] I have attached "resume-base-2026.docx" to a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)
	attached := mustAttach(t, h, id, "resume-base-2026.docx", KindResume)
	attachedCopy := h.store.docs[attached.ID].StoredPath

	// @when [features/document-library.feature] I remove "resume-base-2026.docx" from the library
	if err := h.svc.RemoveLibraryDocument(h.ctx, lib.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/document-library.feature] the job still lists "resume-base-2026.docx"
	if got := listed(t, h, id); !slices.Equal(got, []string{"resume-base-2026.docx as resume"}) {
		t.Errorf("the job lists %v", got)
	}
	if _, ok := h.files.copies[attachedCopy]; !ok {
		t.Errorf("the job's copy %q was deleted with the library's", attachedCopy)
	}
}

func TestLibraryDocumentLookup(t *testing.T) {
	h := newHarness(t)
	id := h.seedLibraryDoc("resume-2026.pdf", "General resume")

	got, err := h.svc.LibraryDocument(h.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if want := (LibraryDocument{ID: id, FileName: "resume-2026.pdf", Description: "General resume"}); got != want {
		t.Errorf("LibraryDocument = %+v, want %+v", got, want)
	}
	if _, err := h.svc.LibraryDocument(h.ctx, 999); !errors.Is(err, ErrLibraryDocumentNotFound) {
		t.Errorf("err = %v, want ErrLibraryDocumentNotFound", err)
	}
}

func TestLibraryFiles(t *testing.T) {
	h := newHarness(t)
	resume := h.seedLibraryDoc("resume-2026.pdf", "General resume")
	letter := h.seedLibraryDoc("cover-letter.docx", "")

	files, err := h.svc.LibraryFiles(h.ctx)
	if err != nil {
		t.Fatal(err)
	}

	// In the order Library lists them, newest first.
	want := []DocumentFile{
		{DocumentID: letter, Path: "/data/" + h.store.library[letter].StoredPath},
		{DocumentID: resume, Path: "/data/" + h.store.library[resume].StoredPath},
	}
	if !slices.Equal(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
}

// @scenario [#cli-list-library-missing]
func TestLibraryFilesMarksAMissingCopy(t *testing.T) {
	h := newHarness(t)
	// @given [features/agent-cli-library.feature] my library holds "resume-2026.pdf"
	id := h.seedLibraryDoc("resume-2026.pdf", "")
	// @and [features/agent-cli-library.feature] the stored copy of "resume-2026.pdf" is gone
	delete(h.files.copies, h.store.library[id].StoredPath)

	// @when [features/agent-cli-library.feature] my agent lists the library
	files, err := h.svc.LibraryFiles(h.ctx)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/agent-cli-library.feature] "resume-2026.pdf" is marked as missing its file
	if len(files) != 1 || files[0].DocumentID != id || !files[0].Missing {
		t.Errorf("files = %v, want %d marked missing", files, id)
	}
	if got := libraryListing(t, h); !slices.Equal(got, []string{"resume-2026.pdf: "}) {
		t.Errorf("the library lists %v, want the document kept", got)
	}
}

// @scenario [#cli-add-library-rules]
func TestAddingToTheLibraryFollowsTheRules(t *testing.T) {
	for _, tc := range []struct {
		file   string
		src    string
		setup  func(h *harness)
		reason string
	}{
		{"a file that doesn't exist", "Documents/gone.pdf", func(h *harness) { h.files.absent = []string{"Documents/gone.pdf"} },
			"there's no file at Documents/gone.pdf"},
		{"\"photo.png\"", "photo.png", func(*harness) {},
			"only these document types can be added: .txt, .pdf, .md, .docx, .doc, .rtf, .odt"},
		{"a PDF larger than 10 MB", "portfolio.pdf", func(h *harness) { h.files.sizes["portfolio.pdf"] = MaxDocumentBytes + 1 },
			"files can be at most 10 MB"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			h := newHarness(t)
			tc.setup(h)

			// @when [features/agent-cli-library.feature] my agent tries to add <file> to the library
			_, err := h.svc.AddLibraryDocument(h.ctx, tc.src, "")

			// @then [features/agent-cli-library.feature] the library has no new document
			if len(h.store.library) != 0 || len(h.files.copies) != 0 {
				t.Errorf("added: rows %v, copies %v", h.store.library, h.files.copies)
			}
			// @and [features/agent-cli-library.feature] my agent is told <reason>
			wantValidation(t, err, tc.reason)
		})
	}
}
