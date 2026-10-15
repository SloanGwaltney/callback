package jobs

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"callback/internal/store"
)

// kindFromSpec maps how the spec names a document kind to its DocumentKind.
var kindFromSpec = map[string]DocumentKind{"resume": KindResume, "cover letter": KindCoverLetter, "other": KindOther}

// listed returns the file names and kinds a job lists, as "file as kind".
func listed(t *testing.T, h *harness, jobID int64) []string {
	t.Helper()
	var out []string
	for _, d := range h.job(t, jobID).Documents {
		out = append(out, d.FileName+" as "+string(d.Kind))
	}
	return out
}

func mustAttach(t *testing.T, h *harness, jobID int64, src string, kind DocumentKind) Document {
	t.Helper()
	doc, err := h.svc.AttachDocument(h.ctx, jobID, src, kind)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// @scenario [#attach-document]
func TestAttachDocument(t *testing.T) {
	tests := []struct{ file, kind string }{
		{"resume-2026.pdf", "resume"},
		{"acme-cover.docx", "cover letter"},
		{"portfolio-links.txt", "other"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)

			// @when [features/job-documents.feature] I attach the file "<file>" as a <kind>
			mustAttach(t, h, id, filepath.Join("Documents", tt.file), kindFromSpec[tt.kind])

			// @then [features/job-documents.feature] the job lists "<file>" as a <kind>
			want := []string{tt.file + " as " + string(kindFromSpec[tt.kind])}
			if got := listed(t, h, id); !slices.Equal(got, want) {
				t.Errorf("job lists %v, want %v", got, want)
			}
		})
	}
}

// @scenario [#attach-multiple-documents]
func TestAttachSeveralDocuments(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume
	mustAttach(t, h, id, "resume-2026.pdf", KindResume)
	// @and [features/job-documents.feature] I attach "acme-cover.docx" as a cover letter
	mustAttach(t, h, id, "acme-cover.docx", KindCoverLetter)
	// @and [features/job-documents.feature] I attach "take-home-notes.md" as other
	mustAttach(t, h, id, "take-home-notes.md", KindOther)

	// @then [features/job-documents.feature] the job lists all three documents with their kinds
	want := []string{"resume-2026.pdf as resume", "acme-cover.docx as cover_letter", "take-home-notes.md as other"}
	if got := listed(t, h, id); !slices.Equal(got, want) {
		t.Errorf("job lists %v, want %v", got, want)
	}
}

// @scenario [#document-kept-if-original-moved]
func TestAttachedDocumentOpensItsOwnCopy(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	doc := mustAttach(t, h, id, filepath.Join("Downloads", "resume-2026.pdf"), KindResume)
	stored := h.store.docs[doc.ID].StoredPath
	// @and [features/job-documents.feature] I have since moved or deleted the original "resume-2026.pdf" on my computer
	// (The original's path is never read again after attaching; the storage test in
	// internal/docfiles deletes a real original and reads the copy.)

	// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
	if err := h.svc.OpenDocument(h.ctx, doc.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-documents.feature] the document opens with the content it had when I attached it
	if want := []string{h.files.Path(stored)}; !slices.Equal(h.opener.opened, want) {
		t.Errorf("opened %v, want the stored copy %v", h.opener.opened, want)
	}
}

// @scenario [#allowed-file-types]
func TestAllowedFileTypes(t *testing.T) {
	for _, file := range []string{"notes.txt", "resume.pdf", "prep.md", "cover.docx", "cover.doc", "resume.rtf", "resume.odt"} {
		t.Run(file, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)

			// @when [features/job-documents.feature] I attach the file "<file>" as other
			mustAttach(t, h, id, file, KindOther)

			// @then [features/job-documents.feature] the job lists "<file>" as other
			if got := listed(t, h, id); !slices.Equal(got, []string{file + " as other"}) {
				t.Errorf("job lists %v", got)
			}
		})
	}
}

// @scenario [#file-type-any-case]
func TestFileTypeAnyCase(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-documents.feature] I attach the file "RESUME.PDF" as other
	mustAttach(t, h, id, "RESUME.PDF", KindOther)

	// @then [features/job-documents.feature] the job lists "RESUME.PDF" as other
	if got := listed(t, h, id); !slices.Equal(got, []string{"RESUME.PDF as other"}) {
		t.Errorf("job lists %v", got)
	}
}

