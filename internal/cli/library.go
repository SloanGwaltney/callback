package cli

import (
	"fmt"
	"io"
	"strings"

	"callback/internal/jobs"
)

// libraryCommands are the commands for the document library, in the order the help lists them.
// The rules are the jobs service's, as in the app.
var libraryCommands = []*command{
	{
		path:    []string{"library", "list"},
		summary: "List the documents in the library, newest first",
		about: "Lists every document in the person's document library, newest first, one line each: " +
			"its ID, file name, description, and the path of its stored copy, which you can read. " +
			"Library documents, like a general resume, aren't attached to any job.\n\n" +
			"Never edit a stored copy: to change a library document, add the new file and remove the " +
			"old one.",
		run: listLibrary,
	},
	{
		path:    []string{"library", "add"},
		summary: "Add a copy of a file on this computer to the library",
		about: "Adds a file to the document library, and shows it with its new ID and the path of its " +
			"stored copy. Callback keeps its own stored copy, so the file you give is never changed, " +
			"and moving or deleting it later doesn't affect the library. Adding the same file again " +
			"adds another copy.",
		options: []option{
			{name: "file", value: "<path>", usage: fmt.Sprintf("the file to add, at most %d MB, of one of these types: %s",
				jobs.MaxDocumentBytes>>20, strings.Join(jobs.AllowedDocumentTypes, ", "))},
			{name: "description", value: "<text>", usage: "what the document is, like \"General resume\"; none if not given"},
		},
		run: addLibraryDocument,
	},
	{
		path:    []string{"library", "describe"},
		takesID: "library-document",
		summary: "Change or clear a library document's description",
		about: "Changes a library document's description. Give an empty value, --description \"\", to " +
			"clear it. Shows the document as it is afterwards.",
		options: []option{{name: "description", value: "<text>", usage: "the new description, or \"\" for none"}},
		run:     describeLibraryDocument,
	},
	{
		path:    []string{"library", "remove"},
		takesID: "library-document",
		summary: "Remove a document from the library and delete its stored copy",
		about: "Removes a document from the library at once, without asking to confirm, and deletes " +
			"Callback's stored copy. The file it was added from isn't touched, and neither are jobs' " +
			"documents, which have their own copies. This can't be undone.",
		run: removeLibraryDocument,
	},
}

// libraryDocumentJSON is a library document with where its stored copy is.
type libraryDocumentJSON struct {
	jobs.LibraryDocument
	StoredPath string `json:"storedPath"`
	// Missing is set when the stored copy is gone.
	Missing bool `json:"missing"`
}

// libraryJSON pairs each library document with its stored copy.
func libraryJSON(docs []jobs.LibraryDocument, files []jobs.DocumentFile) []libraryDocumentJSON {
	byID := map[int64]jobs.DocumentFile{}
	for _, f := range files {
		byID[f.DocumentID] = f
	}
	out := make([]libraryDocumentJSON, 0, len(docs))
	for _, doc := range docs {
		f := byID[doc.ID]
		out = append(out, libraryDocumentJSON{LibraryDocument: doc, StoredPath: f.Path, Missing: f.Missing})
	}
	return out
}

// libraryLine is a library document on one line, starting with its ID, with where its stored copy
// is.
func libraryLine(d libraryDocumentJSON) string {
	description := d.Description
	if description == "" {
		description = "(no description)"
	}
	missing := ""
	if d.Missing {
		missing = " (missing: the stored copy is gone)"
	}
	return fmt.Sprintf("#%d %s | %s | stored copy: %s%s", d.ID, d.FileName, description, d.StoredPath, missing)
}

// @scenario [features/agent-cli-library.feature#cli-list-library] List the library
// @scenario [features/agent-cli-library.feature#cli-list-library-missing] The list marks a library document whose file is missing
// @scenario [features/agent-cli-library.feature#cli-list-library-empty] My agent is told when the library is empty
func listLibrary(c *call) error {
	docs, err := c.jobs.Library(c.ctx)
	if err != nil {
		return err
	}
	files, err := c.jobs.LibraryFiles(c.ctx)
	if err != nil {
		return err
	}
	listed := libraryJSON(docs, files)
	return c.output(listed, func(w io.Writer) {
		if len(listed) == 0 {
			fmt.Fprintln(w, "The library has no documents.")
		}
		for _, d := range listed {
			fmt.Fprintln(w, libraryLine(d))
		}
	})
}

// @scenario [features/agent-cli-library.feature#cli-add-library-document] Add a document to the library
// @scenario [features/agent-cli-library.feature#cli-add-library-rules] Adding a library document follows the app's rules
func addLibraryDocument(c *call) error {
	path, ok := c.value("file")
	if !ok {
		return usageErr("`library add` needs --file with the path of the file to add")
	}
	description, _ := c.value("description")
	doc, err := c.jobs.AddLibraryDocument(c.ctx, path, description)
	if err != nil {
		return err
	}
	return c.printLibraryDocument(doc)
}

// @scenario [features/agent-cli-library.feature#cli-describe-library-document] Change a library document's description
// @scenario [features/agent-cli-library.feature#cli-clear-library-description] Clear a library document's description
func describeLibraryDocument(c *call) error {
	description, ok := c.value("description")
	if !ok {
		return usageErr("`library describe` needs --description with the new description, or --description \"\" to clear it")
	}
	doc, err := c.jobs.SetLibraryDescription(c.ctx, c.id, description)
	if err != nil {
		return missingErr("library document", c.id, err, jobs.ErrLibraryDocumentNotFound)
	}
	return c.printLibraryDocument(doc)
}

// printLibraryDocument shows a library document with where its stored copy is.
func (c *call) printLibraryDocument(doc jobs.LibraryDocument) error {
	files, err := c.jobs.LibraryFiles(c.ctx)
	if err != nil {
		return err
	}
	out := libraryJSON([]jobs.LibraryDocument{doc}, files)[0]
	return c.output(out, func(w io.Writer) { fmt.Fprintf(w, "Library document: %s\n", libraryLine(out)) })
}

// @scenario [features/agent-cli-library.feature#cli-remove-library-document] Remove a library document
func removeLibraryDocument(c *call) error {
	doc, err := c.jobs.LibraryDocument(c.ctx, c.id)
	if err != nil {
		return missingErr("library document", c.id, err, jobs.ErrLibraryDocumentNotFound)
	}
	if err := c.jobs.RemoveLibraryDocument(c.ctx, c.id); err != nil {
		return missingErr("library document", c.id, err, jobs.ErrLibraryDocumentNotFound)
	}
	return c.output(removedJSON{doc}, func(w io.Writer) {
		fmt.Fprintf(w, "Removed library document #%d, %s, and deleted its stored copy.\n", doc.ID, doc.FileName)
	})
}
