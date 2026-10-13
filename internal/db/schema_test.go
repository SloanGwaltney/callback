package db

import (
	"context"
	"database/sql"
	"testing"

	"callback/internal/db/migrations"
	"callback/internal/store"
)

// Storage-tier tests for the real schema in migrations/. See
// docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

func openSchema(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := Open(context.Background(), t.TempDir(), migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func mustExec(t *testing.T, conn *sql.DB, query string, args ...any) sql.Result {
	t.Helper()
	res, err := conn.Exec(query, args...)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return res
}

func TestSchemaAcceptsValidJobs(t *testing.T) {
	conn := openSchema(t)

	mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Acme', 'Software Engineer', 0)`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, status, date_applied) VALUES ('Globex', 'Developer', 0, 'applied', '2026-10-01')`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, status, closing_reason) VALUES ('Initech', 'Engineer', 0, 'closed', 'rejected')`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, status, closing_reason, closing_reason_other) VALUES ('Umbrella', 'Engineer', 1, 'closed', 'other', 'Company went through layoffs')`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, interest) VALUES ('Hooli', 'Engineer', 1, 1)`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, interest) VALUES ('Vandelay', 'Engineer', 2, 5)`)

	var status string
	if err := conn.QueryRow(`SELECT status FROM jobs WHERE company = 'Acme'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "saved" {
		t.Errorf("default status = %q, want %q", status, "saved")
	}

	var interest sql.NullInt64
	if err := conn.QueryRow(`SELECT interest FROM jobs WHERE company = 'Acme'`).Scan(&interest); err != nil {
		t.Fatal(err)
	}
	if interest.Valid {
		t.Errorf("default interest = %d, want NULL", interest.Int64)
	}
}

func TestSchemaRejectsInvalidJobs(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"empty company", `INSERT INTO jobs (company, role, position) VALUES ('  ', 'Engineer', 0)`},
		{"empty role", `INSERT INTO jobs (company, role, position) VALUES ('Acme', '', 0)`},
		{"unknown status", `INSERT INTO jobs (company, role, position, status) VALUES ('Acme', 'Engineer', 0, 'ghosted')`},
		{"negative position", `INSERT INTO jobs (company, role, position) VALUES ('Acme', 'Engineer', -1)`},
		{"malformed date applied", `INSERT INTO jobs (company, role, position, date_applied) VALUES ('Acme', 'Engineer', 0, '10/01/2026')`},
		{"impossible date applied", `INSERT INTO jobs (company, role, position, date_applied) VALUES ('Acme', 'Engineer', 0, '2026-02-30')`},
		{"closed without a reason", `INSERT INTO jobs (company, role, position, status) VALUES ('Acme', 'Engineer', 0, 'closed')`},
		{"reason without being closed", `INSERT INTO jobs (company, role, position, closing_reason) VALUES ('Acme', 'Engineer', 0, 'rejected')`},
		{"unknown closing reason", `INSERT INTO jobs (company, role, position, status, closing_reason) VALUES ('Acme', 'Engineer', 0, 'closed', 'bored')`},
		{"other without its text", `INSERT INTO jobs (company, role, position, status, closing_reason) VALUES ('Acme', 'Engineer', 0, 'closed', 'other')`},
		{"other with blank text", `INSERT INTO jobs (company, role, position, status, closing_reason, closing_reason_other) VALUES ('Acme', 'Engineer', 0, 'closed', 'other', ' ')`},
		{"interest below one star", `INSERT INTO jobs (company, role, position, interest) VALUES ('Acme', 'Engineer', 0, 0)`},
		{"interest above five stars", `INSERT INTO jobs (company, role, position, interest) VALUES ('Acme', 'Engineer', 0, 6)`},
		{"text without other", `INSERT INTO jobs (company, role, position, status, closing_reason, closing_reason_other) VALUES ('Acme', 'Engineer', 0, 'closed', 'rejected', 'layoffs')`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := openSchema(t)
			if _, err := conn.Exec(tt.query); err == nil {
				t.Error("insert succeeded, want a constraint error")
			}
		})
	}
}

func TestSchemaDocuments(t *testing.T) {
	conn := openSchema(t)
	// Foreign keys are set per connection, so keep every statement on one.
	conn.SetMaxOpenConns(1)

	res := mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Acme', 'Software Engineer', 0)`)
	jobID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, conn, `INSERT INTO documents (job_id, kind, file_name, stored_path) VALUES (?, 'resume', 'resume-2026.pdf', 'documents/a.pdf')`, jobID)
	mustExec(t, conn, `INSERT INTO documents (job_id, kind, file_name, stored_path) VALUES (?, 'cover_letter', 'acme-cover.docx', 'documents/b.docx')`, jobID)

	t.Run("rejects an unknown kind", func(t *testing.T) {
		if _, err := conn.Exec(`INSERT INTO documents (job_id, kind, file_name, stored_path) VALUES (?, 'photo', 'me.png', 'documents/c.png')`, jobID); err == nil {
			t.Error("insert succeeded, want a constraint error")
		}
	})

	t.Run("rejects a reused stored path", func(t *testing.T) {
		if _, err := conn.Exec(`INSERT INTO documents (job_id, kind, file_name, stored_path) VALUES (?, 'other', 'copy.pdf', 'documents/a.pdf')`, jobID); err == nil {
			t.Error("insert succeeded, want a unique constraint error")
		}
	})

	t.Run("rejects a missing job", func(t *testing.T) {
		if _, err := conn.Exec(`INSERT INTO documents (job_id, kind, file_name, stored_path) VALUES (999, 'other', 'x.pdf', 'documents/d.pdf')`); err == nil {
			t.Error("insert succeeded, want a foreign key error")
		}
	})

	t.Run("deleting a job deletes its document rows", func(t *testing.T) {
		mustExec(t, conn, `DELETE FROM jobs WHERE id = ?`, jobID)
		// Covers the database rows only; deleting the files is the app's job. See
		// docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
		// @and [features/job-cards.feature] its attached documents are removed too
		var n int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%d document rows left after deleting the job", n)
		}
	})
}

// @scenario [features/job-board.feature#card-order-persists] Card order is the same after reopening the app
func TestListJobsOrdersByPositionWithinStatus(t *testing.T) {
	conn := openSchema(t)

	// @given [features/job-board.feature] the "Saved" column holds the jobs at "Initech", "Acme", and "Globex" in that order
	mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Globex', 'Developer', 2)`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, status) VALUES ('Hooli', 'Engineer', 0, 'applied')`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Initech', 'Engineer', 0)`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Acme', 'Engineer', 1)`)

	jobs, err := store.New(conn).ListJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/job-board.feature] the "Saved" column shows "Initech", "Acme", and "Globex" in that order
	var saved []string
	for _, j := range jobs {
		if j.Status == "saved" {
			saved = append(saved, j.Company)
		}
	}
	want := []string{"Initech", "Acme", "Globex"}
	if len(jobs) != 4 || len(saved) != 3 || saved[0] != want[0] || saved[1] != want[1] || saved[2] != want[2] {
		t.Errorf("saved column = %v (of %d jobs), want %v", saved, len(jobs), want)
	}
}
