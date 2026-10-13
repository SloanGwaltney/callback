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
	tables
	// fail makes the named method return errFake.
	fail string
}

// tables is the fake's data, which InTx snapshots. A new table goes here and in clone, and
// nowhere else, for rollback to cover it.
type tables struct {
	jobs       map[int64]store.Job
	docs       map[int64]store.Document
	contacts   map[int64]store.Contact
	interviews map[int64]store.Interview
	nextID     int64
}

func (t tables) clone() tables {
	return tables{
		jobs:       maps.Clone(t.jobs),
		docs:       maps.Clone(t.docs),
		contacts:   maps.Clone(t.contacts),
		interviews: maps.Clone(t.interviews),
		nextID:     t.nextID,
	}
}

var errFake = errors.New("fake failure")

var _ Store = (*fakeStore)(nil)

func newFakeStore() *fakeStore {
	return &fakeStore{tables: tables{
		jobs:       map[int64]store.Job{},
		docs:       map[int64]store.Document{},
		contacts:   map[int64]store.Contact{},
		interviews: map[int64]store.Interview{},
	}}
}

func (f *fakeStore) InTx(ctx context.Context, fn func(store.Querier) error) error {
	saved := f.tables.clone()
	if err := fn(f); err != nil {
		f.tables = saved
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

func (f *fakeStore) CreateContact(ctx context.Context, arg store.CreateContactParams) (store.Contact, error) {
	if err := f.failing("CreateContact"); err != nil {
		return store.Contact{}, err
	}
	if _, ok := f.jobs[arg.JobID]; !ok {
		return store.Contact{}, errors.New("FOREIGN KEY constraint failed")
	}
	f.nextID++
	c := store.Contact{
		ID: f.nextID, JobID: arg.JobID, Name: arg.Name, Relationship: arg.Relationship, Email: arg.Email,
		Phone: arg.Phone, ProfileLink: arg.ProfileLink, Notes: arg.Notes,
	}
	f.contacts[c.ID] = c
	return c, nil
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

func (f *fakeStore) CreateInterview(ctx context.Context, arg store.CreateInterviewParams) (store.Interview, error) {
	if err := f.failing("CreateInterview"); err != nil {
		return store.Interview{}, err
	}
	if _, ok := f.jobs[arg.JobID]; !ok {
		return store.Interview{}, errors.New("FOREIGN KEY constraint failed")
	}
	f.nextID++
	i := store.Interview{ID: f.nextID, JobID: arg.JobID, Date: arg.Date, Time: arg.Time, Kind: arg.Kind, Notes: arg.Notes}
	f.interviews[i.ID] = i
	return i, nil
}

func (f *fakeStore) CreateJob(ctx context.Context, arg store.CreateJobParams) (int64, error) {
	if err := f.failing("CreateJob"); err != nil {
		return 0, err
	}
	f.nextID++
	f.jobs[f.nextID] = store.Job{
		ID: f.nextID, Company: arg.Company, Role: arg.Role, Description: arg.Description, Link: arg.Link,
		Salary: arg.Salary, Location: arg.Location, Notes: arg.Notes, Status: arg.Status,
		Position: arg.Position, DateApplied: arg.DateApplied, Interest: arg.Interest,
	}
	return f.nextID, nil
}

func (f *fakeStore) DeleteContact(ctx context.Context, id int64) error {
	delete(f.contacts, id)
	return nil
}

func (f *fakeStore) DeleteDocument(ctx context.Context, id int64) error {
	delete(f.docs, id)
	return nil
}

func (f *fakeStore) DeleteInterview(ctx context.Context, id int64) error {
	delete(f.interviews, id)
	return nil
}

func (f *fakeStore) DeleteJob(ctx context.Context, id int64) error {
	delete(f.jobs, id)
	maps.DeleteFunc(f.docs, func(_ int64, d store.Document) bool { return d.JobID == id })
	maps.DeleteFunc(f.contacts, func(_ int64, c store.Contact) bool { return c.JobID == id })
	maps.DeleteFunc(f.interviews, func(_ int64, i store.Interview) bool { return i.JobID == id })
	return nil
}

func (f *fakeStore) GetContact(ctx context.Context, id int64) (store.Contact, error) {
	c, ok := f.contacts[id]
	if !ok {
		return store.Contact{}, sql.ErrNoRows
	}
	return c, nil
}

func (f *fakeStore) GetDocument(ctx context.Context, id int64) (store.Document, error) {
	doc, ok := f.docs[id]
	if !ok {
		return store.Document{}, sql.ErrNoRows
	}
	return doc, nil
}

func (f *fakeStore) GetInterview(ctx context.Context, id int64) (store.Interview, error) {
	i, ok := f.interviews[id]
	if !ok {
		return store.Interview{}, sql.ErrNoRows
	}
	return i, nil
}

func (f *fakeStore) GetJob(ctx context.Context, id int64) (store.Job, error) {
	job, ok := f.jobs[id]
	if !ok {
		return store.Job{}, sql.ErrNoRows
	}
	return job, nil
}

func (f *fakeStore) ListContactsForJob(ctx context.Context, jobID int64) ([]store.Contact, error) {
	var out []store.Contact
	for _, c := range f.contacts {
		if c.JobID == jobID {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b store.Contact) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
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

// ListInterviewsForJob orders like the real query: by date, then time with untimed last, then ID.
func (f *fakeStore) ListInterviewsForJob(ctx context.Context, jobID int64) ([]store.Interview, error) {
	var out []store.Interview
	for _, i := range f.interviews {
		if i.JobID == jobID {
			out = append(out, i)
		}
	}
	slices.SortFunc(out, interviewOrder)
	return out, nil
}

func (f *fakeStore) ListUpcomingInterviews(ctx context.Context, date string) ([]store.Interview, error) {
	if err := f.failing("ListUpcomingInterviews"); err != nil {
		return nil, err
	}
	var out []store.Interview
	for _, i := range f.interviews {
		if i.Date >= date {
			out = append(out, i)
		}
	}
	slices.SortFunc(out, func(a, b store.Interview) int {
		return cmp.Or(cmp.Compare(a.JobID, b.JobID), interviewOrder(a, b))
	})
	return out, nil
}

func interviewOrder(a, b store.Interview) int {
	untimed := func(i store.Interview) int {
		if i.Time == "" {
			return 1
		}
		return 0
	}
	return cmp.Or(strings.Compare(a.Date, b.Date), cmp.Compare(untimed(a), untimed(b)),
		strings.Compare(a.Time, b.Time), cmp.Compare(a.ID, b.ID))
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
			Interest: j.Interest, DocumentCount: count,
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

func (f *fakeStore) UpdateContact(ctx context.Context, arg store.UpdateContactParams) error {
	if err := f.failing("UpdateContact"); err != nil {
		return err
	}
	c, ok := f.contacts[arg.ID]
	if ok {
		c.Name, c.Relationship, c.Email = arg.Name, arg.Relationship, arg.Email
		c.Phone, c.ProfileLink, c.Notes = arg.Phone, arg.ProfileLink, arg.Notes
		f.contacts[arg.ID] = c
	}
	return nil
}

func (f *fakeStore) UpdateInterview(ctx context.Context, arg store.UpdateInterviewParams) error {
	if err := f.failing("UpdateInterview"); err != nil {
		return err
	}
	i, ok := f.interviews[arg.ID]
	if ok {
		i.Date, i.Time, i.Kind, i.Notes = arg.Date, arg.Time, arg.Kind, arg.Notes
		f.interviews[arg.ID] = i
	}
	return nil
}

func (f *fakeStore) UpdateJobDetails(ctx context.Context, arg store.UpdateJobDetailsParams) error {
	return f.update(arg.ID, func(j *store.Job) {
		j.Company, j.Role, j.Description, j.Link = arg.Company, arg.Role, arg.Description, arg.Link
		j.Salary, j.Location, j.Notes, j.DateApplied = arg.Salary, arg.Location, arg.Notes, arg.DateApplied
		j.Interest = arg.Interest
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

// seedContact adds a contact straight in the store, bypassing the service.
func (h *harness) seedContact(jobID int64, c store.Contact) int64 {
	h.store.nextID++
	c.ID, c.JobID = h.store.nextID, jobID
	if c.Relationship == "" {
		c.Relationship = string(RelationshipRecruiter)
	}
	h.store.contacts[c.ID] = c
	return c.ID
}

// seedInterview adds an interview straight in the store, bypassing the service.
func (h *harness) seedInterview(jobID int64, i store.Interview) int64 {
	h.store.nextID++
	i.ID, i.JobID = h.store.nextID, jobID
	h.store.interviews[i.ID] = i
	return i.ID
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