// @scenario [#disallowed-file-types]
func TestDisallowedFileTypes(t *testing.T) {
	for _, file := range []string{"headshot.png", "take-home.zip", "setup.exe", "salaries.xlsx", "no-extension"} {
		t.Run(file, func(t *testing.T) {
			h := newHarness(t)
			// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
			id := h.acme(StatusSaved)

			// @when [features/job-documents.feature] I try to attach the file "<file>"
			_, err := h.svc.AttachDocument(h.ctx, id, file, KindOther)

			// @then [features/job-documents.feature] the file is not attached
			if len(h.store.docs) != 0 || len(h.files.copies) != 0 {
				t.Errorf("attached: rows %v, copies %v", h.store.docs, h.files.copies)
			}
			// @and [features/job-documents.feature] I am told which document types are allowed
			wantValidation(t, err, "only these document types can be attached: .txt, .pdf, .md, .docx, .doc, .rtf, .odt")
		})
	}
}

// @scenario [#file-size-limit]
func TestFileSizeLimit(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-documents.feature] I try to attach a 12 MB file "portfolio.pdf"
	h.files.sizes["portfolio.pdf"] = 12 << 20
	_, err := h.svc.AttachDocument(h.ctx, id, "portfolio.pdf", KindOther)

	// @then [features/job-documents.feature] the file is not attached
	if len(h.store.docs) != 0 || len(h.files.copies) != 0 {
		t.Errorf("attached: rows %v, copies %v", h.store.docs, h.files.copies)
	}
	// @and [features/job-documents.feature] I am told files can be at most 10 MB
	wantValidation(t, err, "files can be at most 10 MB")
}

// @scenario [#file-size-at-limit]
func TestFileSizeAtLimit(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme"
	id := h.acme(StatusSaved)

	// @when [features/job-documents.feature] I attach a 10 MB file "portfolio.pdf" as other
	h.files.sizes["portfolio.pdf"] = 10 << 20
	mustAttach(t, h, id, "portfolio.pdf", KindOther)

	// @then [features/job-documents.feature] the job lists "portfolio.pdf" as other
	if got := listed(t, h, id); !slices.Equal(got, []string{"portfolio.pdf as other"}) {
		t.Errorf("job lists %v", got)
	}
}

func TestOneByteOverTheLimitIsRejected(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	h.files.sizes["portfolio.pdf"] = 10<<20 + 1
	if _, err := h.svc.AttachDocument(h.ctx, id, "portfolio.pdf", KindOther); err == nil {
		t.Error("attached a file one byte over 10 MB")
	}
}

func TestAttachRejectsUnknownKindAndJob(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	if _, err := h.svc.AttachDocument(h.ctx, id, "resume.pdf", "photo"); err == nil {
		t.Error("attached with an unknown kind")
	}
	if _, err := h.svc.AttachDocument(h.ctx, 99, "resume.pdf", KindResume); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("err = %v, want ErrJobNotFound", err)
	}
	if len(h.files.copies) != 0 {
		t.Errorf("copies left: %v", h.files.copies)
	}
}

func TestAttachRemovesTheCopyIfTheRowFails(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	h.store.fail = "CreateDocument"

	if _, err := h.svc.AttachDocument(h.ctx, id, "resume.pdf", KindResume); !errors.Is(err, errFake) {
		t.Fatalf("err = %v, want errFake", err)
	}
	if len(h.files.copies) != 0 {
		t.Errorf("orphaned copies: %v", h.files.copies)
	}
}

// @scenario [#open-document]
func TestOpenDocument(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	doc := mustAttach(t, h, id, "resume-2026.pdf", KindResume)

	// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
	if err := h.svc.OpenDocument(h.ctx, doc.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-documents.feature] it opens in my computer's default app for that file type
	if len(h.opener.opened) != 1 || !strings.HasSuffix(h.opener.opened[0], ".pdf") {
		t.Errorf("opened %v, want the stored .pdf copy", h.opener.opened)
	}
}

