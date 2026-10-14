package db

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"callback/internal/store"
)

// Storage-tier tests for the status_changes table and its queries.

func addStatusChange(t *testing.T, q store.Querier, jobID int64, status, date string) {
	t.Helper()
	arg := store.AddStatusChangeParams{JobID: jobID, Status: status, Date: date}
	if status == "closed" {
		arg.ClosingReason = sql.NullString{String: "no_response", Valid: true}
	}
	if err := q.AddStatusChange(context.Background(), arg); err != nil {
		t.Fatal(err)
	}
}

// historyOf returns a job's status history as "<status> <date>" entries, in the query's order.
func historyOf(t *testing.T, q store.Querier, jobID int64) []string {
	t.Helper()
	changes, err := q.ListStatusChangesForJob(context.Background(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, c := range changes {
		out = append(out, c.Status+" "+c.Date)
	}
	return out
}

func TestSchemaStatusChanges(t *testing.T) {
	conn := openSchema(t)
	// Foreign keys are set per connection, so keep every statement on one.
	conn.SetMaxOpenConns(1)

	res := mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Acme', 'Software Engineer', 0)`)
	jobID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"saved", "applied", "interviewing", "offer"} {
		mustExec(t, conn, `INSERT INTO status_changes (job_id, status, date) VALUES (?, ?, '2026-10-09')`, jobID, status)
	}
	mustExec(t, conn, `INSERT INTO status_changes (job_id, status, date, closing_reason) VALUES (?, 'closed', '2026-10-09', 'rejected')`, jobID)
	mustExec(t, conn, `INSERT INTO status_changes (job_id, status, date, closing_reason, closing_reason_other) VALUES (?, 'closed', '2026-10-09', 'other', 'Company went through layoffs')`, jobID)

	for _, tt := range []struct{ name, query string }{
		{"missing status", `INSERT INTO status_changes (job_id, date) VALUES (?, '2026-10-09')`},
		{"unknown status", `INSERT INTO status_changes (job_id, status, date) VALUES (?, 'ghosted', '2026-10-09')`},
		{"missing date", `INSERT INTO status_changes (job_id, status) VALUES (?, 'saved')`},
		{"malformed date", `INSERT INTO status_changes (job_id, status, date) VALUES (?, 'saved', 'Oct 9')`},
		{"impossible date", `INSERT INTO status_changes (job_id, status, date) VALUES (?, 'saved', '2026-02-30')`},
		{"closed without a reason", `INSERT INTO status_changes (job_id, status, date) VALUES (?, 'closed', '2026-10-09')`},
		{"a reason when not closed", `INSERT INTO status_changes (job_id, status, date, closing_reason) VALUES (?, 'applied', '2026-10-09', 'rejected')`},
		{"unknown reason", `INSERT INTO status_changes (job_id, status, date, closing_reason) VALUES (?, 'closed', '2026-10-09', 'ghosted')`},
		{"other without text", `INSERT INTO status_changes (job_id, status, date, closing_reason) VALUES (?, 'closed', '2026-10-09', 'other')`},
		{"other with blank text", `INSERT INTO status_changes (job_id, status, date, closing_reason, closing_reason_other) VALUES (?, 'closed', '2026-10-09', 'other', '  ')`},
		{"text without other", `INSERT INTO status_changes (job_id, status, date, closing_reason, closing_reason_other) VALUES (?, 'closed', '2026-10-09', 'rejected', 'Layoffs')`},
	} {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			if _, err := conn.Exec(tt.query, jobID); err == nil {
				t.Error("insert succeeded, want a constraint error")
			}
		})
	}

	t.Run("rejects a missing job", func(t *testing.T) {
		if _, err := conn.Exec(`INSERT INTO status_changes (job_id, status, date) VALUES (999, 'saved', '2026-10-09')`); err == nil {
			t.Error("insert succeeded, want a foreign key error")
		}
	})
}

