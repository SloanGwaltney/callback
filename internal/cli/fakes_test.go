package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"callback/internal/backups"
	"callback/internal/jobs"
)

// Hand-written fakes for what the command line runs over. The rules themselves are the services'
// and are tested in internal/jobs and internal/backups; these fakes only keep what they're given.
// See docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

// fakeJobs is an in-memory jobs service. order holds the job IDs top to bottom; each column is
// the jobs in order with its status.
type fakeJobs struct {
	jobs  map[int64]*jobs.Job
	order []int64
	days  map[int64]int
	files map[int64][]jobs.DocumentFile
	// library is the document library, newest first, and libraryFiles its stored copies.
	library      []jobs.LibraryDocument
	libraryFiles map[int64]jobs.DocumentFile
	// fail is returned by the next change, which then changes nothing, as the service does for a
	// value that breaks one of its rules.
	fail error
	// updated is the details the last UpdateJob was given.
	updated jobs.Details
	moves   []jobs.Move
	nextID  int64
}

var _ Jobs = (*fakeJobs)(nil)

func newFakeJobs() *fakeJobs {
	return &fakeJobs{jobs: map[int64]*jobs.Job{}, days: map[int64]int{}, files: map[int64][]jobs.DocumentFile{}, libraryFiles: map[int64]jobs.DocumentFile{}, nextID: 100}
}

// add puts a job at the bottom of its column, Saved if status is "".
func (f *fakeJobs) add(company, role string, status jobs.Status) *jobs.Job {
	if status == "" {
		status = jobs.StatusSaved
	}
	f.nextID++
	job := &jobs.Job{
		ID: f.nextID, Company: company, Role: role, Status: status,
		Documents: []jobs.Document{}, Contacts: []jobs.Contact{}, Interviews: []jobs.Interview{}, StatusHistory: []jobs.StatusChange{},
	}
	f.jobs[job.ID] = job
	f.order = append(f.order, job.ID)
	return job
}

// change returns the next change's failure, clearing it.
func (f *fakeJobs) change() error {
	err := f.fail
	f.fail = nil
	return err
}

func (f *fakeJobs) toTop(id int64) {
	f.order = slices.Insert(slices.DeleteFunc(f.order, func(o int64) bool { return o == id }), 0, id)
}

func (f *fakeJobs) Board(context.Context) (jobs.Board, error) {
	var b jobs.Board
	for _, status := range jobs.Statuses {
		col := jobs.Column{Status: status, Label: status.Label(), Cards: []jobs.Card{}}
		for _, id := range f.order {
			job := f.jobs[id]
			if job.Status != status {
				continue
			}
			card := jobs.Card{
				ID: job.ID, Company: job.Company, Role: job.Role, Location: job.Location, DateApplied: job.DateApplied,
				Interest: job.Interest, ClosingReason: job.ClosingReason, ClosingReasonText: job.ClosingReasonText,
				DocumentCount: len(job.Documents), NextAction: job.NextAction,
			}
			if d, ok := f.days[id]; ok {
				card.DaysInStatus = &d
			}
			col.Cards = append(col.Cards, card)
		}
		col.Count = len(col.Cards)
		b.Columns = append(b.Columns, col)
	}
	return b, nil
}

func (f *fakeJobs) Job(_ context.Context, id int64) (jobs.Job, error) {
	job, ok := f.jobs[id]
	if !ok {
		return jobs.Job{}, jobs.ErrJobNotFound
	}
	return *job, nil
}

func (f *fakeJobs) JobDocumentFiles(_ context.Context, jobID int64) ([]jobs.DocumentFile, error) {
	if _, ok := f.jobs[jobID]; !ok {
		return nil, jobs.ErrJobNotFound
	}
	return f.files[jobID], nil
}

func (f *fakeJobs) AddJob(_ context.Context, d jobs.Details, status jobs.Status) (jobs.Job, error) {
	if err := f.change(); err != nil {
		return jobs.Job{}, err
	}
	job := f.add(d.Company, d.Role, status)
	setDetails(job, d)
	f.toTop(job.ID)
	return *job, nil
}

