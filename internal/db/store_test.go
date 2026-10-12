package db

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

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

// @scenario [features/job-documents.feature#card-shows-document-count] The card shows that documents are attached
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

// @scenario [features/job-board.feature#board-persists] Board is the same after reopening the app
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

// @scenario [features/job-interest.feature#interest-persists] The interest rating is the same after reopening the app
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

// @scenario [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
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
