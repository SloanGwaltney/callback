package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"callback/internal/jobs"
)

// The scenarios in features/agent-cli-library.feature. The fake jobs service keeps what it's
// given; the library's rules, like which files can be added, are the jobs service's and are
// tested in internal/jobs.

// inLibrary adds fileName to the top of the library, as the newest document, with a stored copy.
func (h *harness) inLibrary(fileName, description string) jobs.LibraryDocument {
	h.jobs.nextID++
	doc := jobs.LibraryDocument{ID: h.jobs.nextID, FileName: fileName, Description: description}
	h.jobs.library = slices.Insert(h.jobs.library, 0, doc)
	h.jobs.libraryFiles[doc.ID] = jobs.DocumentFile{DocumentID: doc.ID, Path: storedPath(doc.ID)}
	return doc
}

// libraryNames lists the library as "file: description", newest first.
func (h *harness) libraryNames() []string {
	var out []string
	for _, d := range h.jobs.library {
		out = append(out, d.FileName+": "+d.Description)
	}
	return out
}

// The order is the store's; internal/db's TestAgentListsTheLibraryNewestFirst checks it.
// @scenario [#cli-list-library]
func TestAgentListsTheLibrary(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-library.feature] my library holds "resume-2026.pdf" described as "General resume"
	resume := h.inLibrary("resume-2026.pdf", "General resume")
	// @and [features/agent-cli-library.feature] my library holds "cover-letter.docx" with no description
	letter := h.inLibrary("cover-letter.docx", "")

	// @when [features/agent-cli-library.feature] my agent lists the library
	r := h.run("library", "list")

	r.wantCode(t, ExitOK)
	// @and [features/agent-cli-library.feature] it sees each one's ID, description, and where its stored copy is
	want := fmt.Sprintf("#%d cover-letter.docx | (no description) | stored copy: %s\n"+
		"#%d resume-2026.pdf | General resume | stored copy: %s\n",
		letter.ID, storedPath(letter.ID), resume.ID, storedPath(resume.ID))
	if r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestLibraryListAsJSON(t *testing.T) {
	h := newHarness()
	doc := h.inLibrary("resume-2026.pdf", "General resume")

	r := h.run("library", "list", "--json")

	r.wantCode(t, ExitOK)
	var got []libraryDocumentJSON
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, r.stdout)
	}
	want := []libraryDocumentJSON{{LibraryDocument: doc, StoredPath: storedPath(doc.ID)}}
	if !slices.Equal(got, want) {
		t.Errorf("JSON = %+v, want %+v", got, want)
	}
	for _, key := range []string{`"id"`, `"fileName"`, `"description"`, `"storedPath"`, `"missing"`} {
		if !strings.Contains(r.stdout, key) {
			t.Errorf("JSON has no %s: %s", key, r.stdout)
		}
	}
}

// @scenario [#cli-list-library-missing]
func TestLibraryListMarksAMissingFile(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-library.feature] my library holds "resume-2026.pdf"
	doc := h.inLibrary("resume-2026.pdf", "")
	// @and [features/agent-cli-library.feature] the stored copy of "resume-2026.pdf" is gone
	h.jobs.libraryFiles[doc.ID] = jobs.DocumentFile{DocumentID: doc.ID, Path: storedPath(doc.ID), Missing: true}

	// @when [features/agent-cli-library.feature] my agent lists the library
	r := h.run("library", "list")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-library.feature] "resume-2026.pdf" is marked as missing its file
	if !strings.Contains(r.stdout, "resume-2026.pdf | (no description) | stored copy: "+storedPath(doc.ID)+" (missing: the stored copy is gone)") {
		t.Errorf("stdout doesn't mark the file missing: %q", r.stdout)
	}
	if json := h.run("library", "list", "--json").stdout; !strings.Contains(json, `"missing": true`) {
		t.Errorf("JSON doesn't mark the file missing: %s", json)
	}
}

// @scenario [#cli-list-library-empty]
func TestAgentIsToldTheLibraryIsEmpty(t *testing.T) {
	// @given [features/agent-cli-library.feature] my library is empty
	h := newHarness()

	// @when [features/agent-cli-library.feature] my agent lists the library
	r := h.run("library", "list")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-library.feature] my agent is told the library has no documents
	if r.stdout != "The library has no documents.\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
	if got := strings.TrimSpace(h.run("library", "list", "--json").stdout); got != "[]" {
		t.Errorf("JSON = %q, want []", got)
	}
}

// @scenario [#cli-add-library-document]
func TestAgentAddsALibraryDocument(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-library.feature] there is a file "resume-2026.pdf" on my computer
	src := "C:/Users/me/Documents/resume-2026.pdf"

	// @when [features/agent-cli-library.feature] my agent adds "resume-2026.pdf" to the library described as "General resume"
	r := h.run("library", "add", "--file", src, "--description", "General resume")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-library.feature] the library lists "resume-2026.pdf" described as "General resume"
	if got := h.libraryNames(); !slices.Equal(got, []string{"resume-2026.pdf: General resume"}) {
		t.Errorf("the library lists %v", got)
	}
	// @and [features/agent-cli-library.feature] my agent is shown the document with its ID and where its stored copy is
	id := h.jobs.library[0].ID
	if want := fmt.Sprintf("Library document: #%d resume-2026.pdf | General resume | stored copy: %s\n", id, storedPath(id)); r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestAddLibraryDocumentWithoutDescription(t *testing.T) {
	h := newHarness()

	r := h.run("library", "add", "--file", "cover-letter.docx", "--json")

	r.wantCode(t, ExitOK)
	var got libraryDocumentJSON
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, r.stdout)
	}
	if got.FileName != "cover-letter.docx" || got.Description != "" || got.StoredPath != storedPath(got.ID) {
		t.Errorf("JSON = %+v", got)
	}
}

