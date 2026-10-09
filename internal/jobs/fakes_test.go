package jobs

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"callback/internal/store"
)

// Hand-written fakes for the service's dependencies. See
// docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md and
// docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md.

// fakeStore is an in-memory Store. InTx snapshots the data and restores it if fn fails, so
// rollback can be checked too.
type fakeStore struct {
	jobs   map[int64]store.Job
	docs   map[int64]store.Document
	nextID int64
	// fail makes the named method return errFake.
	fail string
}

var errFake = errors.New("fake failure")

var _ Store = (*fakeStore)(nil)

func newFakeStore() *fakeStore {
	return &fakeStore{jobs: map[int64]store.Job{}, docs: map[int64]store.Document{}}
}

func (f *fakeStore) InTx(ctx context.Context, fn func(store.Querier) error) error {
	jobs, docs, next := maps.Clone(f.jobs), maps.Clone(f.docs), f.nextID
	if err := fn(f); err != nil {
		f.jobs, f.docs, f.nextID = jobs, docs, next
		return err
	}
	return nil
}

func (f *fakeStore) failing(method string) error {
	if f.fail == method {
		return errFake
	}
	return nil
}

func (f *fakeStore) CreateDocument(ctx context.Context, arg store.CreateDocumentParams) (store.Document, error) {
	if err := f.failing("CreateDocument"); err != nil {
		return store.Document{}, err
	}
	if _, ok := f.jobs[arg.JobID]; !ok {
		return store.Document{}, errors.New("FOREIGN KEY constraint failed")
	}
	f.nextID++
	doc := store.Document{ID: f.nextID, JobID: arg.JobID, Kind: arg.Kind, FileName: arg.FileName, StoredPath: arg.StoredPath}
	f.docs[doc.ID] = doc
	return doc, nil
}

func (f *fakeStore) CreateJob(ctx context.Context, arg store.CreateJobParams) (int64, error) {
	if err := f.failing("CreateJob"); err != nil {
		return 0, err
	}
	f.nextID++
	f.jobs[f.nextID] = store.Job{
		ID: f.nextID, Company: arg.Company, Role: arg.Role, Description: arg.Description, Link: arg.Link,
		Salary: arg.Salary, Location: arg.Location, Notes: arg.Notes, Status: arg.Status,
		Position: arg.Position, DateApplied: arg.DateApplied,
	}
	return f.nextID, nil
}

func (f *fakeStore) DeleteDocument(ctx context.Context, id int64) error {
	delete(f.docs, id)
	return nil
}

func (f *fakeStore) DeleteJob(ctx context.Context, id int64) error {
	delete(f.jobs, id)
	maps.DeleteFunc(f.docs, func(_ int64, d store.Document) bool { return d.JobID == id })
	return nil
}

func (f *fakeStore) GetDocument(ctx context.Context, id int64) (store.Document, error) {
	doc, ok := f.docs[id]
	if !ok {
		return store.Document{}, sql.ErrNoRows
	}
	return doc, nil
}

func (f *fakeStore) GetJob(ctx context.Context, id int64) (store.Job, error) {
	job, ok := f.jobs[id]
	if !ok {
		return store.Job{}, sql.ErrNoRows
	}
	return job, nil
}

