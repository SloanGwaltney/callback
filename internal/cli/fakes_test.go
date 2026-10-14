package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"slices"
	"strings"
	"testing"

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
	return &fakeJobs{jobs: map[int64]*jobs.Job{}, days: map[int64]int{}, files: map[int64][]jobs.DocumentFile{}, nextID: 100}
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

// fakeBackups counts the daily backups it was asked for.
type fakeBackups struct {
	calls int
	err   error
}

func (b *fakeBackups) TakeDaily(context.Context) (bool, error) {
	b.calls++
	return b.err == nil, b.err
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