func TestAddLibraryDocumentNeedsAFile(t *testing.T) {
	h := newHarness()

	r := h.run("library", "add", "--description", "General resume")

	r.wantStderr(t, ExitUsage, "needs --file")
	if len(h.jobs.library) != 0 {
		t.Errorf("the library holds %v", h.jobs.library)
	}
}

// @scenario [#cli-add-library-rules]
func TestAddingALibraryDocumentFollowsTheAppsRules(t *testing.T) {
	for _, tc := range []struct {
		file   string
		src    string
		reason string
	}{
		{"a file that doesn't exist", "gone.pdf", "there's no file at gone.pdf"},
		{"\"photo.png\"", "photo.png", "only these document types can be added: .txt, .pdf, .md, .docx, .doc, .rtf, .odt"},
		{"a PDF larger than 10 MB", "portfolio.pdf", "files can be at most 10 MB"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			h := newHarness()
			h.jobs.fail = &jobs.ValidationError{Msg: tc.reason}

			// @when [features/agent-cli-library.feature] my agent tries to add <file> to the library
			r := h.run("library", "add", "--file", tc.src)

			// @then [features/agent-cli-library.feature] the library has no new document
			if len(h.jobs.library) != 0 {
				t.Errorf("the library holds %v", h.jobs.library)
			}
			// @and [features/agent-cli-library.feature] my agent is told <reason>
			r.wantStderr(t, ExitValidation, tc.reason)
		})
	}
}

// @scenario [#cli-describe-library-document]
func TestAgentChangesALibraryDescription(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-library.feature] my library holds "resume-2026.pdf" described as "General resume"
	doc := h.inLibrary("resume-2026.pdf", "General resume")

	// @when [features/agent-cli-library.feature] my agent changes its description to "Resume for backend roles"
	r := h.run("library", "describe", fmt.Sprint(doc.ID), "--description", "Resume for backend roles")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-library.feature] the library lists "resume-2026.pdf" described as "Resume for backend roles"
	if got := h.libraryNames(); !slices.Equal(got, []string{"resume-2026.pdf: Resume for backend roles"}) {
		t.Errorf("the library lists %v", got)
	}
	if want := fmt.Sprintf("Library document: #%d resume-2026.pdf | Resume for backend roles | stored copy: %s\n", doc.ID, storedPath(doc.ID)); r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

// @scenario [#cli-clear-library-description]
func TestAgentClearsALibraryDescription(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-library.feature] my library holds "resume-2026.pdf" described as "General resume"
	doc := h.inLibrary("resume-2026.pdf", "General resume")

	// @when [features/agent-cli-library.feature] my agent clears its description
	r := h.run("library", "describe", fmt.Sprint(doc.ID), "--description", "")

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-library.feature] the library lists "resume-2026.pdf" with no description
	if got := h.libraryNames(); !slices.Equal(got, []string{"resume-2026.pdf: "}) {
		t.Errorf("the library lists %v", got)
	}
}

func TestDescribeNeedsADescription(t *testing.T) {
	h := newHarness()
	doc := h.inLibrary("resume-2026.pdf", "General resume")

	r := h.run("library", "describe", fmt.Sprint(doc.ID))

	r.wantStderr(t, ExitUsage, "needs --description")
	if got := h.libraryNames(); !slices.Equal(got, []string{"resume-2026.pdf: General resume"}) {
		t.Errorf("the library lists %v", got)
	}
}

// @scenario [#cli-remove-library-document]
func TestAgentRemovesALibraryDocument(t *testing.T) {
	h := newHarness()
	// @given [features/agent-cli-library.feature] my library holds "resume-2026.pdf"
	doc := h.inLibrary("resume-2026.pdf", "")
	h.inLibrary("cover-letter.docx", "")

	// @when [features/agent-cli-library.feature] my agent removes it from the library
	r := h.run("library", "remove", fmt.Sprint(doc.ID))

	r.wantCode(t, ExitOK)
	// @then [features/agent-cli-library.feature] the library doesn't list "resume-2026.pdf"
	if got := h.libraryNames(); !slices.Equal(got, []string{"cover-letter.docx: "}) {
		t.Errorf("the library lists %v", got)
	}
	if want := fmt.Sprintf("Removed library document #%d, resume-2026.pdf, and deleted its stored copy.\n", doc.ID); r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestUnknownLibraryDocument(t *testing.T) {
	for _, args := range [][]string{
		{"library", "describe", "999", "--description", "x"},
		{"library", "remove", "999"},
	} {
		h := newHarness()
		h.run(args...).wantStderr(t, ExitNotFound, "there's no library document with ID 999")
	}
}
