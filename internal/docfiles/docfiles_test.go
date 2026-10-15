package docfiles

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Storage-tier tests against a real temporary directory. See
// docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

// writeOriginal writes a file outside the data directory, standing in for one on the user's
// computer.
func writeOriginal(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func readStored(t *testing.T, d *Dir, stored string) string {
	t.Helper()
	b, err := os.ReadFile(d.Path(stored))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSaveCopiesIntoTheDocumentsFolder(t *testing.T) {
	dataDir := t.TempDir()
	d := New(dataDir)
	src := writeOriginal(t, "Resume-2026.PDF", "my resume")

	stored, err := d.Save(src)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(stored, "documents/") || !strings.HasSuffix(stored, ".pdf") || strings.Contains(stored, `\`) {
		t.Errorf("stored = %q, want documents/<id>.pdf with forward slashes", stored)
	}
	if strings.Contains(stored, "Resume") {
		t.Errorf("stored = %q, want a generated name, not the original", stored)
	}
	if got := d.Path(stored); filepath.Dir(got) != filepath.Join(dataDir, Folder) {
		t.Errorf("Path = %q, want it in %q", got, filepath.Join(dataDir, Folder))
	}
	if got := readStored(t, d, stored); got != "my resume" {
		t.Errorf("copy holds %q", got)
	}
}

// @scenario [#document-kept-if-original-moved]
func TestCopySurvivesTheOriginalBeingDeleted(t *testing.T) {
	d := New(t.TempDir())
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	original := writeOriginal(t, "resume-2026.pdf", "resume as sent to Acme")
	stored, err := d.Save(original)
	if err != nil {
		t.Fatal(err)
	}

	// @and [features/job-documents.feature] I have since moved or deleted the original "resume-2026.pdf" on my computer
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}

	// @when [features/job-documents.feature] I open the attached "resume-2026.pdf"
	got := readStored(t, d, stored)

	// @then [features/job-documents.feature] the document opens with the content it had when I attached it
	if got != "resume as sent to Acme" {
		t.Errorf("copy holds %q", got)
	}
}

// @scenario [#remove-document]
func TestRemoveDeletesOnlyTheCopy(t *testing.T) {
	d := New(t.TempDir())
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	original := writeOriginal(t, "resume-2026.pdf", "resume")
	stored, err := d.Save(original)
	if err != nil {
		t.Fatal(err)
	}

	// @when [features/job-documents.feature] I remove the attached "resume-2026.pdf"
	if err := d.Remove(stored); err != nil {
		t.Fatal(err)
	}

	// @then [features/job-documents.feature] the job no longer lists "resume-2026.pdf"
	if _, err := os.Stat(d.Path(stored)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("copy still there: %v", err)
	}
	// @and [features/job-documents.feature] the original "resume-2026.pdf" on my computer is not deleted
	if _, err := os.Stat(original); err != nil {
		t.Errorf("original gone: %v", err)
	}
}

// @scenario [#same-document-many-jobs]
func TestSameFileSavedTwiceMakesTwoCopies(t *testing.T) {
	d := New(t.TempDir())
	original := writeOriginal(t, "resume-2026.pdf", "resume")

	// @when [features/job-documents.feature] I attach "resume-2026.pdf" as a resume to both jobs
	forAcme, err := d.Save(original)
	if err != nil {
		t.Fatal(err)
	}
	forGlobex, err := d.Save(original)
	if err != nil {
		t.Fatal(err)
	}
	if forAcme == forGlobex {
		t.Fatalf("both saves returned %q, want separate copies", forAcme)
	}

	// @and [features/job-documents.feature] removing it from one job leaves it attached to the other
	if err := d.Remove(forAcme); err != nil {
		t.Fatal(err)
	}
	if got := readStored(t, d, forGlobex); got != "resume" {
		t.Errorf("Globex's copy holds %q", got)
	}
}

// @scenario [#remove-missing-document]
func TestRemoveMissingCopyIsNotAnError(t *testing.T) {
	d := New(t.TempDir())
	// @given [features/job-documents.feature] I have attached "resume-2026.pdf" to a job for "Software Engineer" at "Acme"
	stored, err := d.Save(writeOriginal(t, "resume-2026.pdf", "resume"))
	if err != nil {
		t.Fatal(err)
	}
	// @and [features/job-documents.feature] its stored copy is no longer in the app's data directory
	if err := os.Remove(d.Path(stored)); err != nil {
		t.Fatal(err)
	}

	// @when [features/job-documents.feature] I remove the attached "resume-2026.pdf"
	// The job stops listing it once this succeeds (TestRemoveMissingDocument in internal/jobs).
	if err := d.Remove(stored); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestSize(t *testing.T) {
	d := New(t.TempDir())
	src := filepath.Join(t.TempDir(), "portfolio.pdf")
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	// Exactly 10 MB, sparse where the file system allows.
	if err := f.Truncate(10 << 20); err != nil {
		t.Fatal(err)
	}
	f.Close()

	size, err := d.Size(src)
	if err != nil {
		t.Fatal(err)
	}
	if size != 10<<20 {
		t.Errorf("size = %d, want %d", size, 10<<20)
	}
	if _, err := d.Size(t.TempDir()); err == nil {
		t.Error("Size of a directory succeeded")
	}
	if _, err := d.Size(filepath.Join(t.TempDir(), "missing.pdf")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Size of a missing file = %v, want fs.ErrNotExist", err)
	}
}

func TestSaveCopiesLargeFilesWhole(t *testing.T) {
	d := New(t.TempDir())
	content := bytes.Repeat([]byte("0123456789abcdef"), 1<<16) // 1 MB
	src := filepath.Join(t.TempDir(), "big.txt")
	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatal(err)
	}
	stored, err := d.Save(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := readStored(t, d, stored); got != string(content) {
		t.Errorf("copy is %d bytes, want %d", len(got), len(content))
	}
}

func TestSaveMissingFileLeavesNothingBehind(t *testing.T) {
	dataDir := t.TempDir()
	d := New(dataDir)
	if _, err := d.Save(filepath.Join(t.TempDir(), "missing.pdf")); err == nil {
		t.Fatal("saving a missing file succeeded")
	}
	entries, _ := os.ReadDir(filepath.Join(dataDir, Folder))
	if len(entries) != 0 {
		t.Errorf("left %d files in the documents folder", len(entries))
	}
}

func TestExists(t *testing.T) {
	d := New(t.TempDir())
	stored, err := d.Save(writeOriginal(t, "resume-base-2026.docx", "resume"))
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := d.Exists(stored); !ok || err != nil {
		t.Errorf("Exists(saved copy) = %v, %v, want true", ok, err)
	}
	if err := d.Remove(stored); err != nil {
		t.Fatal(err)
	}
	if ok, err := d.Exists(stored); ok || err != nil {
		t.Errorf("Exists(removed copy) = %v, %v, want false", ok, err)
	}
	if ok, err := d.Exists(Folder); ok || err != nil {
		t.Errorf("Exists(the folder itself) = %v, %v, want false", ok, err)
	}
}

// @scenario [#library-document-kept-if-original-moved]
func TestLibraryCopySurvivesTheOriginalBeingDeleted(t *testing.T) {
	d := New(t.TempDir())
	// @given [features/document-library.feature] I added "resume-base-2026.docx" to the library
	original := writeOriginal(t, "resume-base-2026.docx", "base resume as added")
	stored, err := d.Save(original)
	if err != nil {
		t.Fatal(err)
	}

	// @and [features/document-library.feature] I have since moved or deleted the original "resume-base-2026.docx" on my computer
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}

	// @when [features/document-library.feature] I open "resume-base-2026.docx" from the library
	got := readStored(t, d, stored)

	// @then [features/document-library.feature] the document opens with the content it had when I added it
	if got != "base resume as added" {
		t.Errorf("copy holds %q", got)
	}
}

// @scenario [#remove-library-document]
func TestRemoveLibraryCopyKeepsTheOriginal(t *testing.T) {
	d := New(t.TempDir())
	// @given [features/document-library.feature] the library lists "resume-base-2026.docx"
	original := writeOriginal(t, "resume-base-2026.docx", "resume")
	stored, err := d.Save(original)
	if err != nil {
		t.Fatal(err)
	}

	// @when [features/document-library.feature] I remove "resume-base-2026.docx" from the library
	if err := d.Remove(stored); err != nil {
		t.Fatal(err)
	}

	// @and [features/document-library.feature] the original "resume-base-2026.docx" on my computer is not deleted
	if _, err := os.Stat(original); err != nil {
		t.Errorf("original gone: %v", err)
	}
}