// The restore that brings back the document without its file is
// TestRestoreListsARemovedDocumentWithoutItsFile in internal/backups.
// @scenario [#restore-missing-document-file]
func TestOpenADocumentRestoredWithoutItsFile(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusApplied)
	// A restored document's row is back, but its stored copy was deleted when it was removed.
	docID := h.seedDoc(id, "resume-2026.pdf", KindResume)
	delete(h.files.copies, h.store.docs[docID].StoredPath)

	err := h.svc.OpenDocument(h.ctx, docID)

	// @but [features/agent-backups.feature] opening it tells me its file can't be found
	var missing *MissingFileError
	if !errors.As(err, &missing) || err.Error() != `the file for "resume-2026.pdf" can't be found` {
		t.Errorf("err = %v, want a MissingFileError", err)
	}
	if len(h.opener.opened) != 0 {
		t.Errorf("opened %v", h.opener.opened)
	}
}

// @scenario [#open-missing-document]
func TestOpenMissingDocument(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	doc := mustAttach(t, h, id, "resume-2026.pdf", KindResume)
	// @and [features/job-documents.feature] its stored copy is no longer in the app's data directory
	delete(h.files.copies, h.store.docs[doc.ID].StoredPath)

	// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
	err := h.svc.OpenDocument(h.ctx, doc.ID)

	// @then [features/job-documents.feature] I am told the file for "resume-2026.pdf" can't be found
	var missing *MissingFileError
	if !errors.As(err, &missing) || err.Error() != `the file for "resume-2026.pdf" can't be found` {
		t.Errorf("err = %v, want a MissingFileError", err)
	}
	if len(h.opener.opened) != 0 {
		t.Errorf("opened %v", h.opener.opened)
	}
	// @and [features/job-documents.feature] the job still lists "resume-2026.pdf"
	if got := listed(t, h, id); !slices.Equal(got, []string{"resume-2026.pdf as resume"}) {
		t.Errorf("job lists %v", got)
	}
}

// @scenario [#remove-missing-document]
func TestRemoveMissingDocument(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	doc := mustAttach(t, h, id, "resume-2026.pdf", KindResume)
	// @and [features/job-documents.feature] its stored copy is no longer in the app's data directory
	delete(h.files.copies, h.store.docs[doc.ID].StoredPath)

	// @when [features/job-documents.feature] I remove the attached "resume-2026.pdf"
	// Removing a copy that's already gone isn't an error (TestRemoveMissingCopyIsNotAnError).
	if err := h.svc.RemoveDocument(h.ctx, doc.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-documents.feature] the job no longer lists "resume-2026.pdf"
	if got := listed(t, h, id); len(got) != 0 {
		t.Errorf("job lists %v", got)
	}
}

// @scenario [#change-document-kind]
func TestChangeDocumentKind(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	// @given [features/job-documents.feature] I have attached "acme-cover.docx" as other to a job for "Software Engineer" at "Acme"
	doc := h.seedDoc(id, "acme-cover.docx", KindOther)

	// @when [features/job-documents.feature] I change its kind to cover letter
	if err := h.svc.SetDocumentKind(h.ctx, doc, KindCoverLetter); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-documents.feature] the job lists "acme-cover.docx" as a cover letter
	if got := listed(t, h, id); !slices.Equal(got, []string{"acme-cover.docx as cover_letter"}) {
		t.Errorf("job lists %v", got)
	}
}

func TestSetDocumentKindRejectsUnknownKindAndDocument(t *testing.T) {
	h := newHarness(t)
	doc := h.seedDoc(h.acme(StatusSaved), "cover.docx", KindOther)
	if err := h.svc.SetDocumentKind(h.ctx, doc, "photo"); err == nil {
		t.Error("set an unknown kind")
	}
	if err := h.svc.SetDocumentKind(h.ctx, 99, KindResume); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound", err)
	}
}

// @scenario [#remove-document]
func TestRemoveDocument(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	original := filepath.Join("Documents", "resume-2026.pdf")
	doc := mustAttach(t, h, id, original, KindResume)
	stored := h.store.docs[doc.ID].StoredPath

	// @when [features/job-documents.feature] I remove the attached "resume-2026.pdf"
	if err := h.svc.RemoveDocument(h.ctx, doc.ID); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-documents.feature] the job no longer lists "resume-2026.pdf"
	if got := listed(t, h, id); len(got) != 0 {
		t.Errorf("job lists %v", got)
	}
	// @and [features/job-documents.feature] the original "resume-2026.pdf" on my computer is not deleted
	if !slices.Equal(h.files.removed, []string{stored}) {
		t.Errorf("removed %v, want only the stored copy %q, never %q", h.files.removed, stored, original)
	}
}