func TestStatusChangeQueries(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	acme := createJob(t, s, "Acme", "applied", 0)

	err := s.AddStatusChange(ctx, store.AddStatusChangeParams{
		JobID: acme, Status: "closed", Date: "2026-10-09",
		ClosingReason:      sql.NullString{String: "other", Valid: true},
		ClosingReasonOther: sql.NullString{String: "Company went through layoffs", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := s.ListStatusChangesForJob(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("history = %+v, want one entry", changes)
	}
	c := changes[0]
	if c.JobID != acme || c.Status != "closed" || c.Date != "2026-10-09" ||
		c.ClosingReason.String != "other" || c.ClosingReasonOther.String != "Company went through layoffs" {
		t.Errorf("entry = %+v", c)
	}
}

// @scenario [#history-oldest-first] [#history-same-day-order]
func TestListStatusChangesForJobIsInTheOrderTheyHappened(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	acme := createJob(t, s, "Acme", "interviewing", 0)
	globex := createJob(t, s, "Globex", "saved", 0)
	addStatusChange(t, s, acme, "saved", "2026-10-01")
	addStatusChange(t, s, globex, "saved", "2026-10-01")
	addStatusChange(t, s, acme, "applied", "2026-10-09")
	addStatusChange(t, s, acme, "interviewing", "2026-10-09")

	// Several on the same day keep the order they were added in, not alphabetical or any other.
	got := historyOf(t, s, acme)
	if want := []string{"saved 2026-10-01", "applied 2026-10-09", "interviewing 2026-10-09"}; !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}
}

func TestListCurrentStatusDatesTakesEachJobsLatestChange(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	acme := createJob(t, s, "Acme", "applied", 0)
	globex := createJob(t, s, "Globex", "saved", 0)
	createJob(t, s, "Initech", "saved", 1)
	addStatusChange(t, s, acme, "saved", "2026-10-01")
	addStatusChange(t, s, globex, "saved", "2026-10-06")
	// A later change on an earlier date still counts as the latest: rows go in the order they
	// happened.
	addStatusChange(t, s, acme, "applied", "2026-10-03")

	rows, err := s.ListCurrentStatusDates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64]string{}
	for _, r := range rows {
		got[r.JobID] = r.Date
	}
	want := map[int64]string{acme: "2026-10-03", globex: "2026-10-06"}
	if len(got) != len(want) || got[acme] != want[acme] || got[globex] != want[globex] {
		t.Errorf("current status dates = %v, want %v (Initech has no history)", got, want)
	}
}

// @scenario [#history-persists]
func TestStatusHistoryPersistsAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	// @given [features/job-status-history.feature] I have a job for "Software Engineer" at "Acme" that moved from "Saved" to "Applied"
	acme := createJob(t, s, "Acme", "applied", 0)
	addStatusChange(t, s, acme, "saved", "2026-10-01")
	addStatusChange(t, s, acme, "applied", "2026-10-09")
	before := historyOf(t, s, acme)

	// @when [features/job-status-history.feature] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	// @when [features/job-status-history.feature] I open the card
	got := historyOf(t, reopened, acme)

	// @then [features/job-status-history.feature] the status history shows "Saved" and "Applied" with the same dates as before
	if want := []string{"saved 2026-10-01", "applied 2026-10-09"}; !slices.Equal(got, want) || !slices.Equal(got, before) {
		t.Errorf("after reopening: %v, want %v", got, want)
	}
}

// @scenario [#history-deleted-with-job]
func TestDeleteJobDeletesItsStatusChangeRows(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	// @given [features/job-status-history.feature#history-deleted-with-job] I have a job for "Software Engineer" at "Acme" with a status history
	acme := createJob(t, s, "Acme", "applied", 0)
	addStatusChange(t, s, acme, "saved", "2026-10-01")
	addStatusChange(t, s, acme, "applied", "2026-10-09")
	globex := createJob(t, s, "Globex", "saved", 0)
	addStatusChange(t, s, globex, "saved", "2026-10-01")

	// @when [features/job-status-history.feature] I delete the job
	err := s.InTx(ctx, func(q store.Querier) error { return q.DeleteJob(ctx, acme) })
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-status-history.feature] its status history is removed too
	var left int
	if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM status_changes WHERE job_id = ?`, acme).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Errorf("%d status changes outlived the job", left)
	}
	if got := historyOf(t, s, globex); len(got) != 1 {
		t.Errorf("Globex's history = %v, want its one entry", got)
	}
}

func TestMigrationStartsHistoryForExistingJobs(t *testing.T) {
	dataDir := t.TempDir()
	ctx := context.Background()
	old := openBefore(t, dataDir, "00007")
	mustExec(t, old, `INSERT INTO jobs (company, role, position, status) VALUES ('Acme', 'Software Engineer', 0, 'applied')`)
	mustExec(t, old, `INSERT INTO jobs (company, role, position, status, closing_reason, closing_reason_other) VALUES ('Globex', 'Developer', 0, 'closed', 'other', 'Layoffs')`)
	old.Close()

	s, _ := openStore(t, dataDir)
	today := time.Now().Format("2006-01-02")
	changes, err := s.ListStatusChangesForJob(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Status != "closed" || changes[0].ClosingReasonOther.String != "Layoffs" {
		t.Errorf("Globex's history = %+v, want Closed with its reason", changes)
	}
	// The migration dates the entry the day it runs, which is today unless the test straddles
	// midnight.
	if got := historyOf(t, s, 1); !slices.Equal(got, []string{"applied " + today}) {
		t.Errorf("Acme's history = %v, want [applied %s]", got, today)
	}
}