func (f *fakeJobs) UpdateJob(_ context.Context, id int64, d jobs.Details) (jobs.Job, error) {
	f.updated = d
	if err := f.change(); err != nil {
		return jobs.Job{}, err
	}
	job, ok := f.jobs[id]
	if !ok {
		return jobs.Job{}, jobs.ErrJobNotFound
	}
	setDetails(job, d)
	return *job, nil
}

func setDetails(job *jobs.Job, d jobs.Details) {
	job.Company, job.Role, job.Description, job.Link = d.Company, d.Role, d.Description, d.Link
	job.Salary, job.Location, job.Notes, job.DateApplied, job.Interest = d.Salary, d.Location, d.Notes, d.DateApplied, d.Interest
}

func (f *fakeJobs) MoveJob(_ context.Context, m jobs.Move) error {
	if err := f.change(); err != nil {
		return err
	}
	job, ok := f.jobs[m.JobID]
	if !ok {
		return jobs.ErrJobNotFound
	}
	f.moves = append(f.moves, m)
	job.Status, job.ClosingReason, job.ClosingReasonOther = m.Status, m.ClosingReason, m.ClosingReasonOther
	job.ClosingReasonText = reasonText(m.ClosingReason, m.ClosingReasonOther)
	f.place(job.ID, m.Index)
	return nil
}

// place puts a job at index in its column, from 0 at the top, or at the bottom when index is
// past it, as the jobs service does.
func (f *fakeJobs) place(id int64, index int) {
	f.order = slices.DeleteFunc(f.order, func(o int64) bool { return o == id })
	at, seen := len(f.order), 0
	for i, o := range f.order {
		if f.jobs[o].Status != f.jobs[id].Status {
			continue
		}
		if seen == index {
			at = i
			break
		}
		seen++
		at = i + 1
	}
	f.order = slices.Insert(f.order, at, id)
}

func (f *fakeJobs) SetClosingReason(_ context.Context, id int64, reason jobs.ClosingReason, other string) error {
	if err := f.change(); err != nil {
		return err
	}
	job, ok := f.jobs[id]
	if !ok {
		return jobs.ErrJobNotFound
	}
	job.ClosingReason, job.ClosingReasonOther, job.ClosingReasonText = reason, other, reasonText(reason, other)
	return nil
}

// reasonText is how a closing reason reads, as the jobs service gives it: Other's own words, or the
// reason's label.
func reasonText(reason jobs.ClosingReason, other string) string {
	if reason == jobs.ReasonOther {
		return other
	}
	return reason.Label()
}

func (f *fakeJobs) DeleteJob(_ context.Context, id int64) error {
	if err := f.change(); err != nil {
		return err
	}
	if _, ok := f.jobs[id]; !ok {
		return jobs.ErrJobNotFound
	}
	delete(f.jobs, id)
	f.order = slices.DeleteFunc(f.order, func(o int64) bool { return o == id })
	return nil
}

func (f *fakeJobs) SetNextAction(_ context.Context, id int64, text, date string) (jobs.Job, error) {
	if err := f.change(); err != nil {
		return jobs.Job{}, err
	}
	job, ok := f.jobs[id]
	if !ok {
		return jobs.Job{}, jobs.ErrJobNotFound
	}
	job.NextAction = &jobs.NextAction{Text: text, ExpectedDate: date}
	return *job, nil
}

func (f *fakeJobs) ClearNextAction(_ context.Context, id int64) (jobs.Job, error) {
	if err := f.change(); err != nil {
		return jobs.Job{}, err
	}
	job, ok := f.jobs[id]
	if !ok {
		return jobs.Job{}, jobs.ErrJobNotFound
	}
	job.NextAction = nil
	return *job, nil
}

// contactAt finds a contact on any job, with the job it's on.
func (f *fakeJobs) contactAt(id int64) (*jobs.Job, int, bool) {
	for _, job := range f.jobs {
		if i := slices.IndexFunc(job.Contacts, func(c jobs.Contact) bool { return c.ID == id }); i >= 0 {
			return job, i, true
		}
	}
	return nil, 0, false
}

func (f *fakeJobs) Contact(_ context.Context, id int64) (jobs.Contact, error) {
	job, i, ok := f.contactAt(id)
	if !ok {
		return jobs.Contact{}, jobs.ErrContactNotFound
	}
	return job.Contacts[i], nil
}

