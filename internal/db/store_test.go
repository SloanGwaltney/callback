package db

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"callback/internal/db/migrations"
	"callback/internal/store"
)

// Storage-tier tests for Store and the sqlc queries, against a real SQLite file in a temporary
// directory. See docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

func openStore(t *testing.T, dataDir string) (*Store, *sql.DB) {
	t.Helper()
	conn, err := Open(context.Background(), dataDir, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return NewStore(conn), conn
}

// openBefore opens a database in dataDir with only the migrations older than version, as an
// earlier release left it, to test a migration that changes data already there. Fill it with
// mustExec, close it, then openStore the same dataDir to run the rest:
//
//	old := openBefore(t, dataDir, "00007")
//	mustExec(t, old, `INSERT INTO jobs ...`)
//	old.Close()
//	s, _ := openStore(t, dataDir)
func openBefore(t *testing.T, dataDir, version string) *sql.DB {
	t.Helper()
	older := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		// Migration file names start with a zero-padded version, so they sort in order.
		if e.Name() >= version {
			continue
		}
		data, err := fs.ReadFile(migrations.FS, e.Name())
		if err != nil {
			t.Fatal(err)
		}
		older[e.Name()] = &fstest.MapFile{Data: data}
	}
	conn, err := Open(context.Background(), dataDir, older)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func createJob(t *testing.T, q store.Querier, company, status string, position int64) int64 {
	t.Helper()
	id, err := q.CreateJob(context.Background(), store.CreateJobParams{
		Company: company, Role: "Software Engineer", Status: status, Position: position,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func companiesIn(t *testing.T, s *Store, status string) []string {
	t.Helper()
	rows, err := s.ListJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range rows {
		if r.Status == status {
			out = append(out, r.Company)
		}
	}
	return out
}

func TestInTxCommits(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()

	err := s.InTx(ctx, func(q store.Querier) error {
		createJob(t, q, "Acme", "saved", 0)
		createJob(t, q, "Globex", "saved", 1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := companiesIn(t, s, "saved"); !slices.Equal(got, []string{"Acme", "Globex"}) {
		t.Errorf("saved = %v", got)
	}
}

func TestInTxRollsBackWhenFnFails(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	boom := errors.New("boom")

	err := s.InTx(ctx, func(q store.Querier) error {
		createJob(t, q, "Acme", "saved", 0)
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if got := companiesIn(t, s, "saved"); len(got) != 0 {
		t.Errorf("rolled-back insert left %v", got)
	}
}

func TestInTxRollsBackOnAConstraintError(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	id := createJob(t, s, "Acme", "interviewing", 0)

	err := s.InTx(ctx, func(q store.Querier) error {
		if err := q.SetJobPosition(ctx, store.SetJobPositionParams{ID: id, Position: 5}); err != nil {
			return err
		}
		// Closed without a reason breaks the schema's CHECK.
		return q.UpdateJobPlacement(ctx, store.UpdateJobPlacementParams{ID: id, Status: "closed"})
	})
	if err == nil {
		t.Fatal("closing without a reason succeeded")
	}
	job, err := s.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "interviewing" || job.Position != 0 {
		t.Errorf("job = %s at %d, want the transaction undone", job.Status, job.Position)
	}
}

func TestGetJobMissingIsErrNoRows(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	if _, err := s.GetJob(context.Background(), 99); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
	if _, err := s.GetDocument(context.Background(), 99); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
	if _, err := s.GetContact(context.Background(), 99); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestListJobIDsInStatusIsInPositionOrder(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	globex := createJob(t, s, "Globex", "applied", 1)
	createJob(t, s, "Hooli", "saved", 0)
	acme := createJob(t, s, "Acme", "applied", 0)

	ids, err := s.ListJobIDsInStatus(ctx, "applied")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ids, []int64{acme, globex}) {
		t.Errorf("ids = %v, want [%d %d]", ids, acme, globex)
	}
}

func TestJobUpdates(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	id := createJob(t, s, "Acme", "interviewing", 0)

	err := s.UpdateJobDetails(ctx, store.UpdateJobDetailsParams{
		ID: id, Company: "Acme", Role: "Senior Software Engineer", Description: "Lead the team.",
		Link: "https://acme.example.com/jobs/123", Salary: "$65/hr", Location: "Remote",
		Notes: "Line one.\nLine two.", DateApplied: sql.NullString{String: "2026-10-01", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = s.UpdateJobPlacement(ctx, store.UpdateJobPlacementParams{
		ID: id, Status: "closed", Position: 3, DateApplied: sql.NullString{String: "2026-10-01", Valid: true},
		ClosingReason:      sql.NullString{String: "other", Valid: true},
		ClosingReasonOther: sql.NullString{String: "Company went through layoffs", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetClosingReason(ctx, store.SetClosingReasonParams{ID: id, ClosingReason: sql.NullString{String: "rejected", Valid: true}}); err != nil {
		t.Fatal(err)
	}

	job, err := s.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	want := store.Job{
		ID: id, Company: "Acme", Role: "Senior Software Engineer", Description: "Lead the team.",
		Link: "https://acme.example.com/jobs/123", Salary: "$65/hr", Location: "Remote",
		Notes: "Line one.\nLine two.", Status: "closed", Position: 3,
		DateApplied:   sql.NullString{String: "2026-10-01", Valid: true},
		ClosingReason: sql.NullString{String: "rejected", Valid: true},
	}
	if job != want {
		t.Errorf("job = %+v\nwant %+v", job, want)
	}
}

func TestDocumentQueries(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "saved", 0)
	globex := createJob(t, s, "Globex", "saved", 1)

	resume, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: acme, Kind: "resume", FileName: "resume-2026.pdf", StoredPath: "documents/a.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	cover, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: acme, Kind: "other", FileName: "acme-cover.docx", StoredPath: "documents/b.docx"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: globex, Kind: "resume", FileName: "resume-2026.pdf", StoredPath: "documents/c.pdf"}); err != nil {
		t.Fatal(err)
	}

	if err := s.SetDocumentKind(ctx, store.SetDocumentKindParams{ID: cover.ID, Kind: "cover_letter"}); err != nil {
		t.Fatal(err)
	}
	docs, err := s.ListDocumentsForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].ID != resume.ID || docs[1].Kind != "cover_letter" {
		t.Errorf("Acme's documents = %+v", docs)
	}

	if err := s.DeleteDocument(ctx, resume.ID); err != nil {
		t.Fatal(err)
	}
	if docs, _ := s.ListDocumentsForJob(ctx, acme); len(docs) != 1 {
		t.Errorf("after delete, Acme has %d documents, want 1", len(docs))
	}
	if docs, _ := s.ListDocumentsForJob(ctx, globex); len(docs) != 1 {
		t.Errorf("deleting Acme's resume touched Globex's: %d left", len(docs))
	}
}

// @scenario [#card-shows-document-count]
func TestListJobsCountsDocuments(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "saved", 0)
	createJob(t, s, "Globex", "saved", 1)
	for i, path := range []string{"documents/a.pdf", "documents/b.docx"} {
		if _, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: acme, Kind: "other", FileName: path, StoredPath: path}); err != nil {
			t.Fatalf("document %d: %v", i, err)
		}
	}

	rows, err := s.ListJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int64{}
	for _, r := range rows {
		counts[r.Company] = r.DocumentCount
	}
	// @then [features/job-documents.feature] the card shows that it has 2 documents
	if counts["Acme"] != 2 || counts["Globex"] != 0 {
		t.Errorf("document counts = %v, want Acme 2 and Globex 0", counts)
	}
}

func TestDeleteJobDeletesItsDocumentRows(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "saved", 0)
	if _, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: acme, Kind: "resume", FileName: "r.pdf", StoredPath: "documents/r.pdf"}); err != nil {
		t.Fatal(err)
	}

	err := s.InTx(ctx, func(q store.Querier) error { return q.DeleteJob(ctx, acme) })
	if err != nil {
		t.Fatal(err)
	}
	if docs, _ := s.ListDocumentsForJob(ctx, acme); len(docs) != 0 {
		t.Errorf("%d document rows left after deleting the job in a transaction", len(docs))
	}
}

func createContact(t *testing.T, q store.Querier, jobID int64, name, relationship string) store.Contact {
	t.Helper()
	c, err := q.CreateContact(context.Background(), store.CreateContactParams{JobID: jobID, Name: name, Relationship: relationship})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestContactQueries(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "saved", 0)
	globex := createJob(t, s, "Globex", "saved", 1)

	dana, err := s.CreateContact(ctx, store.CreateContactParams{
		JobID: acme, Name: "Dana Smith", Relationship: "recruiter", Email: "dana@acme.example.com",
		Phone: "(555) 010-4477", ProfileLink: "https://www.linkedin.com/in/dana-smith", Notes: "Prefers email.",
	})
	if err != nil {
		t.Fatal(err)
	}
	globexDana := createContact(t, s, globex, "Dana Smith", "recruiter")

	err = s.UpdateContact(ctx, store.UpdateContactParams{
		ID: dana.ID, Name: "Dana Smith", Relationship: "hiring_manager", Email: "dana.smith@acme.example.com",
		Phone: "(555) 010-4477", ProfileLink: "https://www.linkedin.com/in/dana-smith", Notes: "Prefers email.",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetContact(ctx, dana.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := store.Contact{
		ID: dana.ID, JobID: acme, Name: "Dana Smith", Relationship: "hiring_manager", Email: "dana.smith@acme.example.com",
		Phone: "(555) 010-4477", ProfileLink: "https://www.linkedin.com/in/dana-smith", Notes: "Prefers email.",
	}
	if got != want {
		t.Errorf("contact = %+v\nwant %+v", got, want)
	}
	if other, _ := s.GetContact(ctx, globexDana.ID); other != globexDana {
		t.Errorf("updating Acme's Dana changed Globex's: %+v", other)
	}

	if err := s.DeleteContact(ctx, dana.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetContact(ctx, dana.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("after delete, err = %v, want sql.ErrNoRows", err)
	}
	if contacts, _ := s.ListContactsForJob(ctx, globex); len(contacts) != 1 {
		t.Errorf("deleting Acme's Dana touched Globex's: %d left", len(contacts))
	}
}

// @scenario [#view-contacts]
func TestListContactsForJobIsInOrderAdded(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme"
	acme := createJob(t, s, "Acme", "saved", 0)
	globex := createJob(t, s, "Globex", "saved", 1)
	// @and [features/job-contacts.feature] I added the contact "Dana Smith" as a recruiter
	createContact(t, s, acme, "Dana Smith", "recruiter")
	createContact(t, s, globex, "Ana Lopez", "referrer")
	// @and [features/job-contacts.feature] I then added the contact "Lee Park" as a hiring manager
	createContact(t, s, acme, "Lee Park", "hiring_manager")

	// @when [features/job-contacts.feature] I open the card
	contacts, err := s.ListContactsForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] I see "Dana Smith" and "Lee Park" in the order I added them
	var names []string
	for _, c := range contacts {
		names = append(names, c.Name)
	}
	if want := []string{"Dana Smith", "Lee Park"}; !slices.Equal(names, want) {
		t.Errorf("contacts = %v, want %v", names, want)
	}
}

// @scenario [#contacts-persist]
func TestContactsPersistAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	ctx := context.Background()
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith" as a recruiter
	acme := createJob(t, s, "Acme", "saved", 0)
	createContact(t, s, acme, "Dana Smith", "recruiter")

	// @when [features/job-contacts.feature] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	// @and [features/job-contacts.feature] I open the card
	contacts, err := reopened.ListContactsForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] the job's contacts show "Dana Smith" as a recruiter
	if len(contacts) != 1 || contacts[0].Name != "Dana Smith" || contacts[0].Relationship != "recruiter" {
		t.Errorf("after reopening: %+v", contacts)
	}
}

// @scenario [#contacts-deleted-with-job]
func TestDeleteJobDeletesItsContactRows(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/job-contacts.feature] I have a job for "Software Engineer" at "Acme" with the contact "Dana Smith"
	acme := createJob(t, s, "Acme", "saved", 0)
	dana := createContact(t, s, acme, "Dana Smith", "recruiter")
	globex := createJob(t, s, "Globex", "saved", 1)
	createContact(t, s, globex, "Dana Smith", "recruiter")

	// @when [features/job-contacts.feature] I delete the job
	err := s.InTx(ctx, func(q store.Querier) error { return q.DeleteJob(ctx, acme) })
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-contacts.feature] its contacts are removed too
	if _, err := s.GetContact(ctx, dana.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Acme's contact is still there: err = %v", err)
	}
	if contacts, _ := s.ListContactsForJob(ctx, globex); len(contacts) != 1 {
		t.Errorf("Globex has %d contacts, want its own kept", len(contacts))
	}
}

// @scenario [#board-persists]
func TestBoardPersistsAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	// @given [features/job-board.feature] I have a job for "Software Engineer" at "Acme" in "Interviewing"
	createJob(t, s, "Acme", "interviewing", 0)

	// @when [features/job-board.feature] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	// @then [features/job-board.feature] the card for "Software Engineer" at "Acme" is in the "Interviewing" column
	rows, err := reopened.ListJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Company != "Acme" || rows[0].Role != "Software Engineer" || rows[0].Status != "interviewing" {
		t.Errorf("after reopening: %+v", rows)
	}
}

// @scenario [#interest-persists]
func TestInterestPersistsAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	ctx := context.Background()
	// @given [features/job-interest.feature] I have a job for "Software Engineer" at "Acme" with an interest rating of 4 stars
	_, err := s.CreateJob(ctx, store.CreateJobParams{
		Company: "Acme", Role: "Software Engineer", Status: "saved", Interest: sql.NullInt64{Int64: 4, Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	// @when [features/job-interest.feature] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	// @then [features/job-interest.feature] the card shows 4 of 5 stars
	rows, err := reopened.ListJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Interest != (sql.NullInt64{Int64: 4, Valid: true}) {
		t.Errorf("after reopening: %+v", rows)
	}
}

func TestUpdateJobDetailsSetsAndClearsInterest(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	id := createJob(t, s, "Acme", "saved", 0)

	for _, want := range []sql.NullInt64{{Int64: 2, Valid: true}, {}} {
		err := s.UpdateJobDetails(ctx, store.UpdateJobDetailsParams{ID: id, Company: "Acme", Role: "Software Engineer", Interest: want})
		if err != nil {
			t.Fatal(err)
		}
		job, err := s.GetJob(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if job.Interest != want {
			t.Errorf("interest = %+v, want %+v", job.Interest, want)
		}
	}
}

// @scenario [#card-order-persists]
func TestCardOrderPersistsAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	// @given [features/job-board.feature] the "Saved" column holds the jobs at "Initech", "Acme", and "Globex" in that order
	// Inserted out of order, so the order can only come from the stored positions.
	createJob(t, s, "Globex", "saved", 2)
	createJob(t, s, "Initech", "saved", 0)
	createJob(t, s, "Acme", "saved", 1)

	// @when [features/job-board.feature] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
	if got, want := companiesIn(t, reopened, "saved"), []string{"Initech", "Acme", "Globex"}; !slices.Equal(got, want) {
		t.Errorf("Saved = %v, want %v", got, want)
	}
}

// The command line's `job delete` calls the jobs service's DeleteJob, which deletes the job's row
// and its documents' stored copies (TestDeleteJob in internal/jobs); the rest go with the row.
// @scenario [#cli-delete-job]
func TestDeleteJobDeletesEverythingOnIt(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "applied", 0)
	createContact(t, s, acme, "Dana Smith", "recruiter")
	createInterview(t, s, acme, "2026-10-14", "", "technical")
	if _, err := s.CreateDocument(ctx, store.CreateDocumentParams{JobID: acme, Kind: "resume", FileName: "r.pdf", StoredPath: "documents/r.pdf"}); err != nil {
		t.Fatal(err)
	}
	addStatusChange(t, s, acme, "applied", "2026-10-09")

	if err := s.InTx(ctx, func(q store.Querier) error { return q.DeleteJob(ctx, acme) }); err != nil {
		t.Fatal(err)
	}

	// @and [features/agent-cli-jobs.feature] its contacts, interviews, attached documents, and status history are deleted with it
	for _, table := range []string{"contacts", "interviews", "documents", "status_changes"} {
		var left int
		if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE job_id = ?`, acme).Scan(&left); err != nil {
			t.Fatal(err)
		}
		if left != 0 {
			t.Errorf("%d rows in %s outlived the job", left, table)
		}
	}
}