func TestRemoveDocumentNotFound(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.RemoveDocument(h.ctx, 99); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound", err)
	}
	if err := h.svc.OpenDocument(h.ctx, 99); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound", err)
	}
}

// @scenario [#same-document-many-jobs]
func TestSameDocumentOnManyJobs(t *testing.T) {
	h := newHarness(t)
	// @given [features/job-documents.feature] I have a job for "Software Engineer" at "Acme" and a job for "Developer" at "Globex"
	acme := h.acme(StatusSaved)
	globex := h.seed(store.Job{Company: "Globex", Role: "Developer"})

	// @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume to both jobs
	fromAcme := mustAttach(t, h, acme, "resume-2026.pdf", KindResume)
	mustAttach(t, h, globex, "resume-2026.pdf", KindResume)

	// @then [features/job-documents.feature] both jobs list "resume-2026.pdf" as a resume
	for _, id := range []int64{acme, globex} {
		if got := listed(t, h, id); !slices.Equal(got, []string{"resume-2026.pdf as resume"}) {
			t.Errorf("job %d lists %v", id, got)
		}
	}

	// @and [features/job-documents.feature] removing it from one job leaves it attached to the other
	if err := h.svc.RemoveDocument(h.ctx, fromAcme.ID); err != nil {
		t.Fatal(err)
	}
	if got := listed(t, h, globex); !slices.Equal(got, []string{"resume-2026.pdf as resume"}) {
		t.Errorf("Globex lists %v after removing Acme's", got)
	}
	if len(h.files.copies) != 1 {
		t.Errorf("copies left: %v, want Globex's", h.files.copies)
	}
}

func TestJobDocumentFiles(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	resume := h.seedDoc(id, "resume-2026.pdf", KindResume)
	letter := h.seedDoc(id, "letter.pdf", KindCoverLetter)
	delete(h.files.copies, h.store.docs[letter].StoredPath)

	files, err := h.svc.JobDocumentFiles(h.ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	want := []DocumentFile{
		{DocumentID: resume, Path: "/data/" + h.store.docs[resume].StoredPath},
		{DocumentID: letter, Path: "/data/" + h.store.docs[letter].StoredPath, Missing: true},
	}
	if !slices.Equal(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
}

func TestJobDocumentFilesUnknownJob(t *testing.T) {
	h := newHarness(t)
	if _, err := h.svc.JobDocumentFiles(h.ctx, 999); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("err = %v, want ErrJobNotFound", err)
	}
}

func TestDocumentKindLabels(t *testing.T) {
	for kind, want := range map[DocumentKind]string{KindResume: "Resume", KindCoverLetter: "Cover letter", KindOther: "Other"} {
		if got := kind.Label(); got != want {
			t.Errorf("%q.Label() = %q, want %q", kind, got, want)
		}
	}
}

func TestDocument(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	docID := h.seedDoc(id, "acme-resume.pdf", KindResume)

	got, err := h.svc.Document(h.ctx, docID)
	if err != nil {
		t.Fatal(err)
	}
	if want := (Document{ID: docID, JobID: id, Kind: KindResume, FileName: "acme-resume.pdf"}); got != want {
		t.Errorf("Document = %+v, want %+v", got, want)
	}
	if _, err := h.svc.Document(h.ctx, 999); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound", err)
	}
}

func TestAttachMissingFile(t *testing.T) {
	h := newHarness(t)
	id := h.acme(StatusSaved)
	h.files.absent = []string{"Documents/gone.pdf"}

	_, err := h.svc.AttachDocument(h.ctx, id, "Documents/gone.pdf", KindResume)

	wantValidation(t, err, "there's no file at Documents/gone.pdf")
	if len(h.store.docs) != 0 || len(h.files.copies) != 0 {
		t.Errorf("attached: rows %v, copies %v", h.store.docs, h.files.copies)
	}
}

func TestDocumentKindsListsEveryOne(t *testing.T) {
	if len(DocumentKinds) != len(documentKindLabels) {
		t.Errorf("DocumentKinds has %d, labels have %d", len(DocumentKinds), len(documentKindLabels))
	}
	for _, k := range DocumentKinds {
		if !k.valid() {
			t.Errorf("%q isn't a document kind", k)
		}
	}
}