func (f *fakeStore) ListDocumentsForJob(ctx context.Context, jobID int64) ([]store.Document, error) {
	var out []store.Document
	for _, d := range f.docs {
		if d.JobID == jobID {
			out = append(out, d)
		}
	}
	slices.SortFunc(out, func(a, b store.Document) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *fakeStore) ListJobIDsInStatus(ctx context.Context, status string) ([]int64, error) {
	var jobs []store.Job
	for _, j := range f.jobs {
		if j.Status == status {
			jobs = append(jobs, j)
		}
	}
	slices.SortFunc(jobs, byPosition)
	ids := make([]int64, len(jobs))
	for i, j := range jobs {
		ids[i] = j.ID
	}
	return ids, nil
}

func (f *fakeStore) ListJobs(ctx context.Context) ([]store.ListJobsRow, error) {
	jobs := slices.Collect(maps.Values(f.jobs))
	slices.SortFunc(jobs, func(a, b store.Job) int {
		return cmp.Or(strings.Compare(a.Status, b.Status), byPosition(a, b))
	})
	rows := make([]store.ListJobsRow, len(jobs))
	for i, j := range jobs {
		var count int64
		for _, d := range f.docs {
			if d.JobID == j.ID {
				count++
			}
		}
		rows[i] = store.ListJobsRow{
			ID: j.ID, Company: j.Company, Role: j.Role, Description: j.Description, Link: j.Link,
			Salary: j.Salary, Location: j.Location, Notes: j.Notes, Status: j.Status, Position: j.Position,
			DateApplied: j.DateApplied, ClosingReason: j.ClosingReason, ClosingReasonOther: j.ClosingReasonOther,
			DocumentCount: count,
		}
	}
	return rows, nil
}

func byPosition(a, b store.Job) int {
	return cmp.Or(cmp.Compare(a.Position, b.Position), cmp.Compare(a.ID, b.ID))
}

func (f *fakeStore) SetClosingReason(ctx context.Context, arg store.SetClosingReasonParams) error {
	return f.update(arg.ID, func(j *store.Job) {
		j.ClosingReason, j.ClosingReasonOther = arg.ClosingReason, arg.ClosingReasonOther
	})
}

func (f *fakeStore) SetDocumentKind(ctx context.Context, arg store.SetDocumentKindParams) error {
	doc, ok := f.docs[arg.ID]
	if ok {
		doc.Kind = arg.Kind
		f.docs[arg.ID] = doc
	}
	return nil
}

func (f *fakeStore) SetJobPosition(ctx context.Context, arg store.SetJobPositionParams) error {
	return f.update(arg.ID, func(j *store.Job) { j.Position = arg.Position })
}

func (f *fakeStore) UpdateJobDetails(ctx context.Context, arg store.UpdateJobDetailsParams) error {
	return f.update(arg.ID, func(j *store.Job) {
		j.Company, j.Role, j.Description, j.Link = arg.Company, arg.Role, arg.Description, arg.Link
		j.Salary, j.Location, j.Notes, j.DateApplied = arg.Salary, arg.Location, arg.Notes, arg.DateApplied
	})
}

func (f *fakeStore) UpdateJobPlacement(ctx context.Context, arg store.UpdateJobPlacementParams) error {
	if err := f.failing("UpdateJobPlacement"); err != nil {
		return err
	}
	return f.update(arg.ID, func(j *store.Job) {
		j.Status, j.Position, j.DateApplied = arg.Status, arg.Position, arg.DateApplied
		j.ClosingReason, j.ClosingReasonOther = arg.ClosingReason, arg.ClosingReasonOther
	})
}

func (f *fakeStore) update(id int64, change func(*store.Job)) error {
	job, ok := f.jobs[id]
	if ok {
		change(&job)
		f.jobs[id] = job
	}
	return nil
}

// fakeFiles records document copies in memory.
type fakeFiles struct {
	// sizes gives a source file's size; files not listed are 1 KB.
	sizes map[string]int64
	// copies maps each stored path to the source it was copied from.
	copies  map[string]string
	removed []string
	next    int
	// failRemove makes Remove fail.
	failRemove bool
}

var _ Files = (*fakeFiles)(nil)

func (f *fakeFiles) Size(src string) (int64, error) {
	if size, ok := f.sizes[src]; ok {
		return size, nil
	}
	return 1 << 10, nil
}

func (f *fakeFiles) Save(src string) (string, error) {
	f.next++
	stored := fmt.Sprintf("documents/%d%s", f.next, strings.ToLower(filepath.Ext(src)))
	f.copies[stored] = src
	return stored, nil
}

func (f *fakeFiles) Remove(stored string) error {
	if f.failRemove {
		return errFake
	}
	f.removed = append(f.removed, stored)
	delete(f.copies, stored)
	return nil
}

func (f *fakeFiles) Path(stored string) string { return "/data/" + stored }

// fakeOpener records what it was asked to open.
type fakeOpener struct {
	opened []string
}

func (o *fakeOpener) Open(target string) error {
	o.opened = append(o.opened, target)
	return nil
}

// harness is a Service wired to fakes, with today fixed at 2026-10-09.
type harness struct {
	svc    *Service
	store  *fakeStore
	files  *fakeFiles
	opener *fakeOpener
	ctx    context.Context
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{
		store:  newFakeStore(),
		files:  &fakeFiles{sizes: map[string]int64{}, copies: map[string]string{}},
		opener: &fakeOpener{},
		ctx:    context.Background(),
	}
	now := func() time.Time { return time.Date(2026, 10, 9, 12, 0, 0, 0, time.Local) }
	h.svc = New(h.store, h.files, h.opener, now)
	return h
}

// seed puts a job straight into the store at the bottom of its column, bypassing the service.
// Status defaults to Saved and role to "Software Engineer"; a closed job gets "No response".
func (h *harness) seed(job store.Job) int64 {
	if job.Status == "" {
		job.Status = string(StatusSaved)
	}
	if job.Role == "" {
		job.Role = "Software Engineer"
	}
	if job.Status == string(StatusClosed) && !job.ClosingReason.Valid {
		job.ClosingReason = nullString(string(ReasonNoResponse))
	}
	ids, _ := h.store.ListJobIDsInStatus(h.ctx, job.Status)
	job.Position = int64(len(ids))
	h.store.nextID++
	job.ID = h.store.nextID
	h.store.jobs[job.ID] = job
	return job.ID
}

// seedDoc attaches a document straight in the store and the fake files.
func (h *harness) seedDoc(jobID int64, fileName string, kind DocumentKind) int64 {
	h.store.nextID++
	stored := fmt.Sprintf("documents/seed-%d%s", h.store.nextID, filepath.Ext(fileName))
	h.files.copies[stored] = fileName
	h.store.docs[h.store.nextID] = store.Document{ID: h.store.nextID, JobID: jobID, Kind: string(kind), FileName: fileName, StoredPath: stored}
	return h.store.nextID
}

func (h *harness) acme(status Status) int64 {
	return h.seed(store.Job{Company: "Acme", Status: string(status)})
}

func (h *harness) board(t *testing.T) Board {
	t.Helper()
	b, err := h.svc.Board(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (h *harness) job(t *testing.T, id int64) Job {
	t.Helper()
	j, err := h.svc.Job(h.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

// column returns the companies in a status column, top to bottom.
func (h *harness) column(t *testing.T, status Status) []string {
	t.Helper()
	for _, c := range h.board(t).Columns {
		if c.Status == status {
			var companies []string
			for _, card := range c.Cards {
				companies = append(companies, card.Company)
			}
			return companies
		}
	}
	t.Fatalf("no %q column", status)
	return nil
}

// card returns the card for a job and the status column it's in.
func (h *harness) card(t *testing.T, id int64) (Card, Status) {
	t.Helper()
	for _, c := range h.board(t).Columns {
		for _, card := range c.Cards {
			if card.ID == id {
				return card, c.Status
			}
		}
	}
	t.Fatalf("job %d isn't on the board", id)
	return Card{}, ""
}

func mustMove(t *testing.T, h *harness, m Move) {
	t.Helper()
	if err := h.svc.MoveJob(h.ctx, m); err != nil {
		t.Fatal(err)
	}
}

// wantValidation checks that err is a ValidationError with the message want.
func wantValidation(t *testing.T, err error, want string) {
	t.Helper()
	var v *ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("err = %v, want a ValidationError %q", err, want)
	}
	if v.Msg != want {
		t.Errorf("message = %q, want %q", v.Msg, want)
	}
}

// statusFromLabel maps a column title from the spec to its Status.
func statusFromLabel(t *testing.T, label string) Status {
	t.Helper()
	for _, s := range Statuses {
		if s.Label() == label {
			return s
		}
	}
	t.Fatalf("no status labelled %q", label)
	return ""
}
