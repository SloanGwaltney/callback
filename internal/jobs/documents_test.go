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

// @scenario [features/job-documents.feature#attach-document] Attach a document of a given kind
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

// @scenario [features/job-documents.feature#attach-multiple-documents] Attach several documents to one job
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

// @scenario [features/job-documents.feature#document-kept-if-original-moved] An attached document still opens after the original file is moved
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

// @scenario [features/job-documents.feature#allowed-file-types] Text-like document types can be attached
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

// @scenario [features/job-documents.feature#file-type-any-case] A document's type is recognized whatever the letter case of its extension
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

// @scenario [features/job-documents.feature#disallowed-file-types] Other file types can't be attached
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

// @scenario [features/job-documents.feature#file-size-limit] A file larger than 10 MB can't be attached
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

// @scenario [features/job-documents.feature#file-size-at-limit] A file of exactly 10 MB can be attached
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

// @scenario [features/job-documents.feature#open-document] Open an attached document
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

// @scenario [features/job-documents.feature#change-document-kind] Change what kind of document an attachment is
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

// @scenario [features/job-documents.feature#remove-document] Remove an attached document
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

// @scenario [features/job-documents.feature#same-document-many-jobs] The same file can be attached to more than one job
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
