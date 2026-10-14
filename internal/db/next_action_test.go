package db

import (
	"context"
	"database/sql"
	"testing"

	"callback/internal/store"
)

// Storage-tier tests for a job's next action: its columns on jobs and their queries.

func TestSchemaNextAction(t *testing.T) {
	conn := openSchema(t)

	mustExec(t, conn, `INSERT INTO jobs (company, role, position) VALUES ('Acme', 'Software Engineer', 0)`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, next_action) VALUES ('Globex', 'Engineer', 1, 'Send a thank-you email')`)
	mustExec(t, conn, `INSERT INTO jobs (company, role, position, next_action, next_action_date) VALUES ('Initech', 'Engineer', 2, 'Waiting for a call back', '2026-10-14')`)

	var action, date sql.NullString
	if err := conn.QueryRow(`SELECT next_action, next_action_date FROM jobs WHERE company = 'Acme'`).Scan(&action, &date); err != nil {
		t.Fatal(err)
	}
	if action.Valid || date.Valid {
		t.Errorf("default next action = %+v %+v, want NULL", action, date)
	}

	for _, tt := range []struct{ name, query string }{
		{"blank next action", `INSERT INTO jobs (company, role, position, next_action) VALUES ('Acme', 'Engineer', 0, '  ')`},
		{"date without a next action", `INSERT INTO jobs (company, role, position, next_action_date) VALUES ('Acme', 'Engineer', 0, '2026-10-14')`},
		{"malformed date", `INSERT INTO jobs (company, role, position, next_action, next_action_date) VALUES ('Acme', 'Engineer', 0, 'Call', '10/14/2026')`},
		{"impossible date", `INSERT INTO jobs (company, role, position, next_action, next_action_date) VALUES ('Acme', 'Engineer', 0, 'Call', '2026-02-30')`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := conn.Exec(tt.query); err == nil {
				t.Errorf("schema accepted %s", tt.name)
			}
		})
	}
}

func nextActionOf(t *testing.T, q store.Querier, id int64) (sql.NullString, sql.NullString) {
	t.Helper()
	job, err := q.GetJob(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return job.NextAction, job.NextActionDate
}

func TestSetAndClearNextAction(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	id := createJob(t, s, "Acme", "applied", 0)
	other := createJob(t, s, "Globex", "applied", 1)

	set := store.SetNextActionParams{
		ID:             id,
		NextAction:     sql.NullString{String: "Waiting for a call back", Valid: true},
		NextActionDate: sql.NullString{String: "2026-10-14", Valid: true},
	}
	if err := s.SetNextAction(ctx, set); err != nil {
		t.Fatal(err)
	}
	if action, date := nextActionOf(t, s, id); action != set.NextAction || date != set.NextActionDate {
		t.Errorf("after setting: %+v %+v", action, date)
	}
	if action, _ := nextActionOf(t, s, other); action.Valid {
		t.Errorf("another job's next action changed to %+v", action)
	}

	// Setting it again with no date clears only the date.
	set.NextActionDate = sql.NullString{}
	if err := s.SetNextAction(ctx, set); err != nil {
		t.Fatal(err)
	}
	if action, date := nextActionOf(t, s, id); action != set.NextAction || date.Valid {
		t.Errorf("after clearing the date: %+v %+v", action, date)
	}

	if err := s.ClearNextAction(ctx, id); err != nil {
		t.Fatal(err)
	}
	if action, date := nextActionOf(t, s, id); action.Valid || date.Valid {
		t.Errorf("after clearing: %+v %+v", action, date)
	}
}

func TestJobUpdatesKeepTheNextAction(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	ctx := context.Background()
	id := createJob(t, s, "Acme", "interviewing", 0)
	want := store.SetNextActionParams{
		ID:             id,
		NextAction:     sql.NullString{String: "Waiting for a call back", Valid: true},
		NextActionDate: sql.NullString{String: "2026-10-14", Valid: true},
	}
	if err := s.SetNextAction(ctx, want); err != nil {
		t.Fatal(err)
	}

	if err := s.UpdateJobDetails(ctx, store.UpdateJobDetailsParams{ID: id, Company: "Acme", Role: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	err := s.UpdateJobPlacement(ctx, store.UpdateJobPlacementParams{
		ID: id, Status: "closed", ClosingReason: sql.NullString{String: "rejected", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetJobPosition(ctx, store.SetJobPositionParams{ID: id, Position: 3}); err != nil {
		t.Fatal(err)
	}

	if action, date := nextActionOf(t, s, id); action != want.NextAction || date != want.NextActionDate {
		t.Errorf("after editing and moving: %+v %+v", action, date)
	}
}

// @scenario [#next-action-persists]
func TestNextActionPersistsAcrossReopen(t *testing.T) {
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	ctx := context.Background()
	// @given [features/job-next-action.feature#next-action-persists] I have a job for "Software Engineer" at "Acme" with the next action "Waiting for a call back" expected on 2026-10-14
	id := createJob(t, s, "Acme", "applied", 0)
	err := s.SetNextAction(ctx, store.SetNextActionParams{
		ID:             id,
		NextAction:     sql.NullString{String: "Waiting for a call back", Valid: true},
		NextActionDate: sql.NullString{String: "2026-10-14", Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	// @when [features/job-next-action.feature#next-action-persists] I close and reopen the app
	conn.Close()
	reopened, _ := openStore(t, dataDir)

	rows, err := reopened.ListJobs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("after reopening: %+v", rows)
	}
	// @then [features/job-next-action.feature#next-action-persists] the job's next action is "Waiting for a call back"
	if rows[0].NextAction != (sql.NullString{String: "Waiting for a call back", Valid: true}) {
		t.Errorf("next action = %+v", rows[0].NextAction)
	}
	// @and [features/job-next-action.feature#next-action-persists] its expected date is 2026-10-14
	if rows[0].NextActionDate != (sql.NullString{String: "2026-10-14", Valid: true}) {
		t.Errorf("expected date = %+v", rows[0].NextActionDate)
	}
}