func (f *fakeJobs) AddContact(_ context.Context, jobID int64, d jobs.ContactDetails) (jobs.Contact, error) {
	if err := f.change(); err != nil {
		return jobs.Contact{}, err
	}
	job, ok := f.jobs[jobID]
	if !ok {
		return jobs.Contact{}, jobs.ErrJobNotFound
	}
	f.nextID++
	c := contactFrom(f.nextID, jobID, d)
	job.Contacts = append(job.Contacts, c)
	return c, nil
}

func contactFrom(id, jobID int64, d jobs.ContactDetails) jobs.Contact {
	return jobs.Contact{
		ID: id, JobID: jobID, Name: d.Name, Relationship: d.Relationship, Email: d.Email,
		Phone: d.Phone, ProfileLink: d.ProfileLink, Notes: d.Notes,
	}
}

func (f *fakeJobs) UpdateContact(_ context.Context, id int64, d jobs.ContactDetails) (jobs.Contact, error) {
	if err := f.change(); err != nil {
		return jobs.Contact{}, err
	}
	job, i, ok := f.contactAt(id)
	if !ok {
		return jobs.Contact{}, jobs.ErrContactNotFound
	}
	job.Contacts[i] = contactFrom(id, job.ID, d)
	return job.Contacts[i], nil
}

func (f *fakeJobs) RemoveContact(_ context.Context, id int64) error {
	if err := f.change(); err != nil {
		return err
	}
	job, i, ok := f.contactAt(id)
	if !ok {
		return jobs.ErrContactNotFound
	}
	job.Contacts = slices.Delete(job.Contacts, i, i+1)
	return nil
}

// interviewAt finds an interview on any job, with the job it's on.
func (f *fakeJobs) interviewAt(id int64) (*jobs.Job, int, bool) {
	for _, job := range f.jobs {
		if i := slices.IndexFunc(job.Interviews, func(v jobs.Interview) bool { return v.ID == id }); i >= 0 {
			return job, i, true
		}
	}
	return nil, 0, false
}

func (f *fakeJobs) Interview(_ context.Context, id int64) (jobs.Interview, error) {
	job, i, ok := f.interviewAt(id)
	if !ok {
		return jobs.Interview{}, jobs.ErrInterviewNotFound
	}
	return job.Interviews[i], nil
}

func (f *fakeJobs) AddInterview(_ context.Context, jobID int64, d jobs.InterviewDetails) (jobs.Interview, error) {
	if err := f.change(); err != nil {
		return jobs.Interview{}, err
	}
	job, ok := f.jobs[jobID]
	if !ok {
		return jobs.Interview{}, jobs.ErrJobNotFound
	}
	f.nextID++
	v := jobs.Interview{ID: f.nextID, JobID: jobID, Date: d.Date, Time: d.Time, Kind: d.Kind, Notes: d.Notes}
	job.Interviews = append(job.Interviews, v)
	return v, nil
}

func (f *fakeJobs) UpdateInterview(_ context.Context, id int64, d jobs.InterviewDetails) (jobs.Interview, error) {
	if err := f.change(); err != nil {
		return jobs.Interview{}, err
	}
	job, i, ok := f.interviewAt(id)
	if !ok {
		return jobs.Interview{}, jobs.ErrInterviewNotFound
	}
	job.Interviews[i] = jobs.Interview{ID: id, JobID: job.ID, Date: d.Date, Time: d.Time, Kind: d.Kind, Notes: d.Notes}
	return job.Interviews[i], nil
}

func (f *fakeJobs) RemoveInterview(_ context.Context, id int64) error {
	if err := f.change(); err != nil {
		return err
	}
	job, i, ok := f.interviewAt(id)
	if !ok {
		return jobs.ErrInterviewNotFound
	}
	job.Interviews = slices.Delete(job.Interviews, i, i+1)
	return nil
}

// documentAt finds an attached document on any job, with the job it's on.
func (f *fakeJobs) documentAt(id int64) (*jobs.Job, int, bool) {
	for _, job := range f.jobs {
		if i := slices.IndexFunc(job.Documents, func(d jobs.Document) bool { return d.ID == id }); i >= 0 {
			return job, i, true
		}
	}
	return nil, 0, false
}

func (f *fakeJobs) Document(_ context.Context, id int64) (jobs.Document, error) {
	job, i, ok := f.documentAt(id)
	if !ok {
		return jobs.Document{}, jobs.ErrDocumentNotFound
	}
	return job.Documents[i], nil
}

// AttachDocument attaches the file at src under its base name, with a stored copy in
// /data/documents.
func (f *fakeJobs) AttachDocument(_ context.Context, jobID int64, src string, kind jobs.DocumentKind) (jobs.Document, error) {
	if err := f.change(); err != nil {
		return jobs.Document{}, err
	}
	job, ok := f.jobs[jobID]
	if !ok {
		return jobs.Document{}, jobs.ErrJobNotFound
	}
	f.nextID++
	doc := jobs.Document{ID: f.nextID, JobID: jobID, Kind: kind, FileName: filepath.Base(src)}
	job.Documents = append(job.Documents, doc)
	f.files[jobID] = append(f.files[jobID], jobs.DocumentFile{DocumentID: doc.ID, Path: storedPath(doc.ID)})
	return doc, nil
}

// storedPath is where the fake keeps a document's stored copy.
func storedPath(id int64) string { return fmt.Sprintf("/data/documents/%d.pdf", id) }

func (f *fakeJobs) SetDocumentKind(_ context.Context, id int64, kind jobs.DocumentKind) error {
	if err := f.change(); err != nil {
		return err
	}
	job, i, ok := f.documentAt(id)
	if !ok {
		return jobs.ErrDocumentNotFound
	}
	job.Documents[i].Kind = kind
	return nil
}

func (f *fakeJobs) RemoveDocument(_ context.Context, id int64) error {
	if err := f.change(); err != nil {
		return err
	}
	job, i, ok := f.documentAt(id)
	if !ok {
		return jobs.ErrDocumentNotFound
	}
	job.Documents = slices.Delete(job.Documents, i, i+1)
	f.files[job.ID] = slices.DeleteFunc(f.files[job.ID], func(file jobs.DocumentFile) bool { return file.DocumentID == id })
	return nil
}

// libraryAt finds a document in the library.
func (f *fakeJobs) libraryAt(id int64) (int, bool) {
	i := slices.IndexFunc(f.library, func(d jobs.LibraryDocument) bool { return d.ID == id })
	return i, i >= 0
}

func (f *fakeJobs) Library(context.Context) ([]jobs.LibraryDocument, error) {
	return slices.Clone(f.library), nil
}

func (f *fakeJobs) LibraryDocument(_ context.Context, id int64) (jobs.LibraryDocument, error) {
	i, ok := f.libraryAt(id)
	if !ok {
		return jobs.LibraryDocument{}, jobs.ErrLibraryDocumentNotFound
	}
	return f.library[i], nil
}

func (f *fakeJobs) LibraryFiles(context.Context) ([]jobs.DocumentFile, error) {
	files := make([]jobs.DocumentFile, 0, len(f.library))
	for _, d := range f.library {
		files = append(files, f.libraryFiles[d.ID])
	}
	return files, nil
}

// AddLibraryDocument adds the file at src, under its base name, at the top of the library, with a
// stored copy in /data/documents.
func (f *fakeJobs) AddLibraryDocument(_ context.Context, src, description string) (jobs.LibraryDocument, error) {
	if err := f.change(); err != nil {
		return jobs.LibraryDocument{}, err
	}
	f.nextID++
	doc := jobs.LibraryDocument{ID: f.nextID, FileName: filepath.Base(src), Description: description}
	f.library = slices.Insert(f.library, 0, doc)
	f.libraryFiles[doc.ID] = jobs.DocumentFile{DocumentID: doc.ID, Path: storedPath(doc.ID)}
	return doc, nil
}

func (f *fakeJobs) SetLibraryDescription(_ context.Context, id int64, description string) (jobs.LibraryDocument, error) {
	if err := f.change(); err != nil {
		return jobs.LibraryDocument{}, err
	}
	i, ok := f.libraryAt(id)
	if !ok {
		return jobs.LibraryDocument{}, jobs.ErrLibraryDocumentNotFound
	}
	f.library[i].Description = description
	return f.library[i], nil
}

func (f *fakeJobs) RemoveLibraryDocument(_ context.Context, id int64) error {
	if err := f.change(); err != nil {
		return err
	}
	i, ok := f.libraryAt(id)
	if !ok {
		return jobs.ErrLibraryDocumentNotFound
	}
	f.library = slices.Delete(f.library, i, i+1)
	delete(f.libraryFiles, id)
	return nil
}

// fakeBackups counts the daily backups and removals of expired backups it was asked for, and keeps
// the agent backups it's given. Which backups expire, and the rules for names and days, are the
// backups service's.
type fakeBackups struct {
	calls int
	err   error
	// removals counts the calls to RemoveExpired, which returns removeErr.
	removals  int
	removeErr error
	agent     []backups.AgentBackup
	// listed are the daily and app backups List lists before the agent backups.
	listed []backups.Backup
	// restored lists the names restored, and undo is the backup each restore makes first.
	restored []string
	undo     backups.AgentBackup
	// fail is returned by the next MakeAgent or Restore, which then changes nothing.
	fail error
}

func (b *fakeBackups) TakeDaily(context.Context) (bool, error) {
	b.calls++
	return b.err == nil, b.err
}

func (b *fakeBackups) RemoveExpired() error {
	b.removals++
	return b.removeErr
}

// failure returns the next call's failure, clearing it.
func (b *fakeBackups) failure() error {
	err := b.fail
	b.fail = nil
	return err
}

// MakeAgent keeps the backup until days after 2026-10-14.
func (b *fakeBackups) MakeAgent(_ context.Context, name string, days int) (backups.AgentBackup, error) {
	if err := b.failure(); err != nil {
		return backups.AgentBackup{}, err
	}
	made := backups.AgentBackup{Name: name, KeptUntil: time.Date(2026, time.October, 14+days, 0, 0, 0, 0, time.UTC).Format("2006-01-02")}
	b.agent = append(b.agent, made)
	return made, nil
}

// List lists the backups in listed, then the agent backups the fake made or was given.
func (b *fakeBackups) List() ([]backups.Backup, error) {
	out := slices.Clone(b.listed)
	for _, a := range b.agent {
		out = append(out, backups.Backup{Kind: backups.Agent, Name: a.Name, KeptUntil: a.KeptUntil})
	}
	return out, nil
}

func (b *fakeBackups) Restore(_ context.Context, name string) (backups.AgentBackup, error) {
	if err := b.failure(); err != nil {
		return backups.AgentBackup{}, err
	}
	b.restored = append(b.restored, name)
	b.agent = append(b.agent, b.undo)
	return b.undo, nil
}

// watchedReader is stdin that records whether anything read it.
type watchedReader struct {
	io.Reader
	read bool
}

func (r *watchedReader) Read(p []byte) (int, error) {
	r.read = true
	return r.Reader.Read(p)
}

// harness runs the command line over fakes.
type harness struct {
	jobs    *fakeJobs
	backups *fakeBackups
	// files are the files ReadFile can read, by path.
	files map[string]string
	stdin *watchedReader
	// openErr is what opening the data returns.
	openErr error
	opened  int
	closed  int
}

func newHarness() *harness {
	return &harness{jobs: newFakeJobs(), backups: &fakeBackups{}, files: map[string]string{}, stdin: &watchedReader{Reader: strings.NewReader("")}}
}

// result is what a run printed and the exit code it ended with.
type result struct {
	stdout, stderr string
	code           int
}

func (h *harness) run(args ...string) result {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), args, Env{
		Stdin:  h.stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		ReadFile: func(name string) ([]byte, error) {
			text, ok := h.files[name]
			if !ok {
				return nil, fs.ErrNotExist
			}
			return []byte(text), nil
		},
		Open: func(context.Context) (Services, func() error, error) {
			h.opened++
			if h.openErr != nil {
				return Services{}, nil, h.openErr
			}
			return Services{Jobs: h.jobs, Backups: h.backups}, func() error { h.closed++; return nil }, nil
		},
	})
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// wantCode fails the test unless r ended with code.
func (r result) wantCode(t *testing.T, code int) {
	t.Helper()
	if r.code != code {
		t.Errorf("exit code = %d, want %d\nstdout: %s\nstderr: %s", r.code, code, r.stdout, r.stderr)
	}
}

// acme adds a job for "Software Engineer" at "Acme" in Saved.
func (h *harness) acme() *jobs.Job {
	return h.jobs.add("Acme", "Software Engineer", jobs.StatusSaved)
}

var errUnexpected = errors.New("disk I/O error")
